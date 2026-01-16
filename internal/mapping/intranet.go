package mapping

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"time"
)

type Strategy string

const (
	RoundRobin     Strategy = "round_robin"
	Random         Strategy = "random"
	FirstAvailable Strategy = "first_available"
)

type Mapping struct {
	Type     string   `json:"type"`     // "single" or "loadbalance"
	IP       string   `json:"ip"`       // For single type
	IPs      []string `json:"ips"`      // For loadbalance type
	Strategy Strategy `json:"strategy"` // Load balancing strategy
}

type failedIP struct {
	failedAt time.Time
	domain   string
}

type IntranetMapper struct {
	mu           sync.RWMutex
	mappings     map[string]*Mapping
	currentIndex map[string]int
	failedIPs    map[string]failedIP // key: "domain:ip"
	configFile   string
}

func New(configFile string) (*IntranetMapper, error) {
	m := &IntranetMapper{
		mappings:     make(map[string]*Mapping),
		currentIndex: make(map[string]int),
		failedIPs:    make(map[string]failedIP),
		configFile:   configFile,
	}

	if err := m.Reload(); err != nil {
		return nil, err
	}

	return m, nil
}

// Reload reads mappings from the config file
func (m *IntranetMapper) Reload() error {
	data, err := os.ReadFile(m.configFile)
	if err != nil {
		return err
	}

	var mappings map[string]*Mapping
	if err := json.Unmarshal(data, &mappings); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.mappings = mappings
	// Reset indices
	m.currentIndex = make(map[string]int)
	for domain := range mappings {
		m.currentIndex[domain] = 0
	}

	log.Printf("Loaded %d intranet mappings from %s", len(mappings), m.configFile)
	return nil
}

// RewriteURL replaces the domain with mapped IP if available
func (m *IntranetMapper) RewriteURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	ip := m.getMapping(parsedURL.Hostname())
	if ip == "" {
		return rawURL
	}

	parsedURL.Host = ip
	if parsedURL.Port() != "" {
		parsedURL.Host = ip + ":" + parsedURL.Port()
	}

	return parsedURL.String()
}

// GetOriginalHost returns the original hostname for setting Host header
func (m *IntranetMapper) GetOriginalHost(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsedURL.Hostname()
}

// GetMappings returns a copy of all mappings
func (m *IntranetMapper) GetMappings() map[string]*Mapping {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*Mapping, len(m.mappings))
	for k, v := range m.mappings {
		copyMapping := *v
		result[k] = &copyMapping
	}
	return result
}

// MarkIPFailed marks an IP as failed for a domain
func (m *IntranetMapper) MarkIPFailed(ip, domain string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := domain + ":" + ip
	m.failedIPs[key] = failedIP{
		failedAt: time.Now(),
		domain:   domain,
	}
	log.Printf("Marked IP as failed: %s for domain: %s", ip, domain)
}

func (m *IntranetMapper) getMapping(domain string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	mapping, ok := m.mappings[domain]
	if !ok {
		return ""
	}

	if mapping.Type == "single" {
		return mapping.IP
	}

	// loadbalance type
	return m.getLoadBalancedIP(domain, mapping)
}

func (m *IntranetMapper) getLoadBalancedIP(domain string, mapping *Mapping) string {
	if len(mapping.IPs) == 0 {
		return ""
	}

	// Filter out failed IPs
	availableIPs := m.filterAvailableIPs(mapping.IPs, domain)
	if len(availableIPs) == 0 {
		// All IPs failed, clear failures and use original list
		m.clearFailedIPs(domain)
		availableIPs = mapping.IPs
	}

	strategy := mapping.Strategy
	if strategy == "" {
		strategy = RoundRobin
	}

	switch strategy {
	case RoundRobin:
		idx := m.currentIndex[domain] % len(availableIPs)
		m.currentIndex[domain] = (idx + 1) % len(availableIPs)
		return availableIPs[idx]
	case Random:
		return availableIPs[rand.Intn(len(availableIPs))]
	case FirstAvailable:
		return availableIPs[0]
	default:
		return availableIPs[0]
	}
}

func (m *IntranetMapper) filterAvailableIPs(ips []string, domain string) []string {
	available := make([]string, 0, len(ips))
	now := time.Now()

	for _, ip := range ips {
		key := domain + ":" + ip
		if failed, ok := m.failedIPs[key]; ok {
			// Auto-recover after 5 minutes
			if now.Sub(failed.failedAt) > 5*time.Minute {
				delete(m.failedIPs, key)
				available = append(available, ip)
			}
		} else {
			available = append(available, ip)
		}
	}

	return available
}

func (m *IntranetMapper) clearFailedIPs(domain string) {
	for key, failed := range m.failedIPs {
		if failed.domain == domain {
			delete(m.failedIPs, key)
		}
	}
	log.Printf("Cleared failed IPs for domain: %s", domain)
}

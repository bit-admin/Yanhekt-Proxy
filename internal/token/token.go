package token

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	cacheTTL = 10 * time.Second
)

type cacheEntry struct {
	videoToken string
	fetchedAt  time.Time
}

type TokenCache struct {
	mu          sync.RWMutex
	cache       map[string]cacheEntry
	upstreamAPI string
	magicKey    string
	httpClient  *http.Client
}

func NewCache(upstreamAPI, magicKey string) *TokenCache {
	return &TokenCache{
		cache:       make(map[string]cacheEntry),
		upstreamAPI: upstreamAPI,
		magicKey:    magicKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetVideoToken returns a cached video token or fetches a new one
func (tc *TokenCache) GetVideoToken(loginToken string) (string, error) {
	tc.mu.RLock()
	entry, ok := tc.cache[loginToken]
	tc.mu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < cacheTTL {
		return entry.videoToken, nil
	}

	// Fetch new token
	videoToken, err := tc.fetchVideoToken(loginToken)
	if err != nil {
		return "", err
	}

	// Update cache
	tc.mu.Lock()
	tc.cache[loginToken] = cacheEntry{
		videoToken: videoToken,
		fetchedAt:  time.Now(),
	}
	tc.mu.Unlock()

	return videoToken, nil
}

// InvalidateToken removes a token from cache (used on 403 errors)
func (tc *TokenCache) InvalidateToken(loginToken string) {
	tc.mu.Lock()
	delete(tc.cache, loginToken)
	tc.mu.Unlock()
}

func (tc *TokenCache) fetchVideoToken(loginToken string) (string, error) {
	url := tc.upstreamAPI + "/v1/auth/video/token?id=0"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set headers matching the Electron app
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signature := tc.md5Hash(tc.magicKey + "_v1_undefined")

	req.Header.Set("Origin", "https://www.yanhekt.cn")
	req.Header.Set("Referer", "https://www.yanhekt.cn/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.3")
	req.Header.Set("xdomain-client", "web_user")
	req.Header.Set("Xdomain-Client", "web_user")
	req.Header.Set("Xclient-Version", "v1")
	req.Header.Set("Xclient-Signature", signature)
	req.Header.Set("Xclient-Timestamp", timestamp)
	req.Header.Set("Authorization", "Bearer "+loginToken)

	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Code    interface{} `json:"code"`
		Message string      `json:"message"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for success (code can be 0 or "0")
	codeOK := false
	switch v := result.Code.(type) {
	case float64:
		codeOK = v == 0
	case string:
		codeOK = v == "0"
	}

	if !codeOK {
		return "", fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data.Token, nil
}

func (tc *TokenCache) md5Hash(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

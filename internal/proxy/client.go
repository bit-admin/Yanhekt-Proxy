package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/autoslides/video-proxy/internal/mapping"
)

const (
	maxRetries = 3
)

var baseHeaders = map[string]string{
	"Origin":     "https://www.yanhekt.cn",
	"Referer":    "https://www.yanhekt.cn/",
	"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.3",
}

type Client struct {
	externalClient *http.Client
	intranetClient *http.Client
	mapper         *mapping.IntranetMapper
}

func NewClient(externalTimeout, intranetTimeout time.Duration, mapper *mapping.IntranetMapper) *Client {
	return &Client{
		externalClient: &http.Client{
			Timeout: externalTimeout,
		},
		intranetClient: &http.Client{
			Timeout: intranetTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // Required for intranet IPs
				},
			},
		},
		mapper: mapper,
	}
}

// FetchM3U8 fetches M3U8 content from the given URL
func (c *Client) FetchM3U8(url string, isIntranet bool, originalHost string) ([]byte, error) {
	client := c.externalClient
	requestURL := url

	if isIntranet && c.mapper != nil {
		client = c.intranetClient
		requestURL = c.mapper.RewriteURL(url)
	}

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	c.setHeaders(req, originalHost, isIntranet)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("M3U8 request failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// ProxyTS streams TS content directly to the response writer
func (c *Client) ProxyTS(url string, w http.ResponseWriter, isIntranet bool, originalHost string) error {
	client := c.externalClient
	requestURL := url

	if isIntranet && c.mapper != nil {
		client = c.intranetClient
		requestURL = c.mapper.RewriteURL(url)
	}

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return err
	}

	c.setHeaders(req, originalHost, isIntranet)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("TS request failed with status %d", resp.StatusCode)
	}

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	return err
}

// FetchM3U8WithRetry fetches M3U8 with retry logic for 403 errors
// The retryFunc is called on 403 to allow token refresh
func (c *Client) FetchM3U8WithRetry(
	getURL func() string,
	isIntranet bool,
	originalHost string,
	onRetry func(attempt int) error,
) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		url := getURL()

		client := c.externalClient
		requestURL := url

		if isIntranet && c.mapper != nil {
			client = c.intranetClient
			requestURL = c.mapper.RewriteURL(url)
		}

		req, err := http.NewRequest("GET", requestURL, nil)
		if err != nil {
			return nil, err
		}

		c.setHeaders(req, originalHost, isIntranet)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				if onRetry != nil {
					onRetry(attempt)
				}
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		if resp.StatusCode == http.StatusForbidden && attempt < maxRetries {
			lastErr = fmt.Errorf("M3U8 request got 403")
			if onRetry != nil {
				if err := onRetry(attempt); err != nil {
					return nil, err
				}
			}
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		return nil, fmt.Errorf("M3U8 request failed with status %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("M3U8 request failed after %d retries: %w", maxRetries, lastErr)
}

// ProxyTSWithRetry streams TS with retry logic for 403 errors
func (c *Client) ProxyTSWithRetry(
	getURL func() string,
	w http.ResponseWriter,
	isIntranet bool,
	originalHost string,
	onRetry func(attempt int) error,
) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		url := getURL()

		client := c.externalClient
		requestURL := url

		if isIntranet && c.mapper != nil {
			client = c.intranetClient
			requestURL = c.mapper.RewriteURL(url)
		}

		req, err := http.NewRequest("GET", requestURL, nil)
		if err != nil {
			return err
		}

		c.setHeaders(req, originalHost, isIntranet)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				if onRetry != nil {
					onRetry(attempt)
				}
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return err
		}

		if resp.StatusCode == http.StatusOK {
			// Copy response headers
			for key, values := range resp.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(resp.StatusCode)
			_, err = io.Copy(w, resp.Body)
			resp.Body.Close()
			return err
		}

		resp.Body.Close()

		if resp.StatusCode == http.StatusForbidden && attempt < maxRetries {
			lastErr = fmt.Errorf("TS request got 403")
			if onRetry != nil {
				if err := onRetry(attempt); err != nil {
					return err
				}
			}
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		return fmt.Errorf("TS request failed with status %d", resp.StatusCode)
	}

	return fmt.Errorf("TS request failed after %d retries: %w", maxRetries, lastErr)
}

func (c *Client) setHeaders(req *http.Request, originalHost string, isIntranet bool) {
	for key, value := range baseHeaders {
		req.Header.Set(key, value)
	}

	if isIntranet && originalHost != "" {
		req.Host = originalHost
		req.Header.Set("Host", originalHost)
	}
}

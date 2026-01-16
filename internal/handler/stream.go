package handler

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/autoslides/video-proxy/internal/crypto"
	"github.com/autoslides/video-proxy/internal/proxy"
	"github.com/autoslides/video-proxy/internal/token"
)

type StreamHandler struct {
	crypto     *crypto.Crypto
	tokenCache *token.TokenCache
	client     *proxy.Client
	videoHost  string
	serverHost string // The proxy server's host for rewriting URLs
}

func NewStreamHandler(
	crypto *crypto.Crypto,
	tokenCache *token.TokenCache,
	client *proxy.Client,
	videoHost string,
) *StreamHandler {
	return &StreamHandler{
		crypto:     crypto,
		tokenCache: tokenCache,
		client:     client,
		videoHost:  videoHost,
	}
}

// SetServerHost sets the proxy server's host for URL rewriting
func (h *StreamHandler) SetServerHost(host string) {
	h.serverHost = host
}

// ServeHTTP handles both /external/stream and /intranet/stream
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Determine mode from path
	isIntranet := strings.HasPrefix(r.URL.Path, "/intranet/")

	// Parse query parameters
	originalURL := r.URL.Query().Get("url")
	loginToken := r.URL.Query().Get("token")

	if originalURL == "" || loginToken == "" {
		http.Error(w, "Missing required parameters: url and token", http.StatusBadRequest)
		return
	}

	// Fix URL escaping
	originalURL = strings.ReplaceAll(originalURL, "\\/", "/")

	// Get video token (cached for 10s)
	videoToken, err := h.tokenCache.GetVideoToken(loginToken)
	if err != nil {
		log.Printf("Failed to get video token: %v", err)
		http.Error(w, "Failed to get video token", http.StatusInternalServerError)
		return
	}

	// Build signed URL function (for retry with fresh signature)
	buildSignedURL := func() string {
		encryptedURL := h.crypto.EncryptURL(originalURL)
		return h.crypto.SignURL(encryptedURL, videoToken)
	}

	// Fetch M3U8 with retry logic
	content, err := h.client.FetchM3U8WithRetry(
		buildSignedURL,
		isIntranet,
		h.videoHost,
		func(attempt int) error {
			log.Printf("M3U8 request retry %d, refreshing token", attempt+1)
			// Invalidate and refresh token
			h.tokenCache.InvalidateToken(loginToken)
			newToken, err := h.tokenCache.GetVideoToken(loginToken)
			if err != nil {
				return err
			}
			videoToken = newToken
			return nil
		},
	)

	if err != nil {
		log.Printf("Failed to fetch M3U8: %v", err)
		http.Error(w, "Failed to fetch M3U8", http.StatusBadGateway)
		return
	}

	// Rewrite TS URLs in M3U8 content
	rewrittenContent := h.rewriteM3U8Content(string(content), originalURL, loginToken, isIntranet, r)

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(rewrittenContent))
}

// rewriteM3U8Content rewrites TS segment URLs to point to our proxy
func (h *StreamHandler) rewriteM3U8Content(content, baseURL, loginToken string, isIntranet bool, r *http.Request) string {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))

	// Determine server host for proxy URLs
	serverHost := h.serverHost
	if serverHost == "" {
		serverHost = r.Host
	}

	// Determine scheme
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwdProto := r.Header.Get("X-Forwarded-Proto"); fwdProto != "" {
		scheme = fwdProto
	}

	// Path prefix based on mode
	pathPrefix := "/external/ts/"
	if isIntranet {
		pathPrefix = "/intranet/ts/"
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			result = append(result, line)
			continue
		}

		// This is a TS file reference
		tsFileName := trimmed
		proxyURL := fmt.Sprintf("%s://%s%s%s?base=%s&token=%s",
			scheme,
			serverHost,
			pathPrefix,
			url.PathEscape(tsFileName),
			url.QueryEscape(baseURL),
			url.QueryEscape(loginToken),
		)
		result = append(result, proxyURL)
	}

	return strings.Join(result, "\n")
}

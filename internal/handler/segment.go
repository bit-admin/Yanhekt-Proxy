package handler

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/autoslides/video-proxy/internal/crypto"
	"github.com/autoslides/video-proxy/internal/proxy"
	"github.com/autoslides/video-proxy/internal/token"
)

type SegmentHandler struct {
	crypto     *crypto.Crypto
	tokenCache *token.TokenCache
	client     *proxy.Client
	videoHost  string
}

func NewSegmentHandler(
	crypto *crypto.Crypto,
	tokenCache *token.TokenCache,
	client *proxy.Client,
	videoHost string,
) *SegmentHandler {
	return &SegmentHandler{
		crypto:     crypto,
		tokenCache: tokenCache,
		client:     client,
		videoHost:  videoHost,
	}
}

// ServeHTTP handles both /external/ts/{path} and /intranet/ts/{path}
func (h *SegmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Determine mode from path
	isIntranet := strings.HasPrefix(r.URL.Path, "/intranet/")

	// Extract TS filename from path
	var tsFileName string
	if isIntranet {
		tsFileName = strings.TrimPrefix(r.URL.Path, "/intranet/ts/")
	} else {
		tsFileName = strings.TrimPrefix(r.URL.Path, "/external/ts/")
	}

	// URL decode the filename
	tsFileName, err := url.PathUnescape(tsFileName)
	if err != nil {
		http.Error(w, "Invalid TS filename", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	baseURL := r.URL.Query().Get("base")
	loginToken := r.URL.Query().Get("token")

	if baseURL == "" || loginToken == "" {
		http.Error(w, "Missing required parameters: base and token", http.StatusBadRequest)
		return
	}

	// Build full TS URL
	tsURL := h.resolveURL(baseURL, tsFileName)

	// Get video token
	videoToken, err := h.tokenCache.GetVideoToken(loginToken)
	if err != nil {
		log.Printf("Failed to get video token: %v", err)
		http.Error(w, "Failed to get video token", http.StatusInternalServerError)
		return
	}

	// Build signed URL function (for retry with fresh signature)
	buildSignedURL := func() string {
		encryptedURL := h.crypto.EncryptURL(tsURL)
		return h.crypto.SignURL(encryptedURL, videoToken)
	}

	// Proxy TS with retry logic
	err = h.client.ProxyTSWithRetry(
		buildSignedURL,
		w,
		isIntranet,
		h.videoHost,
		func(attempt int) error {
			log.Printf("TS request retry %d, refreshing token", attempt+1)
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
		log.Printf("Failed to proxy TS: %v", err)
		// Only write error if headers haven't been sent
		// (the proxy client might have already started writing)
	}
}

// resolveURL resolves a relative URL against a base URL
func (h *SegmentHandler) resolveURL(base, relative string) string {
	if strings.HasPrefix(relative, "http") {
		return relative
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return relative
	}

	if strings.HasPrefix(relative, "/") {
		return baseURL.Scheme + "://" + baseURL.Host + relative
	}

	// Relative path - append to base directory
	basePath := baseURL.Path
	lastSlash := strings.LastIndex(basePath, "/")
	if lastSlash >= 0 {
		basePath = basePath[:lastSlash+1]
	}

	return baseURL.Scheme + "://" + baseURL.Host + basePath + relative
}

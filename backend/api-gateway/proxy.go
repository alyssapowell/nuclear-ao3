package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// SECURITY HELPERS
// =============================================================================

// isStaticAsset determines if a request is for a static asset that can be safely cached
// SECURITY POLICY: Only cache static assets (CSS, JS, images). Never cache API content.
func isStaticAsset(path string) bool {
	// Only cache static assets - files with extensions
	staticExtensions := []string{
		".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
		".woff", ".woff2", ".ttf", ".eot", // fonts
		".map", // source maps
	}

	// Check for static file extensions
	for _, ext := range staticExtensions {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return true
		}
	}

	// Also allow caching for specific static paths
	staticPaths := []string{
		"/static/", "/assets/", "/public/",
	}

	for _, staticPath := range staticPaths {
		if strings.HasPrefix(path, staticPath) {
			return true
		}
	}

	// NEVER cache any API endpoints or dynamic content
	// All /api/* paths are dynamic and should never be server-cached
	return false
}

// =============================================================================
// SERVICE PROXY HANDLERS
// =============================================================================

// ProxyToAuth forwards requests to the auth service
func (gw *APIGateway) ProxyToAuth(c *gin.Context) {
	gw.proxyRequest(c, gw.authService, "/api/v1/auth")
}

// ProxyToWork forwards requests to the work service
func (gw *APIGateway) ProxyToWork(c *gin.Context) {
	gw.proxyRequest(c, gw.workService, "/api/v1/works")
}

// ProxyToTag forwards requests to the tag service
func (gw *APIGateway) ProxyToTag(c *gin.Context) {
	gw.proxyRequest(c, gw.tagService, "/api/v1/tags")
}

// ProxyToSearch forwards requests to the search service
func (gw *APIGateway) ProxyToSearch(c *gin.Context) {
	gw.proxyRequest(c, gw.searchService, "/api/v1/search")
}

// proxyRequest handles the actual proxying logic
func (gw *APIGateway) proxyRequest(c *gin.Context, service *ServiceClient, basePath string) {
	start := time.Now()

	// Check service health
	if !service.Health.IsHealthy {
		gw.handleServiceUnavailable(c, service.Name)
		return
	}

	// Build target URL - handle different base paths correctly
	var targetURL string
	requestPath := c.Request.URL.Path

	// For /my, /users, /series, /collections, /bookmarks, /comments, /pseuds routes,
	// we want to preserve the full API path structure
	if strings.HasPrefix(requestPath, "/api/v1/my/") ||
		strings.HasPrefix(requestPath, "/api/v1/users/") ||
		strings.HasPrefix(requestPath, "/api/v1/series/") ||
		strings.HasPrefix(requestPath, "/api/v1/collections/") ||
		strings.HasPrefix(requestPath, "/api/v1/bookmarks/") ||
		strings.HasPrefix(requestPath, "/api/v1/comments/") ||
		strings.HasPrefix(requestPath, "/api/v1/pseuds/") {
		// Use the full request path for these routes
		targetURL = service.BaseURL + requestPath
	} else {
		// Original logic for other routes
		targetPath := strings.TrimPrefix(requestPath, basePath)
		if targetPath == "" {
			targetPath = "/"
		}
		targetURL = service.BaseURL + basePath + targetPath
	}

	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	// SECURITY: Only cache static assets (CSS, JS, images). Never cache API content.
	shouldCache := isStaticAsset(c.Request.URL.Path)

	if c.Request.Method == "GET" && gw.cache != nil && shouldCache {
		cacheKey := gw.cache.GenerateCacheKey("proxy", service.Name, c.Request.URL.Path, c.Request.URL.RawQuery)

		if cached, err := gw.cache.Get(c.Request.Context(), cacheKey); err == nil && cached != nil {
			var response interface{}
			if err := gw.cache.GetJSON(c.Request.Context(), cacheKey, &response); err == nil {
				// Cache hit
				c.Header("X-Cache", "HIT")
				c.JSON(http.StatusOK, response)
				return
			}
		}
	}

	// Read request body
	var bodyBytes []byte
	if c.Request.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}
		c.Request.Body.Close()
	}

	// Create target request
	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create proxy request"})
		return
	}

	// Copy headers (excluding hop-by-hop headers)
	gw.copyHeaders(req.Header, c.Request.Header)

	// Set additional headers
	req.Header.Set("X-Forwarded-For", c.ClientIP())
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-Forwarded-Host", c.Request.Host)
	req.Header.Set("X-Gateway-Request-ID", c.GetHeader("X-Request-ID"))

	// Forward user context if available
	if userID := c.GetHeader("X-User-ID"); userID != "" {
		req.Header.Set("X-User-ID", userID)
	}
	if authHeader := c.GetHeader("Authorization"); authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	// Make the request
	resp, err := service.HTTPClient.Do(req)
	if err != nil {
		gw.handleProxyError(c, service.Name, err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response body"})
		return
	}

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Add cache headers
	c.Header("X-Cache", "MISS")
	c.Header("X-Proxy-Service", service.Name)
	c.Header("X-Response-Time", fmt.Sprintf("%.2fms", float64(time.Since(start).Nanoseconds())/1e6))

	// SECURITY: Only cache static assets (CSS, JS, images). Never cache API content.
	if c.Request.Method == "GET" && gw.cache != nil && isStaticAsset(c.Request.URL.Path) {
		cacheKey := gw.cache.GenerateCacheKey("proxy", service.Name, c.Request.URL.Path, c.Request.URL.RawQuery)

		// Cache static assets for longer periods since they don't change often
		ttl := 1 * time.Hour // Static assets can be cached longer

		gw.cache.Set(c.Request.Context(), cacheKey, respBody, ttl)
	}

	// Return response
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)

	// Record metrics
	if gw.metrics != nil {
		duration := time.Since(start)
		gw.metrics.RecordRequest(c.Request.Method, c.Request.URL.Path, resp.StatusCode, duration)
	}
}

// copyHeaders copies HTTP headers, excluding hop-by-hop headers
func (gw *APIGateway) copyHeaders(dst, src http.Header) {
	// Hop-by-hop headers that shouldn't be forwarded
	hopByHopHeaders := map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Te":                  true,
		"Trailers":            true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
	}

	for key, values := range src {
		if !hopByHopHeaders[key] {
			for _, value := range values {
				dst.Add(key, value)
			}
		}
	}
}

// handleServiceUnavailable returns a service unavailable response
func (gw *APIGateway) handleServiceUnavailable(c *gin.Context, serviceName string) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error":       "Service temporarily unavailable",
		"service":     serviceName,
		"code":        "SERVICE_UNAVAILABLE",
		"message":     fmt.Sprintf("The %s service is currently unavailable. Please try again later.", serviceName),
		"retry_after": 30,
	})

	// Record metrics
	if gw.metrics != nil {
		gw.metrics.RecordRequest(c.Request.Method, c.Request.URL.Path, http.StatusServiceUnavailable, 0)
		gw.metrics.RecordServiceHealth(serviceName, false)
	}
}

// handleProxyError handles errors during proxying
func (gw *APIGateway) handleProxyError(c *gin.Context, serviceName string, err error) {
	// Check if it's a timeout error
	if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
		c.JSON(http.StatusGatewayTimeout, gin.H{
			"error":   "Service timeout",
			"service": serviceName,
			"code":    "SERVICE_TIMEOUT",
			"message": fmt.Sprintf("The %s service took too long to respond.", serviceName),
		})

		if gw.metrics != nil {
			gw.metrics.RecordRequest(c.Request.Method, c.Request.URL.Path, http.StatusGatewayTimeout, 0)
		}
		return
	}

	// Generic proxy error
	c.JSON(http.StatusBadGateway, gin.H{
		"error":   "Proxy error",
		"service": serviceName,
		"code":    "PROXY_ERROR",
		"message": fmt.Sprintf("Failed to communicate with %s service: %v", serviceName, err),
	})

	if gw.metrics != nil {
		gw.metrics.RecordRequest(c.Request.Method, c.Request.URL.Path, http.StatusBadGateway, 0)
		gw.metrics.RecordServiceHealth(serviceName, false)
	}
}

// =============================================================================
// ADVANCED PROXY FEATURES
// =============================================================================

// RoundRobinProxy implements round-robin load balancing for multiple service instances
type RoundRobinProxy struct {
	services []*ServiceClient
	current  int
}

// NewRoundRobinProxy creates a new round-robin proxy
func NewRoundRobinProxy(services []*ServiceClient) *RoundRobinProxy {
	return &RoundRobinProxy{
		services: services,
		current:  0,
	}
}

// NextService returns the next service in round-robin fashion
func (rr *RoundRobinProxy) NextService() *ServiceClient {
	if len(rr.services) == 0 {
		return nil
	}

	// Find next healthy service
	start := rr.current
	for {
		service := rr.services[rr.current]
		rr.current = (rr.current + 1) % len(rr.services)

		if service.Health.IsHealthy {
			return service
		}

		// If we've gone full circle, return the original choice (even if unhealthy)
		if rr.current == start {
			return service
		}
	}
}

// CircuitBreakerProxy implements circuit breaker pattern for resilient proxying
type CircuitBreakerProxy struct {
	service      *ServiceClient
	failureCount int
	lastFailure  time.Time
	state        CircuitState
	threshold    int
	timeout      time.Duration
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewCircuitBreakerProxy creates a new circuit breaker proxy
func NewCircuitBreakerProxy(service *ServiceClient) *CircuitBreakerProxy {
	return &CircuitBreakerProxy{
		service:   service,
		state:     CircuitClosed,
		threshold: 5,                // Open after 5 failures
		timeout:   30 * time.Second, // Try again after 30 seconds
	}
}

// IsAvailable checks if the service is available according to circuit breaker logic
func (cb *CircuitBreakerProxy) IsAvailable() bool {
	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = CircuitHalfOpen
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreakerProxy) RecordSuccess() {
	cb.failureCount = 0
	cb.state = CircuitClosed
}

// RecordFailure records a failed request
func (cb *CircuitBreakerProxy) RecordFailure() {
	cb.failureCount++
	cb.lastFailure = time.Now()

	if cb.state == CircuitHalfOpen {
		cb.state = CircuitOpen
	} else if cb.failureCount >= cb.threshold {
		cb.state = CircuitOpen
	}
}

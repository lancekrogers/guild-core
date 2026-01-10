// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package sandbox

import (
	"context"
	"net"
	"net/url"
	"strings"
	"sync"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/observability"
)

// GuildNetworkFilter implements network access filtering
type GuildNetworkFilter struct {
	allowedHosts []string
	allowedPorts []int
	blockedHosts []string
	logger       observability.Logger
	statsMu      sync.RWMutex
	stats        NetworkStats
}

// NetworkStats tracks network filtering statistics
type NetworkStats struct {
	AllowedRequests int64 `json:"allowed_requests"`
	BlockedRequests int64 `json:"blocked_requests"`
	TotalRequests   int64 `json:"total_requests"`
}

// NewNetworkFilter creates a new network filter
func NewNetworkFilter(ctx context.Context, allowedHosts []string) (NetworkFilter, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("NetworkFilter")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("NetworkFilter").
			WithOperation("NewNetworkFilter")
	}

	filter := &GuildNetworkFilter{
		allowedHosts: make([]string, len(allowedHosts)),
		allowedPorts: []int{80, 443, 22, 3000, 8000, 8080}, // Common development ports
		blockedHosts: []string{
			"169.254.169.254",          // AWS metadata service
			"metadata.google.internal", // GCP metadata service
			"10.0.0.0/8",               // Private networks
			"172.16.0.0/12",            // Private networks
			"192.168.0.0/16",           // Private networks
		},
		logger: logger,
	}

	// Normalize and validate allowed hosts
	for i, host := range allowedHosts {
		normalized, err := filter.normalizeHost(host)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid allowed host").
				WithComponent("NetworkFilter").
				WithOperation("NewNetworkFilter").
				WithDetails("host", host)
		}
		filter.allowedHosts[i] = normalized
	}

	logger.Info("Network filter initialized",
		"allowed_hosts", len(filter.allowedHosts),
		"blocked_hosts", len(filter.blockedHosts),
	)

	return filter, nil
}

// IsHostAllowed checks if a host is allowed for network access
func (nf *GuildNetworkFilter) IsHostAllowed(host string) bool {
	nf.updateStats(func(stats *NetworkStats) {
		stats.TotalRequests++
	})

	// Normalize the host
	normalizedHost, err := nf.normalizeHost(host)
	if err != nil {
		nf.logger.Warn("Invalid host format", "host", host, "error", err)
		nf.updateStats(func(stats *NetworkStats) {
			stats.BlockedRequests++
		})
		return false
	}

	// Check if host is explicitly blocked
	if nf.isBlocked(normalizedHost) {
		nf.logger.Debug("Host blocked by security policy", "host", normalizedHost)
		nf.updateStats(func(stats *NetworkStats) {
			stats.BlockedRequests++
		})
		return false
	}

	// Check if host is in allowed list
	for _, allowedHost := range nf.allowedHosts {
		if nf.matchesHost(normalizedHost, allowedHost) {
			nf.updateStats(func(stats *NetworkStats) {
				stats.AllowedRequests++
			})
			return true
		}
	}

	// Default deny
	nf.logger.Debug("Host not in allowed list", "host", normalizedHost)
	nf.updateStats(func(stats *NetworkStats) {
		stats.BlockedRequests++
	})
	return false
}

// FilterRequest filters an outgoing network request
func (nf *GuildNetworkFilter) FilterRequest(ctx context.Context, req NetworkRequest) error {
	logger := nf.logger.WithOperation("FilterRequest")

	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("NetworkFilter").
			WithOperation("FilterRequest")
	}

	// Validate the request
	if err := nf.validateRequest(req); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid network request").
			WithComponent("NetworkFilter").
			WithOperation("FilterRequest")
	}

	// Check host access
	if !nf.IsHostAllowed(req.Host) {
		logger.Warn("Network request blocked",
			"host", req.Host,
			"method", req.Method,
			"url", req.URL,
		)
		return gerror.New(gerror.ErrCodeSecurityViolation, "network access denied", nil).
			WithComponent("NetworkFilter").
			WithOperation("FilterRequest").
			WithDetails("host", req.Host).
			WithDetails("url", req.URL)
	}

	// Check port if specified
	if req.Port > 0 && !nf.isPortAllowed(req.Port) {
		logger.Warn("Network request blocked - invalid port",
			"host", req.Host,
			"port", req.Port,
		)
		return gerror.New(gerror.ErrCodeSecurityViolation, "port access denied", nil).
			WithComponent("NetworkFilter").
			WithOperation("FilterRequest").
			WithDetails("port", req.Port)
	}

	// Check for suspicious headers
	if err := nf.validateHeaders(req.Headers); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeSecurityViolation, "suspicious request headers").
			WithComponent("NetworkFilter").
			WithOperation("FilterRequest")
	}

	logger.Debug("Network request allowed",
		"host", req.Host,
		"method", req.Method,
		"url", req.URL,
	)

	return nil
}

// GetAllowedHosts returns the list of allowed hosts
func (nf *GuildNetworkFilter) GetAllowedHosts() []string {
	return append([]string(nil), nf.allowedHosts...) // Return a copy
}

// AddAllowedHost adds a host to the allowed list
func (nf *GuildNetworkFilter) AddAllowedHost(host string) error {
	normalized, err := nf.normalizeHost(host)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid host").
			WithComponent("NetworkFilter").
			WithOperation("AddAllowedHost").
			WithDetails("host", host)
	}

	// Check if already exists
	for _, existing := range nf.allowedHosts {
		if existing == normalized {
			return nil // Already exists
		}
	}

	nf.allowedHosts = append(nf.allowedHosts, normalized)
	nf.logger.Info("Host added to allowed list", "host", normalized)

	return nil
}

// RemoveAllowedHost removes a host from the allowed list
func (nf *GuildNetworkFilter) RemoveAllowedHost(host string) error {
	normalized, err := nf.normalizeHost(host)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid host").
			WithComponent("NetworkFilter").
			WithOperation("RemoveAllowedHost").
			WithDetails("host", host)
	}

	for i, existing := range nf.allowedHosts {
		if existing == normalized {
			// Remove from slice
			nf.allowedHosts = append(nf.allowedHosts[:i], nf.allowedHosts[i+1:]...)
			nf.logger.Info("Host removed from allowed list", "host", normalized)
			return nil
		}
	}

	return gerror.New(gerror.ErrCodeNotFound, "host not found in allowed list", nil).
		WithComponent("NetworkFilter").
		WithOperation("RemoveAllowedHost").
		WithDetails("host", normalized)
}

// GetStats returns network filtering statistics
func (nf *GuildNetworkFilter) GetStats() NetworkStats {
	nf.statsMu.RLock()
	defer nf.statsMu.RUnlock()
	return nf.stats
}

// Helper methods

func (nf *GuildNetworkFilter) updateStats(fn func(*NetworkStats)) {
	nf.statsMu.Lock()
	defer nf.statsMu.Unlock()
	fn(&nf.stats)
}

func (nf *GuildNetworkFilter) normalizeHost(host string) (string, error) {
	// Remove protocol if present
	if strings.Contains(host, "://") {
		parsedURL, err := url.Parse(host)
		if err != nil {
			return "", err
		}
		host = parsedURL.Host
	}

	// Remove port if present for comparison
	if strings.Contains(host, ":") {
		host, _, _ = net.SplitHostPort(host)
	}

	// Convert to lowercase
	host = strings.ToLower(strings.TrimSpace(host))

	// Validate host format
	if host == "" {
		return "", gerror.New(gerror.ErrCodeValidation, "empty host", nil)
	}

	// Check if it's a valid IP or hostname
	if net.ParseIP(host) == nil {
		// Not an IP, validate as hostname
		if !nf.isValidHostname(host) {
			return "", gerror.New(gerror.ErrCodeValidation, "invalid hostname format", nil)
		}
	}

	return host, nil
}

func (nf *GuildNetworkFilter) isValidHostname(hostname string) bool {
	// Basic hostname validation
	if len(hostname) > 253 {
		return false
	}

	for _, part := range strings.Split(hostname, ".") {
		if len(part) == 0 || len(part) > 63 {
			return false
		}
		for _, char := range part {
			if !((char >= 'a' && char <= 'z') ||
				(char >= '0' && char <= '9') ||
				char == '-') {
				return false
			}
		}
		if strings.HasPrefix(part, "-") || strings.HasSuffix(part, "-") {
			return false
		}
	}

	return true
}

func (nf *GuildNetworkFilter) isBlocked(host string) bool {
	for _, blocked := range nf.blockedHosts {
		if nf.matchesHost(host, blocked) {
			return true
		}
	}
	return false
}

func (nf *GuildNetworkFilter) matchesHost(host, pattern string) bool {
	// Exact match
	if host == pattern {
		return true
	}

	// Wildcard subdomain match (*.example.com)
	if strings.HasPrefix(pattern, "*.") {
		domain := pattern[2:]
		return strings.HasSuffix(host, "."+domain) || host == domain
	}

	// CIDR match for IP ranges
	if strings.Contains(pattern, "/") {
		return nf.matchesCIDR(host, pattern)
	}

	return false
}

func (nf *GuildNetworkFilter) matchesCIDR(host, cidr string) bool {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	return network.Contains(ip)
}

func (nf *GuildNetworkFilter) isPortAllowed(port int) bool {
	for _, allowedPort := range nf.allowedPorts {
		if port == allowedPort {
			return true
		}
	}
	return false
}

func (nf *GuildNetworkFilter) validateRequest(req NetworkRequest) error {
	if req.Host == "" {
		return gerror.New(gerror.ErrCodeValidation, "host cannot be empty", nil)
	}

	if req.Method == "" {
		req.Method = "GET" // Default method
	}

	// Validate method
	allowedMethods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}
	validMethod := false
	for _, method := range allowedMethods {
		if strings.ToUpper(req.Method) == method {
			validMethod = true
			break
		}
	}
	if !validMethod {
		return gerror.New(gerror.ErrCodeValidation, "invalid HTTP method", nil).
			WithDetails("method", req.Method)
	}

	return nil
}

func (nf *GuildNetworkFilter) validateHeaders(headers map[string]string) error {
	// Check for suspicious headers that might indicate attacks
	suspiciousHeaders := []string{
		"x-forwarded-for",
		"x-real-ip",
		"x-cluster-client-ip",
	}

	for header, value := range headers {
		headerLower := strings.ToLower(header)

		// Check for suspicious headers with internal IP addresses
		for _, suspicious := range suspiciousHeaders {
			if headerLower == suspicious {
				if nf.containsPrivateIP(value) {
					return gerror.New(gerror.ErrCodeSecurityViolation, "suspicious header with private IP", nil).
						WithDetails("header", header).
						WithDetails("value", value)
				}
			}
		}

		// Check for potential injection attempts
		if nf.containsSuspiciousContent(value) {
			return gerror.New(gerror.ErrCodeSecurityViolation, "suspicious header content", nil).
				WithDetails("header", header).
				WithDetails("value", value)
		}
	}

	return nil
}

func (nf *GuildNetworkFilter) containsPrivateIP(value string) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
	}

	// Extract IPs from the value
	for _, ip := range strings.Fields(strings.Replace(value, ",", " ", -1)) {
		parsedIP := net.ParseIP(strings.TrimSpace(ip))
		if parsedIP != nil {
			for _, cidr := range privateRanges {
				if nf.matchesCIDR(parsedIP.String(), cidr) {
					return true
				}
			}
		}
	}

	return false
}

func (nf *GuildNetworkFilter) containsSuspiciousContent(value string) bool {
	suspiciousPatterns := []string{
		"<script",
		"javascript:",
		"data:text/html",
		"../",
		"..%2f",
		"%3cscript",
	}

	valueLower := strings.ToLower(value)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(valueLower, pattern) {
			return true
		}
	}

	return false
}

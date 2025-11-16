package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// CheckResult contains the result of a health check
type CheckResult struct {
	IsUp         bool
	ResponseTime time.Duration
	StatusCode   int    // For HTTP checks
	Error        string // Error message if check failed
	CertExpiry   *time.Time // SSL certificate expiry for HTTPS
}

// CheckEndpoint performs a health check based on the endpoint type
// Supports: TCP (host:port), HTTP (http://...), HTTPS (https://...)
func CheckEndpoint(endpoint string) CheckResult {
	startTime := time.Now()

	// Determine check type based on endpoint format
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return checkHTTP(endpoint, startTime)
	}

	// Default to TCP check
	return checkTCP(endpoint, startTime)
}

// checkTCP performs a TCP connection check
func checkTCP(endpoint string, startTime time.Time) CheckResult {
	timeout := 5 * time.Second
	conn, err := net.DialTimeout("tcp", endpoint, timeout)
	responseTime := time.Since(startTime)

	if err != nil {
		return CheckResult{
			IsUp:         false,
			ResponseTime: responseTime,
			Error:        err.Error(),
		}
	}
	defer conn.Close()

	return CheckResult{
		IsUp:         true,
		ResponseTime: responseTime,
	}
}

// checkHTTP performs an HTTP/HTTPS health check
func checkHTTP(endpoint string, startTime time.Time) CheckResult {
	timeout := 10 * time.Second

	// Custom transport to capture SSL certificate info
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		DialContext: (&net.Dialer{
			Timeout: timeout,
		}).DialContext,
		TLSHandshakeTimeout: timeout,
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 5 redirects
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Get(endpoint)
	responseTime := time.Since(startTime)

	if err != nil {
		return CheckResult{
			IsUp:         false,
			ResponseTime: responseTime,
			Error:        err.Error(),
		}
	}
	defer resp.Body.Close()

	result := CheckResult{
		ResponseTime: responseTime,
		StatusCode:   resp.StatusCode,
	}

	// Extract SSL certificate expiry for HTTPS
	if strings.HasPrefix(endpoint, "https://") && resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		result.CertExpiry = &cert.NotAfter
	}

	// Consider 2xx and 3xx as successful
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.IsUp = true
	} else {
		result.IsUp = false
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}

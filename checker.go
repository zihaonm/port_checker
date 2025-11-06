package main

import (
	"net"
	"time"
)

// CheckEndpoint attempts to establish a TCP connection to the given endpoint
// Returns true if connection is successful, false otherwise
func CheckEndpoint(endpoint string) bool {
	timeout := 5 * time.Second
	conn, err := net.DialTimeout("tcp", endpoint, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

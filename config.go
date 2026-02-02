package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Target represents a monitoring target with its schedule and endpoint
type Target struct {
	Schedule string // cron schedule
	Endpoint string // IP:port
	Name     string // optional friendly name for the target
}

// LoadTargets reads the configuration file and parses targets
func LoadTargets(filename string) ([]Target, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var targets []Target
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse line: first 5 fields are cron schedule, 6th is endpoint, 7th is optional name
		fields := strings.Fields(line)
		if len(fields) < 6 {
			return nil, fmt.Errorf("invalid format at line %d: expected at least 6 fields (5 cron + endpoint)", lineNum)
		}

		// Cron schedule: first 5 fields
		schedule := strings.Join(fields[0:5], " ")
		// Endpoint: 6th field
		endpoint := fields[5]
		// Name: optional 7th field
		name := ""
		if len(fields) >= 7 {
			name = fields[6]
		}

		// Validate endpoint format (should contain ':')
		if !strings.Contains(endpoint, ":") {
			return nil, fmt.Errorf("invalid endpoint at line %d: %s (expected format IP:port)", lineNum, endpoint)
		}

		targets = append(targets, Target{
			Schedule: schedule,
			Endpoint: endpoint,
			Name:     name,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no valid targets found in config file")
	}

	return targets, nil
}

// LoadTargetsFromString parses targets from a string (for embedded content)
func LoadTargetsFromString(content string) ([]Target, error) {
	var targets []Target
	lineNum := 0

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse line: first 5 fields are cron schedule, 6th is endpoint, 7th is optional name
		fields := strings.Fields(line)
		if len(fields) < 6 {
			return nil, fmt.Errorf("invalid format at line %d: expected at least 6 fields (5 cron + endpoint)", lineNum)
		}

		// Cron schedule: first 5 fields
		schedule := strings.Join(fields[0:5], " ")
		// Endpoint: 6th field
		endpoint := fields[5]
		// Name: optional 7th field
		name := ""
		if len(fields) >= 7 {
			name = fields[6]
		}

		// Validate endpoint format (should contain ':')
		if !strings.Contains(endpoint, ":") && !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			return nil, fmt.Errorf("invalid endpoint at line %d: %s", lineNum, endpoint)
		}

		targets = append(targets, Target{
			Schedule: schedule,
			Endpoint: endpoint,
			Name:     name,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no valid targets found in config")
	}

	return targets, nil
}

package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

//go:embed .env
var embeddedEnv string

//go:embed targets.txt
var embeddedTargets string

// EndpointStatus tracks the status and metrics of an endpoint
type EndpointStatus struct {
	IsUp             bool
	LastCheck        time.Time
	LastStatusChange time.Time
	ResponseTime     time.Duration
	FailureCount     int       // Consecutive failures
	LastAlertTime    time.Time // Last time we sent an alert
	StatusCode       int       // HTTP status code
	CertExpiry       *time.Time
}

// Monitor manages the monitoring of endpoints
type Monitor struct {
	notifier   *TelegramNotifier
	logger     *Logger
	statusMap  map[string]*EndpointStatus // tracks status and metrics of each endpoint
	nameMap    map[string]string          // maps endpoint to friendly name
	statusMu   sync.RWMutex
}

// NewMonitor creates a new Monitor instance
func NewMonitor(notifier *TelegramNotifier, logger *Logger) *Monitor {
	return &Monitor{
		notifier:  notifier,
		logger:    logger,
		statusMap: make(map[string]*EndpointStatus),
		nameMap:   make(map[string]string),
	}
}

// CheckAndNotify checks an endpoint and sends notifications on status changes
func (m *Monitor) CheckAndNotify(endpoint string, name string) {
	result := CheckEndpoint(endpoint)
	now := time.Now()

	m.statusMu.Lock()
	defer m.statusMu.Unlock()

	status, exists := m.statusMap[endpoint]
	if !exists {
		status = &EndpointStatus{
			LastStatusChange: now,
		}
		m.statusMap[endpoint] = status
	}

	// Store the name mapping
	if name != "" {
		m.nameMap[endpoint] = name
	}

	// Update metrics
	previousStatus := status.IsUp
	previousFailureCount := status.FailureCount
	previousLastStatusChange := status.LastStatusChange
	status.LastCheck = now
	status.ResponseTime = result.ResponseTime
	status.StatusCode = result.StatusCode
	status.CertExpiry = result.CertExpiry

	if result.IsUp {
		status.FailureCount = 0
		status.IsUp = true
	} else {
		status.FailureCount++
		status.IsUp = false
	}

	// Log with response time
	if result.IsUp {
		m.logger.LogInfo(fmt.Sprintf("%s is UP (response time: %v)", endpoint, result.ResponseTime))
	} else {
		m.logger.LogError(fmt.Sprintf("%s is DOWN (error: %s, response time: %v)", endpoint, result.Error, result.ResponseTime))
	}

	// Determine if we should send alert (with throttling)
	shouldAlert := false
	statusChanged := !exists || previousStatus != result.IsUp

	if statusChanged {
		status.LastStatusChange = now
		shouldAlert = true
	} else if !result.IsUp {
		// For ongoing failures, escalate alerts based on failure count
		timeSinceLastAlert := now.Sub(status.LastAlertTime)

		// Escalation rules:
		// 3 failures: alert immediately
		// 6 failures: alert after 15 min
		// 10+ failures: alert every 30 min
		if status.FailureCount == 3 {
			shouldAlert = true
		} else if status.FailureCount == 6 && timeSinceLastAlert > 15*time.Minute {
			shouldAlert = true
		} else if status.FailureCount >= 10 && timeSinceLastAlert > 30*time.Minute {
			shouldAlert = true
		}
	}

	// Check SSL certificate expiry (warn if less than 30 days)
	if result.CertExpiry != nil {
		daysUntilExpiry := time.Until(*result.CertExpiry).Hours() / 24
		if daysUntilExpiry <= 30 && daysUntilExpiry > 0 {
			timeSinceLastCertAlert := now.Sub(status.LastAlertTime)
			// Send cert warning once per day
			if timeSinceLastCertAlert > 24*time.Hour {
				m.notifier.SendCertExpiryWarning(endpoint, name, *result.CertExpiry)
				m.logger.LogWarning(fmt.Sprintf("%s SSL certificate expires in %.0f days", endpoint, daysUntilExpiry))
			}
		}
	}

	// Send status change notifications
	if shouldAlert {
		status.LastAlertTime = now

		if result.IsUp {
			if exists && previousStatus == false {
				// Service recovered
				downtime := now.Sub(previousLastStatusChange)
				if err := m.notifier.SendUpAlert(endpoint, name, previousFailureCount, downtime); err != nil {
					m.logger.LogError(fmt.Sprintf("Failed to send UP notification for %s: %v", endpoint, err))
				}
			}
		} else {
			// Service is down
			if err := m.notifier.SendDownAlert(endpoint, name, status.FailureCount, result.Error); err != nil {
				m.logger.LogError(fmt.Sprintf("Failed to send DOWN notification for %s: %v", endpoint, err))
			}
		}
	}
}

func main() {
	// Load environment variables with priority: system env > file .env > embedded .env
	// Step 1: Parse embedded .env into a map
	envMap := make(map[string]string)
	if embeddedEnv != "" {
		for _, line := range strings.Split(embeddedEnv, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				envMap[key] = value
			}
		}
	}
	// Step 2: Parse file system .env and override embedded values
	if fileEnv, err := godotenv.Read(); err == nil {
		for key, value := range fileEnv {
			envMap[key] = value
		}
	}
	// Step 3: Apply merged map, but never override system env vars
	for key, value := range envMap {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	// Get configuration from environment
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	configFile := os.Getenv("CONFIG_FILE")
	logFile := os.Getenv("LOG_FILE")

	// Set defaults
	if configFile == "" {
		configFile = "targets.txt"
	}
	if logFile == "" {
		logFile = "checker.log"
	}

	// Validate required configuration
	if botToken == "" || chatID == "" {
		fmt.Println("ERROR: TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID must be set")
		os.Exit(1)
	}

	// Initialize logger
	logger, err := NewLogger(logFile)
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.LogInfo("Starting service port monitor")

	// Load targets with priority: file system > embedded
	var targets []Target
	targets, err = LoadTargets(configFile)
	if err == nil {
		logger.LogInfo(fmt.Sprintf("Loaded targets from file: %s", configFile))
	} else if embeddedTargets != "" {
		targets, err = LoadTargetsFromString(embeddedTargets)
		if err == nil {
			logger.LogInfo("Loaded targets from embedded configuration")
		}
	}
	if len(targets) == 0 {
		logger.LogError("Failed to load targets from file or embedded configuration")
		os.Exit(1)
	}

	logger.LogInfo(fmt.Sprintf("Loaded %d targets from %s", len(targets), configFile))

	// Initialize Telegram notifier
	notifier := NewTelegramNotifier(botToken, chatID)

	// Initialize monitor
	monitor := NewMonitor(notifier, logger)

	// Create cron scheduler
	c := cron.New()

	// Register jobs for each target
	for _, target := range targets {
		endpoint := target.Endpoint
		name := target.Name
		schedule := target.Schedule

		_, err := c.AddFunc(schedule, func() {
			monitor.CheckAndNotify(endpoint, name)
		})

		if err != nil {
			logger.LogError(fmt.Sprintf("Failed to schedule job for %s with schedule '%s': %v", endpoint, schedule, err))
			continue
		}

		logger.LogInfo(fmt.Sprintf("Scheduled monitoring for %s with schedule: %s", endpoint, schedule))
	}

	// Start the cron scheduler
	c.Start()
	logger.LogInfo("Cron scheduler started")

	// Run initial checks for all endpoints
	logger.LogInfo("Running initial health checks")
	for _, target := range targets {
		monitor.CheckAndNotify(target.Endpoint, target.Name)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.LogInfo("Monitor is running. Press Ctrl+C to stop.")
	<-sigChan

	// Graceful shutdown
	logger.LogInfo("Shutting down...")
	c.Stop()
	logger.LogInfo("Monitor stopped")
}

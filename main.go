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
	statusMu   sync.RWMutex
}

// NewMonitor creates a new Monitor instance
func NewMonitor(notifier *TelegramNotifier, logger *Logger) *Monitor {
	return &Monitor{
		notifier:  notifier,
		logger:    logger,
		statusMap: make(map[string]*EndpointStatus),
	}
}

// CheckAndNotify checks an endpoint and sends notifications on status changes
func (m *Monitor) CheckAndNotify(endpoint string) {
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

	// Update metrics
	previousStatus := status.IsUp
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
				m.notifier.SendCertExpiryWarning(endpoint, *result.CertExpiry)
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
				downtime := now.Sub(status.LastStatusChange)
				if err := m.notifier.SendUpAlert(endpoint, status.FailureCount, downtime); err != nil {
					m.logger.LogError(fmt.Sprintf("Failed to send UP notification for %s: %v", endpoint, err))
				}
			}
		} else {
			// Service is down
			if err := m.notifier.SendDownAlert(endpoint, status.FailureCount, result.Error); err != nil {
				m.logger.LogError(fmt.Sprintf("Failed to send DOWN notification for %s: %v", endpoint, err))
			}
		}
	}
}

// GetAllStatuses returns a snapshot of all endpoint statuses (for web dashboard)
func (m *Monitor) GetAllStatuses() map[string]EndpointStatus {
	m.statusMu.RLock()
	defer m.statusMu.RUnlock()

	snapshot := make(map[string]EndpointStatus)
	for endpoint, status := range m.statusMap {
		snapshot[endpoint] = *status
	}
	return snapshot
}

func main() {
	// Load environment variables from embedded .env first, then try file system
	if embeddedEnv != "" {
		// Parse embedded env vars
		for _, line := range strings.Split(embeddedEnv, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Only set if not already set in environment
				if os.Getenv(key) == "" {
					os.Setenv(key, value)
				}
			}
		}
	}
	// Try to load from file system (overrides embedded)
	_ = godotenv.Load()

	// Get configuration from environment
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	configFile := os.Getenv("CONFIG_FILE")
	logFile := os.Getenv("LOG_FILE")
	dashboardPort := os.Getenv("DASHBOARD_PORT")

	// Set defaults
	if configFile == "" {
		configFile = "targets.txt"
	}
	if logFile == "" {
		logFile = "checker.log"
	}
	if dashboardPort == "" {
		dashboardPort = "8080"
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

	// Load targets from embedded content first, then try file
	var targets []Target
	if embeddedTargets != "" {
		targets, err = LoadTargetsFromString(embeddedTargets)
		if err == nil {
			logger.LogInfo("Loaded targets from embedded configuration")
		}
	}

	// If embedded failed or doesn't exist, try file system
	if len(targets) == 0 {
		targets, err = LoadTargets(configFile)
		if err != nil {
			logger.LogError(fmt.Sprintf("Failed to load targets: %v", err))
			os.Exit(1)
		}
	}

	logger.LogInfo(fmt.Sprintf("Loaded %d targets from %s", len(targets), configFile))

	// Initialize Telegram notifier
	notifier := NewTelegramNotifier(botToken, chatID)

	// Initialize monitor
	monitor := NewMonitor(notifier, logger)

	// Start dashboard server in background
	dashboard := NewDashboardServer(monitor, dashboardPort)
	go func() {
		if err := dashboard.Start(); err != nil {
			logger.LogError(fmt.Sprintf("Dashboard server error: %v", err))
		}
	}()

	// Create cron scheduler
	c := cron.New()

	// Register jobs for each target
	for _, target := range targets {
		endpoint := target.Endpoint
		schedule := target.Schedule

		_, err := c.AddFunc(schedule, func() {
			monitor.CheckAndNotify(endpoint)
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
		monitor.CheckAndNotify(target.Endpoint)
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

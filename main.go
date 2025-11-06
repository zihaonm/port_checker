package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

// Monitor manages the monitoring of endpoints
type Monitor struct {
	notifier   *TelegramNotifier
	logger     *Logger
	statusMap  map[string]bool // tracks last known status of each endpoint
	statusMu   sync.RWMutex
}

// NewMonitor creates a new Monitor instance
func NewMonitor(notifier *TelegramNotifier, logger *Logger) *Monitor {
	return &Monitor{
		notifier:  notifier,
		logger:    logger,
		statusMap: make(map[string]bool),
	}
}

// CheckAndNotify checks an endpoint and sends notifications on status changes
func (m *Monitor) CheckAndNotify(endpoint string) {
	isUp := CheckEndpoint(endpoint)

	m.statusMu.Lock()
	lastStatus, exists := m.statusMap[endpoint]
	m.statusMap[endpoint] = isUp
	m.statusMu.Unlock()

	// If status changed or this is the first check
	if !exists || lastStatus != isUp {
		m.logger.LogStatusChange(endpoint, isUp)

		// Send notification
		if isUp {
			if exists { // Only send UP notification if we previously knew it was down
				if err := m.notifier.SendUpAlert(endpoint); err != nil {
					m.logger.LogError(fmt.Sprintf("Failed to send UP notification for %s: %v", endpoint, err))
				}
			}
		} else {
			if err := m.notifier.SendDownAlert(endpoint); err != nil {
				m.logger.LogError(fmt.Sprintf("Failed to send DOWN notification for %s: %v", endpoint, err))
			}
		}
	}
}

func main() {
	// Load environment variables from .env file if it exists
	_ = godotenv.Load()

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

	// Load targets from config file
	targets, err := LoadTargets(configFile)
	if err != nil {
		logger.LogError(fmt.Sprintf("Failed to load targets: %v", err))
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

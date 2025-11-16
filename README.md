# Service Port Monitor

A Go-based background monitoring tool that checks if IP:port endpoints are reachable based on cron schedules and sends Telegram notifications when services go down or come back up.

## Features

- **Multiple Check Types:**
  - TCP port connectivity checks
  - HTTP/HTTPS endpoint monitoring with status code validation
  - SSL certificate expiration monitoring (30-day warning)

- **Advanced Monitoring:**
  - Response time tracking and metrics
  - Alert throttling and escalation (prevents spam)
  - Configurable check schedules using cron expressions

- **Web Dashboard:**
  - Real-time status visualization at http://localhost:8080
  - Service uptime statistics
  - Response time metrics with color-coded alerts
  - SSL certificate expiration warnings

- **Smart Notifications:**
  - Telegram alerts for status changes
  - Escalation alerts for persistent failures (3, 6, 10+ failures)
  - Detailed downtime and recovery information
  - SSL certificate expiration warnings

- **Deployment Options:**
  - Run as standalone binary
  - systemd service integration
  - Graceful shutdown on SIGINT/SIGTERM

## Requirements

- Go 1.21+ (for building from source)
- Telegram Bot Token and Chat ID
- systemd (optional, for service deployment)

## Configuration

### 1. Telegram Setup

Create a Telegram bot and get your credentials:

1. Talk to [@BotFather](https://t.me/botfather) on Telegram
2. Create a new bot with `/newbot`
3. Copy the bot token
4. Get your chat ID by messaging [@userinfobot](https://t.me/userinfobot)

### 2. Environment Variables

Create a `.env` file (copy from `.env.example`):

```bash
cp .env.example .env
```

Edit `.env` with your credentials:

```env
TELEGRAM_BOT_TOKEN=1234567890:ABCdefGHIjklMNOpqrsTUVwxyz
TELEGRAM_CHAT_ID=123456789
CONFIG_FILE=targets.txt
LOG_FILE=checker.log
DASHBOARD_PORT=8080
```

### 3. Targets Configuration

Edit `targets.txt` to define your monitoring targets:

```
# Format: <minute> <hour> <day> <month> <weekday> <endpoint>

# TCP port checks
*/10 4-23 * * * 127.0.0.1:3390

# HTTP/HTTPS checks (includes SSL certificate monitoring for HTTPS)
*/5 * * * * https://example.com
*/15 * * * * http://192.168.1.100:8080/health

# Mix of different check types
*/10 * * * 1-6 32.0.12.2:9090
```

**Supported endpoint types:**
- **TCP:** `IP:port` (e.g., `127.0.0.1:3390`)
- **HTTP:** `http://domain.com` or `http://IP:port/path`
- **HTTPS:** `https://domain.com` (automatically monitors SSL certificate expiration)

#### Cron Format Reference

```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of week (0 - 6) (Sunday=0)
│ │ │ │ │
* * * * *
```

Examples:
- `*/5 * * * *` - Every 5 minutes
- `0 */2 * * *` - Every 2 hours
- `0 9-17 * * 1-5` - Every hour from 9am to 5pm, Monday to Friday
- `30 2 * * *` - At 2:30 AM every day

## Installation & Usage

### Option 1: Run from Source

```bash
# Install dependencies
go mod download

# Run the monitor
go run .
```

### Option 2: Build Binary

```bash
# Quick build using script
./build.sh

# Or build manually
go build -o port-checker

# Run
./port-checker
```

### Option 3: Build for Linux Server

The build process **embeds .env and targets.txt directly into the binary**, so you don't need to copy config files separately!

```bash
# 1. Configure your .env and targets.txt locally
nano .env          # Add your Telegram credentials
nano targets.txt   # Add your monitoring targets

# 2. Build for Linux (from macOS/any OS) - embeds the config files
./build-linux.sh

# 3. Copy single binary to your Ubuntu server
scp port-checker-linux user@your-server:~/

# 4. SSH and run - no config files needed!
ssh user@your-server
./port-checker-linux
```

**How it works:**
- The build script embeds `.env` and `targets.txt` into the binary
- Deploy just one file - no need to manage config files on the server
- You can still override by creating `.env` or `targets.txt` on the server if needed

### Option 4: systemd Service

```bash
# Build the binary
go build -o service-port-monitor

# Create installation directory
sudo mkdir -p /opt/service-port-monitor
sudo mkdir -p /var/log/service-port-monitor

# Copy files
sudo cp service-port-monitor /opt/service-port-monitor/
sudo cp targets.txt /opt/service-port-monitor/

# Edit the service file with your credentials
sudo nano service-port-monitor.service

# Install service
sudo cp service-port-monitor.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable service-port-monitor
sudo systemctl start service-port-monitor

# Check status
sudo systemctl status service-port-monitor

# View logs
sudo journalctl -u service-port-monitor -f
```

## Web Dashboard

Access the real-time dashboard at `http://localhost:8080` (or your configured port).

**Features:**
- Live status of all monitored services
- Response time metrics with color-coded indicators:
  - Green: < 100ms (excellent)
  - Orange: 100-500ms (acceptable)
  - Red: > 500ms (slow)
- Uptime statistics
- SSL certificate expiration warnings
- Auto-refresh every 5 seconds

## Notifications

The monitor sends enhanced Telegram messages:

- **When service goes down:**
  ```
  ⚠️ [DOWN] https://example.com is not reachable

  Error: HTTP 503
  ```

- **When service comes back up:**
  ```
  ✅ [UP] https://example.com is now reachable

  Was down for: 15.3 minutes
  Failed checks: 3
  ```

- **Escalation alerts for persistent failures:**
  ```
  🚨 [DOWN] https://example.com is not reachable

  Failure count: 10
  Error: connection timeout
  ```

- **SSL certificate warnings (30 days before expiry):**
  ```
  ⚠️ [SSL WARNING] https://example.com

  SSL certificate expires in 25 days
  Expiry date: 2025-12-07 14:30
  ```

## Logging

All status changes are logged to `checker.log` (or path specified in `LOG_FILE`):

```
[2025-11-07 10:30:00] INFO: Starting service port monitor
[2025-11-07 10:30:00] INFO: Loaded 4 targets from targets.txt
[2025-11-07 10:30:00] INFO: Scheduled monitoring for https://example.com with schedule: */5 * * * *
[2025-11-07 10:30:00] Dashboard server starting on http://localhost:8080
[2025-11-07 10:30:00] INFO: Running initial health checks
[2025-11-07 10:30:05] INFO: https://example.com is UP (response time: 125ms)
[2025-11-07 10:30:10] ERROR: 92.10.12.2:8080 is DOWN (error: connection refused, response time: 5s)
[2025-11-07 10:30:11] WARNING: https://example.com SSL certificate expires in 25 days
```

## Project Structure

```
.
├── main.go              # Entry point and orchestration
├── config.go            # Configuration file parser
├── checker.go           # TCP/HTTP/HTTPS health checkers
├── telegram.go          # Telegram notification handler
├── logger.go            # Logging functionality
├── dashboard.go         # Web dashboard server
├── targets.txt          # Monitoring targets configuration
├── .env.example         # Environment variables template
├── service-port-monitor.service  # systemd service file
└── README.md            # This file
```

## Dependencies

- [github.com/robfig/cron/v3](https://github.com/robfig/cron) - Cron scheduler
- [github.com/joho/godotenv](https://github.com/joho/godotenv) - Environment variable loader

## Troubleshooting

### Connection Timeouts

If legitimate services are being marked as down, check:
- Firewall rules between monitor and target
- Network connectivity
- Target service actually listening on specified port

### Telegram Notifications Not Sending

- Verify `TELEGRAM_BOT_TOKEN` is correct
- Verify `TELEGRAM_CHAT_ID` is correct
- Ensure bot has been started (send `/start` to your bot)
- Check logs for error messages

### Service Not Starting

- Check log file permissions
- Verify config file exists and is readable
- Ensure environment variables are set correctly

## License

MIT License - feel free to use and modify as needed.

# Service Port Monitor

A Go-based background monitoring tool that checks if IP:port endpoints are reachable based on cron schedules and sends Telegram notifications when services go down or come back up.

## Features

- Scheduled health checks using cron expressions
- TCP connection testing with 5-second timeout
- Telegram notifications for status changes
- Continuous logging of all status changes
- Support for running as systemd service or Docker container
- Graceful shutdown on SIGINT/SIGTERM

## Requirements

- Go 1.21+ (for building from source)
- Telegram Bot Token and Chat ID
- Docker (optional, for containerized deployment)
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
```

### 3. Targets Configuration

Edit `targets.txt` to define your monitoring targets:

```
# Format: <minute> <hour> <day> <month> <weekday> <IP:port>

# Check every 10 minutes between 4am-11pm
*/10 4-23 * * * 127.0.0.1:3390

# Check every 10 minutes on weekdays (Mon-Sat)
*/10 * * * 1-6 32.0.12.2:9090

# Check every 30 minutes, every day
*/30 * * * * 92.10.12.2:8080
```

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
# Build
go build -o service-port-monitor

# Run
./service-port-monitor
```

### Option 3: Docker

```bash
# Build image
docker build -t service-port-monitor .

# Run with docker-compose (recommended)
docker-compose up -d

# Or run directly
docker run -d \
  --name service-port-monitor \
  -e TELEGRAM_BOT_TOKEN="your_token" \
  -e TELEGRAM_CHAT_ID="your_chat_id" \
  -v $(pwd)/targets.txt:/app/targets.txt:ro \
  -v $(pwd)/logs:/app/logs \
  service-port-monitor
```

View logs:
```bash
docker-compose logs -f
# or
docker logs -f service-port-monitor
```

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

## Notifications

The monitor sends Telegram messages on status changes:

- **When service goes down:**
  ```
  ⚠️ [DOWN] 92.10.12.2:8080 is not reachable
  ```

- **When service comes back up:**
  ```
  ✅ [UP] 92.10.12.2:8080 is now reachable
  ```

## Logging

All status changes are logged to `checker.log` (or path specified in `LOG_FILE`):

```
[2025-11-07 10:30:00] INFO: Starting service port monitor
[2025-11-07 10:30:00] INFO: Loaded 3 targets from targets.txt
[2025-11-07 10:30:00] INFO: Scheduled monitoring for 127.0.0.1:3390 with schedule: */10 4-23 * * *
[2025-11-07 10:30:00] INFO: Running initial health checks
[2025-11-07 10:30:05] 92.10.12.2:8080 is DOWN
[2025-11-07 10:30:10] 127.0.0.1:3390 is UP
```

## Project Structure

```
.
├── main.go              # Entry point and orchestration
├── config.go            # Configuration file parser
├── checker.go           # TCP connection checker
├── telegram.go          # Telegram notification handler
├── logger.go            # Logging functionality
├── targets.txt          # Monitoring targets configuration
├── .env.example         # Environment variables template
├── Dockerfile           # Docker image definition
├── docker-compose.yml   # Docker Compose configuration
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

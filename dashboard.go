package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
)

// DashboardServer serves the web dashboard
type DashboardServer struct {
	monitor *Monitor
	port    string
}

// NewDashboardServer creates a new dashboard server
func NewDashboardServer(monitor *Monitor, port string) *DashboardServer {
	return &DashboardServer{
		monitor: monitor,
		port:    port,
	}
}

// Start starts the web server
func (d *DashboardServer) Start() error {
	http.HandleFunc("/", d.handleDashboard)
	http.HandleFunc("/api/status", d.handleAPIStatus)

	fmt.Printf("Dashboard server starting on http://localhost:%s\n", d.port)
	return http.ListenAndServe(":"+d.port, nil)
}

// handleDashboard serves the HTML dashboard
func (d *DashboardServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Service Port Monitor Dashboard</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
        }
        header {
            background: white;
            padding: 25px;
            border-radius: 10px;
            margin-bottom: 30px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            font-size: 32px;
            margin-bottom: 10px;
        }
        .subtitle {
            color: #666;
            font-size: 14px;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: white;
            padding: 20px;
            border-radius: 10px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        .stat-value {
            font-size: 36px;
            font-weight: bold;
            margin-bottom: 5px;
        }
        .stat-label {
            color: #666;
            font-size: 14px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .stat-up { color: #10b981; }
        .stat-down { color: #ef4444; }
        .stat-total { color: #667eea; }
        .endpoints {
            display: grid;
            gap: 20px;
        }
        .endpoint-card {
            background: white;
            padding: 25px;
            border-radius: 10px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            transition: transform 0.2s;
        }
        .endpoint-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 12px rgba(0,0,0,0.15);
        }
        .endpoint-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
            padding-bottom: 15px;
            border-bottom: 2px solid #f0f0f0;
        }
        .endpoint-name {
            font-size: 20px;
            font-weight: 600;
            color: #333;
            font-family: 'Monaco', 'Courier New', monospace;
        }
        .status-badge {
            padding: 8px 16px;
            border-radius: 20px;
            font-weight: 600;
            font-size: 14px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .status-up {
            background: #d1fae5;
            color: #065f46;
        }
        .status-down {
            background: #fee2e2;
            color: #991b1b;
        }
        .endpoint-details {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
        }
        .detail-item {
            padding: 10px;
            background: #f9fafb;
            border-radius: 6px;
        }
        .detail-label {
            font-size: 12px;
            color: #666;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 5px;
        }
        .detail-value {
            font-size: 16px;
            font-weight: 600;
            color: #333;
        }
        .response-time {
            color: #10b981;
        }
        .response-time-slow {
            color: #f59e0b;
        }
        .response-time-very-slow {
            color: #ef4444;
        }
        .error-message {
            margin-top: 15px;
            padding: 12px;
            background: #fee2e2;
            border-left: 4px solid #ef4444;
            border-radius: 4px;
            color: #991b1b;
            font-size: 14px;
        }
        .cert-warning {
            margin-top: 15px;
            padding: 12px;
            background: #fef3c7;
            border-left: 4px solid #f59e0b;
            border-radius: 4px;
            color: #92400e;
            font-size: 14px;
        }
        .last-update {
            text-align: center;
            color: white;
            margin-top: 30px;
            font-size: 14px;
        }
        .loading {
            text-align: center;
            padding: 40px;
            color: white;
            font-size: 18px;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🔍 Service Port Monitor</h1>
            <p class="subtitle">Real-time monitoring dashboard</p>
        </header>

        <div class="stats" id="stats">
            <div class="stat-card">
                <div class="stat-value stat-up" id="upCount">-</div>
                <div class="stat-label">Services Up</div>
            </div>
            <div class="stat-card">
                <div class="stat-value stat-down" id="downCount">-</div>
                <div class="stat-label">Services Down</div>
            </div>
            <div class="stat-card">
                <div class="stat-value stat-total" id="totalCount">-</div>
                <div class="stat-label">Total Services</div>
            </div>
        </div>

        <div class="endpoints" id="endpoints">
            <div class="loading">Loading endpoints...</div>
        </div>

        <div class="last-update" id="lastUpdate"></div>
    </div>

    <script>
        function formatDuration(ms) {
            if (ms < 1000) return ms.toFixed(0) + 'ms';
            if (ms < 1000 * 60) return (ms / 1000).toFixed(1) + 's';
            if (ms < 1000 * 60 * 60) return (ms / 1000 / 60).toFixed(1) + 'm';
            return (ms / 1000 / 60 / 60).toFixed(1) + 'h';
        }

        function formatTime(dateStr) {
            if (!dateStr) return 'Never';
            const date = new Date(dateStr);
            return date.toLocaleString();
        }

        function getResponseTimeClass(ms) {
            if (ms < 100000000) return 'response-time'; // < 100ms
            if (ms < 500000000) return 'response-time-slow'; // < 500ms
            return 'response-time-very-slow'; // >= 500ms
        }

        function renderEndpoints(data) {
            const upCount = Object.values(data).filter(s => s.IsUp).length;
            const totalCount = Object.keys(data).length;
            const downCount = totalCount - upCount;

            document.getElementById('upCount').textContent = upCount;
            document.getElementById('downCount').textContent = downCount;
            document.getElementById('totalCount').textContent = totalCount;

            const endpointsHTML = Object.entries(data).map(([endpoint, status]) => {
                const statusClass = status.IsUp ? 'status-up' : 'status-down';
                const statusText = status.IsUp ? '✓ Up' : '✗ Down';
                const responseTimeClass = getResponseTimeClass(status.ResponseTime);

                let errorHTML = '';
                if (!status.IsUp) {
                    errorHTML = '<div class="error-message">Service is currently unreachable</div>';
                }

                let certWarning = '';
                if (status.CertExpiry) {
                    const expiryDate = new Date(status.CertExpiry);
                    const daysUntilExpiry = (expiryDate - new Date()) / (1000 * 60 * 60 * 24);
                    if (daysUntilExpiry <= 30 && daysUntilExpiry > 0) {
                        certWarning = '<div class="cert-warning">⚠️ SSL certificate expires in ' +
                            Math.floor(daysUntilExpiry) + ' days (' + expiryDate.toLocaleDateString() + ')</div>';
                    }
                }

                let httpStatusHTML = '';
                if (status.StatusCode > 0) {
                    httpStatusHTML = '<div class="detail-item"><div class="detail-label">HTTP Status</div>' +
                        '<div class="detail-value">' + status.StatusCode + '</div></div>';
                }

                return '<div class="endpoint-card">' +
                    '<div class="endpoint-header">' +
                    '<div class="endpoint-name">' + endpoint + '</div>' +
                    '<div class="status-badge ' + statusClass + '">' + statusText + '</div>' +
                    '</div>' +
                    '<div class="endpoint-details">' +
                    '<div class="detail-item"><div class="detail-label">Response Time</div>' +
                    '<div class="detail-value ' + responseTimeClass + '">' +
                    formatDuration(status.ResponseTime / 1000000) + '</div></div>' +
                    '<div class="detail-item"><div class="detail-label">Last Check</div>' +
                    '<div class="detail-value">' + formatTime(status.LastCheck) + '</div></div>' +
                    '<div class="detail-item"><div class="detail-label">Last Change</div>' +
                    '<div class="detail-value">' + formatTime(status.LastStatusChange) + '</div></div>' +
                    httpStatusHTML +
                    '</div>' +
                    errorHTML +
                    certWarning +
                    '</div>';
            }).join('');

            document.getElementById('endpoints').innerHTML = endpointsHTML;
            document.getElementById('lastUpdate').textContent = 'Last updated: ' + new Date().toLocaleTimeString();
        }

        function fetchStatus() {
            fetch('/api/status')
                .then(response => response.json())
                .then(data => renderEndpoints(data))
                .catch(error => {
                    console.error('Error fetching status:', error);
                    document.getElementById('endpoints').innerHTML =
                        '<div class="loading">Error loading endpoints</div>';
                });
        }

        // Initial fetch
        fetchStatus();

        // Refresh every 5 seconds
        setInterval(fetchStatus, 5000);
    </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, _ := template.New("dashboard").Parse(tmpl)
	t.Execute(w, nil)
}

// handleAPIStatus serves the JSON status data
func (d *DashboardServer) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	statuses := d.monitor.GetAllStatuses()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

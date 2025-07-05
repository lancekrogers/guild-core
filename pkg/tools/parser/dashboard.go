// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// DashboardServer provides HTTP endpoints for parser monitoring
type DashboardServer struct {
	parser   *MonitoredParser
	mux      *http.ServeMux
	server   *http.Server
	
	// WebSocket connections for live updates
	mu          sync.RWMutex
	connections map[string]*websocketConn
}

// websocketConn represents a WebSocket connection (simplified)
type websocketConn struct {
	send chan []byte
}

// NewDashboardServer creates a monitoring dashboard server
func NewDashboardServer(parser *MonitoredParser, addr string) *DashboardServer {
	ds := &DashboardServer{
		parser:      parser,
		mux:         http.NewServeMux(),
		connections: make(map[string]*websocketConn),
	}
	
	// Set up routes
	ds.setupRoutes()
	
	// Create HTTP server
	ds.server = &http.Server{
		Addr:         addr,
		Handler:      ds.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	
	return ds
}

// setupRoutes configures all HTTP routes
func (ds *DashboardServer) setupRoutes() {
	// Health endpoints
	ds.mux.HandleFunc("/health", ds.handleHealth)
	ds.mux.HandleFunc("/health/live", ds.handleLiveness)
	ds.mux.HandleFunc("/health/ready", ds.handleReadiness)
	
	// Metrics endpoints
	ds.mux.Handle("/metrics", promhttp.Handler())
	ds.mux.HandleFunc("/metrics/summary", ds.handleMetricsSummary)
	
	// Alert endpoints
	ds.mux.HandleFunc("/alerts", ds.handleAlerts)
	ds.mux.HandleFunc("/alerts/active", ds.handleActiveAlerts)
	
	// Dashboard UI
	ds.mux.HandleFunc("/", ds.handleDashboard)
	ds.mux.HandleFunc("/api/stats", ds.handleStats)
	ds.mux.HandleFunc("/api/formats", ds.handleFormats)
}

// handleHealth returns comprehensive health check
func (ds *DashboardServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := ds.parser.GetHealth()
	
	// Set status code based on health
	statusCode := http.StatusOK
	switch health.Status {
	case HealthStatusDegraded:
		statusCode = http.StatusOK // Still operational
	case HealthStatusUnhealthy:
		statusCode = http.StatusServiceUnavailable
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// handleLiveness checks if the service is alive
func (ds *DashboardServer) handleLiveness(w http.ResponseWriter, r *http.Request) {
	// Simple liveness check - service is running
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleReadiness checks if the service is ready to handle requests
func (ds *DashboardServer) handleReadiness(w http.ResponseWriter, r *http.Request) {
	health := ds.parser.GetHealth()
	
	if health.Status == HealthStatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Not Ready"))
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}

// handleMetricsSummary returns a summary of key metrics
func (ds *DashboardServer) handleMetricsSummary(w http.ResponseWriter, r *http.Request) {
	health := ds.parser.GetHealth()
	
	summary := map[string]interface{}{
		"timestamp": time.Now(),
		"uptime":    health.Uptime.String(),
		"metrics":   health.Metrics,
		"status":    health.Status,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// handleAlerts returns all alerts
func (ds *DashboardServer) handleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := ds.parser.alertManager.GetAllAlerts()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// handleActiveAlerts returns only active alerts
func (ds *DashboardServer) handleActiveAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := ds.parser.GetAlerts()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// handleStats returns real-time statistics
func (ds *DashboardServer) handleStats(w http.ResponseWriter, r *http.Request) {
	health := ds.parser.GetHealth()
	
	stats := map[string]interface{}{
		"timestamp":       time.Now(),
		"parse_rate":      health.Metrics.ParseRate,
		"success_rate":    health.Metrics.SuccessRate,
		"avg_latency_ms":  health.Metrics.AverageLatency,
		"p95_latency_ms":  health.Metrics.P95Latency,
		"p99_latency_ms":  health.Metrics.P99Latency,
		"active_parsers":  health.Metrics.ActiveParsers,
		"total_parses":    health.Metrics.TotalParses,
		"total_successes": health.Metrics.TotalSuccesses,
		"total_failures":  health.Metrics.TotalFailures,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleFormats returns format distribution
func (ds *DashboardServer) handleFormats(w http.ResponseWriter, r *http.Request) {
	health := ds.parser.GetHealth()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health.Metrics.FormatDistribution)
}

// handleDashboard serves the monitoring dashboard HTML
func (ds *DashboardServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(dashboardHTML))
}

// Start starts the dashboard server
func (ds *DashboardServer) Start() error {
	return ds.server.ListenAndServe()
}

// Stop gracefully shuts down the server
func (ds *DashboardServer) Stop() error {
	return ds.server.Close()
}

// dashboardHTML is a simple monitoring dashboard
const dashboardHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>Guild Parser Monitoring Dashboard</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .header {
            background-color: #333;
            color: white;
            padding: 20px;
            border-radius: 5px;
            margin-bottom: 20px;
        }
        .metric-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .metric-card {
            background-color: white;
            padding: 20px;
            border-radius: 5px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .metric-value {
            font-size: 2em;
            font-weight: bold;
            color: #333;
            margin: 10px 0;
        }
        .metric-label {
            color: #666;
            font-size: 0.9em;
        }
        .status-healthy {
            color: #4CAF50;
        }
        .status-degraded {
            color: #FF9800;
        }
        .status-unhealthy {
            color: #F44336;
        }
        .chart-container {
            background-color: white;
            padding: 20px;
            border-radius: 5px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        .alerts-container {
            background-color: white;
            padding: 20px;
            border-radius: 5px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .alert-item {
            padding: 10px;
            margin: 5px 0;
            border-radius: 3px;
        }
        .alert-critical {
            background-color: #ffebee;
            border-left: 4px solid #F44336;
        }
        .alert-warning {
            background-color: #fff3e0;
            border-left: 4px solid #FF9800;
        }
        .alert-info {
            background-color: #e3f2fd;
            border-left: 4px solid #2196F3;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Guild Parser Monitoring Dashboard</h1>
            <div id="status">Loading...</div>
        </div>
        
        <div class="metric-grid">
            <div class="metric-card">
                <div class="metric-label">Parse Rate</div>
                <div class="metric-value" id="parse-rate">-</div>
                <div class="metric-label">per second</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Success Rate</div>
                <div class="metric-value" id="success-rate">-</div>
                <div class="metric-label">percentage</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Avg Latency</div>
                <div class="metric-value" id="avg-latency">-</div>
                <div class="metric-label">milliseconds</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Active Parsers</div>
                <div class="metric-value" id="active-parsers">-</div>
                <div class="metric-label">concurrent</div>
            </div>
        </div>
        
        <div class="chart-container">
            <h2>Format Distribution</h2>
            <canvas id="format-chart"></canvas>
        </div>
        
        <div class="alerts-container">
            <h2>Active Alerts</h2>
            <div id="alerts-list">No active alerts</div>
        </div>
    </div>
    
    <script>
        // Update dashboard every 5 seconds
        function updateDashboard() {
            // Fetch stats
            fetch('/api/stats')
                .then(response => response.json())
                .then(data => {
                    document.getElementById('parse-rate').textContent = data.parse_rate.toFixed(2);
                    document.getElementById('success-rate').textContent = (data.success_rate * 100).toFixed(1) + '%';
                    document.getElementById('avg-latency').textContent = data.avg_latency_ms.toFixed(1);
                    document.getElementById('active-parsers').textContent = data.active_parsers;
                });
            
            // Fetch health
            fetch('/health')
                .then(response => response.json())
                .then(data => {
                    const statusEl = document.getElementById('status');
                    statusEl.textContent = 'Status: ' + data.status.toUpperCase();
                    statusEl.className = 'status-' + data.status;
                });
            
            // Fetch alerts
            fetch('/alerts/active')
                .then(response => response.json())
                .then(alerts => {
                    const alertsList = document.getElementById('alerts-list');
                    if (alerts.length === 0) {
                        alertsList.innerHTML = '<p>No active alerts</p>';
                    } else {
                        alertsList.innerHTML = alerts.map(alert => {
                            return '<div class="alert-item alert-' + alert.severity + '">' +
                                '<strong>' + alert.title + '</strong><br>' +
                                alert.description +
                                '</div>';
                        }).join('');
                    }
                });
        }
        
        // Initial update
        updateDashboard();
        
        // Update every 5 seconds
        setInterval(updateDashboard, 5000);
    </script>
</body>
</html>
`

// CreateFullyInstrumentedParser creates a parser with all monitoring features
func CreateFullyInstrumentedParser(version string) (*MonitoredParser, *DashboardServer) {
	// Create base parser
	baseParser := NewResponseParser()
	
	// Add tracing and metrics
	instrumentedParser := InstrumentParser(baseParser)
	
	// Add monitoring and alerting
	monitoredParser := NewMonitoredParser(instrumentedParser, version)
	
	// Create dashboard
	dashboardServer := NewDashboardServer(monitoredParser, ":8080")
	
	return monitoredParser, dashboardServer
}
// Dashboard main script
// Handles routing, page rendering, and event display

let currentRoute = 'home';
let statusCheckInterval = null;

// Initialize dashboard
document.addEventListener('DOMContentLoaded', () => {
    initRouting();
    initStatusCheck();
    initMetrics();
    initActivity();
    initDevMode();
    loadInitialData();
});

// Routing
function initRouting() {
    // Set initial route from hash
    const hash = window.location.hash.slice(1) || 'home';
    navigateTo(hash);

    // Handle hash changes
    window.addEventListener('hashchange', () => {
        const route = window.location.hash.slice(1) || 'home';
        navigateTo(route);
    });

    // Handle nav link clicks
    document.querySelectorAll('.nav-link').forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const route = link.dataset.route;
            window.location.hash = route;
        });
    });
}

function navigateTo(route) {
    currentRoute = route;

    // Update nav links
    document.querySelectorAll('.nav-link').forEach(link => {
        link.classList.remove('active');
        if (link.dataset.route === route) {
            link.classList.add('active');
        }
    });

    // Update pages
    document.querySelectorAll('.page').forEach(page => {
        page.classList.remove('active');
    });

    const pageEl = document.getElementById(`page-${route}`);
    if (pageEl) {
        pageEl.classList.add('active');
    }

    // Load page-specific data
    if (route === 'metrics') {
        loadMetrics();
    } else if (route === 'activity') {
        loadActivity();
    }
}

// Status checking
async function initStatusCheck() {
    // Check status immediately
    await updateStatus();

    // Check status every 5 seconds
    statusCheckInterval = setInterval(updateStatus, 5000);
    
    // Reload metrics every 10 seconds if on metrics page
    setInterval(async () => {
        if (currentRoute === 'metrics') {
            await loadMetrics();
        }
    }, 10000);
}

async function updateStatus() {
    try {
        const response = await fetch('http://localhost:8080/api/video-monitor/status');
        if (response.ok) {
            const status = await response.json();
            updateStatusPill(status.status);
            
            // Store status event if status changed
            const lastStatus = sessionStorage.getItem('last_video_status');
            if (lastStatus !== status.status) {
                sessionStorage.setItem('last_video_status', status.status);
                // Optionally store status change event
                if (status.status === 'running' && lastStatus !== 'running') {
                    await appendEvent({
                        id: generateEventId(),
                        ts: new Date().toISOString(),
                        type: 'status',
                        severity: 'info',
                        source: 'video',
                        message: status.message || 'Video monitoring started',
                        metadata: { status: status.status, started_at: status.started_at }
                    });
                }
            }
        } else {
            updateStatusPill('stopped');
        }
    } catch (error) {
        updateStatusPill('stopped');
    }
}

function updateStatusPill(status) {
    const pill = document.getElementById('status-pill');
    if (!pill) return;

    pill.className = 'status-pill';
    pill.textContent = status.charAt(0).toUpperCase() + status.slice(1);

    if (status === 'running') {
        pill.classList.add('status-running');
    } else if (status === 'degraded') {
        pill.classList.add('status-degraded');
    } else {
        pill.classList.add('status-stopped');
    }
}

// Metrics page
async function initMetrics() {
    // Metrics are loaded when page is shown
}

async function loadMetrics() {
    // Fetch video monitor status for real-time data from screen_guard_service
    let videoStatus = null;
    try {
        const response = await fetch('http://localhost:8080/api/video-monitor/status');
        if (response.ok) {
            videoStatus = await response.json();
            console.log('[DASHBOARD] Video monitor status:', videoStatus);
        }
    } catch (error) {
        console.error('[DASHBOARD] Failed to fetch video monitor status:', error);
    }

    // Compute metrics from stored events (which include data from screen_guard_service via SSE)
    const metrics = await computeMetrics();

    // Update monitors started (from events + video status)
    const monitors7d = videoStatus && videoStatus.status === 'running' && videoStatus.started_at 
        ? metrics.monitorsStarted.last7Days + (metrics.monitorsStarted.last7Days === 0 ? 1 : 0)
        : metrics.monitorsStarted.last7Days;
    
    document.getElementById('metric-monitors-7d').textContent = monitors7d;
    document.getElementById('metric-monitors-all').textContent = metrics.monitorsStarted.allTime;

    // Update alerts (from stored events - these come from screen_guard_service via HTTP bridge)
    document.getElementById('metric-alerts-critical').textContent = metrics.alertsDetected.critical;
    document.getElementById('metric-alerts-warn').textContent = metrics.alertsDetected.warn;
    document.getElementById('metric-alerts-info').textContent = metrics.alertsDetected.info;

    // Update warnings and errors
    document.getElementById('metric-warnings').textContent = metrics.warnings;
    document.getElementById('metric-errors').textContent = metrics.errors;

    // Update latency (from detection events metadata)
    const latencyEl = document.getElementById('metric-latency');
    if (metrics.avgLatency !== null) {
        latencyEl.textContent = `${metrics.avgLatency.toFixed(0)}ms`;
    } else {
        latencyEl.textContent = 'N/A';
    }

    // Update trend chart with daily alerts from screen_guard_service
    updateTrendChart(metrics.dailyAlerts);

    // Log video monitor runtime if available
    if (videoStatus && videoStatus.started_at) {
        const runtime = calculateRuntime(videoStatus.started_at);
        if (runtime) {
            console.log('[DASHBOARD] Video monitor runtime:', runtime);
        }
    }
}

function calculateRuntime(startedAt) {
    if (!startedAt) return null;
    const start = new Date(startedAt);
    const now = new Date();
    const diff = now - start;
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    return `${hours}h ${minutes}m`;
}

function updateTrendChart(dailyData) {
    const chartEl = document.getElementById('trend-chart');
    if (!chartEl) return;

    if (!dailyData || dailyData.length === 0 || dailyData.every(v => v === 0)) {
        chartEl.innerHTML = '<div class="trend-placeholder">No data available</div>';
        return;
    }

    const maxValue = Math.max(...dailyData, 1);
    const bars = dailyData.map((value, index) => {
        const height = maxValue > 0 ? (value / maxValue) * 100 : 0;
        return `
            <div class="trend-bar" style="height: ${height}%" data-value="${value}" title="Day ${index + 1}: ${value} alerts">
            </div>
        `;
    }).join('');

    chartEl.innerHTML = bars;
}

// Activity page
function initActivity() {
    const severityFilter = document.getElementById('filter-severity');
    const searchFilter = document.getElementById('filter-search');
    const exportBtn = document.getElementById('export-btn');

    if (severityFilter) {
        severityFilter.addEventListener('change', loadActivity);
    }

    if (searchFilter) {
        searchFilter.addEventListener('input', debounce(loadActivity, 300));
    }

    if (exportBtn) {
        exportBtn.addEventListener('click', exportEvents);
    }
}

async function loadActivity() {
    const severityFilter = document.getElementById('filter-severity');
    const searchFilter = document.getElementById('filter-search');
    const activityList = document.getElementById('activity-list');

    if (!activityList) return;

    const filters = {
        severity: severityFilter ? severityFilter.value : 'all',
        search: searchFilter ? searchFilter.value.trim() : '',
        limit: 50
    };

    const events = await getFilteredEvents(filters);

    if (events.length === 0) {
        activityList.innerHTML = '<div class="empty-state">No events match your filters.</div>';
        return;
    }

    activityList.innerHTML = events.map(event => renderEvent(event)).join('');
}

function renderEvent(event) {
    const time = new Date(event.ts).toLocaleString();
    const severityClass = event.severity || 'info';
    
    return `
        <div class="activity-item">
            <div class="severity-badge ${severityClass}">${severityClass}</div>
            <div class="activity-content">
                <div class="activity-header">
                    <span class="activity-type">${event.type}</span>
                    <span class="activity-time">${time}</span>
                </div>
                <div class="activity-message">${escapeHtml(event.message)}</div>
                <div class="activity-source">Source: ${event.source}</div>
            </div>
        </div>
    `;
}

async function exportEvents() {
    const events = await getEvents();
    const dataStr = JSON.stringify(events, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `dashboard-events-${new Date().toISOString().split('T')[0]}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
}

// Dev mode
function initDevMode() {
    // Check if dev mode is enabled
    chrome.storage.local.get(['dev_mode']).then(result => {
        if (result.dev_mode) {
            const devControls = document.getElementById('dev-controls');
            if (devControls) {
                devControls.style.display = 'block';
            }

            const generateBtn = document.getElementById('dev-generate-events');
            const clearBtn = document.getElementById('dev-clear-events');

            if (generateBtn) {
                generateBtn.addEventListener('click', generateSampleEvents);
            }

            if (clearBtn) {
                clearBtn.addEventListener('click', async () => {
                    if (confirm('Clear all events?')) {
                        await clearEvents();
                        await loadActivity();
                        await loadMetrics();
                    }
                });
            }
        }
    });
}

async function generateSampleEvents() {
    const sampleEvents = [
        {
            id: generateEventId(),
            ts: new Date(Date.now() - 1000 * 60 * 5).toISOString(),
            type: 'status',
            severity: 'info',
            source: 'video',
            message: 'Video monitoring started successfully'
        },
        {
            id: generateEventId(),
            ts: new Date(Date.now() - 1000 * 60 * 4).toISOString(),
            type: 'log',
            severity: 'info',
            source: 'monitor',
            message: 'Scan completed - no detections'
        },
        {
            id: generateEventId(),
            ts: new Date(Date.now() - 1000 * 60 * 3).toISOString(),
            type: 'detection',
            severity: 'critical',
            source: 'video',
            message: 'Password detected: password_assignment',
            metadata: { rule_name: 'password_assignment', confidence: 0.95, latency: 120 }
        },
        {
            id: generateEventId(),
            ts: new Date(Date.now() - 1000 * 60 * 2).toISOString(),
            type: 'detection',
            severity: 'warn',
            source: 'video',
            message: 'Token detected: token_heuristics',
            metadata: { rule_name: 'token_heuristics', confidence: 0.75, latency: 95 }
        },
        {
            id: generateEventId(),
            ts: new Date(Date.now() - 1000 * 60 * 1).toISOString(),
            type: 'log',
            severity: 'info',
            source: 'scanner',
            message: 'OCR confidence: 0.92'
        },
        {
            id: generateEventId(),
            ts: new Date(Date.now() - 1000 * 30).toISOString(),
            type: 'status',
            severity: 'info',
            source: 'extension',
            message: 'Monitoring active'
        }
    ];

    for (const event of sampleEvents) {
        await appendEvent(event);
    }

    // Reload current page
    if (currentRoute === 'activity') {
        await loadActivity();
    } else if (currentRoute === 'metrics') {
        await loadMetrics();
    }

    alert('Generated 6 sample events!');
}

// Initial data load
async function loadInitialData() {
    // Load status
    await updateStatus();

    // If on metrics page, load metrics
    if (currentRoute === 'metrics') {
        await loadMetrics();
    }

    // If on activity page, load activity
    if (currentRoute === 'activity') {
        await loadActivity();
    }
}

// Utility functions
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Listen for events from background/popup
chrome.runtime.onMessage?.addListener((message, sender, sendResponse) => {
    if (message.type === 'DASHBOARD_EVENT') {
        appendEvent(message.event).then(() => {
            if (currentRoute === 'activity') {
                loadActivity();
            } else if (currentRoute === 'metrics') {
                loadMetrics();
            }
        });
    }
});

// Fetch video monitor metrics from service
async function fetchVideoMonitorMetrics() {
    try {
        const response = await fetch('http://localhost:8080/api/video-monitor/status');
        if (response.ok) {
            const status = await response.json();
            return {
                status: status.status,
                startedAt: status.started_at,
                stoppedAt: status.stopped_at,
                message: status.message,
                error: status.error
            };
        }
    } catch (error) {
        console.error('Failed to fetch video monitor metrics:', error);
    }
    return null;
}

// Enhanced metrics display with video service data
async function enhanceMetricsWithVideoData() {
    const videoMetrics = await fetchVideoMonitorMetrics();
    
    if (videoMetrics && videoMetrics.status === 'running' && videoMetrics.startedAt) {
        // Calculate uptime
        const startTime = new Date(videoMetrics.startedAt);
        const now = new Date();
        const uptimeMs = now - startTime;
        const uptimeHours = Math.floor(uptimeMs / (1000 * 60 * 60));
        const uptimeMinutes = Math.floor((uptimeMs % (1000 * 60 * 60)) / (1000 * 60));
        
        // Could add uptime display to metrics if needed
        console.log(`Video monitor uptime: ${uptimeHours}h ${uptimeMinutes}m`);
    }
    
    return videoMetrics;
}


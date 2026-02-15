let MALICIOUS_DOMAINS = [];

const ALLOWED_EXTENSIONS = [
    // Documents
    '.pdf', '.docx', '.doc', '.xlsx', '.xls', '.csv', '.pptx', '.txt', '.rtf',
    // Credentials/Config
    '.pem', '.key', '.env', '.json', '.xml', '.yaml', '.yml',
    // Archives
    '.zip', '.tar', '.gz'
];

const API_ENDPOINT = "http://localhost:8080/api";

document.addEventListener('DOMContentLoaded', () => {
    loadBlocklist();
    initSettings();
    loadStats();
    initVideoMonitoring();

    const dropZone = document.getElementById('pg-drop-zone');
    const hiddenInput = document.getElementById('pg-hidden-file-input');
    const progressDiv = document.getElementById('pg-upload-progress');
    const progressBar = document.getElementById('pg-progress-bar');
    const progressText = document.getElementById('pg-progress-text');
    const resultDiv = document.getElementById('pg-link-result');
    const copyBtn = document.getElementById('pg-copy-btn');
    const openBtn = document.getElementById('pg-open-btn');
    const linkText = document.getElementById('pg-link-text');
    const linkInput = document.getElementById('pg-link-input');
    const checkBtn = document.getElementById('pg-check-btn');
    const checkResult = document.getElementById('pg-check-result');

    if (!dropZone || !hiddenInput) {
        console.error("Required elements not found");
        return;
    }

    if (checkBtn && linkInput && checkResult) {
        checkBtn.addEventListener('click', () => checkLink(linkInput.value.trim(), checkResult));
    }

    dropZone.addEventListener('click', () => hiddenInput.click());

    hiddenInput.addEventListener('change', (e) => {
        if (e.target.files.length > 0) {
            const file = e.target.files[0];
            handleFileUpload(file, dropZone, progressDiv, progressBar, progressText, resultDiv, linkText, copyBtn);
        }
    });

    copyBtn.addEventListener('click', () => {
        navigator.clipboard.writeText(linkText.textContent).then(() => {
            const orig = copyBtn.textContent;
            copyBtn.textContent = "Copied!";
            setTimeout(() => copyBtn.textContent = orig, 2000);
        });
    });

    if (openBtn) {
        openBtn.addEventListener('click', () => {
            const link = linkText.textContent;
            if (link) chrome.tabs.create({ url: link });
        });
    }
});

const UPLOAD_URL = "http://localhost:8080/api/upload";

async function handleFileUpload(file, dropZone, progressDiv, progressBar, progressText, resultDiv, linkText, copyBtn) {
    dropZone.style.display = 'none';
    progressDiv.style.display = 'block';
    resultDiv.style.display = 'none';
    progressText.textContent = "Uploading...";
    progressBar.style.width = '10%';

    try {
        const formData = new FormData();
        formData.append('file', file);
        progressBar.style.width = '30%';

        const result = await new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();
            xhr.open('POST', UPLOAD_URL);
            xhr.onload = () => {
                if (xhr.status >= 200 && xhr.status < 300) {
                    try {
                        resolve(JSON.parse(xhr.responseText));
                    } catch {
                        reject(new Error('Invalid response from server'));
                    }
                } else {
                    let msg = `Upload failed (${xhr.status})`;
                    try {
                        const j = JSON.parse(xhr.responseText);
                        if (j && j.error) msg = j.error;
                        else if (xhr.responseText) msg = xhr.responseText;
                    } catch {
                        if (xhr.responseText) msg = xhr.responseText;
                    }
                    reject(new Error(msg));
                }
            };
            xhr.onerror = () => reject(new Error("Could not connect to backend. Is it running on port 8080?"));
            xhr.upload.onprogress = (e) => {
                if (e.lengthComputable) {
                    progressBar.style.width = Math.max(30, 30 + (e.loaded / e.total) * 65) + '%';
                }
            };
            xhr.send(formData);
        });

        progressText.textContent = "Finalizing...";
        progressBar.style.width = '100%';
        await new Promise(r => setTimeout(r, 300));

        progressDiv.style.display = 'none';
        resultDiv.style.display = 'block';
        linkText.textContent = result.viewLink;
        document.getElementById('pg-hidden-file-input').value = '';
    } catch (error) {
        console.error("Upload failed:", error);
        progressText.textContent = `Error: ${error.message}`;
        progressText.style.color = "#e74c3c";
        progressBar.style.backgroundColor = "#e74c3c";
        setTimeout(() => {
            resetUI(dropZone, progressDiv, resultDiv, progressBar, progressText);
        }, 3000);
    }
}

function resetUI(dropZone, progressDiv, resultDiv, progressBar, progressText) {
    dropZone.style.display = 'block';
    progressDiv.style.display = 'none';
    resultDiv.style.display = 'none';
    progressBar.style.backgroundColor = '#2ecc71';
    progressBar.style.width = '0%';
    progressText.style.color = '#666';
}

function loadBlocklist() {
    return fetch(chrome.runtime.getURL('malicious_domains.json'))
        .then(r => r.json())
        .then(data => { MALICIOUS_DOMAINS = data; })
        .catch(() => { MALICIOUS_DOMAINS = []; });
}

async function ensureBlocklistLoaded() {
    if (MALICIOUS_DOMAINS.length === 0) {
        await loadBlocklist();
    }
}

function checkLink(input, resultEl) {
    resultEl.style.display = 'block';
    resultEl.className = 'check-result';

    if (!input) {
        resultEl.className += ' invalid';
        resultEl.textContent = 'Please paste a URL first.';
        return;
    }

    let url;
    try {
        url = new URL(input.startsWith('http') ? input : 'https://' + input);
    } catch {
        resultEl.className += ' invalid';
        resultEl.textContent = 'Invalid URL. Use a full link (e.g. https://example.com).';
        return;
    }

    resultEl.textContent = 'Checking…';
    ensureBlocklistLoaded().then(() => {
        const domain = url.hostname.replace(/^www\./, '').toLowerCase();
        const isBlocklisted = MALICIOUS_DOMAINS.some(
            bad => domain === bad || domain.endsWith('.' + bad)
        );
        const isInsecure = url.protocol === 'http:';

        if (isBlocklisted) {
            resultEl.className += ' flagged';
            resultEl.textContent = '⚠️ Flagged — this domain is on our blocklist. Proceed with caution.';
        } else if (isInsecure) {
            resultEl.className += ' flagged';
            resultEl.textContent = '⚠️ Insecure — link uses HTTP (not HTTPS). Prefer HTTPS when possible.';
        } else {
            resultEl.className += ' legit';
            resultEl.textContent = '✓ Looks legit — not on our blocklist. Still use normal caution.';
        }
    });
}

// Settings management
const DEFAULT_SETTINGS = {
    pasteGuardEnabled: true,
    pasteBlockThreshold: 'HIGH',
    pasteAllowConvertToLink: true
};

async function initSettings() {
    const enabledToggle = document.getElementById('pg-setting-enabled');
    const thresholdSelect = document.getElementById('pg-setting-threshold');
    const convertLinkToggle = document.getElementById('pg-setting-convert-link');

    if (!enabledToggle || !thresholdSelect || !convertLinkToggle) {
        return;
    }

    // Load settings from storage
    const settings = await chrome.storage.sync.get(DEFAULT_SETTINGS);
    
    enabledToggle.checked = settings.pasteGuardEnabled ?? DEFAULT_SETTINGS.pasteGuardEnabled;
    thresholdSelect.value = settings.pasteBlockThreshold ?? DEFAULT_SETTINGS.pasteBlockThreshold;
    convertLinkToggle.checked = settings.pasteAllowConvertToLink ?? DEFAULT_SETTINGS.pasteAllowConvertToLink;

    // Save defaults if not present
    await chrome.storage.sync.set({
        pasteGuardEnabled: enabledToggle.checked,
        pasteBlockThreshold: thresholdSelect.value,
        pasteAllowConvertToLink: convertLinkToggle.checked
    });

    // Add event listeners
    enabledToggle.addEventListener('change', async () => {
        await chrome.storage.sync.set({ pasteGuardEnabled: enabledToggle.checked });
    });

    thresholdSelect.addEventListener('change', async () => {
        await chrome.storage.sync.set({ pasteBlockThreshold: thresholdSelect.value });
    });

    convertLinkToggle.addEventListener('change', async () => {
        await chrome.storage.sync.set({ pasteAllowConvertToLink: convertLinkToggle.checked });
    });
}

// Load and display stats
async function loadStats() {
    try {
        const stats = await chrome.runtime.sendMessage({ type: 'GET_STATS' });
        if (stats) {
            const analyzedEl = document.getElementById('pg-stat-analyzed');
            const blockedEl = document.getElementById('pg-stat-blocked');
            const linksEl = document.getElementById('pg-stat-links');
            
            if (analyzedEl) analyzedEl.textContent = stats.pastes_analyzed || 0;
            if (blockedEl) blockedEl.textContent = stats.pastes_blocked || 0;
            if (linksEl) linksEl.textContent = stats.secure_links_created || 0;
        }
    } catch (error) {
        console.error('Failed to load stats:', error);
    }
}

// Video monitoring state
let videoMonitorEventSource = null;
let videoAlerts = [];
let videoLogs = [];
const MAX_ALERTS = 10;
const MAX_LOGS = 50;

// Initialize video monitoring UI
function initVideoMonitoring() {
    const startBtn = document.getElementById('pg-video-start-btn');
    const stopBtn = document.getElementById('pg-video-stop-btn');
    
    if (!startBtn || !stopBtn) return;
    
    startBtn.addEventListener('click', startVideoMonitoring);
    stopBtn.addEventListener('click', stopVideoMonitoring);
    
    // Load initial status
    loadVideoStatus();
    
    // Start SSE connection if monitoring is active
    checkAndConnectSSE();
}

async function loadVideoStatus() {
    try {
        const response = await fetch('http://localhost:8080/api/video-monitor/status');
        if (response.ok) {
            const status = await response.json();
            updateVideoStatus(status);
        }
    } catch (error) {
        console.error('Failed to load video status:', error);
    }
}

function updateVideoStatus(status) {
    const statusEl = document.getElementById('pg-video-status');
    const startBtn = document.getElementById('pg-video-start-btn');
    const stopBtn = document.getElementById('pg-video-stop-btn');
    const errorEl = document.getElementById('pg-video-error');
    const alertsDiv = document.getElementById('pg-video-alerts');
    const logsDiv = document.getElementById('pg-video-logs');
    
    if (!statusEl) return;
    
    statusEl.textContent = status.status.charAt(0).toUpperCase() + status.status.slice(1);
    
    // Update status color
    const statusColors = {
        'stopped': { bg: '#ecf0f1', color: '#7f8c8d' },
        'starting': { bg: '#fff3cd', color: '#856404' },
        'running': { bg: '#d5f5e3', color: '#1e8449' },
        'stopping': { bg: '#fff3cd', color: '#856404' },
        'degraded': { bg: '#fadbd8', color: '#c0392b' }
    };
    const colors = statusColors[status.status] || statusColors.stopped;
    statusEl.style.background = colors.bg;
    statusEl.style.color = colors.color;
    
    // Update buttons
    if (status.status === 'running' || status.status === 'starting') {
        startBtn.style.display = 'none';
        stopBtn.style.display = 'block';
        if (alertsDiv) alertsDiv.style.display = 'block';
        if (logsDiv) logsDiv.style.display = 'block';
    } else {
        startBtn.style.display = 'block';
        stopBtn.style.display = 'none';
    }
    
    // Show error if present
    if (status.error && errorEl) {
        errorEl.textContent = status.error;
        errorEl.style.display = 'block';
        if (status.recoverable) {
            errorEl.style.background = '#fff3cd';
            errorEl.style.borderColor = '#f39c12';
            errorEl.style.color = '#856404';
        }
    } else if (errorEl) {
        errorEl.style.display = 'none';
    }
}

async function startVideoMonitoring() {
    console.log('[VIDEO_MONITOR] Start button clicked');
    const startBtn = document.getElementById('pg-video-start-btn');
    if (startBtn) {
        startBtn.disabled = true;
        startBtn.textContent = 'Starting...';
    }
    
    try {
        console.log('[VIDEO_MONITOR] Sending POST to /api/video-monitor/start');
        const response = await fetch('http://localhost:8080/api/video-monitor/start', {
            method: 'POST'
        });
        
        console.log('[VIDEO_MONITOR] Response status:', response.status);
        
        if (response.ok) {
            const status = await response.json();
            console.log('[VIDEO_MONITOR] Start successful, status:', status);
            updateVideoStatus(status);
            connectSSE();
        } else {
            const error = await response.json();
            console.error('[VIDEO_MONITOR] Start failed:', error);
            showVideoError(error.error || 'Failed to start video monitoring');
        }
    } catch (error) {
        console.error('[VIDEO_MONITOR] Network error:', error);
        showVideoError('Could not connect to backend. Is it running on port 8080?');
    } finally {
        if (startBtn) {
            startBtn.disabled = false;
            startBtn.textContent = 'Start';
        }
    }
}

async function stopVideoMonitoring() {
    const stopBtn = document.getElementById('pg-video-stop-btn');
    if (stopBtn) {
        stopBtn.disabled = true;
        stopBtn.textContent = 'Stopping...';
    }
    
    try {
        const response = await fetch('http://localhost:8080/api/video-monitor/stop', {
            method: 'POST'
        });
        
        if (response.ok) {
            const status = await response.json();
            updateVideoStatus(status);
            disconnectSSE();
        } else {
            const error = await response.json();
            showVideoError(error.error || 'Failed to stop video monitoring');
        }
    } catch (error) {
        showVideoError('Could not connect to backend');
    } finally {
        if (stopBtn) {
            stopBtn.disabled = false;
            stopBtn.textContent = 'Stop';
        }
    }
}

function showVideoError(message) {
    const errorEl = document.getElementById('pg-video-error');
    if (errorEl) {
        errorEl.textContent = message;
        errorEl.style.display = 'block';
    }
}

function connectSSE() {
    if (videoMonitorEventSource) {
        return; // Already connected
    }
    
    try {
        const eventSource = new EventSource('http://localhost:8080/api/video-monitor/stream');
        videoMonitorEventSource = eventSource;
        
        eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                handleVideoEvent(data);
            } catch (error) {
                console.error('Error parsing SSE event:', error);
            }
        };
        
        eventSource.onerror = (error) => {
            console.error('SSE connection error:', error);
            // Attempt reconnect after delay
            setTimeout(() => {
                if (videoMonitorEventSource) {
                    disconnectSSE();
                    connectSSE();
                }
            }, 3000);
        };
    } catch (error) {
        console.error('Failed to connect to SSE stream:', error);
    }
}

function disconnectSSE() {
    if (videoMonitorEventSource) {
        videoMonitorEventSource.close();
        videoMonitorEventSource = null;
    }
}

function checkAndConnectSSE() {
    loadVideoStatus().then(() => {
        // Check if status is running, then connect
        const statusEl = document.getElementById('pg-video-status');
        if (statusEl && (statusEl.textContent.toLowerCase() === 'running' || statusEl.textContent.toLowerCase() === 'starting')) {
            connectSSE();
        }
    });
}

async function handleVideoEvent(event) {
    // Store event in dashboard
    const dashboardEvent = {
        id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
        ts: new Date().toISOString(),
        type: event.type,
        severity: mapSeverity(event),
        source: 'video',
        message: getEventMessage(event),
        metadata: event.data
    };
    
    // Store in background (which stores in chrome.storage.local)
    try {
        await chrome.runtime.sendMessage({
            type: 'STORE_DASHBOARD_EVENT',
            event: dashboardEvent
        });
    } catch (error) {
        console.error('Failed to store dashboard event:', error);
    }
    
    switch (event.type) {
        case 'status':
            updateVideoStatus(event.data);
            addLog('info', 'monitor', event.data.message || 'Status update');
            break;
        case 'detection':
            addAlert(event.data);
            addLog('warning', 'detector', `Detection: ${event.data.rule_name} (${event.data.severity})`);
            break;
        case 'log':
            addLog(event.data.level, event.data.component, event.data.message);
            break;
        case 'error':
            showVideoError(event.data.message || 'An error occurred');
            addLog('error', 'system', event.data.message || 'Error occurred');
            break;
    }
}

function mapSeverity(event) {
    if (event.type === 'error') return 'critical';
    if (event.type === 'detection') {
        return event.data.severity === 'critical' || event.data.severity === 'high' ? 'critical' : 'warn';
    }
    if (event.data && event.data.level === 'error') return 'critical';
    if (event.data && event.data.level === 'warning') return 'warn';
    return 'info';
}

function getEventMessage(event) {
    if (event.data && event.data.message) return event.data.message;
    if (event.type === 'detection' && event.data.rule_name) {
        return `Detection: ${event.data.rule_name} (${event.data.severity || 'unknown'})`;
    }
    return `${event.type} event`;
}

function addAlert(data) {
    videoAlerts.unshift({
        timestamp: new Date().toLocaleTimeString(),
        rule: data.rule_name || 'unknown',
        severity: data.severity || 'medium',
        confidence: data.confidence || 0,
        text: data.matched_text || ''
    });
    
    if (videoAlerts.length > MAX_ALERTS) {
        videoAlerts.pop();
    }
    
    updateAlertsDisplay();
}

function updateAlertsDisplay() {
    const alertsList = document.getElementById('pg-video-alerts-list');
    if (!alertsList) return;
    
    if (videoAlerts.length === 0) {
        alertsList.innerHTML = '<div style="color: #999; font-style: italic;">No alerts yet</div>';
        return;
    }
    
    alertsList.innerHTML = videoAlerts.map(alert => {
        const severityColors = {
            'critical': '#e74c3c',
            'high': '#e67e22',
            'medium': '#f39c12',
            'low': '#95a5a6'
        };
        const color = severityColors[alert.severity] || '#95a5a6';
        
        return `
            <div style="margin-bottom: 6px; padding: 6px; background: #fff; border-left: 3px solid ${color}; border-radius: 2px;">
                <div style="display: flex; justify-content: space-between; margin-bottom: 2px;">
                    <span style="font-weight: 600; color: ${color};">${alert.rule}</span>
                    <span style="color: #999; font-size: 10px;">${alert.timestamp}</span>
                </div>
                <div style="font-size: 10px; color: #666;">
                    ${alert.severity.toUpperCase()} • ${(alert.confidence * 100).toFixed(0)}% confidence
                </div>
                ${alert.text ? `<div style="font-size: 10px; color: #999; font-family: monospace; margin-top: 2px;">${alert.text}</div>` : ''}
            </div>
        `;
    }).join('');
}

function addLog(level, component, message) {
    videoLogs.unshift({
        timestamp: new Date().toLocaleTimeString(),
        level,
        component,
        message
    });
    
    if (videoLogs.length > MAX_LOGS) {
        videoLogs.pop();
    }
    
    updateLogsDisplay();
}

function updateLogsDisplay() {
    const logsList = document.getElementById('pg-video-logs-list');
    if (!logsList) return;
    
    if (videoLogs.length === 0) {
        logsList.innerHTML = '<div style="color: #999; font-style: italic;">No logs yet</div>';
        return;
    }
    
    logsList.innerHTML = videoLogs.map(log => {
        const levelColors = {
            'error': '#e74c3c',
            'warning': '#f39c12',
            'info': '#3498db'
        };
        const color = levelColors[log.level] || '#95a5a6';
        
        return `
            <div style="margin-bottom: 2px; color: ${color};">
                <span style="color: #999;">[${log.timestamp}]</span>
                <span style="font-weight: 600;">[${log.component}]</span>
                ${log.message}
            </div>
        `;
    }).join('');
}
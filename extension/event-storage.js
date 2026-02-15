// Event storage utility for dashboard
// Stores events in chrome.storage.local and provides metrics computation

const MAX_EVENTS = 500;
const STORAGE_KEY = 'dashboard_events';

/**
 * Event data model
 * @typedef {Object} Event
 * @property {string} id - Unique event ID
 * @property {string} ts - ISO timestamp
 * @property {string} type - Event type: 'status' | 'detection' | 'log' | 'error'
 * @property {string} severity - Severity: 'info' | 'warn' | 'critical'
 * @property {string} source - Source: 'proxy' | 'video' | 'extension'
 * @property {string} message - Event message
 * @property {Object} [metadata] - Optional metadata
 */

/**
 * Append a new event to storage
 * @param {Event} event - Event to append
 * @returns {Promise<void>}
 */
async function appendEvent(event) {
    const events = await getEvents();
    events.unshift(event); // Add to beginning
    
    // Cap at MAX_EVENTS
    if (events.length > MAX_EVENTS) {
        events.splice(MAX_EVENTS);
    }
    
    await chrome.storage.local.set({ [STORAGE_KEY]: events });
}

/**
 * Get all events from storage
 * @returns {Promise<Event[]>}
 */
async function getEvents() {
    const result = await chrome.storage.local.get([STORAGE_KEY]);
    return result[STORAGE_KEY] || [];
}

/**
 * Get events filtered by criteria
 * @param {Object} filters - Filter criteria
 * @param {string} [filters.severity] - Filter by severity
 * @param {string} [filters.type] - Filter by type
 * @param {string} [filters.source] - Filter by source
 * @param {string} [filters.search] - Text search in message
 * @param {number} [filters.limit] - Maximum number of events to return
 * @returns {Promise<Event[]>}
 */
async function getFilteredEvents(filters = {}) {
    let events = await getEvents();
    
    if (filters.severity && filters.severity !== 'all') {
        events = events.filter(e => e.severity === filters.severity);
    }
    
    if (filters.type) {
        events = events.filter(e => e.type === filters.type);
    }
    
    if (filters.source) {
        events = events.filter(e => e.source === filters.source);
    }
    
    if (filters.search) {
        const searchLower = filters.search.toLowerCase();
        events = events.filter(e => 
            e.message.toLowerCase().includes(searchLower) ||
            (e.metadata && JSON.stringify(e.metadata).toLowerCase().includes(searchLower))
        );
    }
    
    if (filters.limit) {
        events = events.slice(0, filters.limit);
    }
    
    return events;
}

/**
 * Compute metrics from events
 * @param {number} days - Number of days to look back (default: 7)
 * @returns {Promise<Object>}
 */
async function computeMetrics(days = 7) {
    const events = await getEvents();
    const now = new Date();
    const cutoffDate = new Date(now.getTime() - (days * 24 * 60 * 60 * 1000));
    
    const allTimeEvents = events;
    const recentEvents = events.filter(e => new Date(e.ts) >= cutoffDate);
    
    // Count monitors started (status events with 'running' or 'starting')
    const monitorsStarted7d = recentEvents.filter(e => 
        e.type === 'status' && 
        (e.message.toLowerCase().includes('start') || e.message.toLowerCase().includes('running'))
    ).length;
    
    const monitorsStartedAll = allTimeEvents.filter(e => 
        e.type === 'status' && 
        (e.message.toLowerCase().includes('start') || e.message.toLowerCase().includes('running'))
    ).length;
    
    // Count alerts by severity
    const alertsBySeverity = {
        critical: 0,
        warn: 0,
        info: 0
    };
    
    allTimeEvents.forEach(e => {
        if (e.type === 'detection' || e.type === 'error') {
            if (e.severity === 'critical') alertsBySeverity.critical++;
            else if (e.severity === 'warn') alertsBySeverity.warn++;
            else alertsBySeverity.info++;
        }
    });
    
    // Count warnings and errors
    const warnings = allTimeEvents.filter(e => e.severity === 'warn').length;
    const errors = allTimeEvents.filter(e => e.severity === 'critical' || e.type === 'error').length;
    
    // Calculate average alert latency (if available in metadata)
    let avgLatency = null;
    const latencies = allTimeEvents
        .filter(e => e.type === 'detection' && e.metadata && e.metadata.latency)
        .map(e => e.metadata.latency);
    
    if (latencies.length > 0) {
        avgLatency = latencies.reduce((a, b) => a + b, 0) / latencies.length;
    }
    
    // Last 7 days alerts count (for trend)
    const alerts7d = recentEvents.filter(e => e.type === 'detection').length;
    
    // Daily breakdown for last 7 days
    const dailyAlerts = {};
    for (let i = 0; i < days; i++) {
        const date = new Date(now.getTime() - (i * 24 * 60 * 60 * 1000));
        const dateKey = date.toISOString().split('T')[0];
        dailyAlerts[dateKey] = recentEvents.filter(e => {
            const eventDate = new Date(e.ts).toISOString().split('T')[0];
            return eventDate === dateKey && e.type === 'detection';
        }).length;
    }
    
    return {
        monitorsStarted: {
            last7Days: monitorsStarted7d,
            allTime: monitorsStartedAll
        },
        alertsDetected: alertsBySeverity,
        warnings,
        errors,
        avgLatency,
        alertsLast7Days: alerts7d,
        dailyAlerts: Object.values(dailyAlerts).reverse() // Most recent first
    };
}

/**
 * Clear all events
 * @returns {Promise<void>}
 */
async function clearEvents() {
    await chrome.storage.local.set({ [STORAGE_KEY]: [] });
}

/**
 * Generate a unique event ID
 * @returns {string}
 */
function generateEventId() {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

// Make functions globally available for dashboard.js
window.appendEvent = appendEvent;
window.getEvents = getEvents;
window.getFilteredEvents = getFilteredEvents;
window.computeMetrics = computeMetrics;
window.clearEvents = clearEvents;
window.generateEventId = generateEventId;


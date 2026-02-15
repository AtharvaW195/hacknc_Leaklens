const UPLOAD_URL = "http://localhost:8080/api/upload";
const ANALYZE_URL = "http://localhost:8080/api/analyze-text";

// In-memory cache for text analysis results
// key: hash of text, value: { report, timestamp }
const analysisCache = new Map();
const CACHE_TTL_MS = 2 * 60 * 1000; // 2 minutes

// Activity ledger counters
const stats = {
  pastes_analyzed: 0,
  pastes_blocked: 0,
  secure_links_created: 0
};

// Clean up expired cache entries periodically
setInterval(() => {
  const now = Date.now();
  for (const [key, value] of analysisCache.entries()) {
    if (now - value.timestamp > CACHE_TTL_MS) {
      analysisCache.delete(key);
    }
  }
}, 30000); // Clean every 30 seconds

// Hash text using SHA-256
async function hashText(text) {
  const encoder = new TextEncoder();
  const data = encoder.encode(text);
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
}

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg.type === "upload") {
    const { arrayBuffer, fileName, contentType } = msg;
    if (!arrayBuffer || !fileName) {
      sendResponse({ error: "Missing file data" });
      return false;
    }
    const blob = new Blob([arrayBuffer], { type: contentType || "application/octet-stream" });
    const formData = new FormData();
    formData.append("file", blob, fileName);
    fetch(UPLOAD_URL, { method: "POST", body: formData })
      .then(async (r) => {
        const text = await r.text();
        if (!r.ok) {
          let msg = r.statusText;
          try {
            const j = JSON.parse(text);
            if (j && j.error) msg = j.error;
          } catch (_) {
            if (text) msg = text;
          }
          throw new Error(msg);
        }
        return JSON.parse(text);
      })
      .then((data) => sendResponse({ viewLink: data.viewLink, fileId: data.fileId }))
      .catch((e) => sendResponse({ error: e.message || "Upload failed" }));
    return true;
  }
  
  if (msg.type === "UPLOAD_SECURE_TEXT") {
    const { filename, text, label } = msg;
    if (!text) {
      sendResponse({ error: "Missing text" });
      return false;
    }
    
    const fileName = filename || "secure-paste.txt";
    const blob = new Blob([text], { type: "text/plain" });
    const formData = new FormData();
    formData.append("file", blob, fileName);
    
    fetch(UPLOAD_URL, { method: "POST", body: formData })
      .then(async (r) => {
        const responseText = await r.text();
        if (!r.ok) {
          let errorMsg = r.statusText;
          try {
            const j = JSON.parse(responseText);
            if (j && j.error) errorMsg = j.error;
          } catch (_) {
            if (responseText) errorMsg = responseText;
          }
          throw new Error(errorMsg);
        }
        return JSON.parse(responseText);
      })
      .then((data) => {
        stats.secure_links_created++;
        sendResponse({ viewLink: data.viewLink, fileId: data.fileId });
      })
      .catch((e) => sendResponse({ error: e.message || "Upload failed" }));
    return true;
  }
  
  if (msg.type === "PASTE_BLOCKED") {
    stats.pastes_blocked++;
    sendResponse({ success: true });
    return false;
  }
  
  if (msg.type === "GET_STATS") {
    sendResponse(stats);
    return false;
  }
  
  if (msg.type === "E2E_SET_MODE") {
    // E2E test mode handler - sets e2e_mode flag in storage
    const enabled = msg.enabled === true;
    chrome.storage.local.set({ e2e_mode: enabled }).then(() => {
      sendResponse({ success: true });
    }).catch((err) => {
      sendResponse({ error: err.message });
    });
    return true; // Keep channel open for async response
  }
  
  if (msg.type === "ANALYZE_TEXT") {
    const { text } = msg;
    if (!text) {
      sendResponse({ error: "Missing text" });
      return false;
    }
    
    // Handle asynchronously for cache lookup and API call
    (async () => {
      try {
        // Compute hash for cache key
        const textHash = await hashText(text);
        
        // Check cache
        const cached = analysisCache.get(textHash);
        const now = Date.now();
        if (cached && (now - cached.timestamp) < CACHE_TTL_MS) {
          // Cache hit
          const findingsCount = cached.report.findings ? cached.report.findings.length : 0;
          console.log(`[ANALYZE_TEXT] cache hit - risk: ${cached.report.overall_risk}, findings: ${findingsCount}`);
          stats.pastes_analyzed++;
          sendResponse({
            overall_risk: cached.report.overall_risk || "low",
            risk_rationale: cached.report.risk_rationale,
            findings: cached.report.findings || []
          });
          return;
        }
        
        // Cache miss - make API call with timeout
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 600);
        
        try {
          const response = await fetch(ANALYZE_URL, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ text: text }),
            signal: controller.signal
          });
          
          clearTimeout(timeoutId);
          
          const responseText = await response.text();
          if (!response.ok) {
            let errorMsg = response.statusText;
            try {
              const j = JSON.parse(responseText);
              if (j && j.error) errorMsg = j.error;
            } catch (_) {
              if (responseText) errorMsg = responseText;
            }
            throw new Error(errorMsg);
          }
          
          const data = JSON.parse(responseText);
          const report = {
            overall_risk: data.overall_risk || "low",
            risk_rationale: data.risk_rationale,
            findings: data.findings || []
          };
          
          // Store in cache
          analysisCache.set(textHash, {
            report: report,
            timestamp: now
          });
          
          const findingsCount = report.findings.length;
          console.log(`[ANALYZE_TEXT] cache miss - risk: ${report.overall_risk}, findings: ${findingsCount}`);
          
          stats.pastes_analyzed++;
          sendResponse(report);
        } catch (fetchError) {
          clearTimeout(timeoutId);
          if (fetchError.name === 'AbortError') {
            // Timeout - fail open
            console.log('[ANALYZE_TEXT] timeout - failing open');
            sendResponse({ error: "Analysis timeout" });
          } else {
            throw fetchError;
          }
        }
      } catch (error) {
        console.log(`[ANALYZE_TEXT] error - ${error.message}`);
        sendResponse({ error: error.message || "Analysis failed" });
      }
    })();
    
    return true; // Keep channel open for async response
  }
  
  return false;
});

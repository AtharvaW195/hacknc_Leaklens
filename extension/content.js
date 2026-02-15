// Link Validation Module
// Malicious Domains List - Loaded from external JSON or E2E override
let MALICIOUS_DOMAINS = [];

// Load malicious domains list with E2E override support
async function loadMaliciousDomainsList() {
    // Check for E2E test override first
    const storage = await chrome.storage.local.get('e2e_malicious_domains');
    if (storage.e2e_malicious_domains && Array.isArray(storage.e2e_malicious_domains)) {
        MALICIOUS_DOMAINS = storage.e2e_malicious_domains;
        console.log("Privacy Guardrail: Using E2E test override for malicious domains.", MALICIOUS_DOMAINS);
        rescanPageAfterListLoaded();
        checkCurrentPage();
        return;
    }
    
    // Normal behavior: load from JSON file
    fetch(chrome.runtime.getURL('malicious_domains.json'))
        .then(response => response.json())
        .then(data => {
            MALICIOUS_DOMAINS = data;
            console.log("Privacy Guardrail: Loaded malicious domains list.", MALICIOUS_DOMAINS);
            rescanPageAfterListLoaded();
            checkCurrentPage();
        })
        .catch(error => console.error("Privacy Guardrail: Failed to load malicious list.", error));
}

// Load the malicious domains list
loadMaliciousDomainsList();

// Re-scan all links once the blocklist has loaded (fixes race where links were checked while list was empty)
function rescanPageAfterListLoaded() {
    document.querySelectorAll('a').forEach(link => {
        delete link.dataset.pgChecked;
        link.classList.remove('pg-malicious-link');
        const icon = link.querySelector('.pg-warning-icon');
        if (icon) icon.remove();
    });
    processNode(document.body);
}

// Add visual status badge
function injectStatusBadge() {
    if (document.getElementById('pg-status-badge')) return;
    const badge = document.createElement('div');
    badge.id = 'pg-status-badge';
    badge.className = 'pg-status-badge';
    badge.title = 'Privacy Guardrail is Active';
    document.body.appendChild(badge);
}

// Check if CURRENT page is malicious
function checkCurrentPage() {
    try {
        const currentDomain = window.location.hostname.replace('www.', '').toLowerCase();

        // Wait for list to load if needed
        if (MALICIOUS_DOMAINS.length === 0) {
            setTimeout(checkCurrentPage, 500);
            return;
        }

        const isMalicious = MALICIOUS_DOMAINS.some(badDomain =>
            currentDomain === badDomain || currentDomain.endsWith('.' + badDomain)
        );

        if (isMalicious) {
            showBlockingPopup(currentDomain);
        }
    } catch (e) {
        console.error("Privacy Guardrail: Error checking current page", e);
    }
}

function showBlockingPopup(domain) {
    // Remove existing content to "block" interaction
    // document.body.innerHTML = ''; // Optional: Clear page content for total block

    const overlay = document.createElement('div');
    overlay.setAttribute('data-testid', 'blocked-overlay');
    overlay.style.position = 'fixed';
    overlay.style.top = '0';
    overlay.style.left = '0';
    overlay.style.width = '100%';
    overlay.style.height = '100%';
    overlay.style.backgroundColor = 'rgba(231, 76, 60, 0.95)'; // Red warning
    overlay.style.zIndex = '2147483647';
    overlay.style.display = 'flex';
    overlay.style.flexDirection = 'column';
    overlay.style.alignItems = 'center';
    overlay.style.justifyContent = 'center';
    overlay.style.color = 'white';
    overlay.style.fontFamily = 'system-ui, sans-serif';
    overlay.style.textAlign = 'center';

    overlay.innerHTML = `
        <div style="background: white; color: #333; padding: 40px; border-radius: 10px; max-width: 500px; box-shadow: 0 10px 25px rgba(0,0,0,0.5);">
            <h1 data-testid="blocked-title" style="color: #c0392b; margin-top: 0;">⚠️ Site Blocked</h1>
            <p style="font-size: 18px;"><b>${domain}</b> is on the blocklist.</p>
            <p>Access to this site has been restricted by Privacy Guardrail.</p>
            <button id="pg-proceed-btn" type="button" style="background: #95a5a6; color: white; border: none; padding: 10px 20px; border-radius: 5px; cursor: pointer; font-size: 14px; margin-top: 20px; pointer-events: auto;">
                Proceed Anyway (Unsafe)
            </button>
        </div>
    `;

    document.documentElement.appendChild(overlay);

    const proceedBtn = document.getElementById('pg-proceed-btn');
    if (proceedBtn) {
        proceedBtn.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            overlay.remove();
        });
    }
}

function processNode(node) {
    if (node.nodeType !== Node.ELEMENT_NODE) return;

    // Check for links
    if (node.tagName === 'A') {
        validateLink(node);
    } else {
        const links = node.querySelectorAll('a');
        links.forEach(validateLink);
    }

    // Check for file inputs
    if (node.tagName === 'INPUT' && node.type === 'file') {
        injectSecureUpload(node);
    } else {
        const inputs = node.querySelectorAll('input[type="file"]');
        inputs.forEach(injectSecureUpload);
    }
}

function validateLink(link) {
    if (link.dataset.pgChecked) return;
    link.dataset.pgChecked = 'true';

    const href = link.href;
    if (!href) return;

    // Check for HTTP on secure pages (heuristic)
    if (window.location.protocol === 'https:' && href.startsWith('http://') && !href.includes('localhost') && !href.includes('127.0.0.1')) {
        flagLink(link, "Insecure HTTP Link");
        return;
    }

    // Check against loaded domain list
    try {
        const url = new URL(href);
        const domain = url.hostname.replace('www.', '').toLowerCase();

        if (MALICIOUS_DOMAINS.some(badDomain => domain === badDomain || domain.endsWith('.' + badDomain))) {
            flagLink(link, " flagged as potentially unsafe");
            return;
        }
    } catch (e) {
        // Invalid URL, ignore
    }
}

function flagLink(link, reason) {
    link.classList.add('pg-malicious-link');
    link.title = `Warning: ${reason}`;

    // Add visual indicator if not already present
    if (!link.querySelector('.pg-warning-icon')) {
        const warningIcon = document.createElement('span');
        warningIcon.className = 'pg-warning-icon';
        warningIcon.setAttribute('data-testid', 'bad-link-icon');
        warningIcon.textContent = ' ⚠️';
        warningIcon.style.color = 'red';
        warningIcon.style.fontSize = '12px';
        link.appendChild(warningIcon);
    }

    // Use capture so we run before the link's default action or other handlers (e.g. JS navigation)
    link.addEventListener('click', (e) => {
        if (!confirm(`This link has been flagged: ${reason}\nAre you sure you want to proceed?`)) {
            e.preventDefault();
            e.stopPropagation();
        }
    }, { capture: true, once: true });

    // Update status badge to alert
    const badge = document.getElementById('pg-status-badge');
    if (badge) badge.classList.add('pg-alert');
}

// Vulnerable File Types Configuration
const ALLOWED_EXTENSIONS = [
    // Documents
    '.pdf', '.docx', '.doc', '.xlsx', '.xls', '.csv', '.pptx', '.txt', '.rtf',
    // Credentials/Config
    '.pem', '.key', '.env', '.json', '.xml', '.yaml', '.yml',
    // Archives
    '.zip', '.tar', '.gz'
];

const API_BASE = "http://localhost:8080";
const UPLOAD_URL = `${API_BASE}/api/upload`;

function injectSecureUpload(input) {
    if (input.dataset.pgSecureInjected) return;

    const container = document.createElement('span');
    container.className = 'pg-secure-upload-container';

    const btn = document.createElement('button');
    btn.className = 'pg-secure-upload-btn';
    btn.setAttribute('data-testid', 'secure-share-btn');
    btn.textContent = '🔒 Secure Share'; // Renamed to clarify intent
    btn.type = 'button';

    btn.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        openSecureUploadModal(input);
    });

    if (input.nextSibling) {
        input.parentNode.insertBefore(container, input.nextSibling);
    } else {
        input.parentNode.appendChild(container);
    }
    container.appendChild(btn);

    input.dataset.pgSecureInjected = 'true';
}

function openSecureUploadModal(originalInput) {
    let modalOverlay = document.getElementById('pg-secure-upload-modal');
    if (modalOverlay) {
        modalOverlay.style.display = 'flex';
        return;
    }

    modalOverlay = document.createElement('div');
    modalOverlay.id = 'pg-secure-upload-modal';
    modalOverlay.className = 'pg-modal-overlay';

    const modal = document.createElement('div');
    modal.className = 'pg-modal';

    modal.innerHTML = `
    <h2>Secure Upload</h2>
    <p>File is uploaded to S3. You get a <b>one-time view-only link</b> to share (recipient can open in browser once).</p>
    <div style="margin-bottom: 20px; border: 2px dashed #ccc; padding: 20px; border-radius: 4px; cursor: pointer;" id="pg-drop-zone">
      Click to select file<br>
      <span style="font-size: 10px; color: #999;">Supports: PDF, Office, Keys, CSV, ZIP</span>
    </div>
    <input type="file" id="pg-hidden-file-input" style="display: none;" accept="${ALLOWED_EXTENSIONS.join(',')}">
    <div id="pg-upload-progress" style="display: none; margin-bottom: 20px;">
      <div id="pg-progress-text">Initializing upload...</div>
      <div style="width: 100%; background: #eee; height: 5px; margin-top: 5px; border-radius: 3px;">
          <div id="pg-progress-bar" style="width: 0%; background: #2ecc71; height: 100%; border-radius: 3px; transition: width 0.2s;"></div>
      </div>
    </div>
    <div id="pg-link-result" class="pg-link-result"></div>
    <div class="pg-modal-actions" id="pg-modal-actions" style="display: none;">
      <button class="pg-btn-cancel" id="pg-modal-close">Close</button>
      <button class="pg-btn-proceed" id="pg-modal-copy">Copy view-only link</button>
      <button class="pg-btn-proceed" id="pg-modal-open">Open link</button>
    </div>
  `;

    modalOverlay.appendChild(modal);
    document.body.appendChild(modalOverlay);

    // Event Listeners
    const closeBtn = modal.querySelector('#pg-modal-close');
    const copyBtn = modal.querySelector('#pg-modal-copy');
    const openBtn = modal.querySelector('#pg-modal-open');
    const actionsDiv = modal.querySelector('#pg-modal-actions');
    const dropZone = modal.querySelector('#pg-drop-zone');
    const hiddenInput = modal.querySelector('#pg-hidden-file-input');
    const progressDiv = modal.querySelector('#pg-upload-progress');
    const progressBar = modal.querySelector('#pg-progress-bar');
    const progressText = modal.querySelector('#pg-progress-text');
    const resultDiv = modal.querySelector('#pg-link-result');

    closeBtn.addEventListener('click', () => {
        modalOverlay.style.display = 'none';
        resetModal(dropZone, progressDiv, resultDiv, actionsDiv, progressBar);
    });

    dropZone.addEventListener('click', () => hiddenInput.click());

    hiddenInput.addEventListener('change', (e) => {
        if (e.target.files.length > 0) {
            const file = e.target.files[0];
            handleFileUpload(file, dropZone, progressDiv, progressBar, progressText, resultDiv, actionsDiv, copyBtn);
        }
    });

    copyBtn.addEventListener('click', () => {
        const link = resultDiv.textContent;
        navigator.clipboard.writeText(link).then(() => {
            copyBtn.textContent = "Copied!";
            setTimeout(() => copyBtn.textContent = "Copy view-only link", 2000);
        });
    });

    openBtn.addEventListener('click', () => {
        const link = resultDiv.textContent;
        if (link) window.open(link, '_blank');
    });
}

async function handleFileUpload(file, dropZone, progressDiv, progressBar, progressText, resultDiv, actionsDiv, copyBtn) {
    dropZone.style.display = 'none';
    progressDiv.style.display = 'block';
    progressText.textContent = "Uploading...";
    progressBar.style.width = '10%';

    try {
        progressBar.style.width = '40%';
        const arrayBuffer = await file.arrayBuffer();
        const result = await chrome.runtime.sendMessage({
            type: 'upload',
            arrayBuffer,
            fileName: file.name,
            contentType: file.type || 'application/octet-stream'
        });
        if (result && result.error) throw new Error(result.error);
        if (!result || !result.viewLink) throw new Error('No link returned from server');

        progressText.textContent = "Finalizing...";
        progressBar.style.width = '100%';
        await new Promise(r => setTimeout(r, 300));

        progressDiv.style.display = 'none';
        resultDiv.style.display = 'block';
        resultDiv.textContent = result.viewLink;
        resultDiv.setAttribute('data-testid', 'last-link');
        document.getElementById('pg-modal-actions').style.display = 'flex';
        document.getElementById('pg-hidden-file-input').value = '';
    } catch (error) {
        console.error("Upload failed:", error);
        progressText.textContent = `Upload failed: ${error.message}`;
        progressText.style.color = "red";
        progressBar.style.backgroundColor = "red";
        resultDiv.style.display = 'block';
        resultDiv.textContent = error.message;
    }
}


function resetModal(dropZone, progressDiv, resultDiv, actionsDiv, progressBar) {
    dropZone.style.display = 'block';
    dropZone.innerHTML = "Click to select file<br><span style=\"font-size: 10px; color: #999;\">Supports: PDF, Office, Keys, CSV, ZIP</span>";
    progressDiv.style.display = 'none';
    resultDiv.style.display = 'none';
    resultDiv.textContent = '';
    if (actionsDiv) actionsDiv.style.display = 'none';
}

// Paste Interception Module
let pendingPaste = null;

// Settings
let pasteGuardSettings = {
    pasteGuardEnabled: true,
    pasteBlockThreshold: 'HIGH',
    pasteAllowConvertToLink: true
};

// E2E mode flag (test-only)
let e2eMode = false;

// Load settings from storage
async function loadPasteGuardSettings() {
    const defaults = {
        pasteGuardEnabled: true,
        pasteBlockThreshold: 'HIGH',
        pasteAllowConvertToLink: true
    };
    const settings = await chrome.storage.sync.get(defaults);
    pasteGuardSettings = {
        pasteGuardEnabled: settings.pasteGuardEnabled ?? defaults.pasteGuardEnabled,
        pasteBlockThreshold: settings.pasteBlockThreshold ?? defaults.pasteBlockThreshold,
        pasteAllowConvertToLink: settings.pasteAllowConvertToLink ?? defaults.pasteAllowConvertToLink
    };
}

// Load E2E mode flag
async function loadE2EMode() {
    const storage = await chrome.storage.local.get('e2e_mode');
    e2eMode = storage.e2e_mode === true;
}

// Listen for settings changes
chrome.storage.onChanged.addListener((changes, areaName) => {
    if (areaName === 'sync') {
        if (changes.pasteGuardEnabled) {
            pasteGuardSettings.pasteGuardEnabled = changes.pasteGuardEnabled.newValue;
        }
        if (changes.pasteBlockThreshold) {
            pasteGuardSettings.pasteBlockThreshold = changes.pasteBlockThreshold.newValue;
        }
        if (changes.pasteAllowConvertToLink) {
            pasteGuardSettings.pasteAllowConvertToLink = changes.pasteAllowConvertToLink.newValue;
        }
    }
    if (areaName === 'local' && changes.e2e_mode) {
        e2eMode = changes.e2e_mode.newValue === true;
    }
});

// Load settings on initialization
loadPasteGuardSettings();
loadE2EMode();

// Build redacted text from original text and findings
function buildRedactedText(originalText, response) {
    // If backend provides redacted_text, use it
    if (response && response.redacted_text) {
        return response.redacted_text;
    }

    // Otherwise, build from findings
    if (!response || !response.findings || response.findings.length === 0) {
        return originalText;
    }

    const lines = originalText.split('\n');
    const linesToRedact = new Set();
    
    // Collect all line numbers that need redaction
    response.findings.forEach(finding => {
        if (finding.line_number && finding.line_number > 0) {
            // line_number is 1-indexed
            const lineIdx = finding.line_number - 1;
            if (lineIdx >= 0 && lineIdx < lines.length) {
                linesToRedact.add(lineIdx);
            }
        }
    });

    // Redact lines
    const redactedLines = lines.map((line, idx) => {
        if (linesToRedact.has(idx)) {
            return '[REDACTED]';
        }
        return line;
    });

    let redactedText = redactedLines.join('\n');

    // For HIGH risk, ensure redacted text is never equal to original
    const risk = (response && response.overall_risk) ? response.overall_risk.toLowerCase() : 'low';
    if (risk === 'high' && redactedText === originalText) {
        // If no lines were redacted, redact the entire text
        redactedText = '[REDACTED]';
    }

    return redactedText;
}

// Helper to insert text at cursor position in input/textarea
function setRangeText(element, text, start, end) {
    if (element.setRangeText) {
        element.setRangeText(text, start, end, 'end');
    } else {
        // Fallback for older browsers
        const value = element.value;
        const newValue = value.substring(0, start) + text + value.substring(end);
        element.value = newValue;
        element.setSelectionRange(start + text.length, start + text.length);
    }
    // Trigger input event to notify frameworks
    element.dispatchEvent(new Event('input', { bubbles: true }));
}

// Show temporary tooltip for error cases
function showTooltip(element, message) {
    const existing = document.getElementById('pg-paste-tooltip');
    if (existing) existing.remove();

    const tooltip = document.createElement('div');
    tooltip.id = 'pg-paste-tooltip';
    tooltip.setAttribute('data-testid', 'paste-tooltip');
    tooltip.textContent = message;
    tooltip.style.position = 'absolute';
    tooltip.style.backgroundColor = '#e74c3c';
    tooltip.style.color = 'white';
    tooltip.style.padding = '8px 12px';
    tooltip.style.borderRadius = '4px';
    tooltip.style.fontSize = '12px';
    tooltip.style.fontFamily = 'system-ui, sans-serif';
    tooltip.style.zIndex = '2147483647';
    tooltip.style.boxShadow = '0 2px 8px rgba(0,0,0,0.3)';
    tooltip.style.pointerEvents = 'none';
    tooltip.style.whiteSpace = 'nowrap';

    const rect = element.getBoundingClientRect();
    tooltip.style.left = rect.left + 'px';
    tooltip.style.top = (rect.top - 35) + 'px';

    document.body.appendChild(tooltip);

    setTimeout(() => {
        if (tooltip.parentNode) {
            tooltip.remove();
        }
    }, 3000);
}

// Show toast notification
function showToast(message) {
    const existing = document.getElementById('pg-toast');
    if (existing) existing.remove();

    const toast = document.createElement('div');
    toast.id = 'pg-toast';
    toast.setAttribute('data-testid', 'toast');
    toast.textContent = message;
    toast.style.position = 'fixed';
    toast.style.bottom = '20px';
    toast.style.left = '50%';
    toast.style.transform = 'translateX(-50%)';
    toast.style.backgroundColor = '#2ecc71';
    toast.style.color = 'white';
    toast.style.padding = '12px 24px';
    toast.style.borderRadius = '6px';
    toast.style.fontSize = '14px';
    toast.style.fontFamily = 'system-ui, -apple-system, sans-serif';
    toast.style.zIndex = '2147483648';
    toast.style.boxShadow = '0 4px 12px rgba(0,0,0,0.3)';
    toast.style.pointerEvents = 'none';
    toast.style.whiteSpace = 'nowrap';

    document.body.appendChild(toast);

    setTimeout(() => {
        if (toast.parentNode) {
            toast.style.opacity = '0';
            toast.style.transition = 'opacity 0.3s';
            setTimeout(() => {
                if (toast.parentNode) {
                    toast.remove();
                }
            }, 300);
        }
    }, 3000);
}

// Show modal for blocked paste
function showPasteBlockModal(element, response) {
    // Remove existing modal if any
    const existing = document.getElementById('pg-paste-modal');
    if (existing) existing.remove();

    const risk = (response && response.overall_risk) ? response.overall_risk.toLowerCase() : 'medium';
    const riskLevel = risk.toUpperCase();
    const rationale = response.risk_rationale || 'Sensitive content detected';
    const findings = (response.findings || []).slice(0, 3); // Top 3 findings
    const originalText = pendingPaste ? pendingPaste.text : '';

    // Create overlay
    const overlay = document.createElement('div');
    overlay.id = 'pg-paste-modal';
    overlay.setAttribute('data-testid', 'paste-modal');
    overlay.style.position = 'fixed';
    overlay.style.top = '0';
    overlay.style.left = '0';
    overlay.style.width = '100%';
    overlay.style.height = '100%';
    overlay.style.backgroundColor = 'rgba(0, 0, 0, 0.6)';
    overlay.style.zIndex = '2147483647';
    overlay.style.display = 'flex';
    overlay.style.alignItems = 'center';
    overlay.style.justifyContent = 'center';
    overlay.style.fontFamily = 'system-ui, -apple-system, sans-serif';

    // Create modal
    const modal = document.createElement('div');
    modal.style.backgroundColor = 'white';
    modal.style.borderRadius = '8px';
    modal.style.padding = '24px';
    modal.style.maxWidth = '500px';
    modal.style.width = '90%';
    modal.style.maxHeight = '80vh';
    modal.style.overflowY = 'auto';
    modal.style.boxShadow = '0 10px 40px rgba(0,0,0,0.3)';
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-labelledby', 'pg-paste-modal-title');
    modal.setAttribute('aria-modal', 'true');

    // Title
    const title = document.createElement('h2');
    title.id = 'pg-paste-modal-title';
    title.textContent = 'Sensitive paste blocked';
    title.style.margin = '0 0 16px 0';
    title.style.fontSize = '20px';
    title.style.fontWeight = '600';
    title.style.color = '#1a1a1a';

    // Risk level and rationale
    const riskInfo = document.createElement('div');
    riskInfo.style.marginBottom = '20px';
    riskInfo.style.padding = '12px';
    riskInfo.style.backgroundColor = risk === 'high' ? '#fee' : '#fff4e6';
    riskInfo.style.borderLeft = `4px solid ${risk === 'high' ? '#e74c3c' : '#f39c12'}`;
    riskInfo.style.borderRadius = '4px';

    const riskLabel = document.createElement('div');
    riskLabel.textContent = `Risk Level: ${riskLevel}`;
    riskLabel.style.fontWeight = '600';
    riskLabel.style.marginBottom = '8px';
    riskLabel.style.color = risk === 'high' ? '#c0392b' : '#d68910';

    const rationaleText = document.createElement('div');
    rationaleText.textContent = rationale;
    rationaleText.style.fontSize = '14px';
    rationaleText.style.color = '#555';

    riskInfo.appendChild(riskLabel);
    riskInfo.appendChild(rationaleText);

    // Findings section
    let findingsSection = null;
    if (findings.length > 0) {
        findingsSection = document.createElement('div');
        findingsSection.style.marginBottom = '20px';

        const findingsTitle = document.createElement('div');
        findingsTitle.textContent = 'Findings:';
        findingsTitle.style.fontWeight = '600';
        findingsTitle.style.marginBottom = '12px';
        findingsTitle.style.fontSize = '14px';
        findingsTitle.style.color = '#333';
        findingsSection.appendChild(findingsTitle);

        findings.forEach((finding, idx) => {
            const findingItem = document.createElement('div');
            findingItem.style.marginBottom = '12px';
            findingItem.style.padding = '10px';
            findingItem.style.backgroundColor = '#f8f9fa';
            findingItem.style.borderRadius = '4px';
            findingItem.style.fontSize = '13px';

            const typeSeverity = document.createElement('div');
            typeSeverity.style.fontWeight = '600';
            typeSeverity.style.marginBottom = '4px';
            typeSeverity.style.color = '#333';
            typeSeverity.textContent = `${finding.type || 'Unknown'} (${(finding.severity || 'medium').toUpperCase()})`;
            findingItem.appendChild(typeSeverity);

            const reason = document.createElement('div');
            reason.style.color = '#666';
            reason.style.fontSize = '12px';
            reason.textContent = finding.reason || 'No reason provided';
            findingItem.appendChild(reason);

            findingsSection.appendChild(findingItem);
        });
    }

    // Redaction note
    const redactionNote = document.createElement('div');
    redactionNote.textContent = 'Redaction is best-effort.';
    redactionNote.style.fontSize = '12px';
    redactionNote.style.color = '#666';
    redactionNote.style.fontStyle = 'italic';
    redactionNote.style.marginTop = '16px';
    redactionNote.style.marginBottom = '8px';

    // Error message area (initially hidden)
    const errorContainer = document.createElement('div');
    errorContainer.id = 'pg-modal-error';
    errorContainer.style.display = 'none';
    errorContainer.style.marginTop = '16px';
    errorContainer.style.padding = '12px';
    errorContainer.style.backgroundColor = '#fee';
    errorContainer.style.borderLeft = '4px solid #e74c3c';
    errorContainer.style.borderRadius = '4px';
    errorContainer.style.color = '#c0392b';
    errorContainer.style.fontSize = '13px';

    // Buttons container
    const buttonsContainer = document.createElement('div');
    buttonsContainer.style.display = 'flex';
    buttonsContainer.style.gap = '12px';
    buttonsContainer.style.justifyContent = 'flex-end';
    buttonsContainer.style.marginTop = '24px';
    buttonsContainer.style.flexWrap = 'wrap';

    const cancelBtn = document.createElement('button');
    cancelBtn.setAttribute('data-testid', 'paste-cancel');
    cancelBtn.textContent = 'Cancel';
    cancelBtn.style.padding = '10px 20px';
    cancelBtn.style.border = '1px solid #ddd';
    cancelBtn.style.borderRadius = '4px';
    cancelBtn.style.backgroundColor = 'white';
    cancelBtn.style.color = '#333';
    cancelBtn.style.cursor = 'pointer';
    cancelBtn.style.fontSize = '14px';
    cancelBtn.style.fontWeight = '500';
    cancelBtn.setAttribute('type', 'button');

    const pasteRedactedBtn = document.createElement('button');
    pasteRedactedBtn.setAttribute('data-testid', 'paste-redacted');
    pasteRedactedBtn.textContent = 'Paste redacted';
    pasteRedactedBtn.style.padding = '10px 20px';
    pasteRedactedBtn.style.border = 'none';
    pasteRedactedBtn.style.borderRadius = '4px';
    pasteRedactedBtn.style.backgroundColor = '#f39c12';
    pasteRedactedBtn.style.color = 'white';
    pasteRedactedBtn.style.cursor = 'pointer';
    pasteRedactedBtn.style.fontSize = '14px';
    pasteRedactedBtn.style.fontWeight = '500';
    pasteRedactedBtn.setAttribute('type', 'button');

    const pasteBtn = document.createElement('button');
    pasteBtn.setAttribute('data-testid', 'paste-anyway');
    pasteBtn.textContent = 'Paste anyway';
    pasteBtn.style.padding = '10px 20px';
    pasteBtn.style.border = 'none';
    pasteBtn.style.borderRadius = '4px';
    pasteBtn.style.backgroundColor = '#3498db';
    pasteBtn.style.color = 'white';
    pasteBtn.style.cursor = 'pointer';
    pasteBtn.style.fontSize = '14px';
    pasteBtn.style.fontWeight = '500';
    pasteBtn.setAttribute('type', 'button');

    // Convert to Secure Share link button (only for HIGH, optionally MEDIUM)
    const convertBtn = document.createElement('button');
    convertBtn.setAttribute('data-testid', 'paste-convert-link');
    convertBtn.textContent = 'Convert to Secure Share link';
    convertBtn.style.padding = '10px 20px';
    convertBtn.style.border = 'none';
    convertBtn.style.borderRadius = '4px';
    convertBtn.style.backgroundColor = '#27ae60';
    convertBtn.style.color = 'white';
    convertBtn.style.cursor = 'pointer';
    convertBtn.style.fontSize = '14px';
    convertBtn.style.fontWeight = '500';
    convertBtn.setAttribute('type', 'button');
    
    // Only show for HIGH risk (and optionally MEDIUM) and if setting allows
    const showConvertBtn = pasteGuardSettings.pasteAllowConvertToLink && (risk === 'high' || risk === 'medium');
    if (showConvertBtn) {
        buttonsContainer.appendChild(convertBtn);
    }
    
    buttonsContainer.appendChild(cancelBtn);
    buttonsContainer.appendChild(pasteRedactedBtn);
    buttonsContainer.appendChild(pasteBtn);

    // Assemble modal
    modal.appendChild(title);
    modal.appendChild(riskInfo);
    if (findingsSection) {
        modal.appendChild(findingsSection);
    }
    modal.appendChild(redactionNote);
    modal.appendChild(errorContainer);
    modal.appendChild(buttonsContainer);

    overlay.appendChild(modal);
    document.body.appendChild(overlay);

    // Focus trap: get all focusable elements
    const focusableElements = modal.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
    const firstFocusable = focusableElements[0];
    const lastFocusable = focusableElements[focusableElements.length - 1];

    // Focus first button
    setTimeout(() => firstFocusable.focus(), 0);

    // Tab key trap
    const handleTabKey = (e) => {
        if (e.key !== 'Tab') return;

        if (e.shiftKey) {
            if (document.activeElement === firstFocusable) {
                e.preventDefault();
                lastFocusable.focus();
            }
        } else {
            if (document.activeElement === lastFocusable) {
                e.preventDefault();
                firstFocusable.focus();
            }
        }
    };

    // Close function
    const closeModal = (mode) => {
        overlay.remove();
        document.removeEventListener('keydown', handleEscKey);
        document.removeEventListener('keydown', handleTabKey);
        
        let textToPaste = null;
        if (mode === 'original' && pendingPaste) {
            textToPaste = pendingPaste.text;
        } else if (mode === 'redacted' && pendingPaste) {
            textToPaste = buildRedactedText(pendingPaste.text, response);
        }
        
        // Paste text if requested (setRangeText handles cursor positioning)
        if (textToPaste !== null && pendingPaste) {
            setRangeText(pendingPaste.element, textToPaste, pendingPaste.selectionStart, pendingPaste.selectionEnd);
        }
        
        // Return focus to original input
        if (element && typeof element.focus === 'function') {
            try {
                element.focus();
                // If we didn't paste, restore original caret position
                if (mode === 'cancel' && pendingPaste && typeof element.setSelectionRange === 'function') {
                    element.setSelectionRange(pendingPaste.selectionStart, pendingPaste.selectionEnd);
                }
            } catch (e) {
                // Ignore focus errors (e.g., element removed from DOM)
            }
        }
        
        pendingPaste = null;
    };

    // ESC key handler
    const handleEscKey = (e) => {
        if (e.key === 'Escape') {
            e.preventDefault();
            closeModal('cancel');
        }
    };

    // Convert button handler
    if (showConvertBtn) {
        convertBtn.addEventListener('click', () => {
            // Hide any existing error
            errorContainer.style.display = 'none';
            errorContainer.textContent = '';
            
            // Disable button during upload
            convertBtn.disabled = true;
            convertBtn.textContent = 'Uploading...';
            convertBtn.style.opacity = '0.6';
            convertBtn.style.cursor = 'not-allowed';
            
            // Upload text as secure file
            chrome.runtime.sendMessage({
                type: 'UPLOAD_SECURE_TEXT',
                filename: 'secure-paste.txt',
                text: originalText,
                label: `paste:${risk}`
            }, (uploadResponse) => {
                // Re-enable button
                convertBtn.disabled = false;
                convertBtn.textContent = 'Convert to Secure Share link';
                convertBtn.style.opacity = '1';
                convertBtn.style.cursor = 'pointer';
                
                if (chrome.runtime.lastError) {
                    const errorMsg = `Upload failed: ${chrome.runtime.lastError.message}`;
                    errorContainer.textContent = errorMsg;
                    errorContainer.style.display = 'block';
                    showToast(errorMsg);
                    return;
                }
                
                if (uploadResponse && uploadResponse.error) {
                    const errorMsg = `Upload failed: ${uploadResponse.error}`;
                    errorContainer.textContent = errorMsg;
                    errorContainer.style.display = 'block';
                    showToast(errorMsg);
                    return;
                }
                
                if (!uploadResponse || !uploadResponse.viewLink) {
                    const errorMsg = 'Upload failed: No view link received';
                    errorContainer.textContent = errorMsg;
                    errorContainer.style.display = 'block';
                    showToast(errorMsg);
                    return;
                }
                
                // Success: insert view link
                const viewLink = uploadResponse.viewLink;
                setRangeText(element, viewLink, pendingPaste.selectionStart, pendingPaste.selectionEnd);
                
                // Close modal
                closeModal('cancel');
                
                // Show toast
                showToast('Secure one-time link inserted');
            });
        });
    }
    
    // Event listeners
    cancelBtn.addEventListener('click', () => closeModal('cancel'));
    pasteRedactedBtn.addEventListener('click', () => closeModal('redacted'));
    pasteBtn.addEventListener('click', () => closeModal('original'));
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) {
            closeModal('cancel');
        }
    });
    document.addEventListener('keydown', handleEscKey);
    document.addEventListener('keydown', handleTabKey);
}

// Check if element is a valid target for paste interception
function isValidPasteTarget(element) {
    if (!element) return false;
    
    if (element.tagName === 'TEXTAREA') {
        return true;
    }
    
    if (element.tagName === 'INPUT') {
        const validTypes = ['text', 'search', 'email', 'url', 'tel', 'password'];
        return validTypes.includes(element.type.toLowerCase());
    }
    
    return false;
}

// Shared paste handling logic (used by both real paste events and E2E bridge)
function handlePasteText(element, text) {
    if (!isValidPasteTarget(element)) {
        return; // Not a target we care about
    }
    
    // Check if paste guard is enabled
    if (!pasteGuardSettings.pasteGuardEnabled) {
        // Paste guard disabled, insert text directly
        setRangeText(element, text, element.selectionStart || 0, element.selectionEnd || 0);
        element.dispatchEvent(new Event("input", { bubbles: true }));
        return;
    }
    
    // Store context
    const selectionStart = element.selectionStart || 0;
    const selectionEnd = element.selectionEnd || 0;
    
    pendingPaste = {
        element: element,
        text: text,
        selectionStart: selectionStart,
        selectionEnd: selectionEnd,
        timestamp: Date.now()
    };
    
    // Send to background for analysis
    chrome.runtime.sendMessage({
        type: 'ANALYZE_TEXT',
        text: text
    }, (response) => {
        if (chrome.runtime.lastError) {
            // Background script error - fail open
            console.error('Privacy Guardrail: Analysis failed', chrome.runtime.lastError);
            if (pendingPaste && pendingPaste.element === element) {
                setRangeText(pendingPaste.element, pendingPaste.text, pendingPaste.selectionStart, pendingPaste.selectionEnd);
                showTooltip(pendingPaste.element, 'Analyzer offline');
                pendingPaste = null;
            }
            return;
        }
        
        if (response && response.error) {
            // Analysis error - fail open
            console.error('Privacy Guardrail: Analysis error', response.error);
            if (pendingPaste && pendingPaste.element === element) {
                setRangeText(pendingPaste.element, pendingPaste.text, pendingPaste.selectionStart, pendingPaste.selectionEnd);
                showTooltip(pendingPaste.element, 'Analyzer offline');
                pendingPaste = null;
            }
            return;
        }
        
        // Handle analysis result
        const risk = (response && response.overall_risk) ? response.overall_risk.toLowerCase() : 'low';
        
        if (risk === 'low') {
            // Allow paste
            if (pendingPaste && pendingPaste.element === element) {
                setRangeText(pendingPaste.element, pendingPaste.text, pendingPaste.selectionStart, pendingPaste.selectionEnd);
                pendingPaste = null;
            }
        } else {
            // Check if risk level should be blocked based on threshold setting
            const shouldBlock = pasteGuardSettings.pasteBlockThreshold === 'HIGH_MEDIUM' || 
                            (pasteGuardSettings.pasteBlockThreshold === 'HIGH' && risk === 'high');
            
            if (shouldBlock) {
                // Block paste - show modal
                if (pendingPaste && pendingPaste.element === element) {
                    // Notify background of blocked paste
                    chrome.runtime.sendMessage({ type: 'PASTE_BLOCKED' }, () => {
                        // Ignore response/errors
                    });
                    showPasteBlockModal(pendingPaste.element, response);
                    // Don't clear pendingPaste here - modal will handle it
                }
            } else {
                // Risk level not blocked by threshold - allow paste
                if (pendingPaste && pendingPaste.element === element) {
                    setRangeText(pendingPaste.element, pendingPaste.text, pendingPaste.selectionStart, pendingPaste.selectionEnd);
                    pendingPaste = null;
                }
            }
        }
    });
}

// Handle paste event
document.addEventListener('paste', (e) => {
    const target = e.target;
    
    if (!isValidPasteTarget(target)) {
        return; // Not a target we care about
    }
    
    // Get clipboard text
    const clipboardData = e.clipboardData || window.clipboardData;
    if (!clipboardData) return;
    
    const pastedText = clipboardData.getData('text');
    if (!pastedText) return;
    
    // Prevent default paste for now
    e.preventDefault();
    e.stopPropagation();
    
    // Use shared paste handling logic
    handlePasteText(target, pastedText);
}, true); // Use capture phase

// E2E bridge handlers (test-only, gated by e2e_mode)
async function handleE2EInit(payload) {
    // Write E2E test overrides to storage
    const defaults = {
        e2e_mode: true,
        e2e_malicious_domains: ["evil.test"],
        pasteGuardEnabled: payload?.pasteGuardEnabled ?? true,
        pasteBlockThreshold: payload?.pasteBlockThreshold ?? 'HIGH',
        pasteAllowConvertToLink: payload?.pasteAllowConvertToLink ?? true
    };
    
    await chrome.storage.local.set({
        e2e_mode: defaults.e2e_mode,
        e2e_malicious_domains: defaults.e2e_malicious_domains
    });
    
    await chrome.storage.sync.set({
        pasteGuardEnabled: defaults.pasteGuardEnabled,
        pasteBlockThreshold: defaults.pasteBlockThreshold,
        pasteAllowConvertToLink: defaults.pasteAllowConvertToLink
    });
    
    // Reload settings from storage
    await loadPasteGuardSettings();
    await loadE2EMode();
    
    // Acknowledge
    window.postMessage({ source: "pg-e2e", type: "INIT_ACK" }, "*");
}

async function handleE2EReset() {
    // Clear all storage
    await chrome.storage.local.clear();
    await chrome.storage.sync.clear();
    
    // Re-apply defaults
    await handleE2EInit({});
    
    // Acknowledge
    window.postMessage({ source: "pg-e2e", type: "RESET_ACK" }, "*");
}

async function handleE2EGetState() {
    const local = await chrome.storage.local.get(['e2e_mode', 'e2e_malicious_domains']);
    const sync = await chrome.storage.sync.get(['pasteGuardEnabled', 'pasteBlockThreshold', 'pasteAllowConvertToLink']);
    
    window.postMessage({
        source: "pg-e2e",
        type: "GET_STATE_ACK",
        payload: { local, sync }
    }, "*");
}

// E2E bridge (test-only, gated by e2e_mode)
window.addEventListener('message', async (ev) => {
    // Only accept E2E messages when e2e_mode is enabled (or during INIT)
    if (ev.data && ev.data.source === 'pg-e2e') {
        if (ev.data.type === 'INIT') {
            // INIT can run even if e2e_mode not set yet (bootstrap)
            await handleE2EInit(ev.data.payload || {});
            return;
        }
        
        // Other messages require e2e_mode
        if (!e2eMode) return;
        
        if (ev.data.type === 'PASTE' && typeof ev.data.text === 'string' && typeof ev.data.selector === 'string') {
            const element = document.querySelector(ev.data.selector);
            if (!element) {
                console.warn('E2E paste: element not found:', ev.data.selector);
                return;
            }
            
            // Focus the element
            element.focus();
            if (element.click) element.click();
            
            // Use shared paste handling logic
            handlePasteText(element, ev.data.text);
        } else if (ev.data.type === 'RESET') {
            await handleE2EReset();
        } else if (ev.data.type === 'GET_STATE') {
            await handleE2EGetState();
        }
    }
});

// Inject content script loaded marker for E2E tests
function injectContentScriptMarker() {
	// Use meta tag approach - stable and safe
	if (document.querySelector('meta[data-testid="pg-content-loaded"]')) {
		return; // Already injected
	}
	const marker = document.createElement('meta');
	marker.setAttribute('data-testid', 'pg-content-loaded');
	marker.setAttribute('content', '1');
	document.head.appendChild(marker);
}

// Inject extension identity markers on documentElement for E2E tests
// This MUST run early and exactly once per page load
function injectExtensionIdentity() {
	// Ensure documentElement exists (should always be true for content scripts)
	if (!document.documentElement) {
		// Fallback: wait for DOM if somehow called too early
		if (document.readyState === "loading") {
			document.addEventListener("DOMContentLoaded", injectExtensionIdentity, { once: true });
			return;
		}
		return;
	}
	
	// Set deterministic markers on documentElement (runs immediately)
	document.documentElement.setAttribute("data-pg-loaded", "1");
	document.documentElement.setAttribute("data-pg-extension-id", chrome.runtime.id);
}

// Initial Run - markers MUST be injected first
console.log("Privacy Guardrail: Initializing...");
injectContentScriptMarker(); // Inject marker first for E2E tests
injectExtensionIdentity(); // Inject extension identity markers (CRITICAL for E2E)
injectStatusBadge();
checkCurrentPage(); // Check the page itself
processNode(document.body);

// Performance-optimized Observer
const observer = new MutationObserver((mutations) => {
    for (const mutation of mutations) {
        if (mutation.type === 'childList') {
            mutation.addedNodes.forEach(processNode);
        }
    }
});

observer.observe(document.body, { childList: true, subtree: true });
console.log("Privacy Guardrail: Active and observing mutations.");

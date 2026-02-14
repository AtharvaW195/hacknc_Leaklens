// Link Validation Module
// Malicious Domains List - Loaded from external JSON
let MALICIOUS_DOMAINS = [];

// Load the malicious domains list, then re-scan page and re-check current URL
fetch(chrome.runtime.getURL('malicious_domains.json'))
    .then(response => response.json())
    .then(data => {
        MALICIOUS_DOMAINS = data;
        console.log("Privacy Guardrail: Loaded malicious domains list.", MALICIOUS_DOMAINS);
        rescanPageAfterListLoaded();
        checkCurrentPage();
    })
    .catch(error => console.error("Privacy Guardrail: Failed to load malicious list.", error));

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
            <h1 style="color: #c0392b; margin-top: 0;">⚠️ Site Blocked</h1>
            <p style="font-size: 18px;"><b>${domain}</b> is on the blocklist.</p>
            <p>Access to this site has been restricted by Privacy Guardrail.</p>
            <button id="pg-proceed-btn" style="background: #95a5a6; color: white; border: none; padding: 10px 20px; border-radius: 5px; cursor: pointer; font-size: 14px; margin-top: 20px;">
                Proceed Anyway (Unsafe)
            </button>
        </div>
    `;

    document.documentElement.appendChild(overlay);

    document.getElementById('pg-proceed-btn').addEventListener('click', () => {
        overlay.remove();
    });
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

// Initial Run
console.log("Privacy Guardrail: Initializing...");
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

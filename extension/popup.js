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

# Project Status Summary

## Overview
**Privacy Guardrail** is a Chrome extension paired with a Go backend that enhances browser privacy and security. It actively scans for malicious links, blocks known bad domains, and provides a "Secure Share" mechanism to upload files securely to S3 with one-time view links.

## Components & Status

### 1. Chrome Extension (`/extension`)
*   **Version**: 1.0 (Manifest V3)
*   **Permissions**: `activeTab`, `scripting`, `storage`, `tabs`, `<all_urls>`
*   **Key Features**:
    *   **Malicious Domain Blocking**:
        *   Loads a list of blocked domains from `malicious_domains.json`.
        *   **Blocking Overlay**: Immediately blocks access to any site matching the list with a "Site Blocked" red modal.
        *   **Link Scanning**: Scans all `<a>` tags on visited pages; flags links to bad domains with a "⚠️" icon and confirmation dialog.
    *   **Secure Upload**:
        *   **In-Page Injection**: Automatically injects a "🔒 Secure Share" button next to standard file inputs (`<input type="file">`).
        *   **Popup UI**: The extension popup also includes a drag-and-drop zone for secure uploads.
        *   **Functionality**: Uploads files to the backend S3 proxy, generates a one-time view link, and copies it to the clipboard.
    *   **Architecture**:
        *   **Content Script**: Handles page scanning, UI injection, and blocking. Offloads file uploads to the Service Worker via `chrome.runtime.sendMessage`.
        *   **Service Worker (`background.js`)**: Listens for upload messages and performs the actual POST request to the backend.
        *   **Popup (`popup.js`)**: Independent UI for uploads and link checking.

### 2. Backend Server (`/backend`)
*   **Language**: Go
*   **Status**: Complete & Logical
*   **Endpoints**:
    *   `POST /api/upload`: Accepts `multipart/form-data` file uploads, saves to S3, and returns a one-time link. (Used by Extension)
    *   `POST /api/generate-upload-url`: Generates a presigned S3 PUT URL for direct client-side uploads.
    *   `GET /view/{fileId}`: Validates if a file has been viewed. If not, redirects to a short-lived presigned S3 GET URL. Enforces **One-Time Access**.
*   **Storage**: ephemeral memory map for metadata (for demo); AWS S3 for actual files.

### 3. CLI Tool (`/detector`)
*   **Language**: Go
*   **Status**: Existing (Not recently modified)
*   **Functionality**: Scans text input for secrets (API keys, PEM files, etc.).

## Recent Changes
1.  **Fixed Malicious Domain List**: Moved from hardcoded regex to a dynamic `malicious_domains.json` file.
2.  **Web Accessible Resources**: Updated `manifest.json` to allow content scripts to load the domain list.
3.  **Blocking Overlay**: Implemented a mandatory blocking screen for users who visit malicious domains.
4.  **Upload Refactoring**: Standardized upload logic to use the backend's `/api/upload` endpoint via the Background Service Worker.

## Next Steps / Known Considerations
*   **Backend Persistence**: The backend currently stores file metadata in memory (`viewStore`). Restarting the server wipes the "viewed" status and file mapping (though files remain in S3). For production, this should be moved to a DB (Redis/Postgres).
*   **S3 Configuration**: Ensure `AWS_BUCKET_NAME` and credentials are correctly set in `backend/.env`.

# Privacy Guardrail & Pasteguard

## Project Components
- **Pasteguard CLI**: Go-based tool for detecting secrets in text (located in root).
- **Chrome Extension**: Privacy guardrail for browser interactions.

## Chrome Extension
Located in the `extension/` directory.

### Features
- **Link Validation**: Flags malicious links with a warning icon and confirmation dialog.
- **Secure Upload**: Intercepts file uploads to offer a secure sharing option.

### Setup
1. Open Chrome and go to `chrome://extensions`.
2. Enable "Developer mode".
3. Click "Load unpacked" and select the `extension` folder inside this directory.
4. Open `extension/test.html` in Chrome to verify functionality.

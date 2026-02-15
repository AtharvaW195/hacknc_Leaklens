# Browser Extension Setup & Management Guide

This guide covers how to load, manage, and verify the Privacy Guardrail browser extension.

---

## 🎯 Quick Start (30 seconds)

1. **Open:** `chrome://extensions/` (or `edge://extensions/`)
2. **Enable:** Developer mode toggle (top-right)
3. **Click:** "Load unpacked"
4. **Select:** `C:\Users\anand\Development\hacknc\extension` folder
5. **Done!** Extension is now loaded

**Verify it's working:**
- Open any webpage
- Press **F12** → Console tab
- Look for: `Privacy Guardrail: Initializing...` ✅

---

## 📦 Loading the Extension

### Step 1: Open Extensions Page

**Chrome:**
- Navigate to: `chrome://extensions/`
- Or: Menu (⋮) → Extensions → Manage Extensions

**Edge:**
- Navigate to: `edge://extensions/`
- Or: Menu (⋯) → Extensions → Manage Extensions

### Step 2: Enable Developer Mode

1. Look for **"Developer mode"** toggle in the top-right corner
2. **Turn it ON** (toggle should be blue/enabled)
3. This enables the "Load unpacked" button

### Step 3: Load the Extension

1. Click **"Load unpacked"** button (appears after enabling Developer mode)
2. Navigate to your project folder: `C:\Users\anand\Development\hacknc\extension`
3. Select the `extension` folder (not files inside it, the folder itself)
4. Click **"Select Folder"** or **"Select"**

**Expected Result:**
- Extension appears in your extensions list
- Shows "Privacy Guardrail" with version 1.0
- Status should show as "Enabled"

---

## ✅ Verifying Extension is Active

### Method 1: Check Extensions Page

1. Go to `chrome://extensions/` (or `edge://extensions/`)
2. Find "Privacy Guardrail" in the list
3. Check:
   - ✅ **Toggle is ON** (blue/enabled)
   - ✅ **No errors** shown in red
   - ✅ **Status** shows "Enabled"

### Method 2: Check Browser Console

1. Open any webpage (e.g., `https://example.com`)
2. Press **F12** to open Developer Tools
3. Go to **Console** tab
4. Look for messages starting with:
   ```
   Privacy Guardrail: Initializing...
   Privacy Guardrail: Loaded malicious domains list.
   Privacy Guardrail: Active and observing mutations.
   ```

**If you see these messages:** ✅ Extension is working!

### Method 3: Visual Indicators

1. Visit any webpage
2. Look for a small badge/icon (usually in bottom-right corner)
   - This is the "status badge" injected by the extension
   - Indicates the extension is active

### Method 4: Test Paste Interception

1. Go to any webpage with an input field
2. Open Developer Tools (F12) → Console tab
3. Try pasting a secret: `password = "secret123"`
4. **Expected:**
   - Paste is blocked
   - Tooltip appears: "Sensitive paste blocked (HIGH)"
   - Console shows: `Privacy Guardrail: Analysis failed` (if server not running) or analysis messages

---

## 🔄 Managing the Extension

### Reload After Code Changes

**After editing extension files (`content.js`, `background.js`, etc.):**

1. Go to `chrome://extensions/`
2. Find "Privacy Guardrail"
3. Click the **🔄 Refresh icon** (circular arrow) on the extension card
4. Extension reloads with your changes

**Or:**
- Toggle the extension OFF and ON again

### Disable/Enable Extension

1. Go to `chrome://extensions/`
2. Find "Privacy Guardrail"
3. Toggle the switch to:
   - **OFF** (gray) = Extension disabled
   - **ON** (blue) = Extension enabled

### Remove Extension

1. Go to `chrome://extensions/`
2. Find "Privacy Guardrail"
3. Click **"Remove"**
4. Confirm removal

**Note:** This removes the extension. You'll need to reload it using "Load unpacked" again.

### View Extension Details

1. Go to `chrome://extensions/`
2. Find "Privacy Guardrail"
3. Click **"Details"** button
4. See:
   - Extension ID
   - Permissions
   - Site access
   - Options (if available)

---

## 🐛 Troubleshooting

### Extension Not Appearing

**Problem:** Extension doesn't show up after "Load unpacked"

**Solutions:**
1. Make sure you selected the `extension` **folder**, not files inside it
2. Check that `manifest.json` exists in the extension folder
3. Look for errors in red on the extensions page
4. Try removing and reloading

### Extension Shows Errors

**Problem:** Red error message on extensions page

**Common Causes:**
1. **Missing files:** Check that all files exist:
   - `manifest.json`
   - `content.js`
   - `background.js`
   - `popup.html` (optional)
   - `popup.js` (optional)
   - `styles.css` (optional)
   - `malicious_domains.json`

2. **Syntax errors:** Check browser console for JavaScript errors
   - Press F12 → Console tab
   - Look for red error messages

3. **Manifest errors:** Check `manifest.json` syntax
   - Valid JSON required
   - All required fields present

### Extension Not Working

**Problem:** Extension loads but doesn't do anything

**Checklist:**
1. ✅ Extension is **enabled** (toggle ON)
2. ✅ No errors in extensions page
3. ✅ No errors in browser console (F12)
4. ✅ Server is running (for paste interception)
5. ✅ Refresh the webpage after loading extension

**Debug Steps:**
1. Open browser console (F12)
2. Go to Console tab
3. Look for "Privacy Guardrail" messages
4. Check for any red error messages
5. Try reloading the extension

### Paste Interception Not Working

**Problem:** Can paste secrets without blocking

**Check:**
1. ✅ Server running on `http://localhost:8080`
2. ✅ Test server: `Invoke-RestMethod -Uri http://localhost:8080/health`
3. ✅ Extension loaded and enabled
4. ✅ Input type is valid (text, email, password, textarea, etc.)
5. ✅ Check browser console for errors

**Test:**
```powershell
# Test server
Invoke-RestMethod -Uri http://localhost:8080/health

# Should return: {"status":"ok"}
```

### Extension Icon Not Visible

**Problem:** Can't find extension icon in toolbar

**Solution:**
1. Go to `chrome://extensions/`
2. Find "Privacy Guardrail"
3. Look for the **puzzle piece icon** (🧩) in toolbar
4. Click it to see all extensions
5. Click the **pin icon** 📌 to pin Privacy Guardrail to toolbar
6. Now you can click the extension icon to open the popup

**Extension Popup:**
- Click the extension icon in toolbar
- Opens `popup.html` with extension status and features
- Shows if extension is active

---

## 🔍 Extension Status Indicators

### Visual Indicators on Webpages

1. **Status Badge:**
   - Small badge/icon injected on pages
   - Indicates extension is active
   - May show alert state if malicious links found

2. **Warning Icons:**
   - ⚠️ Icons next to malicious links
   - Red warning indicators

3. **Secure Share Button:**
   - 🔒 "Secure Share" button next to file inputs
   - Appears when extension detects file upload fields

### Console Messages

**Normal operation messages:**
```
Privacy Guardrail: Initializing...
Privacy Guardrail: Loaded malicious domains list.
Privacy Guardrail: Active and observing mutations.
```

**Error messages:**
```
Privacy Guardrail: Failed to load malicious list. [error]
Privacy Guardrail: Analysis failed [error]
```

---

## 📋 Quick Reference

### Extension Files Location
```
C:\Users\anand\Development\hacknc\extension\
├── manifest.json          (Extension configuration)
├── content.js            (Main content script - paste interception)
├── background.js         (Background service worker)
├── popup.html            (Extension popup UI)
├── popup.js              (Popup functionality)
├── styles.css            (Styling)
└── malicious_domains.json (Blocklist)
```

### Extension Permissions

The extension requests:
- `activeTab` - Access to active tab
- `scripting` - Inject scripts
- `storage` - Store data
- `tabs` - Access tab information
- `<all_urls>` - Access all websites
- `http://localhost:8080/*` - Access to local server

### Extension ID

After loading, the extension gets a unique ID. You can find it:
1. Go to `chrome://extensions/`
2. Enable "Developer mode"
3. Find "Privacy Guardrail"
4. Click "Details"
5. See "ID" field

---

## 🧪 Testing Checklist

After loading the extension, verify:

- [ ] Extension appears in `chrome://extensions/`
- [ ] Extension is enabled (toggle ON)
- [ ] No red errors on extensions page
- [ ] Console shows "Privacy Guardrail: Initializing..."
- [ ] Status badge appears on webpages
- [ ] Paste interception works (with server running)
- [ ] Link scanning works (malicious links show warnings)
- [ ] Domain blocking works (blocked domains show overlay)
- [ ] Secure Share button appears on file inputs

---

## 💡 Tips

1. **Keep Developer Mode ON** while developing
   - Makes reloading easier
   - Shows more debugging info

2. **Check Console Regularly**
   - Press F12 on any webpage
   - Look for extension messages
   - Watch for errors

3. **Reload After Changes**
   - Always reload extension after editing files
   - Refresh webpage to see content script changes

4. **Test on Multiple Sites**
   - Extension works on all websites
   - Test paste interception on different input types

5. **Server Must Be Running**
   - Paste interception requires server on `localhost:8080`
   - Other features (link scanning, domain blocking) work without server

---

## 🆘 Still Having Issues?

1. **Check browser console** (F12 → Console)
2. **Check extension errors** (`chrome://extensions/`)
3. **Verify server is running** (`http://localhost:8080/health`)
4. **Reload extension** (refresh icon)
5. **Restart browser** (sometimes helps)
6. **Remove and reload extension** (clean start)

---

For server setup, see [STARTUP_AND_TESTING.md](STARTUP_AND_TESTING.md)


const path = require("path");
const os = require("os");
const fs = require("fs");
const { chromium } = require("@playwright/test");

// Ensure userDataDir directory exists
const userDataBaseDir = path.join(__dirname, ".pw-user-data");
if (!fs.existsSync(userDataBaseDir)) {
  fs.mkdirSync(userDataBaseDir, { recursive: true });
}

async function launchWithExtension(extensionPath) {
  // Validate extension path exists and contains manifest.json
  if (!extensionPath) {
    throw new Error(
      "E2E_EXTENSION_PATH is not set. " +
      "This should be set by global-setup.js. " +
      "Check that prepareExtension() is working correctly."
    );
  }

  const manifestPath = path.join(extensionPath, "manifest.json");
  if (!fs.existsSync(extensionPath)) {
    throw new Error(
      `Extension path does not exist: ${extensionPath}\n` +
      `Check that prepareExtension() created the extension directory correctly.`
    );
  }

  if (!fs.existsSync(manifestPath)) {
    throw new Error(
      `Extension manifest.json not found at: ${manifestPath}\n` +
      `Extension directory exists but is missing manifest.json. ` +
      `Verify extension source files are present.`
    );
  }

  // Create unique userDataDir per context to avoid Windows locking collisions
  const userDataDir = fs.mkdtempSync(path.join(userDataBaseDir, "pw-user-"));

  // ALWAYS include host resolver rules - this is critical for tests
  // Map base domains and common subdomains to 127.0.0.1
  // Note: Mapping base domain should cover subdomains, but explicitly map www.evil.test for reliability
  const hostRules = "MAP good.test 127.0.0.1, MAP evil.test 127.0.0.1, MAP www.evil.test 127.0.0.1";

  // Normalize Windows paths for Chromium args (replace backslashes with forward slashes)
  const extensionPathArg = extensionPath.replace(/\\/g, "/");

  // CRITICAL: Extension tests MUST run in headed mode (headless:false)
  // Chromium extensions often fail to load in headless mode
  const isHeadless = false;

  // Build Chromium launch args
  const chromiumArgs = [
    `--disable-extensions-except=${extensionPathArg}`,
    `--load-extension=${extensionPathArg}`,
    `--host-resolver-rules=${hostRules}`,
    // Stability flags for Windows
    "--no-first-run",
    "--no-default-browser-check",
    "--disable-component-update"
  ];

  const context = await chromium.launchPersistentContext(userDataDir, {
    headless: isHeadless,
    args: chromiumArgs
  });

  await context.grantPermissions(["clipboard-read", "clipboard-write"]);

  // Cleanup userDataDir on context close (best-effort)
  context.on("close", () => {
    try {
      fs.rmSync(userDataDir, { recursive: true, force: true });
    } catch (e) {
      // Ignore cleanup errors
    }
  });

  // Open a page and navigate to verify extension is loaded
  const initPage = await context.newPage();
  
  try {
    const response = await initPage.goto("http://good.test:4173/", {
      waitUntil: "domcontentloaded",
      timeout: 15000
    });

    if (!response || response.status() !== 200) {
      throw new Error(
        `Post-launch navigation check failed. ` +
        `Status: ${response ? response.status() : "null"}. ` +
        `Testapp may not be running on port 4173.`
      );
    }

    // Wait for content script marker (proves extension is active)
    // This is a hard handshake - fail fast if extension doesn't inject
    let extensionLoaded = false;
    try {
      await initPage.waitForFunction(
        () => document.documentElement.getAttribute("data-pg-loaded") === "1",
        { timeout: 10000 }
      );
      extensionLoaded = true;
    } catch (waitError) {
      // Extension failed to inject - capture diagnostics
      const currentUrl = initPage.url();
      const screenshot = await initPage.screenshot({ fullPage: false }).catch(() => null);
      const pageContent = await initPage.content().catch(() => "");
      const pageTitle = await initPage.title().catch(() => "");
      
      // Save screenshot for debugging
      const screenshotPath = path.join(__dirname, "..", "test-results", "extension-load-failure.png");
      if (screenshot) {
        try {
          fs.mkdirSync(path.dirname(screenshotPath), { recursive: true });
          fs.writeFileSync(screenshotPath, screenshot);
        } catch (e) {
          // Ignore screenshot save errors
        }
      }

      throw new Error(
        `Extension failed to inject within 10 seconds.\n` +
        `\n` +
        `Current URL: ${currentUrl}\n` +
        `Page title: ${pageTitle}\n` +
        `Page content length: ${pageContent.length} bytes\n` +
        `Screenshot saved to: ${screenshotPath}\n` +
        `\n` +
        `Chromium launch args:\n` +
        `  ${chromiumArgs.join("\n  ")}\n` +
        `Extension path (raw): ${extensionPath}\n` +
        `Extension path (normalized): ${extensionPathArg}\n` +
        `\n` +
        `Most common causes:\n` +
        `1. Running in headless mode (extensions often fail in headless)\n` +
        `2. Bad --load-extension path (check path normalization)\n` +
        `3. Extension manifest.json is invalid\n` +
        `4. Content script not injecting (check manifest.json content_scripts)\n` +
        `5. Extension crashed during load (check browser console)\n` +
        `\n` +
        `Verify:\n` +
        `- Extension path exists: ${fs.existsSync(extensionPath)}\n` +
        `- Manifest exists: ${fs.existsSync(manifestPath)}\n` +
        `- Browser is running in headed mode (headless: ${isHeadless})\n`
      );
    }

    // Read extension ID from content script marker
    const extensionId = await initPage.evaluate(() => {
      return document.documentElement.getAttribute("data-pg-extension-id");
    });

    if (!extensionId) {
      // Extension marker present but ID missing - still a problem
      const screenshot = await initPage.screenshot({ fullPage: false }).catch(() => null);
      const screenshotPath = path.join(__dirname, "..", "test-results", "extension-id-missing.png");
      if (screenshot) {
        try {
          fs.mkdirSync(path.dirname(screenshotPath), { recursive: true });
          fs.writeFileSync(screenshotPath, screenshot);
        } catch (e) {
          // Ignore screenshot save errors
        }
      }

      throw new Error(
        `Extension loaded marker found but extension ID is missing.\n` +
        `Content script may be partially injecting.\n` +
        `Screenshot saved to: ${screenshotPath}\n` +
        `Check content.js injectExtensionIdentity() function.`
      );
    }

    // Initialize test mode via E2E bridge
    await initPage.evaluate(() => {
      return new Promise((resolve) => {
        const handler = (ev) => {
          if (ev.data && ev.data.source === "pg-e2e" && ev.data.type === "INIT_ACK") {
            window.removeEventListener("message", handler);
            resolve();
          }
        };
        window.addEventListener("message", handler);
        window.postMessage({
          source: "pg-e2e",
          type: "INIT",
          payload: {
            e2e_mode: true,
            e2e_malicious_domains: ["evil.test"]
          }
        }, "*");
      });
    });

    // Best-effort: log if service worker is available (but don't block)
    const sw = context.serviceWorkers()[0];
    if (sw) {
      console.log(`[launchWithExtension] Service worker found: ${sw.url()}`);
    }

    return { context, page: initPage, extensionId };
  } catch (error) {
    await initPage.close();
    await context.close();
    throw new Error(
      `Browser launcher initialization failed.\n` +
      `Error: ${error.message}\n` +
      `\n` +
      `This indicates a fundamental problem:\n` +
      `- Testapp server may not be running (check global-setup.js)\n` +
      `- Extension content script may not be loading\n` +
      `- Port 4173 may be blocked or in use\n` +
      `\n` +
      `Do not proceed with tests until this is resolved.`
    );
  }
}

module.exports = { launchWithExtension };


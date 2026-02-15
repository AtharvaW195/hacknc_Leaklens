const { test, expect } = require("@playwright/test");
const { chromium } = require("@playwright/test");
const { launchWithExtension } = require("../utils/launch-extension");

/**
 * Sanity tests that run FIRST to verify all prerequisites are met.
 * If any of these fail, the test suite should not continue.
 * 
 * These tests verify:
 * - Testapp server is reachable (without extension)
 * - Host mapping works (good.test/evil.test -> 127.0.0.1)
 * - Extension is loaded and content script is active
 */
test.describe("E2E Prerequisites Sanity Check", () => {
	test("Testapp is reachable at 127.0.0.1:4173", async () => {
		// Use normal Playwright page without extension for this check
		const browser = await chromium.launch();
		const context = await browser.newContext();
		const page = await context.newPage();

		try {
			const response = await page.goto("http://127.0.0.1:4173/", {
				waitUntil: "domcontentloaded",
				timeout: 10000
			});

			if (!response) {
				throw new Error("Navigation returned null - page may not have loaded");
			}

			if (response.status() !== 200) {
				throw new Error(
					`Testapp returned status ${response.status()} instead of 200. ` +
					`Is the testapp server running on port 4173?`
				);
			}

			// Verify page content
			await expect(page.locator("h1")).toBeVisible({ timeout: 5000 });
			const h1Text = await page.locator("h1").textContent();
			expect(h1Text).toContain("PasteGuard E2E Test");

		} catch (error) {
			const currentUrl = page.url();
			const pageContent = await page.content();
			throw new Error(
				`SANITY CHECK FAILED: Testapp is not reachable.\n` +
				`Error: ${error.message}\n` +
				`Current URL: ${currentUrl}\n` +
				`Page content length: ${pageContent.length} bytes\n` +
				`\n` +
				`Possible causes:\n` +
				`- Testapp server is not running on port 4173\n` +
				`- Firewall blocking port 4173\n` +
				`- Testapp server crashed during startup\n` +
				`\n` +
				`Check global-setup.js logs for testapp startup errors.`
			);
		} finally {
			await context.close();
			await browser.close();
		}
	});

	test("Host mapping works: good.test resolves to 127.0.0.1", async () => {
		// Use normal Playwright page with host resolver rules
		const browser = await chromium.launch({
			args: ["--host-resolver-rules=MAP good.test 127.0.0.1, MAP evil.test 127.0.0.1"]
		});
		const context = await browser.newContext();
		const page = await context.newPage();

		try {
			const response = await page.goto("http://good.test:4173/", {
				waitUntil: "domcontentloaded",
				timeout: 10000
			});

			if (!response) {
				throw new Error("Navigation returned null - host mapping may be broken");
			}

			if (response.status() !== 200) {
				throw new Error(
					`good.test returned status ${response.status()} instead of 200. ` +
					`Host mapping may be broken. Check --host-resolver-rules in launch-extension.js`
				);
			}

			// Verify same page content as direct IP
			await expect(page.locator("h1")).toBeVisible({ timeout: 5000 });
			const h1Text = await page.locator("h1").textContent();
			expect(h1Text).toContain("PasteGuard E2E Test");

		} catch (error) {
			const currentUrl = page.url();
			const pageContent = await page.content();
			throw new Error(
				`SANITY CHECK FAILED: Host mapping for good.test is broken.\n` +
				`Error: ${error.message}\n` +
				`Current URL: ${currentUrl}\n` +
				`Page content length: ${pageContent.length} bytes\n` +
				`\n` +
				`Possible causes:\n` +
				`- --host-resolver-rules not set correctly in Chromium args\n` +
				`- Chromium ignored the host resolver rules\n` +
				`- DNS resolution is interfering\n` +
				`\n` +
				`Check launch-extension.js for correct host-resolver-rules format.`
			);
		} finally {
			await context.close();
			await browser.close();
		}
	});

	test("Extension content script is loaded and active", async () => {
		const { context, page: initPage, extensionId } = await launchWithExtension(process.env.E2E_EXTENSION_PATH);

		// Capture errors for diagnostics
		const errors = [];
		initPage.on("pageerror", (error) => {
			errors.push(`Page error: ${error.message}`);
		});
		initPage.on("console", (msg) => {
			if (msg.type() === "error") {
				errors.push(`Console error: ${msg.text()}`);
			}
		});

		try {
			// Verify extension identity markers
			const loaded = await initPage.evaluate(() => {
				return document.documentElement.getAttribute("data-pg-loaded");
			});
			expect(loaded).toBe("1");

			const extId = await initPage.evaluate(() => {
				return document.documentElement.getAttribute("data-pg-extension-id");
			});
			expect(extId).toBeTruthy();
			expect(extId).toBe(extensionId);

		} catch (error) {
			const currentUrl = initPage.url();
			let pageContentLength = 0;
			try {
				const content = await initPage.content();
				pageContentLength = content.length;
			} catch (e) {
				pageContentLength = -1;
			}

			throw new Error(
				`SANITY CHECK FAILED: Extension content script is not loaded.\n` +
				`Error: ${error.message}\n` +
				`Current URL: ${currentUrl}\n` +
				`Page content length: ${pageContentLength} bytes\n` +
				`Errors captured: ${errors.length > 0 ? errors.join("; ") : "none"}\n` +
				`Extension ID from launcher: ${extensionId}\n` +
				`\n` +
				`Possible causes:\n` +
				`- Extension not loaded (check E2E_EXTENSION_PATH)\n` +
				`- Content script not injected (check manifest.json)\n` +
				`- Content script crashed (check console errors)\n` +
				`- Identity markers not injected (check content.js)\n` +
				`\n` +
				`Verify extension is built and E2E_EXTENSION_PATH points to extension directory.`
			);
		} finally {
			await context.close();
		}
	});
});


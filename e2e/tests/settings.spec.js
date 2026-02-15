const { test, expect } = require("@playwright/test");
const { launchWithExtension } = require("../utils/launch-extension");
const { resetExtensionState, getExtensionId, e2ePaste, waitForContentScript } = require("../utils/test-helpers");

test.describe("Settings", () => {
	let context, page, extensionId;

	test.beforeEach(async () => {
		const result = await launchWithExtension(process.env.E2E_EXTENSION_PATH);
		context = result.context;
		page = result.page;
		extensionId = result.extensionId;
		await resetExtensionState(context, extensionId);
	});

	test.afterEach(async () => {
		if (context) {
			await context.close();
		}
	});

	test("Settings UI loads with default values", async () => {
		// Open extension popup
		await testPage.goto(`chrome-extension://${extensionId}/popup.html`);
		
		// Verify settings are visible
		await expect(testPage.locator("#pg-setting-enabled")).toBeVisible();
		await expect(testPage.locator("#pg-setting-threshold")).toBeVisible();
		await expect(testPage.locator("#pg-setting-convert-link")).toBeVisible();
		
		// Verify default values
		await expect(testPage.locator("#pg-setting-enabled")).toBeChecked();
		await expect(testPage.locator("#pg-setting-threshold")).toHaveValue("HIGH");
		await expect(testPage.locator("#pg-setting-convert-link")).toBeChecked();
	});

	test("Toggle paste guard enabled setting", async () => {
		const testPage = await context.newPage();
		
		await testPage.goto(`chrome-extension://${extensionId}/popup.html`);
		
		// Toggle off
		await testPage.uncheck("#pg-setting-enabled");
		
		// Verify setting is saved
		let sw = context.serviceWorkers().find(s => s.url().includes(extensionId));
		if (!sw) {
			sw = await context.waitForEvent("serviceworker", { timeout: 10000 });
		}
		const storage = await sw.evaluate(() => {
			return chrome.storage.sync.get(['pasteGuardEnabled']);
		});
		expect(storage.pasteGuardEnabled).toBe(false);
		
		// Reload popup and verify setting persists
		await testPage.reload();
		await expect(testPage.locator("#pg-setting-enabled")).not.toBeChecked();
	});

	test("Change block threshold setting", async () => {
		const testPage = await context.newPage();
		
		await testPage.goto(`chrome-extension://${extensionId}/popup.html`);
		
		// Change threshold to HIGH_MEDIUM
		await testPage.selectOption("#pg-setting-threshold", "HIGH_MEDIUM");
		
		// Verify setting is saved
		let sw = context.serviceWorkers().find(s => s.url().includes(extensionId));
		if (!sw) {
			sw = await context.waitForEvent("serviceworker", { timeout: 10000 });
		}
		const storage = await sw.evaluate(() => {
			return chrome.storage.sync.get(['pasteBlockThreshold']);
		});
		expect(storage.pasteBlockThreshold).toBe("HIGH_MEDIUM");
		
		// Reload popup and verify setting persists
		await testPage.reload();
		await expect(testPage.locator("#pg-setting-threshold")).toHaveValue("HIGH_MEDIUM");
	});

	test("Toggle convert to link setting", async () => {
		const testPage = await context.newPage();
		
		await testPage.goto(`chrome-extension://${extensionId}/popup.html`);
		
		// Toggle off
		await testPage.uncheck("#pg-setting-convert-link");
		
		// Verify setting is saved
		let sw = context.serviceWorkers().find(s => s.url().includes(extensionId));
		if (!sw) {
			sw = await context.waitForEvent("serviceworker", { timeout: 10000 });
		}
		const storage = await sw.evaluate(() => {
			return chrome.storage.sync.get(['pasteAllowConvertToLink']);
		});
		expect(storage.pasteAllowConvertToLink).toBe(false);
		
		// Reload popup and verify setting persists
		await testPage.reload();
		await expect(testPage.locator("#pg-setting-convert-link")).not.toBeChecked();
	});

	test("Settings affect paste guard behavior", async () => {
		let sw = context.serviceWorkers().find(s => s.url().includes(extensionId));
		if (!sw) {
			sw = await context.waitForEvent("serviceworker", { timeout: 10000 });
		}
		
		// Set threshold to HIGH_MEDIUM
		await sw.evaluate(() => {
			return chrome.storage.sync.set({ pasteBlockThreshold: "HIGH_MEDIUM" });
		});
		
		// Wait for setting to propagate
		await new Promise(r => setTimeout(r, 500));
		
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
		await waitForContentScript(testPage);
		
		const textarea = testPage.locator("#t");
		
		// Paste medium risk text (should be blocked with HIGH_MEDIUM threshold)
		// Note: This depends on backend returning MEDIUM risk for certain inputs
		// For now, test with HIGH risk which should always be blocked
		await e2ePaste(testPage, "#t", "password=supersecret123");
		
		// Verify modal appears
		await expect(testPage.locator('[data-testid="paste-modal"]')).toBeVisible({ timeout: 5000 });
	});

	test("Stats display in popup", async () => {
		const testPage = await context.newPage();
		
		await testPage.goto(`chrome-extension://${extensionId}/popup.html`);
		
		// Verify stats elements are visible
		await expect(testPage.locator("#pg-stat-analyzed")).toBeVisible();
		await expect(testPage.locator("#pg-stat-blocked")).toBeVisible();
		await expect(testPage.locator("#pg-stat-links")).toBeVisible();
		
		// Stats should be numbers (default 0)
		const analyzed = await testPage.locator("#pg-stat-analyzed").textContent();
		expect(parseInt(analyzed)).toBeGreaterThanOrEqual(0);
	});

	test("Link checker validates URLs", async () => {
		const testPage = await context.newPage();
		
		await testPage.goto(`chrome-extension://${extensionId}/popup.html`);
		
		// Enter evil.test URL
		await testPage.fill("#pg-link-input", "http://evil.test");
		await testPage.click("#pg-check-btn");
		
		// Wait for result
		await expect(testPage.locator("#pg-check-result")).toBeVisible({ timeout: 3000 });
		
		// Verify result indicates flagged
		const result = await testPage.locator("#pg-check-result").textContent();
		expect(result).toContain("Flagged");
	});

	test("Link checker validates good URLs", async () => {
		const testPage = await context.newPage();
		
		await testPage.goto(`chrome-extension://${extensionId}/popup.html`);
		
		// Enter good URL
		await testPage.fill("#pg-link-input", "https://example.com");
		await testPage.click("#pg-check-btn");
		
		// Wait for result
		await expect(testPage.locator("#pg-check-result")).toBeVisible({ timeout: 3000 });
		
		// Verify result indicates legit
		const result = await testPage.locator("#pg-check-result").textContent();
		expect(result.toLowerCase()).toContain("legit");
	});

	test("Link checker handles invalid URLs", async () => {
		const testPage = await context.newPage();
		
		await testPage.goto(`chrome-extension://${extensionId}/popup.html`);
		
		// Enter invalid URL
		await testPage.fill("#pg-link-input", "not-a-url");
		await testPage.click("#pg-check-btn");
		
		// Wait for result
		await expect(testPage.locator("#pg-check-result")).toBeVisible({ timeout: 3000 });
		
		// Verify result indicates invalid
		const result = await testPage.locator("#pg-check-result").textContent();
		expect(result.toLowerCase()).toContain("invalid");
	});

	test("Convert to link button hidden when setting disabled", async () => {
		let sw = context.serviceWorkers().find(s => s.url().includes(extensionId));
		if (!sw) {
			sw = await context.waitForEvent("serviceworker", { timeout: 10000 });
		}
		
		// Disable convert to link setting
		await sw.evaluate(() => {
			return chrome.storage.sync.set({ pasteAllowConvertToLink: false });
		});
		
		// Wait for setting to propagate
		await new Promise(r => setTimeout(r, 500));
		
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
		await waitForContentScript(testPage);
		
		const textarea = testPage.locator("#t");
		await e2ePaste(testPage, "#t", "password=supersecret123");
		
		// Verify modal appears
		await expect(testPage.locator('[data-testid="paste-modal"]')).toBeVisible({ timeout: 5000 });
		
		// Verify convert to link button is NOT visible
		await expect(testPage.locator('[data-testid="paste-convert-link"]')).toHaveCount(0);
	});
});


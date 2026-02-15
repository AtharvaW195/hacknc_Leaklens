const { test, expect } = require("@playwright/test");
const { launchWithExtension } = require("../utils/launch-extension");
const { resetExtensionState, getExtensionId, waitForContentScript, setupDiagnostics, getDiagnosticInfo } = require("../utils/test-helpers");

test.describe("Malicious Domain Blocking", () => {
	let context, page, extensionId;

	test.beforeEach(async () => {
		const result = await launchWithExtension(process.env.E2E_EXTENSION_PATH);
		context = result.context;
		page = result.page;
		extensionId = result.extensionId;
		await resetExtensionState(page);
	});

	test.afterEach(async () => {
		if (context) {
			await context.close();
		}
	});

	test("Block evil.test with overlay shown", async () => {
		const testPage = await context.newPage();
		const diagnostics = setupDiagnostics(testPage);
		
		try {
			await testPage.goto("http://evil.test:4173/", { waitUntil: "domcontentloaded" });
			
			// Wait for content script to load and check domain
			await waitForContentScript(testPage);
			
			// Verify blocking overlay appears
			await expect(testPage.locator('[data-testid="blocked-overlay"]')).toBeVisible();
			await expect(testPage.locator('[data-testid="blocked-title"]')).toBeVisible();
			await expect(testPage.locator('[data-testid="blocked-title"]')).toContainText("Site Blocked");
			
			// Verify overlay contains domain name
			const overlayText = await testPage.locator('[data-testid="blocked-overlay"]').textContent();
			expect(overlayText).toContain("evil.test");
		} catch (error) {
			const diag = await getDiagnosticInfo(testPage, diagnostics);
			throw new Error(`${error.message}\n\n${diag}`);
		} finally {
			await testPage.close();
		}
	});

	test("Allow good.test (not blocked)", async ({ context }) => {
		const page = await context.newPage();
		const diagnostics = setupDiagnostics(page);
		
		try {
			await page.goto("http://good.test:4173/", { waitUntil: "domcontentloaded" });
			
			await waitForContentScript(page);
			
			// Verify no blocking overlay appears
			await expect(page.locator('[data-testid="blocked-overlay"]')).toHaveCount(0);
			
			// Verify page content is accessible
			await expect(page.locator("body")).toBeVisible();
		} catch (error) {
			const diag = await getDiagnosticInfo(page, diagnostics);
			throw new Error(`${error.message}\n\n${diag}`);
		}
	});

	test("Proceed anyway button removes overlay", async ({ context }) => {
		const page = await context.newPage();
		const diagnostics = setupDiagnostics(page);
		
		try {
			await page.goto("http://evil.test:4173/", { waitUntil: "domcontentloaded" });
			
			await waitForContentScript(page);
			await expect(page.locator('[data-testid="blocked-overlay"]')).toBeVisible();
			
			// Click proceed anyway button
			await page.click("#pg-proceed-btn");
			
			// Verify overlay is removed
			await expect(page.locator('[data-testid="blocked-overlay"]')).toHaveCount(0);
		} catch (error) {
			const diag = await getDiagnosticInfo(page, diagnostics);
			throw new Error(`${error.message}\n\n${diag}`);
		}
	});

	test("Subdomain of evil.test is blocked", async ({ context }) => {
		const page = await context.newPage();
		const diagnostics = setupDiagnostics(page);
		
		try {
			await page.goto("http://www.evil.test:4173/", { waitUntil: "domcontentloaded" });
			
			await waitForContentScript(page);
			
			// Verify blocking overlay appears for subdomain
			await expect(page.locator('[data-testid="blocked-overlay"]')).toBeVisible();
		} catch (error) {
			const diag = await getDiagnosticInfo(page, diagnostics);
			throw new Error(`${error.message}\n\n${diag}`);
		}
	});

	test("Extension boot: e2e_mode enabled and malicious domain list override set", async () => {
		// Verify e2e_mode is set via GET_STATE
		const state = await page.evaluate(() => {
			return new Promise((resolve) => {
				const handler = (ev) => {
					if (ev.data && ev.data.source === "pg-e2e" && ev.data.type === "GET_STATE_ACK") {
						window.removeEventListener("message", handler);
						resolve(ev.data.payload);
					}
				};
				window.addEventListener("message", handler);
				window.postMessage({ source: "pg-e2e", type: "GET_STATE" }, "*");
			});
		});
		
		expect(state.local.e2e_mode).toBe(true);
		expect(state.local.e2e_malicious_domains).toEqual(["evil.test"]);
	});

	test("Storage reset between tests", async () => {
		// Set some test data via bridge
		await page.evaluate(() => {
			return new Promise((resolve) => {
				chrome.storage.local.set({ test_key: "test_value" }, () => {
					resolve();
				});
			});
		});
		
		// Reset state
		await resetExtensionState(page);
		
		// Verify test data is cleared via GET_STATE
		const state = await page.evaluate(() => {
			return new Promise((resolve) => {
				const handler = (ev) => {
					if (ev.data && ev.data.source === "pg-e2e" && ev.data.type === "GET_STATE_ACK") {
						window.removeEventListener("message", handler);
						resolve(ev.data.payload);
					}
				};
				window.addEventListener("message", handler);
				window.postMessage({ source: "pg-e2e", type: "GET_STATE" }, "*");
			});
		});
		
		expect(state.local.test_key).toBeUndefined();
		expect(state.local.e2e_mode).toBe(true);
		expect(state.local.e2e_malicious_domains).toEqual(["evil.test"]);
	});
});


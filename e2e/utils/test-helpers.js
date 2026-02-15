const { expect } = require("@playwright/test");

/**
 * E2E paste helper - uses content script message bridge instead of clipboard.
 * 
 * Why we avoid navigator.clipboard in E2E:
 * - navigator.clipboard is undefined in Playwright extension contexts
 * - Clipboard APIs require secure context and user interaction
 * - Synthetic clipboardData doesn't cross isolated world boundary into content scripts
 * - Message bridge allows E2E to trigger paste handling directly in content script context
 *
 * @param {Page} page - Playwright page object
 * @param {string} selector - CSS selector for the target element
 * @param {string} text - Text to paste
 */
async function e2ePaste(page, selector, text) {
	await page.evaluate(
		({ selector, text }) => {
			window.postMessage(
				{ source: "pg-e2e", type: "PASTE", selector, text },
				"*"
			);
		},
		{ selector, text }
	);
}

/**
 * Reset extension state between tests.
 * Clears extension storage and re-applies E2E defaults via service worker.
 *
 * @param {Page|BrowserContext} pageOrContext - Playwright page or context object
 * @param {string} extensionId - Optional extension ID (if context is passed)
 */
async function resetExtensionState(pageOrContext, extensionId = null) {
	let context;
	let page;
	
	// Handle different input types
	if (pageOrContext && typeof pageOrContext.url === 'function') {
		// It's a Page object
		page = pageOrContext;
		context = page.context();
	} else if (pageOrContext && typeof pageOrContext.newPage === 'function') {
		// It's a BrowserContext
		context = pageOrContext;
		// Get or create a page with extension loaded
		const pages = context.pages();
		page = pages.find(p => {
			try {
				if (p && typeof p.url === 'function') {
					const url = p.url();
					return url && (url.includes('good.test') || url.includes('127.0.0.1'));
				}
			} catch {
				// Ignore errors
			}
			return false;
		}) || pages[0];
		
		if (!page || typeof page.url !== 'function') {
			page = await context.newPage();
			await page.goto("http://good.test:4173/", { waitUntil: "domcontentloaded" });
			await waitForContentScript(page);
		}
	} else {
		throw new Error(
			`resetExtensionState: Invalid argument type. ` +
			`Expected Page or BrowserContext, got: ${typeof pageOrContext}`
		);
	}

	// Ensure we have a valid page
	if (!page || typeof page.url !== 'function') {
		throw new Error(
			`resetExtensionState: Could not obtain a valid Page object from context`
		);
	}

	// Get extension ID if not provided
	if (!extensionId) {
		// Try to get from service worker
		let sw = context.serviceWorkers()[0];
		if (!sw) {
			// Wait for service worker with timeout
			try {
				sw = await context.waitForEvent("serviceworker", { timeout: 5000 });
			} catch {
				// Service worker not available, use E2E bridge instead
				const currentUrl = page.url();
				if (!currentUrl || currentUrl === "about:blank") {
					await page.goto("http://good.test:4173/", { waitUntil: "domcontentloaded" });
					await waitForContentScript(page);
				}
				
				// Use E2E bridge RESET
				await page.evaluate(() => {
					return new Promise((resolve) => {
						const handler = (ev) => {
							if (ev.data && ev.data.source === "pg-e2e" && ev.data.type === "RESET_ACK") {
								window.removeEventListener("message", handler);
								resolve();
							}
						};
						window.addEventListener("message", handler);
						window.postMessage({
							source: "pg-e2e",
							type: "RESET"
						}, "*");
					});
				});
				return;
			}
		}
		extensionId = new URL(sw.url()).host;
	}

	// Use service worker to reset storage (more reliable)
	let sw = context.serviceWorkers().find(s => {
		try {
			return new URL(s.url()).host === extensionId;
		} catch {
			return false;
		}
	});
	
	if (!sw) {
		// Wait for service worker
		try {
			sw = await context.waitForEvent("serviceworker", { timeout: 10000 });
		} catch {
			// Fallback to E2E bridge
			const currentUrl = page.url();
			if (!currentUrl || currentUrl === "about:blank") {
				await page.goto("http://good.test:4173/", { waitUntil: "domcontentloaded" });
				await waitForContentScript(page);
			}
			
			await page.evaluate(() => {
				return new Promise((resolve) => {
					const handler = (ev) => {
						if (ev.data && ev.data.source === "pg-e2e" && ev.data.type === "RESET_ACK") {
							window.removeEventListener("message", handler);
							resolve();
						}
					};
					window.addEventListener("message", handler);
					window.postMessage({
						source: "pg-e2e",
						type: "RESET"
					}, "*");
				});
			});
			return;
		}
	}

	// Clear storage and set E2E defaults via service worker
	await sw.evaluate(() => {
		return Promise.all([
			chrome.storage.local.clear(),
			chrome.storage.sync.clear(),
			chrome.storage.local.set({
				e2e_mode: true,
				e2e_malicious_domains: ["evil.test"]
			}),
			chrome.storage.sync.set({
				pasteGuardEnabled: true,
				pasteBlockThreshold: 'HIGH',
				pasteAllowConvertToLink: true
			})
		]);
	});
}

/**
 * Get extension ID from service worker.
 *
 * @param {BrowserContext} context - Playwright browser context
 * @returns {Promise<string>} Extension ID
 */
async function getExtensionId(context) {
	let sw = context.serviceWorkers()[0];
	if (!sw) {
		sw = await context.waitForEvent("serviceworker", { timeout: 5000 });
	}
	return new URL(sw.url()).host;
}

/**
 * Wait for toast notification to appear and contain substring.
 * Uses data-testid="toast" selector.
 *
 * @param {Page} page - Playwright page object
 * @param {string} substring - Substring to match in toast text
 * @param {number} timeout - Timeout in milliseconds (default: 5000)
 */
async function waitForToast(page, substring, timeout = 5000) {
	const toast = page.locator('[data-testid="toast"]');
	await expect(toast).toBeVisible({ timeout });
	if (substring) {
		await expect(toast).toContainText(substring, { timeout });
	}
	return toast;
}

/**
 * Wait for content script to be loaded by checking for the deterministic marker.
 * This is the primary way to ensure content script is active before assertions.
 * Uses the same marker as launchWithExtension: data-pg-loaded attribute on documentElement.
 *
 * @param {Page} page - Playwright page object
 * @param {number} timeout - Timeout in milliseconds (default: 10000)
 */
async function waitForContentScript(page, timeout = 10000) {
	// Wait for the same marker used in launchWithExtension
	await page.waitForFunction(
		() => document.documentElement.getAttribute("data-pg-loaded") === "1",
		{ timeout }
	);
}

/**
 * Setup diagnostics for a page (capture errors and console logs).
 * Call this in test beforeEach or at the start of each test.
 *
 * @param {Page} page - Playwright page object
 * @returns {Object} Diagnostics object with errors array
 */
function setupDiagnostics(page) {
	const diagnostics = {
		errors: [],
		consoleLogs: []
	};

	page.on("pageerror", (error) => {
		diagnostics.errors.push({
			type: "pageerror",
			message: error.message,
			stack: error.stack
		});
	});

	page.on("console", (msg) => {
		const text = msg.text();
		const type = msg.type();
		diagnostics.consoleLogs.push({ type, text });
		
		if (type === "error") {
			diagnostics.errors.push({
				type: "console",
				message: text
			});
		}
	});

	return diagnostics;
}

/**
 * Get diagnostic information for a failed test.
 * Call this when a test fails to get context.
 *
 * @param {Page} page - Playwright page object
 * @param {Object} diagnostics - Diagnostics object from setupDiagnostics
 * @returns {string} Diagnostic message
 */
async function getDiagnosticInfo(page, diagnostics) {
	const currentUrl = page.url();
	let pageContentLength = 0;
	try {
		const content = await page.content();
		pageContentLength = content.length;
	} catch (e) {
		pageContentLength = -1;
	}

	const errorSummary = diagnostics.errors.length > 0
		? diagnostics.errors.map(e => `${e.type}: ${e.message}`).join("; ")
		: "none";

	return (
		`Test failure diagnostics:\n` +
		`Current URL: ${currentUrl}\n` +
		`Page content length: ${pageContentLength} bytes\n` +
		`Errors captured: ${errorSummary}\n` +
		`Console logs: ${diagnostics.consoleLogs.length} total`
	);
}

module.exports = {
	e2ePaste,
	resetExtensionState,
	getExtensionId,
	waitForToast,
	waitForContentScript,
	setupDiagnostics,
	getDiagnosticInfo
};


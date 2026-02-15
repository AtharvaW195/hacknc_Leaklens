const { test, expect } = require("@playwright/test");
const { launchWithExtension } = require("../utils/launch-extension");
const { resetExtensionState, getExtensionId, waitForContentScript, e2ePaste, waitForToast } = require("../utils/test-helpers");

test.describe("Paste Guard", () => {
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

	test("Low risk paste passes through", async () => {
		await page.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(page);
		
		const textarea = page.locator("#t");
		
		// Paste low risk text
		await e2ePaste(page, "#t", "hello world");
		
		// Wait for async analysis to complete and text to be inserted
		await expect(textarea).toHaveValue("hello world", { timeout: 5000 });
		
		// Verify no modal appears
		await expect(page.locator('[data-testid="paste-modal"]')).toHaveCount(0);
	});

	test("High risk paste shows blocking modal", async () => {
		const p = await context.newPage();
		await p.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(p);
		
		const textarea = p.locator("#t");
		
		// Paste high risk text
		await e2ePaste(p, "#t", "password=supersecret123");
		
		// Wait for modal to appear
		await expect(p.locator('[data-testid="paste-modal"]')).toBeVisible({ timeout: 5000 });
		
		// Verify modal contains risk information
		await expect(p.locator('[data-testid="paste-modal"]')).toContainText("Sensitive paste blocked");
	});

	test("Redacted paste does not include secret", async () => {
		const p = await context.newPage();
		await p.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(p);
		
		const textarea = p.locator("#t");
		await textarea.fill("");
		
		await e2ePaste(p, "#t", "password=supersecret123");
		await expect(p.locator('[data-testid="paste-modal"]')).toBeVisible();
		
		// Click redacted paste button
		await p.click('[data-testid="paste-redacted"]');
		
		// Wait for modal to close and text to be inserted
		await expect(p.locator('[data-testid="paste-modal"]')).toHaveCount(0);
		
		const value = await textarea.inputValue();
		expect(value).not.toContain("supersecret123");
	});

	test("Paste anyway inserts original text", async () => {
		const p = await context.newPage();
		await p.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(p);
		
		const textarea = p.locator("#t");
		await textarea.fill("");
		
		await e2ePaste(p, "#t", "password=supersecret123");
		await expect(p.locator('[data-testid="paste-modal"]')).toBeVisible();
		
		// Click paste anyway button
		await p.click('[data-testid="paste-anyway"]');
		
		// Verify original text is inserted
		await expect(textarea).toHaveValue("password=supersecret123", { timeout: 2000 });
	});

	test("Cancel button closes modal without pasting", async () => {
		const p = await context.newPage();
		await p.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(p);
		
		const textarea = p.locator("#t");
		await textarea.fill("existing text");
		
		await e2ePaste(p, "#t", "password=supersecret123");
		await expect(p.locator('[data-testid="paste-modal"]')).toBeVisible();
		
		// Click cancel button
		await p.click('[data-testid="paste-cancel"]');
		
		// Verify modal is closed
		await expect(p.locator('[data-testid="paste-modal"]')).toHaveCount(0);
		
		// Verify textarea still has original value
		await expect(textarea).toHaveValue("existing text");
	});

	test("ESC key closes modal", async () => {
		const p = await context.newPage();
		await p.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(p);
		
		const textarea = p.locator("#t");
		await e2ePaste(p, "#t", "password=supersecret123");
		await expect(p.locator('[data-testid="paste-modal"]')).toBeVisible();
		
		// Press ESC key
		await p.keyboard.press("Escape");
		
		// Verify modal is closed
		await expect(p.locator('[data-testid="paste-modal"]')).toHaveCount(0);
	});

	test("Convert to Secure Share link inserts view link", async () => {
		const p = await context.newPage();
		await p.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(p);
		
		const textarea = p.locator("#t");
		await textarea.fill("");
		
		await e2ePaste(p, "#t", "password=supersecret123");
		await expect(p.locator('[data-testid="paste-modal"]')).toBeVisible();
		
		// Click convert to link button
		await p.click('[data-testid="paste-convert-link"]');
		
		// Wait for link to be inserted
		await p.waitForFunction(
			(selector) => {
				const el = document.querySelector(selector);
				return el && el.value.includes("http://localhost:8080/view/");
			},
			"#t",
			{ timeout: 10000 }
		);
		
		const value = await textarea.inputValue();
		expect(value).toContain("http://localhost:8080/view/");
		expect(value).not.toContain("supersecret123");
	});

	test("Convert to link failure shows error", async () => {
		const p = await context.newPage();
		await p.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(p);
		
		// Mock upload endpoint to fail
		await context.route("http://localhost:8080/api/upload", async (route) => {
			await route.fulfill({
				status: 500,
				contentType: "application/json",
				body: JSON.stringify({ error: "Upload failed" }),
			});
		});
		
		const textarea = p.locator("#t");
		await textarea.fill("");
		
		await e2ePaste(p, "#t", "password=supersecret123");
		await expect(p.locator('[data-testid="paste-modal"]')).toBeVisible();
		
		// Click convert to link button
		await p.click('[data-testid="paste-convert-link"]');
		
		// Wait for error message in modal
		await expect(p.locator("#pg-modal-error")).toBeVisible({ timeout: 5000 });
		await expect(p.locator("#pg-modal-error")).toContainText("Upload failed");
		
		// Verify toast appears
		await waitForToast(p, "Upload failed");
		
		// Verify secret is not leaked
		const value = await textarea.inputValue();
		expect(value).not.toContain("supersecret123");
	});

	test("Paste guard respects disabled setting", async () => {
		// Use extensionId from beforeEach
		let sw = context.serviceWorkers().find(s => s.url().includes(extensionId));
		if (!sw) {
			// Wait for service worker if not available
			sw = await context.waitForEvent("serviceworker", { timeout: 10000 });
		}
		
		// Disable paste guard
		await sw.evaluate(() => {
			return chrome.storage.sync.set({ pasteGuardEnabled: false });
		});
		
		// Wait for setting to propagate
		await new Promise(r => setTimeout(r, 500));
		
		const p = await context.newPage();
		await p.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(p);
		
		const textarea = p.locator("#t");
		
		// Paste high risk text
		await e2ePaste(p, "#t", "password=supersecret123");
		
		// Verify text is pasted directly (no modal)
		await expect(textarea).toHaveValue("password=supersecret123", { timeout: 5000 });
		await expect(p.locator('[data-testid="paste-modal"]')).toHaveCount(0);
	});

	test("Analysis timeout fails open", async () => {
		const p = await context.newPage();
		await p.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(p);
		
		// Delay analyze responses to exceed timeout
		await context.route(
			"http://localhost:8080/api/analyze-text",
			async (route) => {
				await new Promise((r) => setTimeout(r, 1500));
				await route.continue();
			},
		);
		
		const textarea = p.locator("#t");
		
		await e2ePaste(p, "#t", "password=supersecret123");
		
		// Should fail open: paste should appear, and modal should NOT appear
		await expect(p.locator('[data-testid="paste-modal"]')).toHaveCount(0, { timeout: 2000 });
		await expect(textarea).toHaveValue(/password=supersecret123/, { timeout: 5000 });
	});

	test("Paste guard only intercepts valid targets (textarea, input)", async () => {
		const p = await context.newPage();
		
		// Navigate to a real matched origin first
		await p.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
		await waitForContentScript(p);
		
		// Replace DOM with test content
		await p.evaluate(() => {
			document.body.innerHTML = `
				<div contenteditable="true" id="editable">Editable div</div>
				<textarea id="textarea"></textarea>
			`;
		});
		
		// Paste into textarea (should be intercepted)
		await e2ePaste(p, "#textarea", "password=supersecret123");
		await expect(p.locator('[data-testid="paste-modal"]')).toBeVisible();
		
		// Close modal
		await p.click('[data-testid="paste-cancel"]');
		
		// Paste into contenteditable (should NOT be intercepted - extension only handles textarea/input)
		// Note: This test verifies the extension doesn't interfere with contenteditable
		const editable = p.locator("#editable");
		await editable.click();
		await e2ePaste(p, "#editable", "password=supersecret123");
		
		// Wait a bit - modal should not appear for contenteditable
		await p.waitForTimeout(2000);
		await expect(p.locator('[data-testid="paste-modal"]')).toHaveCount(0);
	});
});


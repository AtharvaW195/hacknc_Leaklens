const { test, expect } = require("@playwright/test");
const { launchWithExtension } = require("../utils/launch-extension");
const { resetExtensionState, getExtensionId, e2ePaste, waitForContentScript, setupDiagnostics, getDiagnosticInfo } = require("../utils/test-helpers");

/**
 * Integration tests - verify basic end-to-end flows work together.
 * Detailed feature tests are in separate spec files:
 * - blocked-domain.spec.js
 * - link-scanner.spec.js
 * - secure-upload.spec.js
 * - paste-guard.spec.js
 * - settings.spec.js
 * - backend-one-time.spec.js
 */
test.describe("Privacy Guardrail E2E Integration", () => {
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

	test("End-to-end: Domain blocking → Link scanning → Paste guard → Secure upload", async () => {
		const diagnostics = setupDiagnostics(testPage);
		
		try {
			// 1. Verify domain blocking
			await testPage.goto("http://evil.test:4173/", { waitUntil: "domcontentloaded" });
			await waitForContentScript(testPage);
			await expect(testPage.locator('[data-testid="blocked-overlay"]')).toBeVisible();

			// 2. Verify link scanning
			await testPage.goto("http://good.test:4173/links", { waitUntil: "domcontentloaded" });
			await waitForContentScript(testPage);
			await testPage.waitForTimeout(500);
			await expect(testPage.locator('[data-testid="bad-link-icon"]').first()).toBeVisible();

			// 3. Verify paste guard
			await testPage.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
			await waitForContentScript(testPage);
			const textarea = testPage.locator("#t");
			await e2ePaste(testPage, "#t", "password=supersecret123");
			await expect(testPage.locator('[data-testid="paste-modal"]')).toBeVisible({ timeout: 5000 });
			await testPage.click('[data-testid="paste-cancel"]');

			// 4. Verify secure upload
			await testPage.goto("http://good.test:4173/upload", { waitUntil: "domcontentloaded" });
			await waitForContentScript(testPage);
			await expect(testPage.locator('[data-testid="secure-share-btn"]')).toBeVisible();
		} catch (error) {
			const diag = await getDiagnosticInfo(testPage, diagnostics);
			throw new Error(`${error.message}\n\n${diag}`);
		}
	});
});

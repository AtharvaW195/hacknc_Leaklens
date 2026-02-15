const path = require("path");
const { test, expect } = require("@playwright/test");
const { launchWithExtension } = require("../utils/launch-extension");
const { resetExtensionState, e2ePaste, waitForContentScript } = require("../utils/test-helpers");

test.describe("Backend One-Time View Semantics", () => {
	// Most tests use request fixture (no extension needed)
	// Only one UI test uses extension for paste conversion flow

	test("Invalid view link returns 404", async ({ request }) => {
		// Backend-only test: no extension needed
		const invalidLink = "http://localhost:8080/view/invalid-id-12345";
		const response = await request.get(invalidLink, { maxRedirects: 0 });
		
		expect(response.status()).toBe(404);
	});

	// Single UI test: paste convert -> link -> view (requires extension)
	test("One-time link from paste conversion works", async ({ request }) => {
		const { context, page: initPage } = await launchWithExtension(process.env.E2E_EXTENSION_PATH);
		
		try {
			const testPage = await context.newPage();
			await testPage.goto("http://good.test:4173/paste", { waitUntil: "domcontentloaded" });
			
			await waitForContentScript(testPage);
			
			const textarea = testPage.locator("#t");
			await textarea.fill("");
			
			await e2ePaste(testPage, "#t", "password=supersecret123");
			await expect(testPage.locator('[data-testid="paste-modal"]')).toBeVisible();
			
			// Convert to link
			await testPage.click('[data-testid="paste-convert-link"]');
			
			// Wait for link to be inserted
			await testPage.waitForFunction(
				(selector) => {
					const el = document.querySelector(selector);
					return el && el.value.includes("http://localhost:8080/view/");
				},
				"#t",
				{ timeout: 10000 }
			);
			
			const link = await textarea.inputValue();
			const viewLink = link.match(/http:\/\/localhost:8080\/view\/[^\s]+/)?.[0];
			expect(viewLink).toBeTruthy();
			
			// First view should succeed
			const firstView = await request.get(viewLink, { maxRedirects: 0 });
			expect([200, 302, 307]).toContain(firstView.status());
			
			// Second view should fail
			const secondView = await request.get(viewLink, { maxRedirects: 0 });
			expect([403, 410]).toContain(secondView.status());
			
			await testPage.close();
		} finally {
			await context.close();
		}
	});
});


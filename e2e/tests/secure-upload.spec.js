const path = require("path");
const { test, expect } = require("@playwright/test");
const { launchWithExtension } = require("../utils/launch-extension");
const { resetExtensionState, getExtensionId, waitForContentScript } = require("../utils/test-helpers");

test.describe("Secure Upload", () => {
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

	test("Secure Share button injected next to file input", async () => {
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/upload", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(testPage);
		
		// Verify Secure Share button is visible
		await expect(testPage.locator('[data-testid="secure-share-btn"]')).toBeVisible();
		
		// Verify button is near file input
		const fileInput = testPage.locator('input[type="file"]');
		await expect(fileInput).toBeVisible();
	});

	test("Click Secure Share button opens upload modal", async () => {
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/upload", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(testPage);
		
		// Click Secure Share button
		await testPage.click('[data-testid="secure-share-btn"]');
		
		// Wait for modal to appear
		await expect(testPage.locator("#pg-drop-zone")).toBeVisible();
		await expect(testPage.locator("#pg-hidden-file-input")).toBeVisible();
	});

	test("Upload file and receive one-time view link", async () => {
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/upload", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(testPage);
		
		// Click Secure Share button to open modal
		await testPage.click('[data-testid="secure-share-btn"]');
		
		// Wait for upload modal
		await expect(testPage.locator("#pg-drop-zone")).toBeVisible();
		
		// Set file directly in the hidden input
		const fileInput = testPage.locator("#pg-hidden-file-input");
		await fileInput.setInputFiles(
			path.join(__dirname, "..", "fixtures", "secret.txt")
		);
		
		// Wait for upload to complete and link to appear
		await expect(testPage.locator('[data-testid="last-link"]')).toBeVisible({
			timeout: 10000,
		});
		
		const link = await testPage.locator('[data-testid="last-link"]').textContent();
		expect(link).toContain("http://localhost:8080/view/");
	});

	test("Upload progress bar shows during upload", async () => {
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/upload", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(testPage);
		
		await testPage.click('[data-testid="secure-share-btn"]');
		await expect(testPage.locator("#pg-drop-zone")).toBeVisible();
		
		// Set up route to delay upload response
		let uploadStarted = false;
		await context.route("http://localhost:8080/api/upload", async (route) => {
			uploadStarted = true;
			await route.continue();
		});
		
		const fileInput = testPage.locator("#pg-hidden-file-input");
		await fileInput.setInputFiles(
			path.join(__dirname, "..", "fixtures", "secret.txt")
		);
		
		// Wait for progress bar to appear
		await testPage.waitForFunction(() => {
			const progress = document.getElementById("pg-upload-progress");
			return progress && progress.style.display !== "none";
		}, { timeout: 5000 });
		
		expect(uploadStarted).toBe(true);
	});

	test("Upload failure shows error message", async () => {
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/upload", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(testPage);
		
		// Mock upload endpoint to return error
		await context.route("http://localhost:8080/api/upload", async (route) => {
			await route.fulfill({
				status: 500,
				contentType: "application/json",
				body: JSON.stringify({ error: "Upload failed" }),
			});
		});
		
		await testPage.click('[data-testid="secure-share-btn"]');
		await expect(testPage.locator("#pg-drop-zone")).toBeVisible();
		
		const fileInput = testPage.locator("#pg-hidden-file-input");
		await fileInput.setInputFiles(
			path.join(__dirname, "..", "fixtures", "secret.txt")
		);
		
		// Wait for error message
		await expect(testPage.locator("#pg-upload-progress")).toBeVisible();
		const errorText = await testPage.locator("#pg-progress-text").textContent();
		expect(errorText).toContain("Upload failed");
	});

	test("Close button closes modal", async () => {
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/upload", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(testPage);
		
		await testPage.click('[data-testid="secure-share-btn"]');
		await expect(testPage.locator("#pg-drop-zone")).toBeVisible();
		
		// Click close button
		await testPage.click("#pg-modal-close");
		
		// Verify modal is closed
		await expect(testPage.locator("#pg-drop-zone")).not.toBeVisible();
	});

	test("Secure Share button appears for multiple file inputs", async () => {
		const testPage = await context.newPage();
		
		// Navigate to a real matched origin first
		await testPage.goto("http://good.test:4173/", { waitUntil: "domcontentloaded" });
		await waitForContentScript(testPage);
		
		// Replace DOM with test content
		await testPage.evaluate(() => {
			document.body.innerHTML = `
				<input type="file" id="input1">
				<input type="file" id="input2">
			`;
		});
		
		await testPage.waitForTimeout(500);
		
		// Verify Secure Share buttons appear for both inputs
		const buttons = testPage.locator('[data-testid="secure-share-btn"]');
		await expect(buttons).toHaveCount(2);
	});

	test("Upload modal handles file type validation", async () => {
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/upload", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(testPage);
		
		await testPage.click('[data-testid="secure-share-btn"]');
		await expect(testPage.locator("#pg-drop-zone")).toBeVisible();
		
		// Mock upload endpoint to reject file type
		await context.route("http://localhost:8080/api/upload", async (route) => {
			await route.fulfill({
				status: 403,
				contentType: "application/json",
				body: JSON.stringify({ error: "File type not allowed" }),
			});
		});
		
		const fileInput = testPage.locator("#pg-hidden-file-input");
		await fileInput.setInputFiles(
			path.join(__dirname, "..", "fixtures", "secret.txt")
		);
		
		// Wait for error message
		await expect(testPage.locator("#pg-progress-text")).toContainText("File type not allowed", {
			timeout: 5000,
		});
	});
});


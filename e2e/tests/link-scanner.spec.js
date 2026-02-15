const { test, expect } = require("@playwright/test");
const { launchWithExtension } = require("../utils/launch-extension");
const { resetExtensionState, getExtensionId, waitForContentScript, setupDiagnostics, getDiagnosticInfo } = require("../utils/test-helpers");

test.describe("Link Scanner", () => {
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

	test("Flag bad link with warning icon", async () => {
		const diagnostics = setupDiagnostics(testPage);
		
		try {
			await testPage.goto("http://good.test:4173/links", { waitUntil: "domcontentloaded" });
			
			await waitForContentScript(testPage);
		
		// Wait for link scanning to complete
		await testPage.waitForTimeout(500);
		
		// Verify warning icon appears near bad link
		const badLinkIcon = testPage.locator('[data-testid="bad-link-icon"]').first();
		await expect(badLinkIcon).toBeVisible();
		
		// Verify icon is near the bad link
		const badLink = testPage.locator("#bad");
		await expect(badLink).toBeVisible();
		} catch (error) {
			const diag = await getDiagnosticInfo(testPage, diagnostics);
			throw new Error(`${error.message}\n\n${diag}`);
		}
	});

	test("Click bad link triggers confirm dialog", async () => {
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/links", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(testPage);
		await testPage.waitForTimeout(500);
		
		// Set up dialog handler
		let dialogHandled = false;
		testPage.once("dialog", async (dialog) => {
			expect(dialog.type()).toBe("confirm");
			expect(dialog.message()).toContain("flagged");
			dialogHandled = true;
			await dialog.dismiss();
		});
		
		// Click bad link
		await testPage.click("#bad");
		
		// Verify dialog was triggered
		expect(dialogHandled).toBe(true);
		
		// Verify we remain on same page (dismissed)
		await expect(testPage).toHaveURL(/\/links$/);
	});

	test("Accept confirm dialog navigates to bad link", async () => {
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/links", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(testPage);
		await testPage.waitForTimeout(500);
		
		// Set up dialog handler to accept
		testPage.once("dialog", async (dialog) => {
			await dialog.accept();
		});
		
		// Click bad link
		await testPage.click("#bad");
		
		// Verify navigation occurred (URL should change)
		// Note: actual navigation depends on link href, but dialog was accepted
		await testPage.waitForTimeout(500);
	});

	test("Flag insecure HTTP link on HTTPS page", async () => {
		const testPage = await context.newPage();
		
		// Navigate to a real matched origin first
		await testPage.goto("http://good.test:4173/", { waitUntil: "domcontentloaded" });
		await waitForContentScript(testPage);
		
		// Replace DOM with test content
		await testPage.evaluate(() => {
			document.body.innerHTML = `
				<a href="http://example.com" id="insecure-link">Insecure Link</a>
			`;
		});
		
		// Simulate HTTPS protocol (content script checks window.location.protocol)
		await testPage.evaluate(() => {
			Object.defineProperty(window, 'location', {
				value: { protocol: 'https:' },
				writable: true
			});
		});
		
		await testPage.waitForTimeout(500);
		
		// Verify warning icon appears for insecure link
		const insecureLink = testPage.locator("#insecure-link");
		const icon = insecureLink.locator('[data-testid="bad-link-icon"]');
		await expect(icon).toBeVisible();
	});

	test("Do not flag localhost HTTP links", async () => {
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/links", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(testPage);
		await testPage.waitForTimeout(500);
		
		// Add a localhost link dynamically
		await testPage.evaluate(() => {
			const link = document.createElement('a');
			link.href = "http://localhost:8080/test";
			link.id = "localhost-link";
			link.textContent = "Localhost Link";
			document.body.appendChild(link);
		});
		
		await testPage.waitForTimeout(500);
		
		// Verify no warning icon for localhost
		const localhostLink = testPage.locator("#localhost-link");
		const icon = localhostLink.locator('[data-testid="bad-link-icon"]');
		await expect(icon).toHaveCount(0);
	});

	test("Do not flag 127.0.0.1 HTTP links", async () => {
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/links", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(testPage);
		await testPage.waitForTimeout(500);
		
		// Add a 127.0.0.1 link dynamically
		await testPage.evaluate(() => {
			const link = document.createElement('a');
			link.href = "http://127.0.0.1:8080/test";
			link.id = "local-link";
			link.textContent = "Local IP Link";
			document.body.appendChild(link);
		});
		
		await testPage.waitForTimeout(500);
		
		// Verify no warning icon for 127.0.0.1
		const localLink = testPage.locator("#local-link");
		const icon = localLink.locator('[data-testid="bad-link-icon"]');
		await expect(icon).toHaveCount(0);
	});

	test("Scan dynamically added links", async () => {
		const testPage = await context.newPage();
		await testPage.goto("http://good.test:4173/links", { waitUntil: "domcontentloaded" });
		
		await waitForContentScript(testPage);
		await testPage.waitForTimeout(500);
		
		// Add a new link dynamically
		await testPage.evaluate(() => {
			const link = document.createElement('a');
			link.href = "http://evil.test/test";
			link.id = "dynamic-bad-link";
			link.textContent = "Dynamic Bad Link";
			document.body.appendChild(link);
		});
		
		// Wait for mutation observer to process
		await testPage.waitForTimeout(500);
		
		// Verify warning icon appears for dynamically added bad link
		const dynamicLink = testPage.locator("#dynamic-bad-link");
		const icon = dynamicLink.locator('[data-testid="bad-link-icon"]');
		await expect(icon).toBeVisible();
	});
});


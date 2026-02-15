const path = require("path");
const os = require("os");

// Calculate workers: min(3, os.cpus().length - 1) or use E2E_WORKERS env var
const cpuCount = os.cpus().length;
const defaultWorkers = Math.min(3, Math.max(1, cpuCount - 1));
const workers = process.env.E2E_WORKERS ? parseInt(process.env.E2E_WORKERS, 10) : defaultWorkers;

// Extension project workers: 1-2 for stability on Windows
const extensionWorkers = Math.min(2, workers);

module.exports = {
  testDir: path.join(__dirname, "tests"),
  timeout: 60_000,
  expect: { timeout: 10_000 },
  retries: 1,
  globalSetup: path.join(__dirname, "global-setup.js"),
  globalTeardown: path.join(__dirname, "global-teardown.js"),
  projects: [
    {
      name: "backend",
      testMatch: /backend-one-time\.spec\.js/,
      fullyParallel: true,
      workers: workers,
      use: {
        headless: true,
        screenshot: "only-on-failure",
        video: "retain-on-failure",
        trace: "retain-on-failure"
      }
    },
    {
      name: "extension",
      testMatch: /(?!backend-one-time\.spec\.js).*\.spec\.js/,
      fullyParallel: true,
      workers: extensionWorkers,
      use: {
        headless: false, // CRITICAL: Extension tests MUST run in headed mode
        screenshot: "only-on-failure",
        video: "retain-on-failure",
        trace: "retain-on-failure"
      }
    }
  ]
};


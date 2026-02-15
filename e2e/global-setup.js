const path = require("path");
const { spawn } = require("child_process");
const fs = require("fs");
const { waitHttp } = require("./utils/wait-http");
const { prepareExtension } = require("./utils/prepare-extension");

module.exports = async () => {
  const repoRoot = path.resolve(__dirname, "..");

  // Prepare extension copy for deterministic malicious_domains.json
  const extensionPath = await prepareExtension({ repoRoot });
  process.env.E2E_EXTENSION_PATH = extensionPath;

  // Check if backend is already running
  let backendSpawned = false;
  let backend = null;
  let backendPid = null;

  try {
    // Try to reach existing backend with short timeout
    await waitHttp("http://localhost:8080/health", { timeoutMs: 2000, intervalMs: 100 });
    console.log("Backend already running on port 8080, reusing it");
  } catch (e) {
    // Backend not running, spawn it
    console.log("Starting backend server...");
    backend = spawn("go", ["run", "main.go", "serve", "--addr", ":8080"], {
      cwd: repoRoot,
      env: {
        ...process.env,
        BACKEND_TEST_MODE: process.env.BACKEND_TEST_MODE || "1"
      },
      stdio: "inherit"
    });
    backendPid = backend.pid;
    backendSpawned = true;
  }

  // Start testapp
  const testappDir = path.join(__dirname, "testapp");
  const testapp = spawn("node", ["server.js"], {
    cwd: testappDir,
    env: { ...process.env, TESTAPP_PORT: "4173" },
    stdio: "inherit"
  });

  // Wait for services - CRITICAL: do not proceed until both are ready
  console.log("Waiting for backend health endpoint...");
  try {
    await waitHttp("http://localhost:8080/health", { timeoutMs: 30000, intervalMs: 500 });
    console.log("✓ Backend is ready");
  } catch (error) {
    console.error("✗ Backend health check failed");
    if (backendSpawned && backend) {
      backend.kill();
    }
    testapp.kill();
    throw new Error(
      `GLOBAL SETUP FAILED: Backend is not responding.\n` +
      `Health endpoint http://localhost:8080/health did not become available.\n` +
      `Error: ${error.message}\n` +
      `\n` +
      `Possible causes:\n` +
      `- Backend failed to start (check Go errors above)\n` +
      `- Port 8080 is already in use by another process\n` +
      `- Backend crashed during startup\n` +
      `\n` +
      `Do not proceed with tests until backend is running.`
    );
  }

  console.log("Waiting for testapp server...");
  try {
    await waitHttp("http://127.0.0.1:4173/", { timeoutMs: 30000, intervalMs: 500 });
    console.log("✓ Testapp is ready");
  } catch (error) {
    console.error("✗ Testapp health check failed");
    if (backendSpawned && backend) {
      backend.kill();
    }
    testapp.kill();
    throw new Error(
      `GLOBAL SETUP FAILED: Testapp is not responding.\n` +
      `Testapp endpoint http://127.0.0.1:4173/ did not become available.\n` +
      `Error: ${error.message}\n` +
      `\n` +
      `Possible causes:\n` +
      `- Testapp failed to start (check Node.js errors above)\n` +
      `- Port 4173 is already in use by another process\n` +
      `- Testapp server crashed during startup\n` +
      `\n` +
      `Do not proceed with tests until testapp is running.`
    );
  }

  // Save PIDs for teardown
  fs.writeFileSync(
    path.join(__dirname, ".pids"),
    JSON.stringify({ 
      backendPid: backendPid, 
      testappPid: testapp.pid, 
      extensionPath,
      backendSpawned: backendSpawned
    }, null, 2)
  );
};


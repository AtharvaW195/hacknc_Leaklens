const path = require("path");
const fs = require("fs");
const { spawnSync } = require("child_process");

function killPidWindows(pid) {
  if (!pid) return;
  try {
    // Use taskkill on Windows for robust process termination
    spawnSync("taskkill", ["/PID", String(pid), "/T", "/F"], {
      stdio: "ignore"
    });
  } catch {}
}

function killPid(pid) {
  if (!pid) return;
  try { 
    process.kill(pid); 
  } catch {}
}

module.exports = async () => {
  const p = path.join(__dirname, ".pids");
  if (!fs.existsSync(p)) return;
  const { backendPid, testappPid, backendSpawned } = JSON.parse(fs.readFileSync(p, "utf-8"));
  
  // Always kill testapp (we always spawn it)
  if (process.platform === "win32") {
    killPidWindows(testappPid);
  } else {
    killPid(testappPid);
  }
  
  // Only kill backend if we spawned it
  if (backendSpawned && backendPid) {
    if (process.platform === "win32") {
      killPidWindows(backendPid);
    } else {
      killPid(backendPid);
    }
  }
};


const fs = require("fs");
const fsp = fs.promises;
const path = require("path");
const os = require("os");

async function copyDir(src, dest) {
  await fsp.mkdir(dest, { recursive: true });
  const entries = await fsp.readdir(src, { withFileTypes: true });
  for (const e of entries) {
    const s = path.join(src, e.name);
    const d = path.join(dest, e.name);
    if (e.isDirectory()) await copyDir(s, d);
    else await fsp.copyFile(s, d);
  }
}

async function prepareExtension({ repoRoot }) {
  const src = path.join(repoRoot, "extension");
  const dest = await fsp.mkdtemp(path.join(os.tmpdir(), "pg-ext-"));
  await copyDir(src, dest);

  // overwrite malicious_domains.json for deterministic tests
  const maliciousPath = path.join(dest, "malicious_domains.json");
  const testList = ["evil.test"]; // keep it minimal so good.test remains unblocked
  await fsp.writeFile(maliciousPath, JSON.stringify(testList, null, 2), "utf-8");

  return dest;
}

module.exports = { prepareExtension };


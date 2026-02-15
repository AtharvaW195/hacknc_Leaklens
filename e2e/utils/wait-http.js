async function waitHttp(url, { timeoutMs = 15_000, intervalMs = 250 } = {}) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(url);
      if (res.ok) return;
    } catch {}
    await new Promise(r => setTimeout(r, intervalMs));
  }
  throw new Error(`Timed out waiting for ${url}`);
}

module.exports = { waitHttp };


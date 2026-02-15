const UPLOAD_URL = "http://localhost:8080/api/upload";
const ANALYZE_URL = "http://localhost:8080/analyze";

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg.type === "upload") {
    const { arrayBuffer, fileName, contentType } = msg;
    if (!arrayBuffer || !fileName) {
      sendResponse({ error: "Missing file data" });
      return false;
    }
    const blob = new Blob([arrayBuffer], { type: contentType || "application/octet-stream" });
    const formData = new FormData();
    formData.append("file", blob, fileName);
    fetch(UPLOAD_URL, { method: "POST", body: formData })
      .then(async (r) => {
        const text = await r.text();
        if (!r.ok) {
          let msg = r.statusText;
          try {
            const j = JSON.parse(text);
            if (j && j.error) msg = j.error;
          } catch (_) {
            if (text) msg = text;
          }
          throw new Error(msg);
        }
        return JSON.parse(text);
      })
      .then((data) => sendResponse({ viewLink: data.viewLink, fileId: data.fileId }))
      .catch((e) => sendResponse({ error: e.message || "Upload failed" }));
    return true;
  }
  
  if (msg.type === "ANALYZE_TEXT") {
    const { text } = msg;
    if (!text) {
      sendResponse({ error: "Missing text" });
      return false;
    }
    fetch(ANALYZE_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ text: text })
    })
      .then(async (r) => {
        const responseText = await r.text();
        if (!r.ok) {
          let errorMsg = r.statusText;
          try {
            const j = JSON.parse(responseText);
            if (j && j.error) errorMsg = j.error;
          } catch (_) {
            if (responseText) errorMsg = responseText;
          }
          throw new Error(errorMsg);
        }
        return JSON.parse(responseText);
      })
      .then((data) => sendResponse({ 
        overall_risk: data.overall_risk || "low",
        risk_rationale: data.risk_rationale,
        findings: data.findings || []
      }))
      .catch((e) => sendResponse({ error: e.message || "Analysis failed" }));
    return true;
  }
  
  return false;
});

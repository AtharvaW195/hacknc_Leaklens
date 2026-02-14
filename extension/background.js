const UPLOAD_URL = "http://localhost:8080/api/upload";

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg.type !== "upload") return false;
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
});

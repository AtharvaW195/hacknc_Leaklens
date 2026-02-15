const express = require("express");
const path = require("path");

const app = express();
const pagesDir = path.join(__dirname, "pages");

app.get("/", (_, res) => res.sendFile(path.join(pagesDir, "index.html")));
app.get("/upload", (_, res) => res.sendFile(path.join(pagesDir, "upload.html")));
app.get("/paste", (_, res) => res.sendFile(path.join(pagesDir, "paste.html")));
app.get("/links", (_, res) => res.sendFile(path.join(pagesDir, "links.html")));

const port = process.env.TESTAPP_PORT || 4173;
app.listen(port, "127.0.0.1", () => {
  console.log(`testapp listening on http://127.0.0.1:${port}`);
});


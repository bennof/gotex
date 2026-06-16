const sourceInput = document.getElementById("sourceInput");
const logOutput = document.getElementById("logOutput");
const buildButton = document.getElementById("buildButton");
const showButton = document.getElementById("showButton");
const downloadButton = document.getElementById("downloadButton");
const assetList = document.getElementById("assetList");
const dropArea = document.getElementById("dropArea");
const fileInput = document.getElementById("fileInput");
const filePickerButton = document.getElementById("filePickerButton");

let sessionId = "";
let sessionUrl = "";
let downloadUrl = "";
const assetEntries = new Map();

function clearSessionState() {
  sessionId = "";
  sessionUrl = "";
  downloadUrl = "";
  showButton.disabled = true;
  downloadButton.disabled = true;

  for (const asset of assetEntries.values()) {
    if (asset.previewUrl) {
      URL.revokeObjectURL(asset.previewUrl);
    }
  }

  assetEntries.clear();
  renderAssetList();
}

async function readErrorMessage(response) {
  const text = (await response.text()).trim();
  return text || `HTTP ${response.status}`;
}

async function ensureSession() {
  if (sessionId) {
    return sessionId;
  }

  const response = await fetch("/session", {
    method: "POST",
  });
  if (!response.ok) {
    throw new Error(await readErrorMessage(response));
  }

  const data = await response.json();
  if (!data.id || !data.path) {
    throw new Error("No session information returned");
  }

  sessionId = data.id;
  sessionUrl = data.path;
  appendLog(`Session ready: ${sessionId}`);
  return sessionId;
}

function setLog(text) {
  logOutput.textContent = text;
}

function appendLog(text) {
  const current = logOutput.textContent.trim();
  logOutput.textContent = current ? `${current}\n${text}` : text;
  logOutput.scrollTop = logOutput.scrollHeight;
}

function setBusy(isBusy) {
  buildButton.disabled = isBusy;
  filePickerButton.disabled = isBusy;
  fileInput.disabled = isBusy;
  sourceInput.disabled = isBusy;
}

function formatBytes(size) {
  if (size < 1024) {
    return `${size} B`;
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function renderAssetList() {
  assetList.innerHTML = "";

  if (assetEntries.size === 0) {
    const emptyItem = document.createElement("li");
    emptyItem.className = "asset-empty";
    emptyItem.textContent = "No uploaded assets yet.";
    assetList.appendChild(emptyItem);
    return;
  }

  for (const asset of assetEntries.values()) {
    const item = document.createElement("li");
    item.className = "asset-item";
    item.draggable = true;

    item.addEventListener("dragstart", (event) => {
      event.dataTransfer.setData("text/plain", asset.name);
      event.dataTransfer.effectAllowed = "copy";
    });

    let thumb;
    if (asset.previewUrl) {
      thumb = document.createElement("img");
      thumb.className = "asset-thumb";
      thumb.src = asset.previewUrl;
      thumb.alt = asset.name;
    } else {
      thumb = document.createElement("div");
      thumb.className = "asset-thumb asset-thumb-placeholder";
      thumb.textContent = "PDF";
    }

    const meta = document.createElement("div");
    meta.className = "asset-meta";

    const name = document.createElement("span");
    name.className = "asset-name";
    name.textContent = asset.name;

    const note = document.createElement("span");
    note.className = "asset-note";
    note.textContent = formatBytes(asset.size);

    meta.appendChild(name);
    meta.appendChild(note);
    item.appendChild(thumb);
    item.appendChild(meta);
    assetList.appendChild(item);
  }
}

function insertAtCursor(text) {
  const start = sourceInput.selectionStart;
  const end = sourceInput.selectionEnd;
  const value = sourceInput.value;
  sourceInput.value = `${value.slice(0, start)}${text}${value.slice(end)}`;
  const nextPos = start + text.length;
  sourceInput.focus();
  sourceInput.setSelectionRange(nextPos, nextPos);
}

async function uploadFile(file) {
  const id = await ensureSession();
  const ext = file.name.split(".").pop()?.toLowerCase() || "";
  if (!["jpg", "jpeg", "png", "pdf"].includes(ext)) {
    appendLog(`Skipped unsupported file: ${file.name}`);
    return;
  }

  const formData = new FormData();
  formData.append("file", file, file.name);

  const response = await fetch(`/session/${id}/upload`, {
    method: "POST",
    body: formData,
  });

  if (!response.ok) {
    throw new Error(`${file.name}: ${await readErrorMessage(response)}`);
  }

  const data = await response.json();
  const previewUrl = file.type === "image/png" || file.type === "image/jpeg"
    ? URL.createObjectURL(file)
    : "";

  const existing = assetEntries.get(data.name);
  if (existing?.previewUrl) {
    URL.revokeObjectURL(existing.previewUrl);
  }

  assetEntries.set(data.name, {
    name: data.name,
    size: data.bytes || file.size,
    previewUrl,
  });

  renderAssetList();
  appendLog(`Uploaded: ${data.name}`);
}

async function handleFiles(fileList) {
  const files = Array.from(fileList);
  if (files.length === 0) {
    return;
  }

  setBusy(true);
  try {
    for (const file of files) {
      await uploadFile(file);
    }
  } catch (error) {
    appendLog(`Error: ${error.message}`);
  } finally {
    setBusy(false);
    fileInput.value = "";
  }
}

async function consumeBuildStream(body) {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { value, done } = await reader.read();
    if (done) {
      buffer += decoder.decode();
      break;
    }

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split("\n");
    buffer = lines.pop() || "";

    for (const line of lines) {
      if (!line.trim()) {
        continue;
      }
      handleBuildMessage(JSON.parse(line));
    }
  }

  if (buffer.trim()) {
    handleBuildMessage(JSON.parse(buffer));
  }
}

function handleBuildMessage(message) {
  if (message.type === "log" && message.data) {
    appendLog(message.data);
    return;
  }

  if (message.type === "error" && message.data) {
    appendLog(`Error: ${message.data}`);
    return;
  }

  if (message.type === "done" && message.url) {
    downloadUrl = message.url;
    if (!sessionUrl) {
      sessionUrl = message.url;
    }
    showButton.disabled = false;
    downloadButton.disabled = false;
    appendLog(`Build finished: ${message.url}`);
  }
}

async function buildDocument() {
  setBusy(true);
  showButton.disabled = true;
  downloadButton.disabled = true;
  downloadUrl = "";
  setLog("Build started ...");

  try {
    const id = await ensureSession();
    appendLog(`Compiling in session ${id} ...`);

    const response = await fetch(`/session/${id}/compile`, {
      method: "POST",
      headers: {
        "Content-Type": "text/plain; charset=utf-8",
      },
      body: sourceInput.value,
    });

    if (!response.ok) {
      throw new Error(await readErrorMessage(response));
    }

    if (!response.body) {
      throw new Error("No response stream available");
    }

    await consumeBuildStream(response.body);
  } catch (error) {
    appendLog(`Error: ${error.message}`);
  } finally {
    setBusy(false);
  }
}

function downloadDocument() {
  if (!downloadUrl) {
    return;
  }

  const targetUrl = `${downloadUrl}?dl=1`;
  clearSessionState();
  appendLog("Download started. Session will be recreated on the next action.");
  window.location.href = targetUrl;
}

function showDocument() {
  if (!downloadUrl) {
    return;
  }

  window.open(downloadUrl, "_blank", "noopener");
}

filePickerButton.addEventListener("click", () => fileInput.click());
fileInput.addEventListener("change", () => handleFiles(fileInput.files));

dropArea.addEventListener("dragover", (event) => {
  event.preventDefault();
  dropArea.classList.add("is-dragover");
});

dropArea.addEventListener("dragleave", () => {
  dropArea.classList.remove("is-dragover");
});

dropArea.addEventListener("drop", (event) => {
  event.preventDefault();
  dropArea.classList.remove("is-dragover");
  handleFiles(event.dataTransfer.files);
});

sourceInput.addEventListener("dragover", (event) => {
  if (event.dataTransfer.types.includes("text/plain")) {
    event.preventDefault();
  }
});

sourceInput.addEventListener("drop", (event) => {
  const text = event.dataTransfer.getData("text/plain");
  if (!text) {
    return;
  }
  event.preventDefault();
  insertAtCursor(text);
});

buildButton.addEventListener("click", buildDocument);
showButton.addEventListener("click", showDocument);
downloadButton.addEventListener("click", downloadDocument);

renderAssetList();

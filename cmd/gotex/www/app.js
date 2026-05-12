const sourceInput = document.getElementById("sourceInput");
const logOutput = document.getElementById("logOutput");
const loginButton = document.getElementById("loginButton");
const buildButton = document.getElementById("buildButton");
const showButton = document.getElementById("showButton");
const downloadButton = document.getElementById("downloadButton");
const assetList = document.getElementById("assetList");
const dropArea = document.getElementById("dropArea");
const fileInput = document.getElementById("fileInput");
const filePickerButton = document.getElementById("filePickerButton");
const loginPanel = document.getElementById("loginPanel");
const usernameInput = document.getElementById("usernameInput");
const passwordInput = document.getElementById("passwordInput");
const loginSubmitButton = document.getElementById("loginSubmitButton");
const loginStatus = document.getElementById("loginStatus");

let downloadUrl = "";
let sessionId = "";
let authUsername = "";
let authPassword = "";
const assetEntries = new Map();

async function readErrorMessage(response) {
  const text = (await response.text()).trim();
  return text || `HTTP ${response.status}`;
}

function authHeaders(extraHeaders = {}) {
  if (!authUsername || !authPassword) {
    return extraHeaders;
  }

  return {
    ...extraHeaders,
    Authorization: `Basic ${btoa(`${authUsername}:${authPassword}`)}`,
  };
}

function setLoginStatus(text) {
  loginStatus.textContent = text;
}

function toggleLoginPanel() {
  loginPanel.hidden = !loginPanel.hidden;
}

async function checkLogin() {
  const username = usernameInput.value.trim();
  const password = passwordInput.value;

  if (!username || !password) {
    setLoginStatus("Please enter username and password.");
    return;
  }

  setLoginStatus("Checking login ...");

  try {
    const response = await fetch("/auth", {
      method: "HEAD",
      headers: authHeaders({
        Authorization: `Basic ${btoa(`${username}:${password}`)}`,
      }),
    });

    if (response.ok) {
      authUsername = username;
      authPassword = password;
      setLoginStatus("Login successful. Credentials will be sent with requests.");
      appendLog("Login successful.");
      return;
    }

    if (response.status === 401 || response.status === 403) {
      authUsername = "";
      authPassword = "";
      setLoginStatus("Login failed. Invalid username or password.");
      appendLog("Login failed: invalid credentials.");
      return;
    }

    authUsername = "";
    authPassword = "";
    setLoginStatus("Auth endpoint unavailable or not configured.");
    appendLog(`Login unavailable: HTTP ${response.status}`);
  } catch (error) {
    authUsername = "";
    authPassword = "";
    setLoginStatus("Auth endpoint unavailable or not configured.");
    appendLog(`Login unavailable: ${error.message}`);
  }
}

async function ensureSession() {
  if (sessionId) {
    return sessionId;
  }

  const response = await fetch("/new", {
    headers: authHeaders(),
  });
  if (!response.ok) {
    throw new Error(`Session HTTP ${response.status}`);
  }

  const data = await response.json();
  if (!data.id) {
    throw new Error("Keine Session-ID erhalten");
  }

  sessionId = data.id;
  appendLog(`Session: ${sessionId}`);
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
    emptyItem.textContent = "Noch keine Dateien hochgeladen.";
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
    appendLog(`Übersprungen: ${file.name}`);
    return;
  }

  const formData = new FormData();
  formData.append("file", file, file.name);

  const response = await fetch(`/assets/${id}`, {
    method: "POST",
    headers: authHeaders(),
    body: formData,
  });

  if (!response.ok) {
    throw new Error(`${file.name}: ${await readErrorMessage(response)}`);
  }

  const resp = await response.json();
  const data = resp.files[0];
  const previewUrl = file.type === "image/png" || file.type === "image/jpeg"
    ? URL.createObjectURL(file)
    : "";

  const existing = assetEntries.get(data.name);
  if (existing?.previewUrl) {
    URL.revokeObjectURL(existing.previewUrl);
  }

  assetEntries.set(data.name, {
    name: data.name,
    size: data.size || file.size,
    previewUrl,
  });

  renderAssetList();
  appendLog(`Hochgeladen: ${data.name}`);
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
    appendLog(`Fehler: ${error.message}`);
  } finally {
    setBusy(false);
    fileInput.value = "";
  }
}

async function buildDocument() {
  setBusy(true);
  showButton.disabled = true;
  downloadButton.disabled = true;
  downloadUrl = "";
  setLog("Build gestartet ...");

  try {
    const id = await ensureSession();
    appendLog(`Build-Session: ${id}`);
    const response = await fetch(`/build/${id}`, {
      method: "POST",
      headers: authHeaders({
        "Content-Type": "text/plain; charset=utf-8",
      }),
      body: sourceInput.value,
    });

    if (!response.ok) {
      throw new Error(await readErrorMessage(response));
    }

    if (!response.body) {
      throw new Error("Kein Response-Stream vorhanden");
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
      const { value, done } = await reader.read();
      if (done) {
        break;
      }

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        if (!line.trim()) {
          continue;
        }

        const message = JSON.parse(line);
        if (message.type === "log" && message.data) {
          appendLog(message.data);
        } else if (message.type === "error" && message.data) {
          appendLog(`Fehler: ${message.data}`);
        } else if (message.type === "done" && message.url) {
          downloadUrl = message.url;
          showButton.disabled = false;
          downloadButton.disabled = false;
          appendLog(`Build abgeschlossen: ${message.url}`);
        }
      }
    }
  } catch (error) {
    appendLog(`Fehler: ${error.message}`);
  } finally {
    setBusy(false);
  }
}

function downloadDocument() {
  if (!downloadUrl) {
    return;
  }
  window.location.href = `${downloadUrl}?dl=1`;
}

function showDocument() {
  if (!downloadUrl) {
    return;
  }
  window.open(downloadUrl, "_blank");
}

filePickerButton.addEventListener("click", () => fileInput.click());
fileInput.addEventListener("change", () => handleFiles(fileInput.files));
loginButton.addEventListener("click", toggleLoginPanel);
loginSubmitButton.addEventListener("click", checkLogin);

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

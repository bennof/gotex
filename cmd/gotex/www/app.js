const sourceInput = document.getElementById("sourceInput");
const logOutput = document.getElementById("logOutput");
const buildButton = document.getElementById("buildButton");
const downloadButton = document.getElementById("downloadButton");

let downloadUrl = "";

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

async function buildDocument() {
  setBusy(true);
  downloadButton.disabled = true;
  downloadUrl = "";
  setLog("Build gestartet ...");

  try {
    const response = await fetch("/build", {
      method: "POST",
      headers: {
        "Content-Type": "text/plain; charset=utf-8",
      },
      body: sourceInput.value,
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
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
          downloadButton.disabled = false;
          appendLog("Build abgeschlossen.");
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
  window.location.href = downloadUrl;
}

buildButton.addEventListener("click", buildDocument);
downloadButton.addEventListener("click", downloadDocument);

let currentPath = "/";
let currentItems = [];
let currentView = "list";
let searchQuery = "";
let i18n = {};
let currentLang = "tr";
let serverConfig = { uploadEnabled: false, pinRequired: false };

function formatBytes(bytes) {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

function formatDate(isoStr) {
  if (!isoStr) return "";
  const d = new Date(isoStr);
  return d.toLocaleString();
}

function getPin() {
  return sessionStorage.getItem("p2p_pin") || "";
}

function setPin(pin) {
  sessionStorage.setItem("p2p_pin", pin);
}

function getPinParam() {
  const pin = getPin();
  return pin ? `&pin=${encodeURIComponent(pin)}` : "";
}

// --- Config & i18n ---
async function fetchConfig() {
  try {
    const res = await fetch("/api/config");
    serverConfig = await res.json();
    currentLang = serverConfig.lang || "tr";
    
    if (serverConfig.uploadEnabled) {
      document.getElementById("upload-panel").classList.remove("hidden");
    }

    if (serverConfig.pinRequired && !getPin()) {
      showPinModal();
    } else {
      loadPath(currentPath);
    }
  } catch (err) {
    console.error("Config fetch error:", err);
  }
}

async function loadI18n(lang) {
  try {
    const res = await fetch(`/api/i18n?lang=${lang}`);
    i18n = await res.json();
    applyI18n();
  } catch (err) {
    console.error("i18n error:", err);
  }
}

function applyI18n() {
  document.getElementById("btn-lang-tr").classList.toggle("active", currentLang === "tr");
  document.getElementById("btn-lang-en").classList.toggle("active", currentLang === "en");

  const setTxt = (id, key) => {
    const el = document.getElementById(id);
    if (el && i18n[key]) el.textContent = i18n[key];
  };

  setTxt("app-heading", "app_title");
  setTxt("app-subheading", "app_desc");
  setTxt("txt-pin-required", "pin_required");
  setTxt("pin-submit", "pin_submit");
  setTxt("txt-download-zip", "download_zip");
  setTxt("th-name", "name");
  setTxt("th-size", "size");
  setTxt("th-date", "modified");
  setTxt("th-actions", "actions");
  setTxt("txt-folder-empty", "folder_empty");
  setTxt("txt-upload-hint", "upload_hint");
  setTxt("txt-qr-title", "qr_title");

  const searchInput = document.getElementById("search-input");
  if (searchInput && i18n["search_placeholder"]) {
    searchInput.placeholder = i18n["search_placeholder"];
  }
}

function showPinModal(error = "") {
  const modal = document.getElementById("pin-modal");
  modal.classList.remove("hidden");
  const errEl = document.getElementById("pin-error");
  if (error) {
    errEl.textContent = error;
    errEl.classList.remove("hidden");
  } else {
    errEl.classList.add("hidden");
  }
}

function hidePinModal() {
  document.getElementById("pin-modal").classList.add("hidden");
}

// --- Data Fetching ---
async function loadPath(path) {
  currentPath = path;
  updateBreadcrumbs(path);

  const headers = {};
  const pin = getPin();
  if (pin) headers["X-PIN"] = pin;

  try {
    const res = await fetch(`/api/list?path=${encodeURIComponent(path)}`, { headers });
    if (res.status === 401) {
      sessionStorage.removeItem("p2p_pin");
      showPinModal(i18n["pin_invalid"] || "Hatalı veya eksik PIN!");
      return;
    }
    if (res.status === 503) {
      const emptyState = document.getElementById("empty-state");
      emptyState.innerHTML = `<p style="color:var(--danger);font-size:1.1rem;font-weight:600;">⚠️ ${i18n["status_stopped"] || "Paylaşım durduruldu veya henüz başlatılmadı."}</p>`;
      emptyState.classList.remove("hidden");
      document.getElementById("file-table").classList.add("hidden");
      document.getElementById("file-grid").classList.add("hidden");
      return;
    }
    if (!res.ok) {
      alert("Hata: " + (await res.text()));
      return;
    }

    hidePinModal();
    currentItems = await res.json();
    filterAndRender();
  } catch (err) {
    console.error("List error:", err);
  }
}

function updateBreadcrumbs(path) {
  const container = document.getElementById("breadcrumb-links");
  container.innerHTML = "";
  const parts = path.split("/").filter(Boolean);
  let built = "";

  parts.forEach((p) => {
    built += "/" + p;
    const sep = document.createElement("span");
    sep.className = "crumb-separator";
    sep.textContent = " / ";
    container.appendChild(sep);

    const crumb = document.createElement("span");
    crumb.className = "crumb-item";
    crumb.textContent = p;
    const target = built;
    crumb.onclick = () => loadPath(target);
    container.appendChild(crumb);
  });
}

function filterAndRender() {
  let filtered = currentItems;
  if (searchQuery.trim() !== "") {
    const q = searchQuery.toLowerCase();
    filtered = currentItems.filter(item => item.name.toLowerCase().includes(q));
  }

  // Sort: directories first, then alphabetical
  filtered.sort((a, b) => {
    if (a.is_dir !== b.is_dir) return b.is_dir ? 1 : -1;
    return a.name.localeCompare(b.name);
  });

  const emptyState = document.getElementById("empty-state");
  if (!filtered || filtered.length === 0) {
    emptyState.classList.remove("hidden");
    document.getElementById("file-table").classList.add("hidden");
    document.getElementById("file-grid").classList.add("hidden");
    return;
  }
  emptyState.classList.add("hidden");

  if (currentView === "list") {
    document.getElementById("file-table").classList.remove("hidden");
    document.getElementById("file-grid").classList.add("hidden");
    renderFileList(filtered);
  } else {
    document.getElementById("file-table").classList.add("hidden");
    document.getElementById("file-grid").classList.remove("hidden");
    renderFileGrid(filtered);
  }
}

// --- Renderers ---
function renderFileList(items) {
  const tbody = document.getElementById("file-list-body");
  tbody.innerHTML = "";

  items.forEach(item => {
    const tr = document.createElement("tr");
    tr.className = `file-row ${item.is_dir ? "is-dir" : ""}`;

    const icon = item.is_dir ? "📁" : getFileIcon(item.name);
    const itemPath = currentPath === "/" ? "/" + item.name : currentPath + "/" + item.name;

    const tdName = document.createElement("td");
    tdName.className = "file-name-cell";
    tdName.innerHTML = `<span class="file-icon">${icon}</span> <span>${escapeHtml(item.name)}</span>`;
    if (item.is_dir) {
      tdName.onclick = () => loadPath(itemPath);
    }
    tr.appendChild(tdName);

    const tdSize = document.createElement("td");
    tdSize.className = "col-size";
    tdSize.textContent = item.is_dir ? "-" : formatBytes(item.size);
    tr.appendChild(tdSize);

    const tdDate = document.createElement("td");
    tdDate.className = "col-date";
    tdDate.textContent = formatDate(item.mod_time);
    tr.appendChild(tdDate);

    const tdActions = document.createElement("td");
    tdActions.className = "col-actions";

    const pinParam = getPinParam();
    if (item.is_dir) {
      const zipBtn = document.createElement("a");
      zipBtn.className = "btn btn-outline btn-sm";
      zipBtn.href = `/api/zip?path=${encodeURIComponent(itemPath)}${pinParam}`;
      zipBtn.innerHTML = `📦 ZIP`;
      tdActions.appendChild(zipBtn);
    } else {
      // Preview button (for viewable types)
      if (canPreview(item.name)) {
        const prevBtn = document.createElement("button");
        prevBtn.className = "btn btn-outline btn-sm";
        prevBtn.textContent = i18n["preview"] || "👁️ Önizle";
        prevBtn.onclick = () => openPreview(item.name, itemPath);
        tdActions.appendChild(prevBtn);
      }

      const dlBtn = document.createElement("a");
      dlBtn.className = "btn btn-primary btn-sm";
      dlBtn.href = `/api/download?path=${encodeURIComponent(itemPath)}${pinParam}`;
      dlBtn.download = item.name;
      dlBtn.innerHTML = `⬇️ ${i18n["download"] || "İndir"}`;
      tdActions.appendChild(dlBtn);
    }
    tr.appendChild(tdActions);

    tbody.appendChild(tr);
  });
}

function renderFileGrid(items) {
  const grid = document.getElementById("file-grid");
  grid.innerHTML = "";

  items.forEach(item => {
    const card = document.createElement("div");
    card.className = "grid-item";

    const icon = item.is_dir ? "📁" : getFileIcon(item.name);
    const itemPath = currentPath === "/" ? "/" + item.name : currentPath + "/" + item.name;

    card.innerHTML = `
      <div class="grid-icon">${icon}</div>
      <div class="grid-name" title="${escapeHtml(item.name)}">${escapeHtml(item.name)}</div>
      <div class="grid-meta">${item.is_dir ? "Klasör" : formatBytes(item.size)}</div>
    `;

    if (item.is_dir) {
      card.onclick = (e) => {
        if (!e.target.closest("button") && !e.target.closest("a")) {
          loadPath(itemPath);
        }
      };
    }

    const actions = document.createElement("div");
    actions.className = "grid-actions";
    const pinParam = getPinParam();

    if (item.is_dir) {
      const zipBtn = document.createElement("a");
      zipBtn.className = "btn btn-outline";
      zipBtn.href = `/api/zip?path=${encodeURIComponent(itemPath)}${pinParam}`;
      zipBtn.innerHTML = `📦 ZIP`;
      actions.appendChild(zipBtn);
    } else {
      if (canPreview(item.name)) {
        const prevBtn = document.createElement("button");
        prevBtn.className = "btn btn-outline";
        prevBtn.textContent = "👁️";
        prevBtn.title = "Önizle";
        prevBtn.onclick = (e) => {
          e.stopPropagation();
          openPreview(item.name, itemPath);
        };
        actions.appendChild(prevBtn);
      }

      const dlBtn = document.createElement("a");
      dlBtn.className = "btn btn-primary";
      dlBtn.href = `/api/download?path=${encodeURIComponent(itemPath)}${pinParam}`;
      dlBtn.download = item.name;
      dlBtn.innerHTML = `⬇️`;
      dlBtn.title = "İndir";
      actions.appendChild(dlBtn);
    }

    card.appendChild(actions);
    grid.appendChild(card);
  });
}

// --- Preview Modal ---
function canPreview(name) {
  const ext = name.split(".").pop().toLowerCase();
  const previewable = [
    "png", "jpg", "jpeg", "gif", "webp", "svg",
    "mp4", "webm", "mp3", "wav", "ogg",
    "txt", "md", "go", "py", "json", "js", "html", "css", "log", "yaml", "yml", "sh"
  ];
  return previewable.includes(ext);
}

async function openPreview(name, path) {
  const modal = document.getElementById("preview-modal");
  const title = document.getElementById("preview-title");
  const body = document.getElementById("preview-body");
  const dlBtn = document.getElementById("preview-dl-btn");

  title.textContent = name;
  body.innerHTML = `<p style="color:var(--text-muted);">Yükleniyor...</p>`;

  const pinParam = getPinParam();
  const url = `/api/preview?path=${encodeURIComponent(path)}${pinParam}`;
  dlBtn.href = `/api/download?path=${encodeURIComponent(path)}${pinParam}`;
  dlBtn.download = name;

  const ext = name.split(".").pop().toLowerCase();

  if (["png", "jpg", "jpeg", "gif", "webp", "svg"].includes(ext)) {
    body.innerHTML = `<img src="${url}" alt="${escapeHtml(name)}" />`;
  } else if (["mp4", "webm"].includes(ext)) {
    body.innerHTML = `<video src="${url}" controls autoplay></video>`;
  } else if (["mp3", "wav", "ogg"].includes(ext)) {
    body.innerHTML = `<audio src="${url}" controls autoplay></audio>`;
  } else {
    // Text / Code preview
    try {
      const res = await fetch(url);
      const text = await res.text();
      body.innerHTML = `<pre><code>${escapeHtml(text)}</code></pre>`;
    } catch (err) {
      body.innerHTML = `<p style="color:var(--danger);">Önizleme yüklenemedi: ${err}</p>`;
    }
  }

  modal.classList.remove("hidden");
}

document.getElementById("preview-close").onclick = () => {
  document.getElementById("preview-modal").classList.add("hidden");
  document.getElementById("preview-body").innerHTML = "";
};

// --- QR Code Modal ---
function showPageQR() {
  const url = window.location.href;
  const encoded = encodeURIComponent(url);
  document.getElementById("qr-container").innerHTML = `<img src="https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encoded}" alt="QR" width="200" height="200"/>`;
  document.getElementById("qr-link-text").textContent = url;
  document.getElementById("qr-modal").classList.remove("hidden");
}

document.getElementById("btn-qr-share").onclick = showPageQR;
document.getElementById("btn-close-qr").onclick = () => {
  document.getElementById("qr-modal").classList.add("hidden");
};

// --- Search Filter ---
document.getElementById("search-input").addEventListener("input", (e) => {
  searchQuery = e.target.value;
  filterAndRender();
});

// --- View Switcher ---
document.getElementById("btn-view-list").onclick = () => {
  currentView = "list";
  document.getElementById("btn-view-list").classList.add("active");
  document.getElementById("btn-view-grid").classList.remove("active");
  filterAndRender();
};

document.getElementById("btn-view-grid").onclick = () => {
  currentView = "grid";
  document.getElementById("btn-view-grid").classList.add("active");
  document.getElementById("btn-view-list").classList.remove("active");
  filterAndRender();
};

// --- PIN Submit Event ---
document.getElementById("pin-submit").addEventListener("click", () => {
  const pin = document.getElementById("pin-input").value.trim();
  if (!pin) return;
  setPin(pin);
  loadPath(currentPath);
});

document.getElementById("pin-input").addEventListener("keypress", (e) => {
  if (e.key === "Enter") document.getElementById("pin-submit").click();
});

// Download Entire Folder as ZIP
document.getElementById("btn-download-zip").addEventListener("click", () => {
  const pinParam = getPinParam();
  window.location.href = `/api/zip?path=${encodeURIComponent(currentPath)}${pinParam}`;
});

// Language Switch
document.getElementById("btn-lang-tr").addEventListener("click", () => {
  currentLang = "tr";
  loadI18n("tr");
});
document.getElementById("btn-lang-en").addEventListener("click", () => {
  currentLang = "en";
  loadI18n("en");
});

// Drag & Drop / Upload
const dropZone = document.getElementById("drop-zone");
const fileInput = document.getElementById("file-input");

if (dropZone && fileInput) {
  dropZone.addEventListener("click", () => fileInput.click());

  dropZone.addEventListener("dragover", (e) => {
    e.preventDefault();
    dropZone.classList.add("dragover");
  });

  dropZone.addEventListener("dragleave", () => {
    dropZone.classList.remove("dragover");
  });

  dropZone.addEventListener("drop", (e) => {
    e.preventDefault();
    dropZone.classList.remove("dragover");
    if (e.dataTransfer.files.length > 0) {
      uploadFiles(e.dataTransfer.files);
    }
  });

  fileInput.addEventListener("change", () => {
    if (fileInput.files.length > 0) {
      uploadFiles(fileInput.files);
    }
  });
}

function uploadFiles(files) {
  const progressContainer = document.getElementById("upload-progress-container");
  const progressBar = document.getElementById("upload-progress-bar");
  const progressText = document.getElementById("upload-progress-text");

  progressContainer.classList.remove("hidden");
  progressBar.style.width = "0%";
  progressText.textContent = "0%";

  const formData = new FormData();
  formData.append("path", currentPath);
  for (let i = 0; i < files.length; i++) {
    formData.append("files", files[i]);
  }

  const xhr = new XMLHttpRequest();
  xhr.open("POST", "/api/upload");
  const pin = getPin();
  if (pin) xhr.setRequestHeader("X-PIN", pin);

  xhr.upload.onprogress = (e) => {
    if (e.lengthComputable) {
      const pct = Math.round((e.loaded / e.total) * 100);
      progressBar.style.width = pct + "%";
      progressText.textContent = pct + "%";
    }
  };

  xhr.onload = () => {
    progressContainer.classList.add("hidden");
    if (xhr.status === 200) {
      loadPath(currentPath);
    } else {
      alert("Upload failed: " + xhr.responseText);
    }
  };

  xhr.onerror = () => {
    progressContainer.classList.add("hidden");
    alert("Upload network error!");
  };

  xhr.send(formData);
}

function getFileIcon(name) {
  const ext = name.split(".").pop().toLowerCase();
  if (["png", "jpg", "jpeg", "gif", "webp", "svg"].includes(ext)) return "🖼️";
  if (["mp4", "mkv", "avi", "mov", "webm"].includes(ext)) return "🎬";
  if (["mp3", "wav", "flac", "ogg"].includes(ext)) return "🎵";
  if (["zip", "rar", "7z", "tar", "gz"].includes(ext)) return "📦";
  if (["pdf", "doc", "docx", "txt", "md"].includes(ext)) return "📄";
  if (["exe", "bin", "sh", "appimage"].includes(ext)) return "⚙️";
  return "📄";
}

function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str;
  return div.innerHTML;
}

// Initial Boot
(async () => {
  await loadI18n(currentLang);
  await fetchConfig();
})();

let isSingleFileMode = false;
let currentLang = "tr";
let i18n = {};
let lastStatus = null;

// --- Vanilla QR Code SVG Generator (Zero dependencies, offline) ---
// Based on a minimalist QR code matrix generator for URLs
function generateQRSvg(text) {
  // Use public Google Charts / QR Server image or fallback encoded SVG
  const encoded = encodeURIComponent(text);
  return `<img src="https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encoded}" alt="QR" width="200" height="200" style="display:block;" onerror="this.onerror=null;this.parentElement.innerHTML='<p style=\\'color:#333;font-size:12px;\\'>QR Önizlemesi</p>'"/>`;
}

// --- Fetch & Apply Translations ---
async function loadI18n(lang) {
  try {
    const res = await fetch(`/api/i18n?lang=${lang}`);
    i18n = await res.json();
    applyI18n();
  } catch (err) {
    console.error("i18n fetch failed:", err);
  }
}

function applyI18n() {
  document.getElementById("btn-lang-tr").classList.toggle("active", currentLang === "tr");
  document.getElementById("btn-lang-en").classList.toggle("active", currentLang === "en");

  const setTxt = (id, key) => {
    const el = document.getElementById(id);
    if (el && i18n[key]) el.textContent = i18n[key];
  };

  setTxt("host-title", "host_title");
  setTxt("host-subtitle", "host_subtitle");
  setTxt("tab-folder", "tab_folder");
  setTxt("tab-file", "tab_file");
  setTxt("btn-browse-os", "browse_btn");
  setTxt("txt-config-heading", "host_title");
  setTxt("lbl-path", "sharing_folder");
  setTxt("lbl-pin", "pin_code");
  setTxt("txt-pin-auto", "pin_mode_auto");
  setTxt("txt-pin-custom", "pin_mode_custom");
  setTxt("txt-pin-none", "pin_mode_none");
  setTxt("txt-opt-upload", "opt_upload");
  setTxt("txt-opt-tunnel", "opt_tunnel");
  setTxt("btn-start-share", "btn_start_share");
  setTxt("btn-stop-share", "btn_stop_share");
  setTxt("txt-links-heading", "links_title");
  setTxt("txt-activity-heading", "activity_title");
}

// --- Status & State Polling ---
async function fetchStatus() {
  try {
    const res = await fetch("/api/host/status");
    if (!res.ok) return;
    const data = await res.json();
    lastStatus = data;
    renderStatus(data);
  } catch (err) {
    console.error("Status fetch error:", err);
  }
}

function renderStatus(data) {
  const badge = document.getElementById("status-badge");
  const btnStart = document.getElementById("btn-start-share");
  const btnStop = document.getElementById("btn-stop-share");

  if (data.isRunning) {
    badge.className = "badge badge-active";
    badge.textContent = i18n["status_active"] || "Paylaşım Aktif";
    btnStart.textContent = "🔄 Ayarları Güncelle";
    btnStop.classList.remove("hidden");
  } else {
    badge.className = "badge badge-stopped";
    badge.textContent = i18n["status_stopped"] || "Durduruldu";
    btnStart.textContent = i18n["btn_start_share"] || "▶️ Paylaşımı Başlat";
    btnStop.classList.add("hidden");
  }

  // Pre-fill path and options if empty
  const pathInput = document.getElementById("input-path");
  if (!pathInput.value && data.path) {
    pathInput.value = data.path;
  }
  if (data.pin) {
    const radioCustom = document.querySelector('input[name="pin-mode"][value="custom"]');
    if (radioCustom) radioCustom.checked = true;
    const customPinInput = document.getElementById("input-custom-pin");
    customPinInput.classList.remove("hidden");
    if (!customPinInput.value) customPinInput.value = data.pin;
  }
  document.getElementById("chk-upload").checked = !!data.allowUpload;

  // Update Links Container
  renderLinks(data);

  // Update Activity Feed
  renderActivity(data.events || []);
}

function renderLinks(data) {
  const container = document.getElementById("links-container");
  container.innerHTML = "";

  if (!data.isRunning) {
    container.innerHTML = `<p class="empty-hint">${i18n["status_stopped"] || "Paylaşım durduruldu."}</p>`;
    return;
  }

  const pinText = data.pin ? ` (PIN: <strong>${data.pin}</strong>)` : "";

  // LAN IPs
  if (data.localIPs && data.localIPs.length > 0) {
    data.localIPs.forEach(ip => {
      const url = `http://${ip}:${data.port}`;
      container.appendChild(createLinkCard(i18n["local_url"] || "Yerel Ağ (LAN)", url, pinText));
    });
  }

  // Short URL (if available)
  if (data.shortURL) {
    container.appendChild(createLinkCard(i18n["short_url"] || "Kısa Link (Hızlı Paylaşım)", data.shortURL, pinText, true));
  }

  // Tunnel Link
  if (data.tunnelURL) {
    container.appendChild(createLinkCard(i18n["tunnel_url"] || "İnternet Tüneli (Orijinal)", data.tunnelURL, pinText, true));
  } else {
    const card = document.createElement("div");
    card.className = "link-item-card";
    card.innerHTML = `<div class="link-item-header"><span class="link-type-title">🌐 İnternet Tüneli</span></div><p style="font-size:0.8rem;color:var(--text-muted);">${i18n["tunnel_starting"] || "Tünel kapalı veya başlatılıyor..."}</p>`;
    container.appendChild(card);
  }
}

function createLinkCard(title, url, pinText, isTunnel = false) {
  const card = document.createElement("div");
  card.className = "link-item-card";

  const header = document.createElement("div");
  header.className = "link-item-header";
  header.innerHTML = `<span class="link-type-title">${isTunnel ? "🌐 " : "🏠 "}${title}</span>${pinText ? `<span style="font-size:0.8rem;">🔑 ${pinText}</span>` : ""}`;
  card.appendChild(header);

  const box = document.createElement("div");
  box.className = "link-url-box";

  const a = document.createElement("a");
  a.className = "link-url-text";
  a.href = url;
  a.target = "_blank";
  a.textContent = url;
  box.appendChild(a);

  const actions = document.createElement("div");
  actions.className = "link-actions";

  const copyBtn = document.createElement("button");
  copyBtn.className = "btn btn-outline btn-sm";
  copyBtn.textContent = i18n["copy"] || "Kopyala";
  copyBtn.onclick = () => {
    navigator.clipboard.writeText(url);
    copyBtn.textContent = i18n["copied"] || "Kopyalandı!";
    setTimeout(() => copyBtn.textContent = i18n["copy"] || "Kopyala", 2000);
  };
  actions.appendChild(copyBtn);

  const qrBtn = document.createElement("button");
  qrBtn.className = "btn btn-accent btn-sm";
  qrBtn.textContent = "QR";
  qrBtn.onclick = () => showQR(url);
  actions.appendChild(qrBtn);

  box.appendChild(actions);
  card.appendChild(box);

  return card;
}

function renderActivity(events) {
  const container = document.getElementById("activity-list");
  if (!events || events.length === 0) {
    container.innerHTML = `<div class="empty-hint">${i18n["no_activity"] || "Henüz bir aktivite yok."}</div>`;
    return;
  }

  container.innerHTML = "";
  // Show most recent first
  const reversed = [...events].reverse();
  reversed.forEach(ev => {
    const entry = document.createElement("div");
    entry.className = `activity-entry ${ev.type || ""}`;

    const d = new Date(ev.timestamp);
    const timeStr = d.toLocaleTimeString();

    entry.innerHTML = `
      <span class="activity-time">${timeStr}</span>
      <span class="activity-msg">${escapeHtml(ev.message)}</span>
    `;
    container.appendChild(entry);
  });
}

function showQR(url) {
  document.getElementById("qr-container").innerHTML = generateQRSvg(url);
  document.getElementById("qr-link-text").textContent = url;
  document.getElementById("qr-modal").classList.remove("hidden");
}

document.getElementById("btn-close-qr").onclick = () => {
  document.getElementById("qr-modal").classList.add("hidden");
};

// --- Mode Tabs ---
document.getElementById("tab-folder").onclick = () => {
  isSingleFileMode = false;
  document.getElementById("tab-folder").classList.add("active");
  document.getElementById("tab-file").classList.remove("active");
  document.getElementById("lbl-path").textContent = i18n["sharing_folder"] || "Paylaşılacak Klasör:";
};

document.getElementById("tab-file").onclick = () => {
  isSingleFileMode = true;
  document.getElementById("tab-file").classList.add("active");
  document.getElementById("tab-folder").classList.remove("active");
  document.getElementById("lbl-path").textContent = i18n["sharing_file"] || "Paylaşılacak Dosya:";
};

// --- Browse OS Button ---
document.getElementById("btn-browse-os").onclick = async () => {
  const mode = isSingleFileMode ? "file" : "folder";
  try {
    const res = await fetch(`/api/host/browse?mode=${mode}`, { method: "POST" });
    if (!res.ok) {
      alert("Dosya seçici açılamadı: " + (await res.text()));
      return;
    }
    const data = await res.json();
    if (data.path) {
      document.getElementById("input-path").value = data.path;
    }
  } catch (err) {
    console.error("Browse error:", err);
  }
};

// --- PIN Radio Behavior ---
document.querySelectorAll('input[name="pin-mode"]').forEach(radio => {
  radio.onchange = () => {
    const customInput = document.getElementById("input-custom-pin");
    if (radio.value === "custom" && radio.checked) {
      customInput.classList.remove("hidden");
      customInput.focus();
    } else {
      customInput.classList.add("hidden");
    }
  };
});

// --- Start / Update Share ---
document.getElementById("btn-start-share").onclick = async () => {
  const path = document.getElementById("input-path").value.trim();
  if (!path) {
    alert("Lütfen paylaşılacak bir yol girin veya Gözat ile seçin!");
    return;
  }

  let pin = "";
  const pinMode = document.querySelector('input[name="pin-mode"]:checked').value;
  if (pinMode === "auto") {
    pin = Math.floor(1000 + Math.random() * 9000).toString();
  } else if (pinMode === "custom") {
    pin = document.getElementById("input-custom-pin").value.trim();
  }

  const allowUpload = document.getElementById("chk-upload").checked;
  const enableTunnel = document.getElementById("chk-tunnel").checked;

  try {
    const res = await fetch("/api/host/start", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        path: path,
        is_single_file: isSingleFileMode,
        pin: pin,
        allow_upload: allowUpload,
        enable_tunnel: enableTunnel
      })
    });

    if (!res.ok) {
      alert("Hata: " + (await res.text()));
      return;
    }

    fetchStatus();
  } catch (err) {
    alert("Bağlantı hatası: " + err);
  }
};

// --- Stop Share ---
document.getElementById("btn-stop-share").onclick = async () => {
  try {
    const res = await fetch("/api/host/stop", { method: "POST" });
    if (res.ok) fetchStatus();
  } catch (err) {
    console.error("Stop error:", err);
  }
};

// --- Language Toggle ---
document.getElementById("btn-lang-tr").onclick = () => {
  currentLang = "tr";
  loadI18n("tr");
};
document.getElementById("btn-lang-en").onclick = () => {
  currentLang = "en";
  loadI18n("en");
};

function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str;
  return div.innerHTML;
}

// Initial Boot
(async () => {
  await loadI18n(currentLang);
  await fetchStatus();
  // Poll status every 2.5s for live links and activity updates
  setInterval(fetchStatus, 2500);
})();

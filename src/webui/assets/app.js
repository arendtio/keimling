(function () {
  var STORAGE_SECRET = "keimling.access_secret";
  var STORAGE_URL = "keimling.connect_url";
  var STORAGE_SEEN = "keimling.seen";

  var gate = document.getElementById("gate");
  var app = document.getElementById("app");
  var secretEl = document.getElementById("secret");
  var connectURLEl = document.getElementById("connect-url");
  var connectBtn = document.getElementById("connect");
  var gateErr = document.getElementById("gate-err");
  var gateTitle = document.getElementById("gate-title");
  var gateCopy = document.getElementById("gate-copy");
  var urlField = document.getElementById("url-field");
  var secretField = document.getElementById("secret-field");
  var localHint = document.getElementById("local-hint");
  var choiceLocal = document.getElementById("choice-local");
  var choiceRemote = document.getElementById("choice-remote");
  var openConnection = document.getElementById("open-connection");
  var openContextBtn = document.getElementById("open-context");
  var contextDialog = document.getElementById("context-dialog");
  var openSkillsBtn = document.getElementById("open-skills");
  var skillsDialog = document.getElementById("skills-dialog");
  var skillsList = document.getElementById("skills-list");
  var skillsResults = document.getElementById("skills-results");
  var skillsPath = document.getElementById("skills-path");
  var skillsScope = document.getElementById("skills-scope");
  var skillsQuery = document.getElementById("skills-query");
  var skillsAddBtn = document.getElementById("skills-add-btn");
  var skillsSearchBtn = document.getElementById("skills-search-btn");
  var skillsErr = document.getElementById("skills-err");
  var skillsPreviewSection = document.getElementById("skills-preview-section");
  var skillsPreviewMeta = document.getElementById("skills-preview-meta");
  var skillsPreviewFiles = document.getElementById("skills-preview-files");
  var skillsPreviewContent = document.getElementById("skills-preview-content");
  var skillsReviewBtn = document.getElementById("skills-review-btn");
  var skillsInstallPreviewBtn = document.getElementById("skills-install-preview-btn");
  var skillsReviewOut = document.getElementById("skills-review-out");
  var currentSkillPreview = null;
  var openMcpBtn = document.getElementById("open-mcp");
  var mcpDialog = document.getElementById("mcp-dialog");
  var mcpList = document.getElementById("mcp-list");
  var mcpName = document.getElementById("mcp-name");
  var mcpKind = document.getElementById("mcp-kind");
  var mcpCommand = document.getElementById("mcp-command");
  var mcpUrl = document.getElementById("mcp-url");
  var mcpStdioRow = document.getElementById("mcp-stdio-row");
  var mcpUrlRow = document.getElementById("mcp-url-row");
  var mcpAddBtn = document.getElementById("mcp-add-btn");
  var mcpReloadBtn = document.getElementById("mcp-reload-btn");
  var mcpErr = document.getElementById("mcp-err");
  var openArtifactsBtn = document.getElementById("open-artifacts");
  var artifactPane = document.getElementById("artifact-pane");
  var artifactBackdrop = document.getElementById("artifact-backdrop");
  var artifactTitle = document.getElementById("artifact-title");
  var artifactFrame = document.getElementById("artifact-frame");
  var artifactHistory = document.getElementById("artifact-history");
  var artifactOpenExt = document.getElementById("artifact-open-ext");
  var artifactFullscreen = document.getElementById("artifact-fullscreen");
  var artifactPaper = document.getElementById("artifact-paper");
  var artifactClose = document.getElementById("artifact-close");
  var artifactResize = document.getElementById("artifact-resize");
  var artifactModeDock = document.getElementById("artifact-mode-dock");
  var artifactModeOverlay = document.getElementById("artifact-mode-overlay");
  var contextCompactBtn = document.getElementById("context-compact-btn");
  var contextCompactStatus = document.getElementById("context-compact-status");
  var shellEl = document.querySelector(".shell");
  var currentArtifact = null;
  var artifactViewMode = "dock";
  var STORAGE_ARTIFACT_WIDTH = "keimling.artifact_width";
  var STORAGE_ARTIFACT_MODE = "keimling.artifact_mode";
  var STORAGE_BG_LIGHT = "keimling.bg_light";
  var STORAGE_BG_DARK = "keimling.bg_dark";
  var BG_LIGHT_DEFAULT = "#eff0f1";
  var BG_DARK_DEFAULT = "#202326";
  var ARTIFACT_WIDTH_MIN = 280;
  var ARTIFACT_WIDTH_MAX_RATIO = 0.72;
  var bgLightEl = document.getElementById("bg-light");
  var bgDarkEl = document.getElementById("bg-dark");
  var bgResetBtn = document.getElementById("bg-reset");
  var newChatBtn = document.getElementById("new-chat");
  var settingsTabs = document.getElementById("settings-tabs");
  var paneConnection = document.getElementById("pane-connection");
  var paneAgent = document.getElementById("pane-agent");
  var paneSkills = document.getElementById("pane-skills");
  var paneMcp = document.getElementById("pane-mcp");
  var saveSettingsBtn = document.getElementById("save-settings");
  var gateCancelBtn = document.getElementById("gate-cancel");
  var gateManage = false;
  var workspaceToggle = document.getElementById("workspace-toggle");
  var workspaceMenu = document.getElementById("workspace-menu");
  var workspaceLabel = document.getElementById("workspace-label");
  var workspaceRootEl = document.getElementById("workspace-root");
  var workspaceList = document.getElementById("workspace-list");
  var workspaceSearch = document.getElementById("workspace-search");
  var workspacePath = document.getElementById("workspace-path");
  var workspaceOpenBtn = document.getElementById("workspace-open-btn");
  var threadListEl = document.getElementById("thread-list");
  var threadSearchEl = document.getElementById("thread-search");
  var chaptersToggle = document.getElementById("chapters-toggle");
  var chaptersFlyout = document.getElementById("chapters-flyout");
  var paperMenuToggle = document.getElementById("paper-menu-toggle");
  var paperMenu = document.getElementById("paper-menu");
  var workspacesCache = [];
  var threadsCache = [];
  var threadsListState = "idle";
  var threadsShowAll = false;
  var threadsSearchTimer = null;
  var threadsSearchActive = false;
  var currentWorkspaceID = "";
  var currentThreadID = "";
  var THREAD_RECENT_LIMIT = 5;
  var setBaseURL = document.getElementById("set-base-url");
  var setAPIKey = document.getElementById("set-api-key");
  var setOpenAIAPIKey = document.getElementById("set-openai-api-key");
  var apiKeyHint = document.getElementById("api-key-hint");
  var openaiKeyHint = document.getElementById("openai-key-hint");
  var setModel = document.getElementById("set-model");
  var setMode = document.getElementById("set-mode");
  var setReasoning = document.getElementById("set-reasoning");
  var setWeb = document.getElementById("set-web");
  var setAutopilot = document.getElementById("set-autopilot");
  var setMaxSteps = document.getElementById("set-max-steps");
  var setContextLimit = document.getElementById("set-context-limit");
  var setVoiceModel = document.getElementById("set-voice-model");

  var thread = document.getElementById("thread");
  var chaptersEl = document.getElementById("chapters");
  var workingWrap = document.getElementById("working-wrap");
  var workingEl = document.getElementById("working");
  var liveWrap = document.getElementById("live-wrap");
  var liveOut = document.getElementById("live-out");
  var stateEl = document.getElementById("state");
  var agentStatus = document.getElementById("agent-status");
  var progressDetail = document.getElementById("progress-detail");
  var modelEl = document.getElementById("model");
  var modeEl = document.getElementById("mode");
  var msg = document.getElementById("msg");
  var send = document.getElementById("send");
  var approve = document.getElementById("approve");
  var deny = document.getElementById("deny");
  var approvalEl = document.getElementById("approval");
  var approvalReason = document.getElementById("approval-reason");
  var approvalCmd = document.getElementById("approval-cmd");
  var approvalPreview = document.getElementById("approval-preview");
  var voiceBtn = document.getElementById("voice");
  var voiceStatus = document.getElementById("voice-status");

  var es = null;
  var messages = [];
  var chapters = [];
  var activeChapter = null;
  var connected = false;
  var agentState = "idle";
  var voiceOK = false;
  var mode = "local"; // local | remote
  var apiBase = ""; // empty = same origin
  var mediaRecorder = null;
  var mediaChunks = [];
  var mediaStream = null;
  var recording = false;
  var voiceSendPending = false;

  function sameOriginLocal() {
    var h = location.hostname;
    return h === "127.0.0.1" || h === "localhost" || h === "::1";
  }

  function secret() {
    return secretEl.value.trim();
  }

  function headers() {
    var h = { "Content-Type": "application/json" };
    var s = secret();
    if (s) h["Authorization"] = "Bearer " + s;
    return h;
  }

  function apiURL(path) {
    if (!apiBase) return path;
    return apiBase.replace(/\/$/, "") + path;
  }

  function storePrefs() {
    try {
      localStorage.setItem(STORAGE_SEEN, "1");
      if (secret()) localStorage.setItem(STORAGE_SECRET, secret());
      else localStorage.removeItem(STORAGE_SECRET);
      if (mode === "remote" && connectURLEl.value.trim()) {
        localStorage.setItem(STORAGE_URL, connectURLEl.value.trim());
      } else {
        localStorage.removeItem(STORAGE_URL);
      }
      storeThemePrefs();
    } catch (e) {}
  }

  function loadPrefs() {
    try {
      return {
        seen: localStorage.getItem(STORAGE_SEEN) === "1",
        secret: localStorage.getItem(STORAGE_SECRET) || "",
        url: localStorage.getItem(STORAGE_URL) || "",
        bgLight: localStorage.getItem(STORAGE_BG_LIGHT) || "",
        bgDark: localStorage.getItem(STORAGE_BG_DARK) || "",
      };
    } catch (e) {
      return { seen: false, secret: "", url: "", bgLight: "", bgDark: "" };
    }
  }

  function normalizeBgHex(v) {
    v = String(v || "").trim().toLowerCase();
    if (!v) return "";
    if (v.charAt(0) !== "#") v = "#" + v;
    if (/^#[0-9a-f]{6}$/.test(v)) return v;
    if (/^#[0-9a-f]{3}$/.test(v)) {
      return "#" + v.charAt(1) + v.charAt(1) + v.charAt(2) + v.charAt(2) + v.charAt(3) + v.charAt(3);
    }
    return "";
  }

  function applyThemeBg(light, dark) {
    var root = document.documentElement;
    var L = normalizeBgHex(light) || BG_LIGHT_DEFAULT;
    var D = normalizeBgHex(dark) || BG_DARK_DEFAULT;
    root.style.setProperty("--bg-light", L);
    root.style.setProperty("--bg-dark", D);
    if (bgLightEl) bgLightEl.value = L;
    if (bgDarkEl) bgDarkEl.value = D;
  }

  function storeThemePrefs() {
    try {
      var L = normalizeBgHex(bgLightEl && bgLightEl.value) || BG_LIGHT_DEFAULT;
      var D = normalizeBgHex(bgDarkEl && bgDarkEl.value) || BG_DARK_DEFAULT;
      if (L === BG_LIGHT_DEFAULT) localStorage.removeItem(STORAGE_BG_LIGHT);
      else localStorage.setItem(STORAGE_BG_LIGHT, L);
      if (D === BG_DARK_DEFAULT) localStorage.removeItem(STORAGE_BG_DARK);
      else localStorage.setItem(STORAGE_BG_DARK, D);
    } catch (e) {}
  }

  function persistThemeBg() {
    var L = normalizeBgHex(bgLightEl && bgLightEl.value) || BG_LIGHT_DEFAULT;
    var D = normalizeBgHex(bgDarkEl && bgDarkEl.value) || BG_DARK_DEFAULT;
    applyThemeBg(L, D);
    storeThemePrefs();
    // Desktop WebView origins are ephemeral — also persist via gui-prefs.
    var body = {
      bg_light: L === BG_LIGHT_DEFAULT ? "" : L,
      bg_dark: D === BG_DARK_DEFAULT ? "" : D,
    };
    saveGuiPrefs(body).catch(function () {});
  }

  function loadThemeFromGuiPrefs() {
    var h = { "Content-Type": "application/json" };
    var s = pageAccessSecret();
    if (s) h["Authorization"] = "Bearer " + s;
    return fetch(apiURL("/api/gui-prefs"), { headers: h })
      .then(function (r) {
        if (!r.ok) return null;
        return r.json();
      })
      .then(function (p) {
        if (!p) return;
        var L = normalizeBgHex(p.bg_light);
        var D = normalizeBgHex(p.bg_dark);
        if (!L && !D) return;
        applyThemeBg(L || BG_LIGHT_DEFAULT, D || BG_DARK_DEFAULT);
        storeThemePrefs();
      })
      .catch(function () {});
  }

  function setChoice(next) {
    mode = next;
    choiceLocal.classList.toggle("active", mode === "local");
    choiceRemote.classList.toggle("active", mode === "remote");
    choiceLocal.setAttribute("aria-pressed", mode === "local" ? "true" : "false");
    choiceRemote.setAttribute("aria-pressed", mode === "remote" ? "true" : "false");
    urlField.hidden = mode !== "remote";
    if (mode === "local" && sameOriginLocal() && !secret()) {
      secretField.hidden = true;
      localHint.hidden = false;
    } else {
      secretField.hidden = false;
      localHint.hidden = true;
    }
  }

  function setSettingsTab(tab) {
    tab = tab || "connection";
    settingsTabs.querySelectorAll(".settings-tab").forEach(function (btn) {
      var on = btn.getAttribute("data-tab") === tab;
      btn.classList.toggle("active", on);
      btn.setAttribute("aria-selected", on ? "true" : "false");
    });
    if (paneConnection) paneConnection.hidden = tab !== "connection";
    if (paneAgent) paneAgent.hidden = tab !== "agent";
    if (paneSkills) paneSkills.hidden = tab !== "skills";
    if (paneMcp) paneMcp.hidden = tab !== "mcp";
    saveSettingsBtn.hidden = tab !== "agent";
    connectBtn.hidden = tab !== "connection";
  }

  function fillAgentSettings(j) {
    if (!j) return;
    setBaseURL.value = j.base_url || "";
    setAPIKey.value = "";
    setOpenAIAPIKey.value = "";
    setAPIKey.placeholder = j.api_key_set ? "Set — leave blank to keep" : "Not set";
    setOpenAIAPIKey.placeholder = j.openai_api_key_set ? "Set — leave blank to keep" : "Not set";
    apiKeyHint.textContent = j.api_key_set
      ? "Provider key is set. Leave blank to keep it, or paste a new key."
      : "No provider key yet. Required for OpenRouter and most remote APIs.";
    openaiKeyHint.textContent = j.openai_api_key_set
      ? "OpenAI key is set. Leave blank to keep it, or paste a new key."
      : "Optional unless you use web search or voice transcription.";
    setModel.value = j.model || "";
    setMode.value = j.mode || "auto";
    setReasoning.value = j.reasoning || "high";
    setWeb.checked = !!j.web;
    setAutopilot.checked = !!j.autopilot;
    setMaxSteps.value = j.max_steps != null ? j.max_steps : 200;
    setContextLimit.value = j.context_limit != null ? j.context_limit : 128000;
    setVoiceModel.value = j.voice_model || "";
  }

  function loadAgentSettings() {
    if (!connected) return Promise.resolve();
    return fetch(apiURL("/api/settings"), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.message) || ("HTTP " + r.status));
          fillAgentSettings(j);
        });
      })
      .catch(function (e) {
        gateErr.hidden = false;
        gateErr.textContent = String(e);
      });
  }

  function saveAgentSettings() {
    gateErr.hidden = true;
    var body = {
      base_url: setBaseURL.value.trim(),
      model: setModel.value.trim(),
      mode: setMode.value,
      reasoning: setReasoning.value,
      web: setWeb.checked,
      autopilot: setAutopilot.checked,
      max_steps: parseInt(setMaxSteps.value, 10),
      context_limit: parseInt(setContextLimit.value, 10),
      voice_model: setVoiceModel.value.trim(),
    };
    var providerKey = setAPIKey.value.trim();
    if (providerKey) body.api_key = providerKey;
    var openaiKey = setOpenAIAPIKey.value.trim();
    if (openaiKey) body.openai_api_key = openaiKey;
    saveSettingsBtn.disabled = true;
    fetch(apiURL("/api/settings"), {
      method: "PUT",
      headers: headers(),
      body: JSON.stringify(body),
    })
      .then(function (r) {
        return r.text().then(function (t) {
          var j = {};
          try { j = JSON.parse(t); } catch (e) {}
          if (!r.ok) throw new Error((j && j.message) || t || ("HTTP " + r.status));
          fillAgentSettings(j);
          modelEl.textContent = j.model || "—";
          modeEl.textContent = j.mode || "—";
          gateErr.hidden = false;
          gateErr.style.color = "var(--accent)";
          gateErr.textContent = "Agent settings saved to the server config.";
        });
      })
      .catch(function (e) {
        gateErr.hidden = false;
        gateErr.style.color = "";
        gateErr.textContent = String(e);
      })
      .finally(function () {
        saveSettingsBtn.disabled = false;
      });
  }

  function showGate(opts) {
    opts = opts || {};
    var prefs = loadPrefs();
    var first = !prefs.seen && !opts.manage;
    gateTitle.textContent = opts.manage ? "Settings" : first ? "Welcome" : "Connect";
    gateCopy.textContent = opts.manage
      ? "Two layers: Connection (UI ↔ agent process) and Agent (model provider & behaviour)."
      : first
        ? "First pick how this UI reaches an agent. Model provider options live under Settings → Agent after you connect."
        : "Resume or pick how to reach the agent.";
    if (prefs.url && !connectURLEl.value.trim()) {
      connectURLEl.value = prefs.url;
    }
    // Do not reset the user's Connection choice while Settings is open.
    if (!opts.manage) {
      if (!sameOriginLocal()) {
        setChoice("remote");
        if (!connectURLEl.value.trim()) {
          try { connectURLEl.value = location.origin + "/"; } catch (e) {}
        }
      } else if (prefs.url) {
        setChoice("remote");
      } else {
        setChoice("local");
      }
    }
    if (opts.prefillSecret) secretEl.value = opts.prefillSecret;
    else if (prefs.secret && !secretEl.value) secretEl.value = prefs.secret;
    gateErr.hidden = !opts.error;
    gateErr.style.color = "";
    gateErr.textContent = opts.error || "";
    settingsTabs.hidden = !opts.manage;
    gateManage = !!opts.manage && (!!opts.keepApp || connected);
    gateCancelBtn.hidden = !gateManage;
    if (opts.manage) {
      setSettingsTab(opts.tab || "connection");
      if (connected) loadAgentSettings();
    } else {
      setSettingsTab("connection");
      saveSettingsBtn.hidden = true;
      connectBtn.hidden = false;
    }
    gate.hidden = false;
    if (!opts.keepApp) app.hidden = true;
  }

  function showApp() {
    gate.hidden = true;
    gateManage = false;
    gateCancelBtn.hidden = true;
    app.hidden = false;
  }

  function closeGate() {
    if (!connected) return;
    gateErr.hidden = true;
    gateErr.style.color = "";
    showApp();
    msg.focus();
  }

  function clearProgress() {
    if (!progressDetail) return;
    progressDetail.hidden = true;
    progressDetail.textContent = "";
    progressDetail.className = "progress-detail";
  }

  function updateProgress(p) {
    if (!progressDetail) return;
    p = p || {};
    if (p.clear) {
      clearProgress();
      return;
    }
    var phase = p.phase || "waiting";
    var label =
      phase === "thinking" ? "thinking" :
      phase === "text" ? "generating" :
      phase === "tool" ? "tool args" :
      "working";
    var parts = [label, (p.elapsed_sec || 0) + "s"];
    var rate = Number(p.tok_per_s) || 0;
    var approx = Number(p.approx_tokens) || 0;
    if (rate > 0) parts.push("~" + rate.toFixed(1) + " tok/s");
    if (approx > 0) parts.push("~" + approx + " tok");
    // While waiting with no stream yet, still show a heartbeat so a dead link is obvious.
    if (phase === "waiting" && approx === 0) {
      parts = ["waiting for model", (p.elapsed_sec || 0) + "s"];
    }
    progressDetail.hidden = false;
    progressDetail.className = "progress-detail" + (phase === "thinking" ? " thinking" : "");
    progressDetail.textContent = parts.join(" · ");
  }

  function setState(s) {
    agentState = s || "idle";
    stateEl.textContent = agentState;
    stateEl.className = "state " + agentState;
    // Keep the indicator next to new output; hide when idle to free vertical space.
    if (agentStatus) {
      var show = agentState === "running" || agentState === "awaiting_approval" || agentState === "compacting";
      agentStatus.hidden = !show;
      agentStatus.className = "agent-status" + (show ? " " + agentState : "");
    }
    if (agentState !== "running" && agentState !== "compacting") clearProgress();
    var waiting = agentState === "awaiting_approval";
    var busyish = agentState === "running" || agentState === "compacting" || waiting;
    approve.disabled = !waiting;
    deny.disabled = !waiting;
    // While recording, Send means “stop + transcribe + send” (same as Enter in CLI).
    send.disabled = !connected || busyish;
    msg.disabled = !connected || busyish;
    voiceBtn.disabled = !connected || !voiceOK || busyish;
    newChatBtn.disabled = !connected || busyish;
    if (contextCompactBtn) contextCompactBtn.disabled = !connected || busyish;
    if (!waiting) hideApproval();
    if (agentState === "running" || agentState === "compacting" || waiting) {
      liveWrap.hidden = false;
      liveWrap.open = true;
    }
    if (threadListEl && currentThreadID) {
      threadsListState = agentState;
      renderThreadListFromCache();
    }
  }

  function setAnsiPre(el, text) {
    el.innerHTML = window.ansiToHtml(text || "");
  }

  function showApproval(payload) {
    payload = payload || {};
    if (payload.clear) {
      hideApproval();
      return;
    }
    var name = payload.name || "tool";
    var cmd = payload.command || name;
    approvalReason.textContent = payload.reason
      ? payload.reason
      : "Keimling wants to run “" + name + "”.";
    setAnsiPre(approvalCmd, cmd);
    if (payload.preview) {
      approvalPreview.hidden = false;
      setAnsiPre(approvalPreview, payload.preview);
    } else {
      approvalPreview.hidden = true;
      approvalPreview.textContent = "";
    }
    approvalEl.hidden = false;
  }

  function hideApproval() {
    approvalEl.hidden = true;
    approvalReason.textContent = "";
    approvalCmd.textContent = "";
    approvalPreview.textContent = "";
    approvalPreview.hidden = true;
  }

  function setVoiceStatus(text) {
    if (!text) {
      voiceStatus.hidden = true;
      voiceStatus.textContent = "";
      return;
    }
    voiceStatus.hidden = false;
    voiceStatus.textContent = text;
  }

  function esc(s) {
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
  }

  function inChapter(m) {
    if (!activeChapter) return true;
    var ev = m.event || 0;
    if (!ev) return true;
    return ev >= activeChapter.start && ev <= activeChapter.end;
  }

  function renderChapters() {
    chaptersEl.innerHTML = "";
    if (!chapters.length) {
      var empty = document.createElement("button");
      empty.type = "button";
      empty.className = "chapter empty";
      empty.textContent = "No chapters yet";
      chaptersEl.appendChild(empty);
      return;
    }
    var all = document.createElement("button");
    all.type = "button";
    all.className = "chapter" + (!activeChapter ? " active" : "");
    all.textContent = "All activity";
    all.addEventListener("click", function () {
      activeChapter = null;
      renderChapters();
      renderThread();
      closeChaptersFlyout();
    });
    chaptersEl.appendChild(all);
    chapters.forEach(function (ch) {
      var btn = document.createElement("button");
      btn.type = "button";
      btn.className = "chapter" + (activeChapter && activeChapter.id === ch.id ? " active" : "");
      var badges = '<span class="status">' + esc(ch.status || "") + "</span>";
      if (ch.memory && ch.memory !== "raw") {
        badges += '<span class="status memory">' + esc(ch.memory) + "</span>";
      } else {
        badges += '<span class="status memory raw">raw</span>';
      }
      btn.innerHTML = esc(ch.title || ch.id) + badges;
      btn.title = ch.summary || "";
      btn.addEventListener("click", function () {
        activeChapter = ch;
        renderChapters();
        renderThread();
        closeChaptersFlyout();
      });
      chaptersEl.appendChild(btn);
    });
  }

  function closeChaptersFlyout() {
    if (!chaptersFlyout || !chaptersToggle) return;
    chaptersFlyout.hidden = true;
    chaptersToggle.setAttribute("aria-expanded", "false");
  }

  function setPaperMenuOpen(open) {
    if (!paperMenu || !paperMenuToggle) return;
    paperMenu.hidden = !open;
    paperMenuToggle.setAttribute("aria-expanded", open ? "true" : "false");
  }

  function closePaperMenu() {
    setPaperMenuOpen(false);
  }

  function setWorkspaceMenuOpen(open) {
    if (!workspaceMenu || !workspaceToggle) return;
    workspaceMenu.hidden = !open;
    workspaceToggle.setAttribute("aria-expanded", open ? "true" : "false");
  }

  function renderWorkspaceList(filter) {
    if (!workspaceList) return;
    filter = String(filter || "").trim().toLowerCase();
    workspaceList.innerHTML = "";
    workspacesCache.forEach(function (ws) {
      var hay = ((ws.name || "") + " " + (ws.root || "") + " " + (ws.id || "")).toLowerCase();
      if (filter && hay.indexOf(filter) < 0) return;
      var li = document.createElement("li");
      var btn = document.createElement("button");
      btn.type = "button";
      btn.className = ws.id === currentWorkspaceID ? "active" : "";
      btn.innerHTML = esc(ws.name || ws.id) + '<span class="ws-path">' + esc(ws.root || "") + "</span>";
      btn.addEventListener("click", function () {
        switchWorkspace(ws.id);
      });
      li.appendChild(btn);
      workspaceList.appendChild(li);
    });
  }

  function loadWorkspaces() {
    if (!connected) return Promise.resolve();
    return fetch(apiURL("/api/workspaces"), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          workspacesCache = j.workspaces || [];
          currentWorkspaceID = j.current || currentWorkspaceID;
          var cur = null;
          for (var i = 0; i < workspacesCache.length; i++) {
            if (workspacesCache[i].id === currentWorkspaceID) {
              cur = workspacesCache[i];
              break;
            }
          }
          if (workspaceLabel) workspaceLabel.textContent = (cur && cur.name) || currentWorkspaceID || "Workspace";
          if (workspaceRootEl) {
            workspaceRootEl.textContent = (cur && cur.root) || "";
            workspaceRootEl.title = workspaceRootEl.textContent;
          }
          renderWorkspaceList(workspaceSearch && workspaceSearch.value);
        });
      })
      .catch(function (err) {
        appendLive("\n[workspace] " + err + "\n");
      });
  }

  function readAPIResult(r) {
    return r.text().then(function (t) {
      var j = null;
      try { j = t ? JSON.parse(t) : null; } catch (e) {}
      if (!r.ok) {
        throw new Error((j && (j.error || j.message)) || t || r.status);
      }
      return j || {};
    });
  }

  function afterWorkspaceAttach(j) {
    j = j || {};
    currentWorkspaceID = j.id || currentWorkspaceID;
    setWorkspaceMenuOpen(false);
    // Host reattached proxy target — reload session/SSE without restarting the app.
    if (es && typeof es.close === "function") {
      try { es.close(); } catch (e) {}
      es = null;
    }
    messages = [];
    chapters = [];
    activeChapter = null;
    clearLive();
    hideApproval();
    return loadSession()
      .then(function () {
        connectEvents();
        return Promise.all([loadWorkspaces(), loadThreads()]);
      })
      .then(function () {
        if (msg) msg.focus();
      });
  }

  function switchWorkspace(id) {
    if (!id || id === currentWorkspaceID) {
      setWorkspaceMenuOpen(false);
      return;
    }
    fetch(apiURL("/api/workspaces/switch"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: id }),
    })
      .then(readAPIResult)
      .then(afterWorkspaceAttach)
      .catch(function (err) {
        appendLive("\n[workspace] " + err + "\n");
      });
  }

  function openWorkspacePath(path) {
    path = String(path || "").trim();
    if (!path) return;
    fetch(apiURL("/api/workspaces/open"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ path: path }),
    })
      .then(readAPIResult)
      .then(afterWorkspaceAttach)
      .catch(function (err) {
        appendLive("\n[workspace] " + err + "\n");
      });
  }

  function visibleThreads(all, currentID) {
    all = all || [];
    if (threadsSearchActive || threadsShowAll) {
      return all.slice();
    }
    var out = [];
    var seen = {};
    all.slice(0, THREAD_RECENT_LIMIT).forEach(function (th) {
      out.push(th);
      seen[th.id] = true;
    });
    if (currentID && !seen[currentID]) {
      for (var i = 0; i < all.length; i++) {
        if (all[i].id === currentID) {
          out.push(all[i]);
          break;
        }
      }
    }
    return out;
  }

  function renderThreadListFromCache() {
    if (!threadListEl) return;
    var state = threadsListState || agentState || "idle";
    var query = threadSearchEl ? String(threadSearchEl.value || "").trim() : "";
    var threads = visibleThreads(threadsCache, currentThreadID);
    threadListEl.innerHTML = "";
    if (!threadsCache.length) {
      var empty = document.createElement("p");
      empty.className = "hint";
      empty.textContent = query ? "No matching threads" : "No threads yet";
      threadListEl.appendChild(empty);
      return;
    }
    if (!threads.length) {
      var none = document.createElement("p");
      none.className = "hint";
      none.textContent = "No threads to show";
      threadListEl.appendChild(none);
      return;
    }
    threads.forEach(function (th) {
      var row = document.createElement("div");
      row.className = "thread-item" + (th.id === currentThreadID || th.active ? " active" : "");
      row.title = th.id;

      var open = document.createElement("button");
      open.type = "button";
      open.className = "thread-open";
      var st = document.createElement("span");
      st.className = "thread-status" + (th.id === currentThreadID || th.active ? " " + state : "");
      var title = document.createElement("span");
      title.className = "thread-title";
      title.textContent = th.title || th.id;
      open.appendChild(st);
      open.appendChild(title);
      open.addEventListener("click", function () {
        switchThread(th.id);
      });

      var del = document.createElement("button");
      del.type = "button";
      del.className = "thread-delete";
      del.textContent = "Delete";
      del.title = "Delete thread permanently";
      del.addEventListener("click", function (e) {
        e.preventDefault();
        e.stopPropagation();
        deleteThread(th.id, th.title || th.id);
      });

      row.appendChild(open);
      row.appendChild(del);
      threadListEl.appendChild(row);
    });
    if (!query && threadsCache.length > THREAD_RECENT_LIMIT) {
      var toggle = document.createElement("button");
      toggle.type = "button";
      toggle.className = "ghost wide thread-show-all";
      toggle.textContent = threadsShowAll
        ? "Show recent only"
        : "Show all (" + threadsCache.length + ")";
      toggle.addEventListener("click", function () {
        threadsShowAll = !threadsShowAll;
        renderThreadListFromCache();
      });
      threadListEl.appendChild(toggle);
      if (!threadsShowAll) {
        var hint = document.createElement("p");
        hint.className = "hint thread-list-hint";
        hint.textContent = "Showing " + THREAD_RECENT_LIMIT + " recent · search title & content";
        threadListEl.appendChild(hint);
      }
    } else if (query) {
      var sh = document.createElement("p");
      sh.className = "hint thread-list-hint";
      sh.textContent = threadsCache.length + " match" + (threadsCache.length === 1 ? "" : "es") + " in titles & content";
      threadListEl.appendChild(sh);
    }
  }

  function renderThreadList(payload) {
    payload = payload || {};
    threadsCache = payload.threads || [];
    threadsListState = payload.state || agentState || "idle";
    currentThreadID = payload.current || currentThreadID;
    threadsSearchActive = !!(payload.query && String(payload.query).trim());
    renderThreadListFromCache();
  }

  function loadThreads(query) {
    if (!connected) return Promise.resolve();
    var q = query == null
      ? (threadSearchEl ? String(threadSearchEl.value || "").trim() : "")
      : String(query || "").trim();
    var url = apiURL("/api/threads");
    if (q) url += "?q=" + encodeURIComponent(q);
    return fetch(url, { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          renderThreadList(j);
        });
      })
      .catch(function (err) {
        appendLive("\n[threads] " + err + "\n");
      });
  }

  function scheduleThreadSearch() {
    if (threadsSearchTimer) clearTimeout(threadsSearchTimer);
    threadsSearchTimer = setTimeout(function () {
      threadsSearchTimer = null;
      loadThreads();
    }, 280);
  }

  function updateThreadTitle(p) {
    p = p || {};
    if (!p.id || !p.title) return;
    for (var i = 0; i < threadsCache.length; i++) {
      if (threadsCache[i].id === p.id) {
        threadsCache[i].title = p.title;
        break;
      }
    }
    renderThreadListFromCache();
  }

  function switchThread(id) {
    if (!id || id === currentThreadID) return;
    if (agentState === "running" || agentState === "compacting") return;
    fetch(apiURL("/api/threads/switch"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: id }),
    })
      .then(function (r) {
        return r.text().then(function (t) {
          if (!r.ok) throw new Error(t || r.status);
        });
      })
      .then(function () {
        return loadSession();
      })
      .then(function () {
        return loadThreads();
      })
      .catch(function (err) {
        appendLive("\n[threads] " + err + "\n");
      });
  }

  function deleteThread(id, label) {
    if (!id) return;
    if (agentState === "running" || agentState === "compacting" || agentState === "awaiting_approval") return;
    var name = label || id;
    if (!window.confirm("Delete thread “" + name + "” permanently?\n\nThis cannot be undone. The conversation and its local data will be removed.")) {
      return;
    }
    fetch(apiURL("/api/threads/delete"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: id }),
    })
      .then(readAPIResult)
      .then(function (j) {
        currentThreadID = j.thread_id || currentThreadID;
        return loadSession();
      })
      .then(function () {
        return loadThreads();
      })
      .catch(function (err) {
        appendLive("\n[threads] " + err + "\n");
      });
  }

  function memoryLabel(m) {
    switch (m) {
      case "distill": return "distilled";
      case "summarize": return "summarized";
      case "collapse": return "collapsed";
      case "keep": return "kept";
      default: return "untouched";
    }
  }

  function toolFirstLine(cmd) {
    var line = String(cmd || "").trim().split("\n")[0] || "";
    if (line.length > 96) return line.slice(0, 93) + "…";
    return line;
  }

  function toolCard(m) {
    var name = m.name || "tool";
    var cmd = m.command || m.text || name;
    var wrap = document.createElement("details");
    wrap.className = "tool-line";
    var summary = document.createElement("summary");
    summary.innerHTML =
      '<span class="tool-line-ran">Ran</span> ' +
      '<span class="tool-line-name">' + esc(name) + "</span> " +
      '<code class="tool-line-cmd">' + esc(toolFirstLine(cmd)) + "</code>";
    wrap.appendChild(summary);

    var card = document.createElement("div");
    card.className = "tool-card";
    var html = '<div class="tool-name">' + esc(name) + '</div>';
    html += '<div class="tool-head">' + esc(cmd) + "</div>";
    if (m.result) {
      html += "<details><summary>Result (model view)</summary>";
      html += '<pre class="tool-body">' + window.ansiToHtml(m.result) + "</pre></details>";
    }
    if (m.raw && m.raw !== m.result) {
      html += "<details><summary>Raw output</summary>";
      html += '<pre class="tool-body">' + window.ansiToHtml(m.raw) + "</pre></details>";
    }
    card.innerHTML = html;
    wrap.appendChild(card);
    return wrap;
  }

  function renderAssistantHTML(text) {
    var raw = text || "";
    try {
      if (typeof marked !== "undefined" && marked.parse) {
        var html = marked.parse(raw, { async: false, gfm: true, breaks: true });
        if (typeof DOMPurify !== "undefined" && DOMPurify.sanitize) {
          return DOMPurify.sanitize(html, { USE_PROFILES: { html: true } });
        }
        return html;
      }
    } catch (e) {}
    return window.ansiToHtml(raw);
  }

  function appendMessageNode(parent, m) {
    if (m.role === "tool_group") {
      parent.appendChild(toolCard(m));
      return;
    }
    var el = document.createElement("article");
    el.className = "msg " + (m.role || "assistant");
    var who = m.role === "user" ? "You" : m.role === "system" ? "System" : "Keimling";
    var body = m.role === "assistant" ? renderAssistantHTML(m.text) : esc(m.text || "");
    el.innerHTML = '<p class="who">' + esc(who) + '</p><div class="body">' + body + "</div>";
    parent.appendChild(el);
  }

  function messagesInChapter(ch) {
    return messages.filter(function (m) {
      var ev = m.event || 0;
      if (!ev) return false;
      return ev >= ch.start && ev <= ch.end;
    });
  }

  function messagesOutsideChapters() {
    if (!chapters.length) return messages.slice();
    return messages.filter(function (m) {
      var ev = m.event || 0;
      if (!ev) return true;
      for (var i = 0; i < chapters.length; i++) {
        if (ev >= chapters[i].start && ev <= chapters[i].end) return false;
      }
      return true;
    });
  }

  function renderChapterOverview(ch) {
    var items = messagesInChapter(ch);
    var wrap = document.createElement("section");
    wrap.className = "chapter-overview";

    var head = document.createElement("header");
    head.className = "chapter-overview-head";
    head.innerHTML =
      '<button type="button" class="chapter-overview-title">' + esc(ch.title || ch.id) + "</button>" +
      '<span class="chapter-overview-badges">' +
        '<span class="status">' + esc(ch.status || "") + "</span>" +
        '<span class="status memory">' + esc(memoryLabel(ch.memory)) + "</span>" +
      "</span>";
    head.querySelector("button").addEventListener("click", function () {
      activeChapter = ch;
      renderChapters();
      renderThread();
    });
    wrap.appendChild(head);

    if (ch.intro) {
      var intro = document.createElement("p");
      intro.className = "chapter-overview-intro";
      intro.textContent = ch.intro;
      wrap.appendChild(intro);
    }
    if (ch.summary) {
      var sum = document.createElement("p");
      sum.className = "chapter-overview-summary";
      sum.textContent = ch.summary;
      wrap.appendChild(sum);
    }
    if (ch.conclusion) {
      var concl = document.createElement("p");
      concl.className = "chapter-overview-conclusion";
      concl.textContent = ch.conclusion;
      wrap.appendChild(concl);
    }

    if (!items.length) {
      var empty = document.createElement("p");
      empty.className = "chapter-overview-summary";
      empty.textContent = "No messages in this chapter range.";
      wrap.appendChild(empty);
      return wrap;
    }

    if (items.length === 1) {
      appendMessageNode(wrap, items[0]);
      return wrap;
    }

    appendMessageNode(wrap, items[0]);

    if (items.length === 2) {
      appendMessageNode(wrap, items[1]);
      return wrap;
    }

    var mid = items.slice(1, -1);
    var fold = document.createElement("details");
    fold.className = "chapter-fold";
    var label = mid.length === 1 ? "1 message in between" : mid.length + " messages in between";
    fold.innerHTML = "<summary>" + esc(label) + "</summary>";
    var inner = document.createElement("div");
    inner.className = "chapter-fold-body";
    mid.forEach(function (m) { appendMessageNode(inner, m); });
    fold.appendChild(inner);
    wrap.appendChild(fold);

    appendMessageNode(wrap, items[items.length - 1]);
    return wrap;
  }

  var SCROLL_STICKY_PX = 100;

  function nearBottom() {
    var el = document.documentElement;
    var gap = el.scrollHeight - (window.scrollY + window.innerHeight);
    return gap <= SCROLL_STICKY_PX;
  }

  function stickScrollIfNearBottom(wasNear) {
    if (!wasNear) return;
    requestAnimationFrame(function () {
      window.scrollTo(0, document.documentElement.scrollHeight);
    });
  }

  function renderThread() {
    var stick = nearBottom();
    thread.innerHTML = "";
    if (!messages.length) {
      var hint = document.createElement("div");
      hint.className = "msg system";
      hint.innerHTML = '<p class="who">Keimling</p><p class="body">Send a message to continue this session.</p>';
      thread.appendChild(hint);
      stickScrollIfNearBottom(stick);
      return;
    }

    // Single chapter: full transcript for that range.
    if (activeChapter) {
      var head = document.createElement("div");
      head.className = "msg system";
      var sum = activeChapter.summary ? "\n\n" + activeChapter.summary : "";
      head.innerHTML = '<p class="who">Chapter</p><div class="body">' + esc(activeChapter.title + sum) + "</div>";
      thread.appendChild(head);
      messages.forEach(function (m) {
        if (!inChapter(m)) return;
        appendMessageNode(thread, m);
      });
      stickScrollIfNearBottom(stick);
      return;
    }

    // All activity: scroll chapters as top-level units (first + last visible, middle folded).
    if (chapters.length) {
      var outside = messagesOutsideChapters();
      outside.forEach(function (m) { appendMessageNode(thread, m); });
      chapters.forEach(function (ch) {
        thread.appendChild(renderChapterOverview(ch));
      });
      stickScrollIfNearBottom(stick);
      return;
    }

    messages.forEach(function (m) { appendMessageNode(thread, m); });
    stickScrollIfNearBottom(stick);
  }

  function appendLive(text) {
    var stick = nearBottom();
    liveWrap.hidden = false;
    liveOut.insertAdjacentHTML("beforeend", window.ansiToHtml(text));
    stickScrollIfNearBottom(stick);
  }

  function clearLive() {
    liveOut.innerHTML = "";
  }

  function applySession(j) {
    modelEl.textContent = j.model || "—";
    modeEl.textContent = j.mode || "—";
    messages = j.messages || [];
    chapters = j.chapters || [];
    voiceOK = !!j.voice;
    voiceBtn.hidden = !voiceOK;
    if (j.workspace_id) currentWorkspaceID = j.workspace_id;
    if (j.thread_id) currentThreadID = j.thread_id;
    if (workspaceLabel && j.workspace_id) {
      // refined when loadWorkspaces returns
      if (!workspaceLabel.textContent || workspaceLabel.textContent === "Workspace") {
        workspaceLabel.textContent = j.workspace_id;
      }
    }
    if (workspaceRootEl && j.workspace_root) {
      workspaceRootEl.textContent = j.workspace_root;
      workspaceRootEl.title = j.workspace_root;
    }
    if (j.working_state) {
      workingWrap.hidden = false;
      workingEl.textContent = j.working_state;
      workingWrap.open = false;
    } else {
      workingWrap.hidden = true;
      workingEl.textContent = "";
    }
    renderChapters();
    renderThread();
    setState(j.busy ? "running" : (j.state || "idle"));
    if (!j.busy) {
      clearLive();
      liveWrap.hidden = true;
      liveWrap.open = false;
    }
  }

  function loadSession() {
    return fetch(apiURL("/api/session"), { headers: headers() }).then(function (r) {
      return r.json().then(function (j) {
        if (!r.ok) throw new Error((j && j.message) || ("HTTP " + r.status));
        applySession(j);
      });
    });
  }

  function connectEvents() {
    if (es) es.close();
    var ctrl = new AbortController();
    fetch(apiURL("/events"), { headers: headers(), signal: ctrl.signal })
      .then(function (r) {
        if (!r.ok) throw new Error("events " + r.status);
        var reader = r.body.getReader();
        var dec = new TextDecoder();
        var buf = "";
        function pump() {
          return reader.read().then(function (res) {
            if (res.done) return;
            buf += dec.decode(res.value, { stream: true });
            var parts = buf.split("\n\n");
            buf = parts.pop();
            parts.forEach(function (block) {
              var ev = "message";
              var data = "";
              block.split("\n").forEach(function (line) {
                if (line.indexOf("event:") === 0) ev = line.slice(6).trim();
                if (line.indexOf("data:") === 0) data += line.slice(5).trim();
              });
              if (!data) return;
              var payload = {};
              try { payload = JSON.parse(data); } catch (e) {}
              if (ev === "line" && payload.text) appendLive(payload.text);
              if (ev === "approval") showApproval(payload);
              if (ev === "progress") updateProgress(payload);
              if (ev === "artifact") showArtifact(payload);
              if (ev === "compaction") updateCompactionProgress(payload);
              if (ev === "thread") updateThreadTitle(payload);
              if (ev === "status" && payload.state) {
                setState(payload.state);
                if (payload.state === "idle") {
                  loadSession().catch(function () {});
                  if (contextDialog && contextDialog.open) openContextDialog();
                }
              }
            });
            return pump();
          });
        }
        return pump();
      })
      .catch(function (e) {
        if (ctrl.signal.aborted) return;
        appendLive("\n[events] " + e + "\n");
      });
    es = { close: function () { ctrl.abort(); } };
  }

  // Remote attach navigates the WebView to the agent origin (same as --connect).
  // Cross-origin fetch from a local UI has no CORS and fails as TypeError: Load failed.
  function remoteUIURL(base, sec) {
    var u = new URL(base);
    if (u.protocol !== "http:" && u.protocol !== "https:") {
      throw new Error("Remote URL must be http or https");
    }
    if (!u.pathname) u.pathname = "/";
    u.searchParams.set("access_secret", sec);
    return u.toString();
  }

  function formatConnectError(e) {
    var s = String(e && e.message ? e.message : e);
    if (/Load failed|Failed to fetch|NetworkError|NETWORK_ERR/i.test(s)) {
      return "Could not reach the agent (network, TLS, or blocked request). Check the URL and that the server is reachable from this machine.";
    }
    return s;
  }

  // Auth against the *page* agent (URL token). The form secret may be the remote
  // server's secret — using that against the ephemeral local agent caused 401 and
  // silently skipped persisting gui-prefs.
  function pageAccessSecret() {
    try {
      var q = new URLSearchParams(window.location.search).get("access_secret");
      if (q) return q;
    } catch (e) {}
    return secret();
  }

  // Persist last desktop connection outside ephemeral-port localStorage.
  function saveGuiPrefs(body) {
    var h = { "Content-Type": "application/json" };
    var s = pageAccessSecret();
    if (s) h["Authorization"] = "Bearer " + s;
    return fetch(apiURL("/api/gui-prefs"), {
      method: "PUT",
      headers: h,
      body: JSON.stringify(body || {}),
    }).then(function (r) {
      if (!r.ok) return r.text().then(function (t) { throw new Error(t || ("HTTP " + r.status)); });
    });
  }

  function rememberRemote(connect, sec) {
    if (typeof keimlingRememberRemote === "function") {
      try { keimlingRememberRemote(connect, sec); } catch (e) {}
    }
    return saveGuiPrefs({ connect: connect, access_secret: sec }).catch(function () {});
  }

  function switchAwayFromRemotePage() {
    connectBtn.disabled = true;
    try {
      localStorage.removeItem(STORAGE_URL);
    } catch (e) {}
    if (typeof keimlingUseLocalAgent !== "function") {
      showGate({
        error: "This page is the remote agent. Close Keimling and start it again without a remote URL (or clear connect in ~/.config/keimling/gui-prefs.json).",
        manage: true,
        keepApp: !app.hidden,
      });
      connectBtn.disabled = false;
      return;
    }
    Promise.resolve(keimlingUseLocalAgent())
      .then(function (home) {
        if (home && home !== "relaunch") {
          location.assign(home);
          return;
        }
        // Relaunch path: native side starts a local process and closes this window.
        gateErr.hidden = false;
        gateErr.textContent = "Switching to local agent…";
      })
      .catch(function (e) {
        showGate({ error: formatConnectError(e), manage: true, keepApp: !app.hidden });
        connectBtn.disabled = false;
      });
  }

  function openSession(opts) {
    opts = opts || {};
    gateErr.hidden = true;
    if (mode === "remote") {
      var remoteBase = connectURLEl.value.trim();
      if (!remoteBase) {
        showGate({ error: "Remote URL required.", manage: true, keepApp: !app.hidden });
        return;
      }
      if (!secret()) {
        showGate({ error: "Access secret required for remote agents.", manage: true, keepApp: !app.hidden });
        return;
      }
      var dest;
      try {
        dest = remoteUIURL(remoteBase, secret());
      } catch (e) {
        showGate({ error: formatConnectError(e), manage: true, keepApp: !app.hidden });
        return;
      }
      try {
        var destOrigin = new URL(dest).origin;
        if (destOrigin === location.origin) {
          // Already on the agent host — drop remote mode and continue same-origin.
          apiBase = "";
          history.replaceState(null, "", dest);
        } else {
          storePrefs();
          connectBtn.disabled = true;
          rememberRemote(remoteBase, secret())
            .then(function () {
              location.assign(dest);
            })
            .catch(function (e) {
              showGate({ error: "Could not remember remote connection: " + formatConnectError(e), manage: true, keepApp: !app.hidden });
              connectBtn.disabled = false;
            });
          return;
        }
      } catch (e) {
        showGate({ error: formatConnectError(e), manage: true, keepApp: !app.hidden });
        return;
      }
    } else {
      // "This window" while still on a remote origin must leave that host.
      if (!sameOriginLocal()) {
        switchAwayFromRemotePage();
        return;
      }
      apiBase = "";
      if (!secret()) {
        // Embedded local UI should pass ?access_secret=; without it we cannot call the API.
        showGate({
          error: "No local access secret. Restart with the desktop UI, or paste the secret.",
          manage: true,
          keepApp: !app.hidden,
        });
        return;
      }
    }
    connectBtn.disabled = true;
    // Clear remembered remote only when the user explicitly chooses local on the
    // desktop agent — not on auto-boot into a remote page (mode "local" there).
    var clearRemote = !opts.auto && mode !== "remote" && sameOriginLocal();
    (clearRemote ? saveGuiPrefs({ connect: "", access_secret: "" }).catch(function () {}) : Promise.resolve())
      .then(function () { return loadSession(); })
      .then(function () {
        connected = true;
        storePrefs();
        showApp();
        // applySession ran setState while connected was still false — refresh controls.
        setState(agentState);
        connectEvents();
        loadThemeFromGuiPrefs();
        loadWorkspaces();
        loadThreads();
        msg.focus();
      })
      .catch(function (e) {
        connected = false;
        setState(agentState);
        showGate({ error: formatConnectError(e), manage: true, keepApp: false });
      })
      .finally(function () {
        connectBtn.disabled = false;
      });
  }

  choiceLocal.addEventListener("click", function () { setChoice("local"); });
  choiceRemote.addEventListener("click", function () { setChoice("remote"); });
  connectBtn.addEventListener("click", openSession);
  secretEl.addEventListener("keydown", function (e) {
    if (e.key === "Enter") openSession();
  });
  connectURLEl.addEventListener("keydown", function (e) {
    if (e.key === "Enter") openSession();
  });
  openConnection.addEventListener("click", function () {
    showGate({ manage: true, keepApp: true, tab: "connection" });
    app.hidden = false;
  });

  function listBlock(items) {
    if (!items || !items.length) return "—";
    return items.map(function (x) { return "• " + x; }).join("\n");
  }
  function renderContextDialog(data) {
    var c = (data && data.charter) || {};
    var b = (data && data.budget) || {};
    document.getElementById("ctx-goal").textContent = c.goal || "—";
    var planLines = [];
    (c.plan || []).forEach(function (step, i) {
      var mark = (c.position === i + 1) ? "→ " : "- ";
      planLines.push(mark + "[" + (step.status || "?") + "] " + (step.step || ""));
    });
    document.getElementById("ctx-plan").textContent = planLines.length ? planLines.join("\n") : "—";
    document.getElementById("ctx-findings").textContent = listBlock(c.findings);
    document.getElementById("ctx-questions").textContent = listBlock(c.open_questions);
    document.getElementById("ctx-constraints").textContent = listBlock(c.constraints);
    document.getElementById("ctx-recent").textContent = listBlock(c.recent);
    var pct = typeof b.used_pct === "number" ? b.used_pct : 0;
    var label = "used " + (b.used || 0) + " / " + (b.limit || 0) +
      " (" + pct.toFixed(1) + "%) · remaining " + (b.remaining || 0) +
      " · " + (b.source || "estimate");
    document.getElementById("budget-label").textContent = label;
    var fill = document.getElementById("budget-fill");
    fill.style.width = Math.min(100, Math.max(0, pct)) + "%";
    fill.classList.toggle("warn", !!b.warn);
  }
  function updateCompactionProgress(p) {
    p = p || {};
    var text = "";
    if (p.error) text = "Failed: " + p.error;
    else if (p.message) text = (p.message || "") + (p.elapsed_sec != null ? " · " + p.elapsed_sec + "s" : "");
    if (contextCompactStatus) contextCompactStatus.textContent = text;
    if (agentStatus && progressDetail && (agentState === "compacting" || !p.done)) {
      agentStatus.hidden = false;
      progressDetail.hidden = false;
      progressDetail.textContent = text ? ("compact · " + text) : "compacting…";
    }
    if (p.done && contextCompactBtn && agentState !== "running" && agentState !== "compacting") {
      contextCompactBtn.disabled = !connected;
    }
  }
  function openContextDialog() {
    if (!connected || !contextDialog) return;
    fetch(apiURL("/api/context"), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          return j;
        });
      })
      .then(function (j) {
        renderContextDialog(j);
        if (typeof contextDialog.showModal === "function") {
          if (!contextDialog.open) contextDialog.showModal();
        } else contextDialog.setAttribute("open", "");
      })
      .catch(function (err) {
        appendLive("\n[context] " + err + "\n");
      });
  }
  if (openContextBtn) {
    openContextBtn.addEventListener("click", function () {
      closePaperMenu();
      openContextDialog();
    });
  }
  if (contextCompactBtn) {
    contextCompactBtn.addEventListener("click", function (e) {
      e.preventDefault();
      if (!connected || agentState === "running" || agentState === "compacting") return;
      contextCompactBtn.disabled = true;
      if (contextCompactStatus) contextCompactStatus.textContent = "Starting…";
      fetch(apiURL("/api/compact"), {
        method: "POST",
        headers: headers(),
        body: JSON.stringify({ guidance: "" }),
      })
        .then(function (r) {
          return r.text().then(function (t) {
            if (!r.ok) throw new Error(t || r.status);
          });
        })
        .catch(function (err) {
          if (contextCompactStatus) contextCompactStatus.textContent = String(err.message || err);
          contextCompactBtn.disabled = false;
        });
    });
  }

  function showSkillsErr(msg) {
    if (!skillsErr) return;
    if (!msg) {
      skillsErr.hidden = true;
      skillsErr.textContent = "";
      return;
    }
    skillsErr.hidden = false;
    skillsErr.textContent = String(msg);
  }
  function renderSkillsInstalled(list) {
    if (!skillsList) return;
    skillsList.innerHTML = "";
    if (!list || !list.length) {
      skillsList.innerHTML = "<li class=\"skill-meta\">No skills installed.</li>";
      return;
    }
    list.forEach(function (sk) {
      var li = document.createElement("li");
      var title = document.createElement("div");
      title.className = "skill-name";
      title.textContent = sk.name || "";
      var meta = document.createElement("div");
      meta.className = "skill-meta";
      meta.textContent = (sk.source || "") + (sk.writable ? "" : " · read-only") +
        (sk.origin_url ? " · " + sk.origin_url : "");
      var desc = document.createElement("div");
      desc.className = "skill-desc";
      desc.textContent = sk.description || "";
      var status = document.createElement("div");
      status.className = "skill-meta";
      var actions = document.createElement("div");
      actions.className = "skills-row";
      function setStatus(text) { status.textContent = text || ""; }
      if (sk.writable) {
        if (sk.origin_url) {
          var checkBtn = document.createElement("button");
          checkBtn.type = "button";
          checkBtn.className = "ghost";
          checkBtn.textContent = "Check update";
          checkBtn.addEventListener("click", function () {
            showSkillsErr("");
            checkBtn.disabled = true;
            setStatus("Checking…");
            fetch(apiURL("/api/skills/check-update"), {
              method: "POST",
              headers: headers(),
              body: JSON.stringify({ name: sk.name }),
            })
              .then(function (r) {
                return r.json().then(function (j) {
                  if (!r.ok) throw new Error((j && (j.error || j.message)) || r.status);
                  return j;
                });
              })
              .then(function (j) {
                var c = j.check || {};
                setStatus((c.status || "?") + (c.message ? " — " + c.message : ""));
              })
              .catch(function (e) { showSkillsErr(e.message || e); setStatus(""); })
              .finally(function () { checkBtn.disabled = false; });
          });
          var updBtn = document.createElement("button");
          updBtn.type = "button";
          updBtn.textContent = "Update";
          updBtn.addEventListener("click", function () {
            showSkillsErr("");
            updBtn.disabled = true;
            setStatus("Updating…");
            fetch(apiURL("/api/skills/update"), {
              method: "POST",
              headers: headers(),
              body: JSON.stringify({ name: sk.name }),
            })
              .then(function (r) {
                return r.json().then(function (j) {
                  if (!r.ok) throw new Error((j && (j.error || j.message)) || r.status);
                  return j;
                });
              })
              .then(function (j) {
                var c = j.check || {};
                setStatus((c.status || "updated") + (c.message ? " — " + c.message : ""));
                return loadSkillsList();
              })
              .catch(function (e) { showSkillsErr(e.message || e); setStatus(""); })
              .finally(function () { updBtn.disabled = false; });
          });
          actions.appendChild(checkBtn);
          actions.appendChild(updBtn);
        }
        var rmBtn = document.createElement("button");
        rmBtn.type = "button";
        rmBtn.className = "ghost danger";
        rmBtn.textContent = "Remove";
        rmBtn.addEventListener("click", function () {
          if (!window.confirm("Remove skill “" + sk.name + "”?")) return;
          showSkillsErr("");
          rmBtn.disabled = true;
          fetch(apiURL("/api/skills/remove"), {
            method: "POST",
            headers: headers(),
            body: JSON.stringify({ name: sk.name }),
          })
            .then(function (r) {
              return r.text().then(function (t) {
                if (!r.ok) throw new Error(t || r.status);
              });
            })
            .then(function () { return loadSkillsList(); })
            .catch(function (e) { showSkillsErr(e.message || e); })
            .finally(function () { rmBtn.disabled = false; });
        });
        actions.appendChild(rmBtn);
      }
      li.appendChild(title);
      li.appendChild(meta);
      li.appendChild(desc);
      if (actions.childNodes.length) {
        li.appendChild(actions);
        li.appendChild(status);
      }
      skillsList.appendChild(li);
    });
  }
  function hideSkillsPreview() {
    currentSkillPreview = null;
    if (skillsPreviewSection) skillsPreviewSection.hidden = true;
    if (skillsReviewOut) {
      skillsReviewOut.hidden = true;
      skillsReviewOut.textContent = "";
    }
    if (skillsPreviewContent) skillsPreviewContent.textContent = "";
    if (skillsPreviewFiles) skillsPreviewFiles.innerHTML = "";
  }
  function loadPreviewFile(path) {
    if (!currentSkillPreview) return;
    fetch(apiURL("/api/skills/preview/file?id=" + encodeURIComponent(currentSkillPreview.id) + "&path=" + encodeURIComponent(path)), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          return j;
        });
      })
      .then(function (j) {
        if (skillsPreviewContent) skillsPreviewContent.textContent = j.content || "";
      })
      .catch(function (e) { showSkillsErr(e.message || e); });
  }
  function renderSkillsPreview(prev) {
    currentSkillPreview = prev;
    if (!skillsPreviewSection || !prev) return;
    skillsPreviewSection.hidden = false;
    var sk = prev.skill || {};
    if (skillsPreviewMeta) {
      skillsPreviewMeta.textContent =
        (sk.name || "?") + " — " + (sk.description || "") +
        " · " + ((prev.files || []).length) + " files · " + (prev.source_url || "");
    }
    if (skillsReviewOut) {
      skillsReviewOut.hidden = true;
      skillsReviewOut.textContent = "";
    }
    if (skillsPreviewFiles) {
      skillsPreviewFiles.innerHTML = "";
      (prev.files || []).forEach(function (f, idx) {
        var li = document.createElement("li");
        var btn = document.createElement("button");
        btn.type = "button";
        btn.className = "ghost wide";
        btn.textContent = f.path + (f.size != null ? " (" + f.size + ")" : "");
        btn.addEventListener("click", function () { loadPreviewFile(f.path); });
        li.appendChild(btn);
        skillsPreviewFiles.appendChild(li);
        if (idx === 0 || f.path === "SKILL.md") loadPreviewFile(f.path);
      });
    }
  }
  function previewSkillCandidate(c) {
    showSkillsErr("");
    return fetch(apiURL("/api/skills/preview"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ candidate_id: c.id, url: c.source_url }),
    })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && (j.error || j.message)) || r.statusText || r.status);
          return j;
        });
      })
      .then(function (j) {
        renderSkillsPreview(j.preview);
      });
  }
  function renderSkillsResults(cands) {
    if (!skillsResults) return;
    skillsResults.innerHTML = "";
    if (!cands || !cands.length) {
      skillsResults.innerHTML = "<li class=\"skill-meta\">No candidates.</li>";
      return;
    }
    cands.forEach(function (c) {
      var li = document.createElement("li");
      var title = document.createElement("div");
      title.className = "skill-name";
      title.textContent = c.title || c.source_url || "skill";
      var meta = document.createElement("div");
      meta.className = "skill-meta";
      meta.textContent = c.source_url || "";
      var desc = document.createElement("div");
      desc.className = "skill-desc";
      desc.textContent = c.description || "";
      var btn = document.createElement("button");
      btn.type = "button";
      btn.textContent = "Preview";
      btn.addEventListener("click", function () {
        showSkillsErr("");
        btn.disabled = true;
        previewSkillCandidate(c)
          .catch(function (e) { showSkillsErr(e.message || e); })
          .finally(function () { btn.disabled = false; });
      });
      li.appendChild(title);
      li.appendChild(meta);
      li.appendChild(desc);
      li.appendChild(btn);
      skillsResults.appendChild(li);
    });
  }
  function loadSkillsList() {
    return fetch(apiURL("/api/skills"), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          return j;
        });
      })
      .then(function (j) {
        renderSkillsInstalled(j.skills || []);
      });
  }
  function openSkillsDialog() {
    if (!connected || !skillsDialog) return;
    showSkillsErr("");
    hideSkillsPreview();
    loadSkillsList()
      .then(function () {
        if (typeof skillsDialog.showModal === "function") skillsDialog.showModal();
        else skillsDialog.setAttribute("open", "");
        if (skillsQuery) skillsQuery.focus();
      })
      .catch(function (err) {
        appendLive("\n[skills] " + err + "\n");
      });
  }
  if (openSkillsBtn) {
    openSkillsBtn.addEventListener("click", openSkillsDialog);
  }
  if (skillsAddBtn) {
    skillsAddBtn.addEventListener("click", function () {
      showSkillsErr("");
      var path = (skillsPath && skillsPath.value || "").trim();
      if (!path) {
        showSkillsErr("Path required.");
        return;
      }
      skillsAddBtn.disabled = true;
      fetch(apiURL("/api/skills/add"), {
        method: "POST",
        headers: headers(),
        body: JSON.stringify({ path: path, scope: (skillsScope && skillsScope.value) || "user" }),
      })
        .then(function (r) {
          return r.text().then(function (t) {
            if (!r.ok) throw new Error(t || r.status);
          });
        })
        .then(function () {
          if (skillsPath) skillsPath.value = "";
          return loadSkillsList();
        })
        .catch(function (e) { showSkillsErr(e.message || e); })
        .finally(function () { skillsAddBtn.disabled = false; });
    });
  }
  if (skillsSearchBtn) {
    skillsSearchBtn.addEventListener("click", function () {
      showSkillsErr("");
      hideSkillsPreview();
      var q = (skillsQuery && skillsQuery.value || "").trim();
      if (!q) {
        showSkillsErr("Query required.");
        return;
      }
      skillsSearchBtn.disabled = true;
      fetch(apiURL("/api/skills/search"), {
        method: "POST",
        headers: headers(),
        body: JSON.stringify({ query: q }),
      })
        .then(function (r) {
          return r.json().then(function (j) {
            if (!r.ok) throw new Error((j && j.error) || j.message || r.status);
            return j;
          });
        })
        .then(function (j) {
          renderSkillsResults(j.candidates || []);
        })
        .catch(function (e) { showSkillsErr(e.message || e); })
        .finally(function () { skillsSearchBtn.disabled = false; });
    });
  }
  if (skillsReviewBtn) {
    skillsReviewBtn.addEventListener("click", function () {
      if (!currentSkillPreview) {
        showSkillsErr("Preview a skill first.");
        return;
      }
      showSkillsErr("");
      skillsReviewBtn.disabled = true;
      fetch(apiURL("/api/skills/review"), {
        method: "POST",
        headers: headers(),
        body: JSON.stringify({ preview_id: currentSkillPreview.id }),
      })
        .then(function (r) {
          return r.json().then(function (j) {
            if (!r.ok) throw new Error((j && (j.error || j.message)) || r.status);
            return j;
          });
        })
        .then(function (j) {
          if (j.preview) currentSkillPreview = j.preview;
          var rev = j.review || {};
          var lines = [];
          lines.push("Risk: " + (rev.risk_level || "?") + (rev.ok_to_install ? " · advisory OK" : " · caution"));
          if (rev.summary) lines.push(rev.summary);
          if (rev.dependencies && rev.dependencies.length) {
            lines.push("Dependencies:");
            rev.dependencies.forEach(function (d) { lines.push("  - " + d); });
          }
          if (rev.concerns && rev.concerns.length) {
            lines.push("Concerns:");
            rev.concerns.forEach(function (d) { lines.push("  - " + d); });
          }
          if (skillsReviewOut) {
            skillsReviewOut.hidden = false;
            skillsReviewOut.textContent = lines.join("\n");
          }
        })
        .catch(function (e) { showSkillsErr(e.message || e); })
        .finally(function () { skillsReviewBtn.disabled = false; });
    });
  }
  if (skillsInstallPreviewBtn) {
    skillsInstallPreviewBtn.addEventListener("click", function () {
      if (!currentSkillPreview) {
        showSkillsErr("Preview a skill first.");
        return;
      }
      showSkillsErr("");
      skillsInstallPreviewBtn.disabled = true;
      fetch(apiURL("/api/skills/install"), {
        method: "POST",
        headers: headers(),
        body: JSON.stringify({
          preview_id: currentSkillPreview.id,
          scope: (skillsScope && skillsScope.value) || "user",
        }),
      })
        .then(function (r) {
          return r.text().then(function (t) {
            if (!r.ok) throw new Error(t || r.status);
          });
        })
        .then(function () {
          hideSkillsPreview();
          return loadSkillsList();
        })
        .catch(function (e) { showSkillsErr(e.message || e); })
        .finally(function () { skillsInstallPreviewBtn.disabled = false; });
    });
  }

  function showMcpErr(msg) {
    if (!mcpErr) return;
    if (!msg) {
      mcpErr.hidden = true;
      mcpErr.textContent = "";
      return;
    }
    mcpErr.hidden = false;
    mcpErr.textContent = msg;
  }
  function syncMcpKindUI() {
    var kind = (mcpKind && mcpKind.value) || "stdio";
    var remote = kind === "http" || kind === "sse";
    if (mcpStdioRow) mcpStdioRow.hidden = remote;
    if (mcpUrlRow) mcpUrlRow.hidden = !remote;
  }
  function renderMcpServers(servers) {
    if (!mcpList) return;
    mcpList.innerHTML = "";
    if (!servers || !servers.length) {
      var empty = document.createElement("li");
      empty.className = "skill-meta";
      empty.textContent = "No MCP servers configured.";
      mcpList.appendChild(empty);
      return;
    }
    servers.forEach(function (s) {
      var li = document.createElement("li");
      var title = document.createElement("div");
      title.className = "skill-name";
      title.textContent = s.name || "";
      var meta = document.createElement("div");
      meta.className = "skill-meta";
      meta.textContent = (s.transport || "") + " · " + (s.status || "") +
        (s.error ? " — " + s.error : "") +
        (s.tools && s.tools.length ? " · " + s.tools.length + " tools" : "");
      var tools = document.createElement("div");
      tools.className = "skill-desc";
      tools.textContent = (s.tools || []).join(", ");
      var rm = document.createElement("button");
      rm.type = "button";
      rm.textContent = "Remove";
      rm.addEventListener("click", function () {
        showMcpErr("");
        rm.disabled = true;
        fetch(apiURL("/api/mcp/remove"), {
          method: "POST",
          headers: headers(),
          body: JSON.stringify({ name: s.name }),
        })
          .then(function (r) {
            return r.text().then(function (t) {
              if (!r.ok) throw new Error(t || r.status);
            });
          })
          .then(function () { return loadMcpList(); })
          .catch(function (e) { showMcpErr(e.message || e); })
          .finally(function () { rm.disabled = false; });
      });
      li.appendChild(title);
      li.appendChild(meta);
      if (s.tools && s.tools.length) li.appendChild(tools);
      li.appendChild(rm);
      mcpList.appendChild(li);
    });
  }
  function loadMcpList() {
    return fetch(apiURL("/api/mcp"), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          return j;
        });
      })
      .then(function (j) {
        renderMcpServers(j.servers || []);
      });
  }
  function openMcpDialog() {
    if (!connected || !mcpDialog) return;
    showMcpErr("");
    syncMcpKindUI();
    loadMcpList()
      .then(function () {
        if (typeof mcpDialog.showModal === "function") mcpDialog.showModal();
        else mcpDialog.setAttribute("open", "");
      })
      .catch(function (err) {
        appendLive("\n[mcp] " + err + "\n");
      });
  }
  if (openMcpBtn) {
    openMcpBtn.addEventListener("click", openMcpDialog);
  }

  function artifactSrc(art) {
    if (!art) return "";
    var u = art.url || "";
    if (/^https?:\/\//i.test(u)) return u;
    if (art.token) return apiURL("/files/" + art.token);
    if (u.charAt(0) === "/") return apiURL(u);
    return u;
  }
  function clampArtifactWidth(px) {
    var max = Math.floor(window.innerWidth * ARTIFACT_WIDTH_MAX_RATIO);
    if (max < ARTIFACT_WIDTH_MIN) max = ARTIFACT_WIDTH_MIN;
    if (px < ARTIFACT_WIDTH_MIN) return ARTIFACT_WIDTH_MIN;
    if (px > max) return max;
    return Math.round(px);
  }
  function applyArtifactWidth(px) {
    if (!shellEl) return;
    var w = clampArtifactWidth(px);
    shellEl.style.setProperty("--artifact-width", w + "px");
    return w;
  }
  function loadArtifactPrefs() {
    try {
      var savedW = parseInt(localStorage.getItem(STORAGE_ARTIFACT_WIDTH) || "", 10);
      if (savedW > 0) applyArtifactWidth(savedW);
      var savedMode = localStorage.getItem(STORAGE_ARTIFACT_MODE);
      if (savedMode === "overlay" || savedMode === "dock") artifactViewMode = savedMode;
    } catch (e) {}
  }
  function saveArtifactWidth(px) {
    try { localStorage.setItem(STORAGE_ARTIFACT_WIDTH, String(px)); } catch (e) {}
  }
  function saveArtifactMode(mode) {
    try { localStorage.setItem(STORAGE_ARTIFACT_MODE, mode); } catch (e) {}
  }
  function syncArtifactModeUI() {
    if (!artifactPane) return;
    var overlay = artifactViewMode === "overlay";
    artifactPane.classList.toggle("mode-overlay", overlay);
    artifactPane.classList.toggle("mode-dock", !overlay);
    if (shellEl) {
      shellEl.classList.toggle("artifact-overlay", overlay && !artifactPane.hidden);
      shellEl.classList.toggle("artifact-open", !artifactPane.hidden && !overlay);
    }
    if (artifactBackdrop) artifactBackdrop.hidden = !overlay || artifactPane.hidden;
    if (artifactModeDock) artifactModeDock.hidden = !overlay;
    if (artifactModeOverlay) artifactModeOverlay.hidden = overlay;
  }
  function setArtifactMode(mode) {
    artifactViewMode = mode === "overlay" ? "overlay" : "dock";
    saveArtifactMode(artifactViewMode);
    syncArtifactModeUI();
  }
  function openArtifactPane() {
    if (!artifactPane) return;
    artifactPane.hidden = false;
    syncArtifactModeUI();
  }
  function renderArtifactHistory(items) {
    if (!artifactHistory) return;
    artifactHistory.innerHTML = "";
    (items || []).forEach(function (art) {
      var li = document.createElement("li");
      var title = document.createElement("div");
      title.className = "skill-name";
      title.textContent = art.title || art.path || "Artifact";
      var meta = document.createElement("div");
      meta.className = "skill-meta";
      meta.textContent = art.path || "";
      var btn = document.createElement("button");
      btn.type = "button";
      btn.textContent = "Show";
      btn.addEventListener("click", function () { showArtifact(art); });
      li.appendChild(title);
      li.appendChild(meta);
      li.appendChild(btn);
      artifactHistory.appendChild(li);
    });
  }
  function showArtifact(art) {
    if (!art || !artifactPane || !artifactFrame) return;
    currentArtifact = art;
    if (artifactTitle) artifactTitle.textContent = art.title || art.path || "Artifact";
    var src = artifactSrc(art);
    if (src) artifactFrame.src = src;
    openArtifactPane();
  }
  function closeArtifactPane() {
    if (artifactFrame) artifactFrame.src = "about:blank";
    if (artifactPane) artifactPane.hidden = true;
    if (artifactBackdrop) artifactBackdrop.hidden = true;
    if (shellEl) {
      shellEl.classList.remove("artifact-open");
      shellEl.classList.remove("artifact-overlay");
      shellEl.classList.remove("artifact-resizing");
    }
  }
  function loadArtifactsList() {
    return fetch(apiURL("/api/artifacts"), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          return j;
        });
      })
      .then(function (j) {
        renderArtifactHistory(j.artifacts || []);
        return j.artifacts || [];
      });
  }
  loadArtifactPrefs();
  if (artifactClose) {
    artifactClose.addEventListener("click", closeArtifactPane);
  }
  if (artifactBackdrop) {
    artifactBackdrop.addEventListener("click", closeArtifactPane);
  }
  if (artifactModeDock) {
    artifactModeDock.addEventListener("click", function () { setArtifactMode("dock"); });
  }
  if (artifactModeOverlay) {
    artifactModeOverlay.addEventListener("click", function () { setArtifactMode("overlay"); });
  }
  if (artifactOpenExt) {
    artifactOpenExt.addEventListener("click", function () {
      var src = artifactSrc(currentArtifact);
      if (!src) return;
      if (typeof keimlingOpenExternal === "function") {
        try { keimlingOpenExternal(src); return; } catch (e) {}
      }
      window.open(src, "_blank", "noopener,noreferrer");
    });
  }
  if (artifactFullscreen) {
    artifactFullscreen.addEventListener("click", function () {
      var el = artifactPaper || artifactFrame;
      if (!el) return;
      var req = el.requestFullscreen || el.webkitRequestFullscreen || el.msRequestFullscreen;
      if (!req) {
        appendLive("\n[artifact] Fullscreen API unavailable\n");
        return;
      }
      Promise.resolve(req.call(el)).catch(function (err) {
        appendLive("\n[artifact] Fullscreen: " + (err && err.message ? err.message : err) + "\n");
      });
    });
  }
  if (artifactResize && shellEl) {
    artifactResize.addEventListener("pointerdown", function (e) {
      if (artifactViewMode === "overlay" || !artifactPane || artifactPane.hidden) return;
      e.preventDefault();
      var startX = e.clientX;
      var startW = artifactPane.getBoundingClientRect().width;
      shellEl.classList.add("artifact-resizing");
      function onMove(ev) {
        var next = applyArtifactWidth(startW + (startX - ev.clientX));
        saveArtifactWidth(next);
      }
      function onUp() {
        shellEl.classList.remove("artifact-resizing");
        window.removeEventListener("pointermove", onMove);
        window.removeEventListener("pointerup", onUp);
      }
      window.addEventListener("pointermove", onMove);
      window.addEventListener("pointerup", onUp);
    });
    artifactResize.addEventListener("keydown", function (e) {
      if (artifactViewMode === "overlay") return;
      var cur = artifactPane ? artifactPane.getBoundingClientRect().width : 512;
      var step = e.shiftKey ? 48 : 16;
      if (e.key === "ArrowLeft") {
        e.preventDefault();
        saveArtifactWidth(applyArtifactWidth(cur + step));
      } else if (e.key === "ArrowRight") {
        e.preventDefault();
        saveArtifactWidth(applyArtifactWidth(cur - step));
      }
    });
  }
  document.addEventListener("keydown", function (e) {
    if (e.key !== "Escape") return;
    if (!artifactPane || artifactPane.hidden) return;
    if (artifactViewMode === "overlay") {
      e.preventDefault();
      closeArtifactPane();
    }
  });
  if (openArtifactsBtn) {
    openArtifactsBtn.addEventListener("click", function () {
      closePaperMenu();
      if (!connected) return;
      loadArtifactsList()
        .then(function (items) {
          if (items.length) showArtifact(items[0]);
          else {
            if (artifactTitle) artifactTitle.textContent = "Artifacts";
            if (artifactFrame) artifactFrame.src = "about:blank";
            openArtifactPane();
          }
        })
        .catch(function (err) {
          appendLive("\n[artifacts] " + err + "\n");
        });
    });
  }
  if (mcpKind) {
    mcpKind.addEventListener("change", syncMcpKindUI);
  }
  if (mcpReloadBtn) {
    mcpReloadBtn.addEventListener("click", function () {
      showMcpErr("");
      mcpReloadBtn.disabled = true;
      fetch(apiURL("/api/mcp/reload"), { method: "POST", headers: headers() })
        .then(function (r) {
          return r.json().then(function (j) {
            if (!r.ok) throw new Error((j && j.error) || r.status);
            return j;
          });
        })
        .then(function (j) { renderMcpServers(j.servers || []); })
        .catch(function (e) { showMcpErr(e.message || e); })
        .finally(function () { mcpReloadBtn.disabled = false; });
    });
  }
  if (mcpAddBtn) {
    mcpAddBtn.addEventListener("click", function () {
      showMcpErr("");
      var name = (mcpName && mcpName.value || "").trim();
      if (!name) {
        showMcpErr("Name required.");
        return;
      }
      var kind = (mcpKind && mcpKind.value) || "stdio";
      var body = { name: name };
      if (kind === "stdio") {
        var line = (mcpCommand && mcpCommand.value || "").trim();
        if (!line) {
          showMcpErr("Command required.");
          return;
        }
        var parts = line.split(/\s+/);
        body.command = parts[0];
        body.args = parts.slice(1);
      } else {
        var url = (mcpUrl && mcpUrl.value || "").trim();
        if (!url) {
          showMcpErr("URL required.");
          return;
        }
        body.url = url;
        body.type = kind === "sse" ? "sse" : "streamableHttp";
      }
      mcpAddBtn.disabled = true;
      fetch(apiURL("/api/mcp/add"), {
        method: "POST",
        headers: headers(),
        body: JSON.stringify(body),
      })
        .then(function (r) {
          return r.text().then(function (t) {
            if (!r.ok) throw new Error(t || r.status);
          });
        })
        .then(function () {
          if (mcpName) mcpName.value = "";
          if (mcpCommand) mcpCommand.value = "";
          if (mcpUrl) mcpUrl.value = "";
          return loadMcpList();
        })
        .catch(function (e) { showMcpErr(e.message || e); })
        .finally(function () { mcpAddBtn.disabled = false; });
    });
  }

  gateCancelBtn.addEventListener("click", closeGate);
  document.addEventListener("keydown", function (e) {
    if (e.key !== "Escape") return;
    if (gate.hidden || !gateManage) return;
    e.preventDefault();
    closeGate();
  });

  function onBgColorInput() {
    applyThemeBg(bgLightEl && bgLightEl.value, bgDarkEl && bgDarkEl.value);
  }
  function onBgColorCommit() {
    persistThemeBg();
  }
  if (bgLightEl) {
    bgLightEl.addEventListener("input", onBgColorInput);
    bgLightEl.addEventListener("change", onBgColorCommit);
  }
  if (bgDarkEl) {
    bgDarkEl.addEventListener("input", onBgColorInput);
    bgDarkEl.addEventListener("change", onBgColorCommit);
  }
  if (bgResetBtn) {
    bgResetBtn.addEventListener("click", function () {
      applyThemeBg(BG_LIGHT_DEFAULT, BG_DARK_DEFAULT);
      persistThemeBg();
    });
  }

  if (threadSearchEl) {
    threadSearchEl.addEventListener("input", function () {
      scheduleThreadSearch();
    });
    threadSearchEl.placeholder = "Search titles & content…";
  }
  if (workspaceToggle) {
    workspaceToggle.addEventListener("click", function () {
      var open = workspaceMenu && workspaceMenu.hidden;
      if (open) loadWorkspaces();
      setWorkspaceMenuOpen(!!open);
    });
  }
  if (workspaceSearch) {
    workspaceSearch.addEventListener("input", function () {
      renderWorkspaceList(workspaceSearch.value);
    });
  }
  if (workspaceOpenBtn) {
    workspaceOpenBtn.addEventListener("click", function () {
      openWorkspacePath(workspacePath && workspacePath.value);
    });
  }
  if (workspacePath) {
    workspacePath.addEventListener("keydown", function (e) {
      if (e.key === "Enter") {
        e.preventDefault();
        openWorkspacePath(workspacePath.value);
      }
    });
  }
  document.addEventListener("click", function (e) {
    if (!workspaceMenu || workspaceMenu.hidden) return;
    var t = e.target;
    if (workspaceSwitchContains(t)) return;
    setWorkspaceMenuOpen(false);
  });
  function workspaceSwitchContains(el) {
    var root = document.getElementById("workspace-switch");
    return root && el && root.contains(el);
  }
  if (chaptersToggle) {
    chaptersToggle.addEventListener("click", function () {
      if (!chaptersFlyout) return;
      closePaperMenu();
      var open = chaptersFlyout.hidden;
      chaptersFlyout.hidden = !open;
      chaptersToggle.setAttribute("aria-expanded", open ? "true" : "false");
      if (open) renderChapters();
    });
  }
  if (paperMenuToggle) {
    paperMenuToggle.addEventListener("click", function () {
      closeChaptersFlyout();
      setPaperMenuOpen(!!(paperMenu && paperMenu.hidden));
    });
  }
  document.addEventListener("click", function (e) {
    if (chaptersFlyout && !chaptersFlyout.hidden) {
      var left = chaptersToggle && chaptersToggle.parentElement;
      if (!(left && left.contains(e.target))) closeChaptersFlyout();
    }
    if (paperMenu && !paperMenu.hidden) {
      var right = paperMenuToggle && paperMenuToggle.parentElement;
      if (!(right && right.contains(e.target))) closePaperMenu();
    }
  });

  settingsTabs.querySelectorAll(".settings-tab").forEach(function (btn) {
    btn.addEventListener("click", function () {
      setSettingsTab(btn.getAttribute("data-tab"));
      if (btn.getAttribute("data-tab") === "agent" && connected) loadAgentSettings();
    });
  });
  saveSettingsBtn.addEventListener("click", saveAgentSettings);

  newChatBtn.addEventListener("click", function () {
    if (!connected || agentState === "running" || agentState === "awaiting_approval" || agentState === "compacting") return;
    if (!window.confirm("Start a new thread in this workspace?")) return;
    newChatBtn.disabled = true;
    fetch(apiURL("/api/threads"), { method: "POST", headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          return j;
        });
      })
      .then(function () {
        messages = [];
        chapters = [];
        activeChapter = null;
        clearLive();
        liveWrap.hidden = true;
        hideApproval();
        renderChapters();
        renderThread();
        setState("idle");
        return loadThreads();
      })
      .then(function () {
        msg.focus();
      })
      .catch(function (err) {
        appendLive("\n[thread] " + err + "\n");
      })
      .finally(function () {
        newChatBtn.disabled = false;
      });
  });

  function sendMessage(text) {
    text = String(text || "").trim();
    if (!text) return;
    send.disabled = true;
    clearLive();
    liveWrap.hidden = false;
    liveWrap.open = true;
    fetch(apiURL("/api/message"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ text: text }),
    })
      .then(function (r) {
        if (!r.ok) return r.text().then(function (t) { throw new Error(t || r.status); });
        msg.value = "";
        messages.push({ role: "user", text: text });
        renderThread();
        setState("running");
      })
      .catch(function (err) {
        appendLive("\n[send] " + err + "\n");
        send.disabled = false;
      });
  }

  document.getElementById("composer").addEventListener("submit", function (e) {
    e.preventDefault();
    if (recording) {
      stopRecording(true);
      return;
    }
    sendMessage(msg.value);
  });

  function decide(allow) {
    fetch(apiURL("/api/approve"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ allow: allow }),
    }).catch(function (err) {
      appendLive("\n[approve] " + err + "\n");
    });
  }
  approve.addEventListener("click", function () { decide(true); });
  deny.addEventListener("click", function () { decide(false); });

  function pickMime() {
    var types = ["audio/webm;codecs=opus", "audio/webm", "audio/ogg;codecs=opus", "audio/mp4"];
    for (var i = 0; i < types.length; i++) {
      if (window.MediaRecorder && MediaRecorder.isTypeSupported && MediaRecorder.isTypeSupported(types[i])) {
        return types[i];
      }
    }
    return "";
  }

  function stopTracks() {
    if (mediaStream) {
      mediaStream.getTracks().forEach(function (t) { t.stop(); });
      mediaStream = null;
    }
  }

  function uploadRecording(blob) {
    var sendAfter = voiceSendPending;
    voiceSendPending = false;
    setVoiceStatus("Transcribing…");
    var fd = new FormData();
    var ext = (blob.type || "").indexOf("mp4") >= 0 ? "mp4" : (blob.type || "").indexOf("ogg") >= 0 ? "ogg" : "webm";
    fd.append("audio", blob, "voice." + ext);
    var h = headers();
    delete h["Content-Type"];
    return fetch(apiURL("/api/transcribe"), { method: "POST", headers: h, body: fd })
      .then(function (r) {
        return r.text().then(function (t) {
          if (!r.ok) throw new Error((t || r.statusText || ("HTTP " + r.status)).slice(0, 240));
          var j = {};
          try { j = JSON.parse(t); } catch (e) { throw new Error("bad transcription response"); }
          return j.text || "";
        });
      })
      .then(function (text) {
        text = String(text || "").trim();
        if (!text) {
          setVoiceStatus("No speech detected.");
          return;
        }
        if (msg.value && !/\s$/.test(msg.value)) msg.value += " ";
        msg.value += text;
        setVoiceStatus("");
        if (sendAfter) {
          sendMessage(msg.value);
          return;
        }
        msg.focus();
      })
      .catch(function (err) {
        voiceSendPending = false;
        setVoiceStatus("Voice: " + err);
      });
  }

  function stopRecording(sendAfter) {
    voiceSendPending = !!sendAfter;
    if (!mediaRecorder || mediaRecorder.state === "inactive") {
      recording = false;
      voiceBtn.classList.remove("recording");
      voiceBtn.textContent = "Mic";
      stopTracks();
      voiceSendPending = false;
      return;
    }
    mediaRecorder.stop();
  }

  function canUseVoice() {
    return connected && voiceOK && agentState !== "running" && agentState !== "awaiting_approval";
  }

  function startRecording() {
    if (!canUseVoice() || recording) return;
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      setVoiceStatus("Microphone API unavailable in this WebView.");
      return;
    }
    var mime = pickMime();
    navigator.mediaDevices.getUserMedia({ audio: true })
      .then(function (stream) {
        mediaStream = stream;
        mediaChunks = [];
        var opts = mime ? { mimeType: mime } : undefined;
        mediaRecorder = new MediaRecorder(stream, opts);
        mediaRecorder.ondataavailable = function (e) {
          if (e.data && e.data.size > 0) mediaChunks.push(e.data);
        };
        mediaRecorder.onstop = function () {
          recording = false;
          voiceBtn.classList.remove("recording");
          voiceBtn.textContent = "Mic";
          setState(agentState);
          var type = mime || (mediaChunks[0] && mediaChunks[0].type) || "audio/webm";
          var blob = new Blob(mediaChunks, { type: type });
          stopTracks();
          if (blob.size < 64) {
            voiceSendPending = false;
            setVoiceStatus("Recording too short.");
            return;
          }
          uploadRecording(blob);
        };
        mediaRecorder.start();
        recording = true;
        voiceSendPending = false;
        voiceBtn.classList.add("recording");
        voiceBtn.textContent = "Stop";
        setVoiceStatus("Recording… Space = draft · Enter = send");
        setState(agentState);
        msg.focus();
      })
      .catch(function (err) {
        setVoiceStatus("Mic permission: " + err);
      });
  }

  voiceBtn.addEventListener("click", function () {
    if (recording) stopRecording(false);
    else startRecording();
  });

  // Voice/send hotkeys only while the chat composer is focused, so dialogs
  // (Skills search, settings, …) can use Space/Enter normally.
  document.addEventListener("keydown", function (e) {
    if (app.hidden || !gate.hidden) return;
    if (e.ctrlKey || e.metaKey || e.altKey) return;
    if (document.activeElement !== msg) return;

    if (recording) {
      if (e.key === " " || e.code === "Space") {
        e.preventDefault();
        stopRecording(false);
        return;
      }
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        stopRecording(true);
        return;
      }
      if (e.key === "F1") {
        e.preventDefault();
        stopRecording(false);
      }
      return;
    }

    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      if (msg.value.trim() === "" && canUseVoice()) {
        startRecording();
        return;
      }
      sendMessage(msg.value);
      return;
    }

    if (!canUseVoice()) return;
    if (msg.value.trim() !== "") return;
    if (e.key === " " || e.code === "Space" || e.key === "F1") {
      e.preventDefault();
      startRecording();
    }
  });

  function persistWindowGeometry() {
    if (typeof keimlingSaveGuiGeometry !== "function") return;
    try {
      keimlingSaveGuiGeometry(Math.round(window.outerWidth), Math.round(window.outerHeight));
    } catch (e) {}
  }
  var geometryTimer = null;
  window.addEventListener("resize", function () {
    if (geometryTimer) clearTimeout(geometryTimer);
    geometryTimer = setTimeout(persistWindowGeometry, 400);
  });
  window.addEventListener("pagehide", persistWindowGeometry);

  // Desktop WebView: open http(s) links in the system browser so the UI is not replaced
  // (share_file URLs are often same-origin as the agent).
  function openLinkExternally(e) {
    if (typeof keimlingOpenExternal !== "function") return;
    var a = e.target && e.target.closest && e.target.closest("a[href]");
    if (!a) return;
    var href = a.getAttribute("href");
    if (!href || href.charAt(0) === "#" || href.indexOf("javascript:") === 0) return;
    e.preventDefault();
    e.stopPropagation();
    try { keimlingOpenExternal(a.href); } catch (err) {}
  }
  document.addEventListener("click", openLinkExternally, true);
  document.addEventListener("auxclick", openLinkExternally, true);

  (function boot() {
    var qSecret = "";
    try {
      qSecret = new URLSearchParams(window.location.search).get("access_secret") || "";
    } catch (e) {}
    var prefs = loadPrefs();
    applyThemeBg(prefs.bgLight || BG_LIGHT_DEFAULT, prefs.bgDark || BG_DARK_DEFAULT);

    // Desktop / resume path: secret in URL → enter session immediately.
    if (qSecret) {
      secretEl.value = qSecret;
      if (sameOriginLocal()) {
        setChoice("local");
      } else {
        setChoice("remote");
        try { connectURLEl.value = location.origin + "/"; } catch (e) {}
      }
      openSession({ auto: true });
      return;
    }

    // Returning visitor with stored secret for this origin → try resume.
    if (prefs.seen && prefs.secret && !prefs.url) {
      secretEl.value = prefs.secret;
      setChoice("local");
      openSession({ auto: true });
      return;
    }
    if (prefs.seen && prefs.secret && prefs.url) {
      secretEl.value = prefs.secret;
      connectURLEl.value = prefs.url;
      setChoice("remote");
      openSession({ auto: true });
      return;
    }

    showGate({ prefillSecret: prefs.secret || "" });
  })();
})();

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
  var newChatBtn = document.getElementById("new-chat");
  var settingsTabs = document.getElementById("settings-tabs");
  var paneConnection = document.getElementById("pane-connection");
  var paneAgent = document.getElementById("pane-agent");
  var saveSettingsBtn = document.getElementById("save-settings");
  var gateCancelBtn = document.getElementById("gate-cancel");
  var gateManage = false;
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
    } catch (e) {}
  }

  function loadPrefs() {
    try {
      return {
        seen: localStorage.getItem(STORAGE_SEEN) === "1",
        secret: localStorage.getItem(STORAGE_SECRET) || "",
        url: localStorage.getItem(STORAGE_URL) || "",
      };
    } catch (e) {
      return { seen: false, secret: "", url: "" };
    }
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
    var agent = tab === "agent";
    settingsTabs.querySelectorAll(".settings-tab").forEach(function (btn) {
      var on = btn.getAttribute("data-tab") === tab;
      btn.classList.toggle("active", on);
      btn.setAttribute("aria-selected", on ? "true" : "false");
    });
    paneConnection.hidden = agent;
    paneAgent.hidden = !agent;
    saveSettingsBtn.hidden = !agent;
    connectBtn.hidden = agent;
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
    if (prefs.url) {
      connectURLEl.value = prefs.url;
      setChoice("remote");
    } else {
      setChoice("local");
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

  function setState(s) {
    agentState = s || "idle";
    stateEl.textContent = agentState;
    stateEl.className = "state " + agentState;
    var waiting = agentState === "awaiting_approval";
    approve.disabled = !waiting;
    deny.disabled = !waiting;
    // While recording, Send means “stop + transcribe + send” (same as Enter in CLI).
    send.disabled = !connected || agentState === "running" || waiting;
    msg.disabled = !connected || agentState === "running" || waiting;
    voiceBtn.disabled = !connected || !voiceOK || agentState === "running" || waiting;
    newChatBtn.disabled = !connected || agentState === "running" || waiting;
    if (!waiting) hideApproval();
    if (agentState === "running" || waiting) {
      liveWrap.hidden = false;
      liveWrap.open = true;
    }
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
    approvalCmd.textContent = cmd;
    if (payload.preview) {
      approvalPreview.hidden = false;
      approvalPreview.textContent = payload.preview;
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
      });
      chaptersEl.appendChild(btn);
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
    setState(j.busy ? "running" : "idle");
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
              if (ev === "status" && payload.state) {
                setState(payload.state);
                if (payload.state === "idle") loadSession().catch(function () {});
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
          saveGuiPrefs({ connect: remoteBase, access_secret: secret() })
            .catch(function (e) {
              showGate({ error: "Could not remember remote connection: " + formatConnectError(e), manage: true, keepApp: !app.hidden });
              connectBtn.disabled = false;
              throw e;
            })
            .then(function () {
              location.assign(dest);
            })
            .catch(function () {});
          return;
        }
      } catch (e) {
        showGate({ error: formatConnectError(e), manage: true, keepApp: !app.hidden });
        return;
      }
    } else {
      apiBase = "";
      // Same-origin server always needs a secret unless Go injected one into the URL / field.
      if (!sameOriginLocal() && !secret()) {
        showGate({ error: "Access secret required.", manage: true, keepApp: !app.hidden });
        return;
      }
      if (sameOriginLocal() && !secret()) {
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

  gateCancelBtn.addEventListener("click", closeGate);
  document.addEventListener("keydown", function (e) {
    if (e.key !== "Escape") return;
    if (gate.hidden || !gateManage) return;
    e.preventDefault();
    closeGate();
  });

  settingsTabs.querySelectorAll(".settings-tab").forEach(function (btn) {
    btn.addEventListener("click", function () {
      setSettingsTab(btn.getAttribute("data-tab"));
      if (btn.getAttribute("data-tab") === "agent" && connected) loadAgentSettings();
    });
  });
  saveSettingsBtn.addEventListener("click", saveAgentSettings);

  newChatBtn.addEventListener("click", function () {
    if (!connected || agentState === "running" || agentState === "awaiting_approval") return;
    if (!window.confirm("Start a new chat? The current session will be archived.")) return;
    newChatBtn.disabled = true;
    fetch(apiURL("/api/newchat"), { method: "POST", headers: headers() })
      .then(function (r) {
        return r.text().then(function (t) {
          if (!r.ok) throw new Error(t || r.status);
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
        msg.focus();
      })
      .catch(function (err) {
        appendLive("\n[newchat] " + err + "\n");
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

  // CLI parity: empty prompt Space/Enter/F1 starts; while recording Space=draft, Enter=send.
  document.addEventListener("keydown", function (e) {
    if (app.hidden || !gate.hidden) return;
    if (e.ctrlKey || e.metaKey || e.altKey) return;

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

  (function boot() {
    var qSecret = "";
    try {
      qSecret = new URLSearchParams(window.location.search).get("access_secret") || "";
    } catch (e) {}
    var prefs = loadPrefs();

    // Desktop / resume path: secret in URL → enter session immediately.
    if (qSecret) {
      secretEl.value = qSecret;
      setChoice("local");
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

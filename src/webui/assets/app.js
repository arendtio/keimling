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
  var loginPanel = document.getElementById("login-panel");
  var loginSecretEl = document.getElementById("login-secret");
  var loginContinue = document.getElementById("login-continue");
  var urlField = document.getElementById("url-field");
  var secretField = document.getElementById("secret-field");
  var localHint = document.getElementById("local-hint");
  var choiceLocal = document.getElementById("choice-local");
  var choiceRemote = document.getElementById("choice-remote");
  var openConnection = document.getElementById("open-connection");
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
  var mcpToken = document.getElementById("mcp-token");
  var mcpStdioRow = document.getElementById("mcp-stdio-row");
  var mcpUrlRow = document.getElementById("mcp-url-row");
  var mcpTokenRow = document.getElementById("mcp-token-row");
  var mcpEnvRow = document.getElementById("mcp-env-row");
  var mcpEnvList = document.getElementById("mcp-env-list");
  var mcpEnvAdd = document.getElementById("mcp-env-add");
  var mcpAuthRow = document.getElementById("mcp-auth-row");
  var mcpAuth = document.getElementById("mcp-auth");
  var mcpHeaderRow = document.getElementById("mcp-headers-row");
  var mcpHeaderList = document.getElementById("mcp-header-list");
  var mcpHeaderAdd = document.getElementById("mcp-header-add");
  var mcpAddBtn = document.getElementById("mcp-add-btn");
  var mcpReloadBtn = document.getElementById("mcp-reload-btn");
  var mcpErr = document.getElementById("mcp-err");
  var artifactPane = document.getElementById("artifact-pane");
  var artifactBackdrop = document.getElementById("artifact-backdrop");
  var artifactFrame = document.getElementById("artifact-frame");
  var artifactHistory = document.getElementById("artifact-history");
  var artifactOpenExt = document.getElementById("artifact-open-ext");
  var artifactFullscreen = document.getElementById("artifact-fullscreen");
  var artifactPaper = document.getElementById("artifact-paper");
  var artifactEmpty = document.getElementById("artifact-empty");
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
  var STORAGE_SIDEBAR_TAB = "keimling.sidebar_tab";
  var STORAGE_SIDEBAR_OPEN = "keimling.sidebar_open";
  var sideTab = "now";
  var sidePhase = "idle";
  var sideRate = 0;
  var sideThinkStart = 0;
  var sideStepResponded = false;
  var sideLastModel = 0;
  var sideMcpStatus = "";
  var sideSamples = [];
  var sideSampleAt = 0;
  var sideSpike = 0;
  var sideClosed = false;
  var SIDE_TYPICAL = 120;
  var SIDE_BAR_KEEP = 400;
  var sideBarWidthCached = 0;
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
  var paneSecrets = document.getElementById("pane-secrets");
  var paneSkills = document.getElementById("pane-skills");
  var paneMcp = document.getElementById("pane-mcp");
  var panePlugins = document.getElementById("pane-plugins");
  var paneThemes = document.getElementById("pane-themes");
  var vaultStatusEl = document.getElementById("vault-status");
  var vaultUnlockEl = document.getElementById("vault-unlock");
  var vaultBodyEl = document.getElementById("vault-body");
  var vaultPasswordEl = document.getElementById("vault-password");
  var vaultUnlockBtn = document.getElementById("vault-unlock-btn");
  var vaultCreateBtn = document.getElementById("vault-create-btn");
  var credentialListEl = document.getElementById("credential-list");
  var credentialNewBtn = document.getElementById("credential-new");
  var credentialEditor = document.getElementById("credential-editor");
  var credEditName = document.getElementById("cred-edit-name");
  var credEditDescription = document.getElementById("cred-edit-description");
  var credEditType = document.getElementById("cred-edit-type");
  var credEditService = document.getElementById("cred-edit-service");
  var credEditTools = document.getElementById("cred-edit-tools");
  var credEditInject = document.getElementById("cred-edit-inject");
  var credEditInjectEnv = document.getElementById("cred-edit-inject-env");
  var credEditSecret = document.getElementById("cred-edit-secret");
  var credEditHint = document.getElementById("cred-edit-hint");
  var credEditCancel = document.getElementById("cred-edit-cancel");
  var credEditSave = document.getElementById("cred-edit-save");
  var credEditDelete = document.getElementById("cred-edit-delete");
  var credentialsCache = [];
  var editingCredentialID = null;
  var vaultUnlocked = false;
  var saveSettingsBtn = document.getElementById("save-settings");
  var gateCancelBtn = document.getElementById("gate-cancel");
  var gateCloseBtn = document.getElementById("gate-close");
  var gateManage = false;
  var workspaceToggle = document.getElementById("workspace-toggle");
  var workspaceMenu = document.getElementById("workspace-menu");
  var workspaceLabel = document.getElementById("workspace-label");
  var workspaceRootEl = document.getElementById("workspace-root");
  var workspaceList = document.getElementById("workspace-list");
  var workspaceSearch = document.getElementById("workspace-search");
  var workspacePath = document.getElementById("workspace-path");
  var workspaceNewName = document.getElementById("workspace-new-name");
  var workspaceOpenBtn = document.getElementById("workspace-open-btn");
  var workspaceManageBtn = document.getElementById("workspace-manage-btn");
  var workspaceDialog = document.getElementById("workspace-dialog");
  var threadListEl = document.getElementById("thread-list");
  var threadSearchEl = document.getElementById("thread-search");
  var chaptersToggle = document.getElementById("chapters-toggle");
  var chaptersFlyout = document.getElementById("chapters-flyout");
  var sidebarToggle = document.getElementById("sidebar-toggle");
  var workspacesCache = [];
  var workspaceUndo = null;
  var threadsCache = [];
  var threadsListState = "idle";
  var threadsShowAll = false;
  var threadsSearchTimer = null;
  var threadsSearchActive = false;
  var agentsCache = [];
  var definitionsCache = [];
  var teamsCache = [];
  var teamSelectIDs = {};
  var currentWorkspaceID = "";
  var currentThreadID = "";
  var workspaceThreadID = "";
  var currentAgentInstanceID = "";
  var selectedDefinitionID = "";
  var editingDefinitionID = "";
  var agentsPollTimer = null;
  var agentsPollInFlight = false;
  var agentListEl = document.getElementById("agent-list");
  var newAgentBtn = document.getElementById("new-agent");
  var agentDefDialog = document.getElementById("agent-def-dialog");
  var agentDefPick = document.getElementById("agent-def-pick");
  var agentDefCreate = document.getElementById("agent-def-create");
  var agentDefEdit = document.getElementById("agent-def-edit");
  var agentDefInstance = document.getElementById("agent-def-instance");
  var agentDefList = document.getElementById("agent-def-list");
  var agentTeamList = document.getElementById("agent-team-list");
  var agentDefErr = document.getElementById("agent-def-err");
  var reviewPromptFactory = "";
  var agentDefName = document.getElementById("agent-def-name");
  var agentDefInstructions = document.getElementById("agent-def-instructions");
  var agentDefEditName = document.getElementById("agent-def-edit-name");
  var agentDefEditInstructions = document.getElementById("agent-def-edit-instructions");
  var agentInstanceName = document.getElementById("agent-instance-name");
  var agentInstanceDefinition = document.getElementById("agent-instance-definition");
  var agentInstanceList = document.getElementById("agent-instance-list");
  var agentToolCatalog = [];
  var agentToolGroups = [];
  var mcpServersCache = [];
  var allInstancesCache = [];
  var manageTab = "definitions";
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
  var setAutopilotInstructions = document.getElementById("set-autopilot-instructions");
  var setAutopilotInstructionsStatus = document.getElementById("set-autopilot-instructions-status");
  var setAutopilotInstructionsErr = document.getElementById("set-autopilot-instructions-err");
  var setAutopilotInstructionsReset = document.getElementById("set-autopilot-instructions-reset");
  var setAutopilotInstructionsUndo = document.getElementById("set-autopilot-instructions-undo");
  var autopilotSettingsCustom = false;
  var autopilotSettingsBaseline = "";
  var autopilotSettingsUndoText = "";
  var autopilotInstructionsEl = document.getElementById("autopilot-instructions");
  var autopilotInstructionsStatus = document.getElementById("autopilot-instructions-status");
  var autopilotInstructionsErr = document.getElementById("autopilot-instructions-err");
  var autopilotInstructionsSave = document.getElementById("autopilot-instructions-save");
  var autopilotInstructionsReset = document.getElementById("autopilot-instructions-reset");
  var autopilotInstructionsUndo = document.getElementById("autopilot-instructions-undo");
  var threadAutopilotCustom = false;
  var threadAutopilotSaved = "";
  var threadAutopilotUndoText = "";
  var setMaxSteps = document.getElementById("set-max-steps");
  var setContextLimit = document.getElementById("set-context-limit");
  var setVoiceModel = document.getElementById("set-voice-model");
  var setVoiceProvider = document.getElementById("set-voice-provider");
  var setRealtimeProvider = document.getElementById("set-realtime-provider");
  var setRealtimeModel = document.getElementById("set-realtime-model");
  var setRealtimeVoice = document.getElementById("set-realtime-voice");
  var setSearchKind = document.getElementById("set-search-kind");
  var setSearchURL = document.getElementById("set-search-url");
  var setSearchKey = document.getElementById("set-search-key");
  var searchURLLabel = document.getElementById("search-url-label");
  var searchURLHint = document.getElementById("search-url-hint");
  var searchKeyLabel = document.getElementById("search-key-label");
  var searchKeyHint = document.getElementById("search-key-hint");
  var providerBanner = document.getElementById("provider-banner");
  var providerDialog = document.getElementById("provider-dialog");
  var needsProvider = false;
  var agentSettingsLoaded = false;
  var clientConnectionsCache = [];
  var clientActiveID = "";
  var serverConnectionsCache = [];
  var serverActiveID = "";
  // Selection for Continue: { source: "current"|"client"|"server"|"local", id?: string }
  var connectTarget = { source: "current" };
  var editingConnectionID = null;
  var editingConnectionScope = "client"; // client | server
  var STORAGE_CLIENT_CONN = "keimling.client_connections";
  var connectionCurrentEl = document.getElementById("connection-current");
  var connectionClientListEl = document.getElementById("connection-client-list");
  var connectionServerListEl = document.getElementById("connection-server-list");
  var connectionEditor = document.getElementById("connection-editor");
  var connectionClientNewBtn = document.getElementById("connection-client-new");
  var connectionServerNewBtn = document.getElementById("connection-server-new");
  var connectionUseLocalBtn = document.getElementById("connection-use-local");
  var connEditScope = document.getElementById("conn-edit-scope");
  var connEditName = document.getElementById("conn-edit-name");
  var connEditURL = document.getElementById("conn-edit-url");
  var connEditSecret = document.getElementById("conn-edit-secret");
  var connEditURLWrap = document.getElementById("conn-edit-url-wrap");
  var connEditSecretWrap = document.getElementById("conn-edit-secret-wrap");
  var connEditHint = document.getElementById("conn-edit-hint");
  var connEditCancel = document.getElementById("conn-edit-cancel");
  var connEditSave = document.getElementById("conn-edit-save");
  var connEditDelete = document.getElementById("conn-edit-delete");
  var providersCache = [];
  var activeProviderID = "";
  var editingProviderID = null;
  var editingProviderModels = [];
  var editingActiveModelID = "";
  var modelFormMode = null; // null | "add" | "edit"
  var modelFormOriginalID = "";
  var catalogCache = [];
  var providerListEl = document.getElementById("provider-list");
  var providerEditor = document.getElementById("provider-editor");
  var providerNewBtn = document.getElementById("provider-new");
  var provEditName = document.getElementById("prov-edit-name");
  var provEditBaseURL = document.getElementById("prov-edit-base-url");
  var provEditKind = document.getElementById("prov-edit-kind");
  var provEditAPIKey = document.getElementById("prov-edit-api-key");
  var provEditCancel = document.getElementById("prov-edit-cancel");
  var provEditSave = document.getElementById("prov-edit-save");
  var provEditDelete = document.getElementById("prov-edit-delete");
  var provModelList = document.getElementById("prov-model-list");
  var provModelNew = document.getElementById("prov-model-new");
  var provModelEditor = document.getElementById("prov-model-editor");
  var provModelEditorTitle = document.getElementById("prov-model-editor-title");
  var provModelID = document.getElementById("prov-model-id");
  var provModelName = document.getElementById("prov-model-name");
  var provModelContext = document.getElementById("prov-model-context");
  var provModelReasoning = document.getElementById("prov-model-reasoning");
  var provModelPromptPrice = document.getElementById("prov-model-prompt-price");
  var provModelCompletionPrice = document.getElementById("prov-model-completion-price");
  var provModelImagesLabel = document.getElementById("prov-model-images-label");
  var provModelImages = document.getElementById("prov-model-images");
  var provModelImagesHint = document.getElementById("prov-model-images-hint");
  var provModelAudioLabel = document.getElementById("prov-model-audio-label");
  var provModelAudio = document.getElementById("prov-model-audio");
  var provModelAudioHint = document.getElementById("prov-model-audio-hint");
  var attachRow = document.getElementById("attach-row");
  var attachFileBtn = document.getElementById("attach-file");
  var attachInput = document.getElementById("attach-input");
  var pendingFiles = [];
  var provModelSave = document.getElementById("prov-model-save");
  var provModelCancel = document.getElementById("prov-model-cancel");
  var provCatalogFetch = document.getElementById("prov-catalog-fetch");
  var provCatalogStatus = document.getElementById("prov-catalog-status");
  var provCatalogDatalist = document.getElementById("prov-catalog-datalist");
  var budgetLineEl = document.getElementById("budget-line");

  var thread = document.getElementById("thread");
  var chaptersEl = document.getElementById("chapters");
  var uiFlash = document.getElementById("ui-flash");
  var stateEl = document.getElementById("state");
  var agentStatus = document.getElementById("agent-status");
  var progressDetail = document.getElementById("progress-detail");
  var modelEl = document.getElementById("model");
  var modelMenu = document.getElementById("model-menu");
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
  var callBtn = document.getElementById("call");
  var callMuteBtn = document.getElementById("call-mute");
  var realtimeReady = false;
  var callLive = false;
  var callMuted = false;
  var callSentAudio = false;
  var callUserSpeaking = false;
  var callRecovering = false;
  var callRecoverCreate = false;
  var callResponseSpeak = 0;
  var callPictureAt = 0;
  var callPicturePending = false;
  var callBgComment = "";
  var stopBtn = document.getElementById("stop");
  var composerMoreBtn = document.getElementById("composer-more");
  var composerMoreMenu = document.getElementById("composer-more-menu");
  var voiceStatus = document.getElementById("voice-status");

  var es = null;
  var messages = [];
  var chapters = [];
  var activeChapter = null;
  var runningThreadID = "";
  var backgroundBusy = false;
  var transcriptSeq = 0;
  var heldOpen = null;
  var connected = false;
  var agentState = "idle";
  var stoppingThreadID = "";
  var stoppingAgents = {};
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

  function askConfirm(message, confirmLabel) {
    var dlg = document.getElementById("confirm-dialog");
    var text = document.getElementById("confirm-text");
    var ok = document.getElementById("confirm-ok");
    var cancel = document.getElementById("confirm-cancel");
    if (!dlg || !text || !ok || typeof dlg.showModal !== "function") {
      return Promise.resolve(false);
    }
    if (dlg.open) dlg.close();
    text.textContent = message == null ? "" : String(message);
    ok.textContent = confirmLabel || "OK";
    dlg.returnValue = "";
    return new Promise(function (resolve) {
      function onClose() {
        dlg.removeEventListener("close", onClose);
        resolve(dlg.returnValue === "ok");
      }
      dlg.addEventListener("close", onClose);
      dlg.showModal();
      if (cancel) cancel.focus();
    });
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

  var THEME_ROLES = [
    ["bg", "Frame"],
    ["surface", "Workplace"],
    ["surface2", "Raised"],
    ["line", "Line"],
    ["fg", "Text"],
    ["muted", "Muted"],
    ["accent", "Accent"],
    ["danger", "Danger"],
    ["bubble", "Bubble"]
  ];
  var THEME_KEIMLING = {
    id: "keimling",
    name: "Keimling",
    builtin: true,
    light: { bg: "#eff0f1", surface: "#fafbfc", surface2: "#e6e8ea", line: "#d4d6d8", fg: "#232629", muted: "#64686d", accent: "#2f8f6b", danger: "#b54b4b", bubble: "#e4e6e8" },
    dark: { bg: "#202326", surface: "#151916", surface2: "#1c221e", line: "#3c453e", fg: "#e7ece8", muted: "#8e998f", accent: "#5d9244", danger: "#d98989", bubble: "#242824" }
  };
  var themeState = { themes: [], active: "keimling", scheme: "auto" };
  var themeEditID = "";

  function themeCopy(theme) {
    return JSON.parse(JSON.stringify(theme || THEME_KEIMLING));
  }

  function themeCatalog() {
    var byID = { keimling: themeCopy(THEME_KEIMLING) };
    (themeState.themes || []).forEach(function (t) {
      if (t && t.id) byID[t.id] = themeCopy(t);
    });
    if (!byID.keimling) byID.keimling = themeCopy(THEME_KEIMLING);
    return Object.keys(byID).map(function (id) { return byID[id]; });
  }

  function themeActive() {
    var id = themeState.active || "keimling";
    var found = null;
    themeCatalog().forEach(function (t) { if (t.id === id) found = t; });
    return found || themeCopy(THEME_KEIMLING);
  }

  function applyThemeRecord(theme) {
    theme = theme || themeCopy(THEME_KEIMLING);
    var root = document.documentElement;
    ["light", "dark"].forEach(function (scheme) {
      var pal = theme[scheme] || {};
      var base = THEME_KEIMLING[scheme];
      THEME_ROLES.forEach(function (role) {
        var hex = normalizeBgHex(pal[role[0]]) || base[role[0]];
        root.style.setProperty("--theme-" + role[0] + "-" + scheme, hex);
      });
    });
  }

  function applyThemeBg(light, dark) {
    var theme = themeActive();
    var L = normalizeBgHex(light);
    var D = normalizeBgHex(dark);
    if (L) theme.light.bg = L;
    if (D) theme.dark.bg = D;
    applyThemeRecord(theme);
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
        if (p.themes && p.themes.length) themeState.themes = p.themes;
        if (p.active_theme) themeState.active = p.active_theme;
        var active = themeActive();
        var L = normalizeBgHex(p.bg_light);
        var D = normalizeBgHex(p.bg_dark);
        if (!(p.themes && p.themes.length)) {
          if (L) active.light.bg = L;
          if (D) active.dark.bg = D;
        }
        if (p.color_scheme === "light" || p.color_scheme === "dark" || p.color_scheme === "auto") themeState.scheme = p.color_scheme;
        applyThemeRecord(active);
        applyColorScheme();
        renderThemeList();
        applySidebarFromGuiPrefs(p);
      })
      .catch(function () {});
  }

  function setChoice(next) {
    mode = next;
    if (choiceLocal) {
      choiceLocal.classList.toggle("active", mode === "local");
      choiceLocal.setAttribute("aria-pressed", mode === "local" ? "true" : "false");
    }
    if (choiceRemote) {
      choiceRemote.classList.toggle("active", mode === "remote");
      choiceRemote.setAttribute("aria-pressed", mode === "remote" ? "true" : "false");
    }
    if (urlField) urlField.hidden = mode !== "remote";
    if (secretField) secretField.hidden = true;
    if (localHint) localHint.hidden = true;
    renderAllConnectionLists();
  }

  function currentSessionLabel() {
    if (sameOriginLocal()) return "Desktop agent (this window)";
    try { return location.origin; } catch (e) { return "This Keimling"; }
  }

  function findIn(list, id) {
    for (var i = 0; i < list.length; i++) {
      if (list[i].id === id) return list[i];
    }
    return null;
  }

  function applyConnectTarget() {
    var t = connectTarget || { source: "current" };
    if (t.source === "local") {
      setChoice("local");
      connectURLEl.value = "";
      return;
    }
    if (t.source === "current") {
      if (sameOriginLocal()) {
        setChoice("local");
        connectURLEl.value = "";
      } else {
        setChoice("remote");
        try { connectURLEl.value = location.origin + "/"; } catch (e) { connectURLEl.value = ""; }
        var s = pageAccessSecret() || secret();
        if (s) secretEl.value = s;
      }
      return;
    }
    var c = t.source === "server"
      ? findIn(serverConnectionsCache, t.id)
      : findIn(clientConnectionsCache, t.id);
    if (!c || c.kind === "local") {
      connectTarget = { source: "current" };
      applyConnectTarget();
      return;
    }
    setChoice("remote");
    connectURLEl.value = c.url || "";
    if (c.access_secret) secretEl.value = c.access_secret;
  }

  function selectConnectTarget(source, id) {
    connectTarget = { source: source, id: id || "" };
    applyConnectTarget();
    renderAllConnectionLists();
  }

  function appendProfileRow(ul, opts) {
    var li = document.createElement("li");
    var pick = document.createElement("button");
    pick.type = "button";
    pick.className = "profile-pick" + (opts.active ? " active" : "");
    pick.innerHTML = esc(opts.title) + '<span class="profile-meta">' + esc(opts.meta || "") + "</span>";
    pick.addEventListener("click", opts.onPick);
    li.appendChild(pick);
    if (opts.onEdit) {
      var edit = document.createElement("button");
      edit.type = "button";
      edit.className = "ghost profile-edit-btn";
      edit.textContent = "Edit";
      edit.addEventListener("click", opts.onEdit);
      li.appendChild(edit);
    }
    ul.appendChild(li);
  }

  function renderAllConnectionLists() {
    var t = connectTarget || { source: "current" };
    if (connectionCurrentEl) {
      connectionCurrentEl.innerHTML = "";
      appendProfileRow(connectionCurrentEl, {
        title: currentSessionLabel(),
        meta: sameOriginLocal() ? "Resume this desktop agent" : "Stay on this server",
        active: t.source === "current",
        onPick: function () { selectConnectTarget("current"); },
      });
    }
    if (connectionClientListEl) {
      connectionClientListEl.innerHTML = "";
      if (!clientConnectionsCache.length) {
        var empty = document.createElement("li");
        empty.className = "hint";
        empty.textContent = "No saved remotes on this machine yet";
        connectionClientListEl.appendChild(empty);
      }
      clientConnectionsCache.forEach(function (c) {
        if (c.kind === "local") return;
        appendProfileRow(connectionClientListEl, {
          title: c.name || c.id,
          meta: c.url || "",
          active: t.source === "client" && t.id === c.id,
          onPick: function () { selectConnectTarget("client", c.id); },
          onEdit: function () { openConnectionEditor("client", c); },
        });
      });
    }
    if (connectionServerListEl) {
      connectionServerListEl.innerHTML = "";
      var remotes = serverConnectionsCache.filter(function (c) { return c.kind === "remote" && c.url; });
      if (!remotes.length) {
        var emptyS = document.createElement("li");
        emptyS.className = "hint";
        emptyS.textContent = "No remote bookmarks on this process";
        connectionServerListEl.appendChild(emptyS);
      }
      remotes.forEach(function (c) {
        appendProfileRow(connectionServerListEl, {
          title: c.name || c.id,
          meta: c.url || "",
          active: t.source === "server" && t.id === c.id,
          onPick: function () { selectConnectTarget("server", c.id); },
          onEdit: function () { openConnectionEditor("server", c); },
        });
      });
    }
    if (connectionUseLocalBtn) {
      connectionUseLocalBtn.hidden = sameOriginLocal() || typeof keimlingUseLocalAgent !== "function";
      connectionUseLocalBtn.classList.toggle("active", t.source === "local");
    }
  }

  function openConnectionEditor(scope, c) {
    editingConnectionScope = scope === "server" ? "server" : "client";
    editingConnectionID = c ? c.id : "";
    if (connectionEditor) connectionEditor.hidden = false;
    if (connEditScope) {
      connEditScope.textContent = editingConnectionScope === "server"
        ? "Editing a bookmark on this Keimling process"
        : "Editing your machine connection book (connections.json)";
    }
    if (connEditName) connEditName.value = c ? (c.name || "") : "Remote";
    if (connEditURL) connEditURL.value = c ? (c.url || "") : "";
    if (connEditSecret) {
      connEditSecret.value = "";
      connEditSecret.placeholder = c && (c.access_secret_set || c.access_secret) ? "Set — leave blank to keep" : "Access secret";
    }
    if (connEditURLWrap) connEditURLWrap.hidden = false;
    if (connEditSecretWrap) connEditSecretWrap.hidden = false;
    if (connEditDelete) connEditDelete.hidden = !c;
    if (connEditHint) {
      connEditHint.textContent = "URL and access secret for Continue / --connect.";
    }
  }

  function closeConnectionEditor() {
    editingConnectionID = null;
    if (connectionEditor) connectionEditor.hidden = true;
  }

  function authHeadersForPrefs() {
    var h = { "Content-Type": "application/json" };
    var s = pageAccessSecret() || secret();
    if (s) h["Authorization"] = "Bearer " + s;
    return h;
  }

  function loadClientConnectionsFromStorage() {
    try {
      var raw = localStorage.getItem(STORAGE_CLIENT_CONN);
      if (!raw) return null;
      return JSON.parse(raw);
    } catch (e) {
      return null;
    }
  }

  function saveClientConnectionsToStorage(payload) {
    try {
      localStorage.setItem(STORAGE_CLIENT_CONN, JSON.stringify(payload));
    } catch (e) {}
  }

  function persistClientConnections() {
    var body = {
      connections: clientConnectionsCache.map(function (c) {
        return {
          id: c.id,
          name: c.name,
          kind: "remote",
          url: c.url || "",
          access_secret: c.access_secret || "",
        };
      }),
      active_id: clientActiveID || "",
    };
    saveClientConnectionsToStorage(body);
    if (typeof keimlingPutClientConnections === "function") {
      try {
        var raw = keimlingPutClientConnections(JSON.stringify(body));
        if (raw) {
          var j = typeof raw === "string" ? JSON.parse(raw) : raw;
          if (j && j.connections) applyClientPayload(j);
        }
      } catch (e) {}
      return Promise.resolve();
    }
    return fetch(apiURL("/api/client-connections"), {
      method: "PUT",
      headers: authHeadersForPrefs(),
      body: JSON.stringify(body),
    }).then(function (r) {
      if (!r.ok) return;
      return r.json().then(applyClientPayload);
    }).catch(function () {});
  }

  function persistServerConnections() {
    var remotes = serverConnectionsCache.filter(function (c) { return c.kind === "remote"; });
    return saveGuiPrefs({
      connections: remotes.map(function (c) {
        return {
          id: c.id,
          name: c.name,
          kind: "remote",
          url: c.url || "",
          access_secret: c.access_secret || "",
        };
      }).concat([{ id: "local", name: "This window", kind: "local" }]),
      active_connection_id: serverActiveID || "local",
    }).catch(function () {});
  }

  function applyClientPayload(j) {
    if (!j) return;
    clientConnectionsCache = (j.connections || []).filter(function (c) {
      return c.kind !== "local" && c.url;
    });
    clientActiveID = j.active_id || "";
  }

  function applyServerPayload(p) {
    if (!p) {
      serverConnectionsCache = [];
      serverActiveID = "local";
      return;
    }
    serverConnectionsCache = p.connections || [];
    serverActiveID = p.active_connection_id || "local";
  }

  function loadClientConnections() {
    if (typeof keimlingGetClientConnections === "function") {
      try {
        var raw = keimlingGetClientConnections();
        var j = typeof raw === "string" ? JSON.parse(raw) : raw;
        applyClientPayload(j);
        return Promise.resolve();
      } catch (e) {}
    }
    return fetch(apiURL("/api/client-connections"), { headers: authHeadersForPrefs() })
      .then(function (r) {
        if (!r.ok) throw new Error("client-connections " + r.status);
        return r.json();
      })
      .then(function (j) {
        applyClientPayload(j);
        saveClientConnectionsToStorage(j);
      })
      .catch(function () {
        var stored = loadClientConnectionsFromStorage();
        if (stored) applyClientPayload(stored);
        else clientConnectionsCache = [];
      });
  }

  function loadServerConnections() {
    return fetch(apiURL("/api/gui-prefs"), { headers: authHeadersForPrefs() })
      .then(function (r) {
        if (!r.ok) return null;
        return r.json();
      })
      .then(function (p) {
        applyServerPayload(p);
      })
      .catch(function () {
        applyServerPayload(null);
      });
  }

  function loadConnectionsFromPrefs() {
    return Promise.all([loadClientConnections(), loadServerConnections()]).then(function () {
      if (!connectTarget || !connectTarget.source) {
        connectTarget = { source: "current" };
      }
      applyConnectTarget();
      renderAllConnectionLists();
    });
  }

  function saveConnectionEditor() {
    var name = connEditName ? connEditName.value.trim() : "";
    var url = connEditURL ? connEditURL.value.trim() : "";
    if (!url) {
      gateErr.hidden = false;
      gateErr.textContent = "Server URL required";
      return;
    }
    var secretVal = connEditSecret ? connEditSecret.value.trim() : "";
    var list = editingConnectionScope === "server" ? serverConnectionsCache : clientConnectionsCache;
    var existing = editingConnectionID ? findIn(list, editingConnectionID) : null;
    if (!existing) {
      var id = "remote-" + Date.now();
      var row = {
        id: id,
        name: name || "Remote",
        kind: "remote",
        url: url,
        access_secret: secretVal,
        access_secret_set: !!secretVal,
      };
      list.push(row);
      if (editingConnectionScope === "server") {
        serverActiveID = id;
        selectConnectTarget("server", id);
      } else {
        clientActiveID = id;
        selectConnectTarget("client", id);
      }
    } else {
      existing.name = name || existing.name;
      existing.url = url;
      if (secretVal) {
        existing.access_secret = secretVal;
        existing.access_secret_set = true;
      }
      selectConnectTarget(editingConnectionScope, existing.id);
    }
    closeConnectionEditor();
    renderAllConnectionLists();
    if (editingConnectionScope === "server") persistServerConnections();
    else persistClientConnections();
  }

  function deleteConnectionEditor() {
    if (!editingConnectionID) return;
    askConfirm("Delete this connection profile?", "Delete").then(function (yes) {
      if (!yes) return;
      if (editingConnectionScope === "server") {
        serverConnectionsCache = serverConnectionsCache.filter(function (c) {
          return c.id !== editingConnectionID;
        });
        if (connectTarget.source === "server" && connectTarget.id === editingConnectionID) {
          selectConnectTarget("current");
        }
        persistServerConnections();
      } else {
        clientConnectionsCache = clientConnectionsCache.filter(function (c) {
          return c.id !== editingConnectionID;
        });
        if (connectTarget.source === "client" && connectTarget.id === editingConnectionID) {
          selectConnectTarget("current");
        }
        persistClientConnections();
      }
      closeConnectionEditor();
      renderAllConnectionLists();
    });
  }

  function adoptServerConnectionToClient(c) {
    if (!c || !c.url) return Promise.resolve();
    var existing = null;
    for (var i = 0; i < clientConnectionsCache.length; i++) {
      var u = (clientConnectionsCache[i].url || "").replace(/\/$/, "");
      var cu = (c.url || "").replace(/\/$/, "");
      if (u === cu) { existing = clientConnectionsCache[i]; break; }
    }
    if (existing) {
      if (c.access_secret) existing.access_secret = c.access_secret;
      if (c.name) existing.name = c.name;
      clientActiveID = existing.id;
    } else {
      var id = "remote-" + Date.now();
      clientConnectionsCache.push({
        id: id,
        name: c.name || c.url,
        kind: "remote",
        url: c.url,
        access_secret: c.access_secret || "",
        access_secret_set: !!(c.access_secret || c.access_secret_set),
      });
      clientActiveID = id;
    }
    return persistClientConnections();
  }

  function showPluginErr(text) {
    var el = document.getElementById("plugin-err");
    if (!el) return;
    el.hidden = !text;
    el.textContent = text || "";
  }

  function loadPlugins() {
    fetch(apiURL("/api/plugins"), { headers: headers() })
      .then(readAPIResult)
      .then(function (j) { renderPlugins((j && j.plugins) || []); })
      .catch(function (err) { showPluginErr(String(err.message || err)); });
  }

  function renderPlugins(list) {
    var ul = document.getElementById("plugin-list");
    if (!ul) return;
    ul.innerHTML = "";
    if (!list.length) {
      var empty = document.createElement("li");
      empty.textContent = "No plugins yet.";
      ul.appendChild(empty);
      return;
    }
    list.forEach(function (p) {
      var li = document.createElement("li");
      var title = document.createElement("div");
      title.textContent = p.name || p.location;
      li.appendChild(title);
      var meta = document.createElement("p");
      meta.className = "profile-meta";
      var bits = [p.layout === "marketplace" ? "Marketplace" : "Plugin", p.kind === "git" ? "Git" : "Directory", p.location];
      if (p.plugins && p.plugins.length) bits.push("Plugins: " + p.plugins.join(", "));
      if (p.skills && p.skills.length) bits.push("Skills: " + p.skills.join(", "));
      if (p.rules && p.rules.length) bits.push("Rules: " + p.rules.join(", "));
      if (p.commands && p.commands.length) bits.push("Commands: " + p.commands.join(", "));
      if (p.agents && p.agents.length) bits.push("Agents: " + p.agents.join(", "));
      if (p.servers && p.servers.length) bits.push("MCP: " + p.servers.join(", "));
      if (p.notes && p.notes.length) bits.push(p.notes.join("; "));
      if (p.error) bits.push(p.error);
      meta.textContent = bits.join(" · ");
      li.appendChild(meta);
      if (p.agents && p.agents.length) {
        var starts = document.createElement("div");
        starts.className = "profile-row-actions";
        p.agents.forEach(function (name) {
          var start = document.createElement("button");
          start.type = "button";
          start.className = "ghost";
          start.textContent = "Start " + name;
          start.addEventListener("click", function () {
            if (!currentWorkspaceID) {
              showPluginErr("Open a workspace first");
              return;
            }
            fetch(apiURL("/api/plugins/agent"), {
              method: "POST",
              headers: headers(),
              body: JSON.stringify({ id: p.id, name: name, workspace_id: currentWorkspaceID }),
            })
              .then(readAPIResult)
              .then(function (j) {
                closeGate();
                currentAgentInstanceID = (j.instance && j.instance.id) || "";
                return afterAgentAttach();
              })
              .catch(function (err) { showPluginErr(String(err.message || err)); });
          });
          starts.appendChild(start);
        });
        li.appendChild(starts);
      }
      if (p.code && p.code.length) {
        var code = document.createElement("div");
        code.className = "plugin-code";
        p.code.forEach(function (entry) {
          var row = document.createElement("label");
          row.className = "plugin-code-row";
          if (!entry.granted) {
            var pick = document.createElement("input");
            pick.type = "checkbox";
            pick.checked = true;
            pick.setAttribute("data-code-name", entry.name);
            row.appendChild(pick);
          }
          var text = document.createElement("span");
          text.textContent = (entry.granted ? "Active" : "Off") + " · " + entry.kind + " " + entry.name + " · " + entry.command;
          row.appendChild(text);
          if (entry.granted) {
            var revoke = document.createElement("button");
            revoke.type = "button";
            revoke.className = "ghost";
            revoke.textContent = "Revoke";
            revoke.addEventListener("click", function () {
              fetch(apiURL("/api/plugins/code"), {
                method: "POST",
                headers: headers(),
                body: JSON.stringify({ id: p.id, revoke: [entry.name] }),
              })
                .then(readAPIResult)
                .then(loadPlugins)
                .catch(function (err) { showPluginErr(String(err.message || err)); });
            });
            row.appendChild(revoke);
          }
          code.appendChild(row);
        });
        if (code.querySelector("input[type=checkbox]")) {
          var activate = document.createElement("button");
          activate.type = "button";
          activate.textContent = "Activate running code";
          activate.addEventListener("click", function () {
            var names = [];
            code.querySelectorAll("input[type=checkbox]").forEach(function (box) {
              if (box.checked && box.getAttribute("data-code-name")) names.push(box.getAttribute("data-code-name"));
            });
            if (!names.length) return;
            fetch(apiURL("/api/plugins/code"), {
              method: "POST",
              headers: headers(),
              body: JSON.stringify({ id: p.id, grant: names }),
            })
              .then(readAPIResult)
              .then(loadPlugins)
              .catch(function (err) { showPluginErr(String(err.message || err)); });
          });
          code.appendChild(activate);
        }
        li.appendChild(code);
      }
      var actions = document.createElement("div");
      actions.className = "profile-row-actions";
      var on = document.createElement("label");
      var box = document.createElement("input");
      box.type = "checkbox";
      box.checked = !!p.enabled;
      box.addEventListener("change", function () {
        fetch(apiURL("/api/plugins/enabled"), { method: "POST", headers: headers(), body: JSON.stringify({ id: p.id, enabled: box.checked }) })
          .then(readAPIResult)
          .then(loadPlugins)
          .catch(function (err) { showPluginErr(String(err.message || err)); });
      });
      on.appendChild(box);
      on.appendChild(document.createTextNode(" Enabled"));
      actions.appendChild(on);
      if (p.kind === "git") {
        var refresh = document.createElement("button");
        refresh.type = "button";
        refresh.className = "ghost";
        refresh.textContent = "Refresh";
        refresh.addEventListener("click", function () {
          fetch(apiURL("/api/plugins/refresh"), { method: "POST", headers: headers(), body: JSON.stringify({ id: p.id }) })
            .then(readAPIResult)
            .then(loadPlugins)
            .catch(function (err) { showPluginErr(String(err.message || err)); });
        });
        actions.appendChild(refresh);
      }
      var remove = document.createElement("button");
      remove.type = "button";
      remove.className = "ghost";
      remove.textContent = "Remove";
      remove.addEventListener("click", function () {
        var label = p.name || p.location;
        askConfirm("Remove plugin " + label + "?", "Remove").then(function (yes) {
          if (!yes) return;
          fetch(apiURL("/api/plugins/remove"), { method: "POST", headers: headers(), body: JSON.stringify({ id: p.id }) })
            .then(readAPIResult)
            .then(loadPlugins)
            .catch(function (err) { showPluginErr(String(err.message || err)); });
        });
      });
      actions.appendChild(remove);
      li.appendChild(actions);
      ul.appendChild(li);
    });
  }

  function setSettingsTab(tab) {
    tab = tab || "connection";
    if (tab === "agent") tab = "models";
    settingsTabs.querySelectorAll(".settings-tab").forEach(function (btn) {
      var on = btn.getAttribute("data-tab") === tab;
      btn.classList.toggle("active", on);
      btn.setAttribute("aria-selected", on ? "true" : "false");
    });
    if (paneConnection) paneConnection.hidden = tab !== "connection";
    if (paneAgent) paneAgent.hidden = tab !== "models";
    if (paneSecrets) paneSecrets.hidden = tab !== "secrets";
    if (paneSkills) paneSkills.hidden = tab !== "skills";
    if (paneMcp) paneMcp.hidden = tab !== "mcp";
    if (panePlugins) panePlugins.hidden = tab !== "plugins";
    if (paneThemes) paneThemes.hidden = tab !== "themes";
    saveSettingsBtn.hidden = tab !== "models";
    connectBtn.hidden = tab !== "connection";
    if (tab === "secrets" && connected) loadCredentialsUI();
    if (tab === "plugins" && connected) loadPlugins();
    if (tab === "themes") renderThemeList();
  }

  function applyVaultPayload(j) {
    var vault = (j && j.vault) || {};
    vaultUnlocked = !!vault.unlocked;
    credentialsCache = (j && j.credentials) || [];
    if (vaultStatusEl) {
      if (!vault.exists) vaultStatusEl.textContent = "No vault yet — create one to store secrets.";
      else if (!vault.unlocked) vaultStatusEl.textContent = "Vault locked (unlock lasts for this process).";
      else vaultStatusEl.textContent = "Vault unlocked · " + (vault.count || credentialsCache.length) + " credential(s).";
    }
    if (vaultUnlockEl) vaultUnlockEl.hidden = vaultUnlocked;
    if (vaultBodyEl) vaultBodyEl.hidden = !vaultUnlocked;
    if (vaultCreateBtn) vaultCreateBtn.hidden = !!vault.exists || vaultUnlocked;
    if (vaultUnlockBtn) vaultUnlockBtn.hidden = !vault.exists || vaultUnlocked;
    renderCredentialList();
  }

  function loadCredentialsUI() {
    if (!connected) return Promise.resolve();
    return fetch(apiURL("/api/credentials"), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.message) || ("HTTP " + r.status));
          applyVaultPayload(j);
        });
      })
      .catch(function (e) {
        if (vaultStatusEl) vaultStatusEl.textContent = String(e);
      });
  }

  function unlockVault(create) {
    var pw = vaultPasswordEl ? vaultPasswordEl.value : "";
    if (!pw) {
      gateErr.hidden = false;
      gateErr.textContent = "Master password required";
      return;
    }
    fetch(apiURL("/api/credentials/unlock"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ password: pw, create: !!create }),
    })
      .then(function (r) {
        return r.text().then(function (t) {
          var j = {};
          try { j = JSON.parse(t); } catch (e) {}
          if (!r.ok) throw new Error((j && j.message) || t || ("HTTP " + r.status));
          if (vaultPasswordEl) vaultPasswordEl.value = "";
          applyVaultPayload(j);
        });
      })
      .catch(function (e) {
        gateErr.hidden = false;
        gateErr.style.color = "";
        gateErr.textContent = String(e);
      });
  }

  function renderCredentialList() {
    if (!credentialListEl) return;
    credentialListEl.innerHTML = "";
    if (!credentialsCache.length) {
      var empty = document.createElement("li");
      empty.className = "hint";
      empty.textContent = "No credentials yet";
      credentialListEl.appendChild(empty);
      return;
    }
    credentialsCache.forEach(function (c) {
      var li = document.createElement("li");
      var pick = document.createElement("button");
      pick.type = "button";
      pick.className = "profile-pick";
      pick.innerHTML = esc(c.name || c.id) +
        '<span class="profile-meta">' + esc((c.type || "") + (c.service ? " · " + c.service : "")) + "</span>";
      pick.addEventListener("click", function () { openCredentialEditor(c); });
      var edit = document.createElement("button");
      edit.type = "button";
      edit.className = "ghost profile-edit-btn";
      edit.textContent = "Edit";
      edit.addEventListener("click", function () { openCredentialEditor(c); });
      li.appendChild(pick);
      li.appendChild(edit);
      credentialListEl.appendChild(li);
    });
  }

  function openCredentialEditor(c) {
    editingCredentialID = c ? c.id : "";
    if (credentialEditor) credentialEditor.hidden = false;
    if (credEditName) credEditName.value = c ? (c.name || "") : "";
    if (credEditDescription) credEditDescription.value = c ? (c.description || "") : "";
    if (credEditType) credEditType.value = (c && c.type) || "password";
    if (credEditService) credEditService.value = c ? (c.service || "") : "";
    if (credEditTools) credEditTools.value = c && c.allowed_tools ? c.allowed_tools.join(", ") : "";
    if (credEditInject) credEditInject.value = c ? (c.inject_arg || "") : "";
    if (credEditInjectEnv) credEditInjectEnv.value = c ? (c.inject_env || "") : "";
    if (credEditSecret) {
      credEditSecret.value = "";
      credEditSecret.placeholder = c && c.secret_set ? "Set — leave blank to keep" : "Secret value (or PEM for ssh_key)";
    }
    if (credEditDelete) credEditDelete.hidden = !c;
    if (credEditHint) {
      credEditHint.textContent = "For exec use {{keimling:<id>}} in commands; inject_env is the vault-defined env name. ssh_key uses an ephemeral ssh-agent. MCP uses inject arg.";
    }
  }

  function closeCredentialEditor() {
    editingCredentialID = null;
    if (credentialEditor) credentialEditor.hidden = true;
  }

  function saveCredentialEditor() {
    var tools = (credEditTools && credEditTools.value || "").split(",").map(function (s) { return s.trim(); }).filter(Boolean);
    var body = {
      name: credEditName ? credEditName.value.trim() : "",
      description: credEditDescription ? credEditDescription.value.trim() : "",
      type: credEditType ? credEditType.value : "password",
      service: credEditService ? credEditService.value.trim() : "",
      allowed_tools: tools,
      inject_arg: credEditInject ? credEditInject.value.trim() : "",
      inject_env: credEditInjectEnv ? credEditInjectEnv.value.trim() : "",
    };
    var secret = credEditSecret ? credEditSecret.value : "";
    var isNew = !editingCredentialID;
    if (isNew || secret) body.secret = secret;
    var url = apiURL(isNew ? "/api/credentials" : "/api/credentials/" + encodeURIComponent(editingCredentialID));
    var method = isNew ? "POST" : "PUT";
    fetch(url, { method: method, headers: headers(), body: JSON.stringify(body) })
      .then(function (r) {
        return r.text().then(function (t) {
          var j = {};
          try { j = JSON.parse(t); } catch (e) {}
          if (!r.ok) throw new Error((j && j.message) || t || ("HTTP " + r.status));
          closeCredentialEditor();
          return loadCredentialsUI();
        });
      })
      .catch(function (e) {
        gateErr.hidden = false;
        gateErr.textContent = String(e);
      });
  }

  function deleteCredentialEditor() {
    if (!editingCredentialID) return;
    askConfirm("Delete this credential?", "Delete").then(function (yes) {
      if (!yes) return;
    fetch(apiURL("/api/credentials/" + encodeURIComponent(editingCredentialID)), {
      method: "DELETE",
      headers: headers(),
    })
      .then(function (r) {
        if (!r.ok) return r.text().then(function (t) { throw new Error(t || ("HTTP " + r.status)); });
        closeCredentialEditor();
        return loadCredentialsUI();
      })
      .catch(function (e) {
        gateErr.hidden = false;
        gateErr.textContent = String(e);
      });
    });
  }

  function activeModelOf(p) {
    if (!p) return null;
    var models = p.models || [];
    var id = p.active_model_id || p.model || "";
    for (var i = 0; i < models.length; i++) {
      if (models[i].id === id) return models[i];
    }
    return models[0] || null;
  }

  function syncHiddenProviderFields() {
    var p = null;
    for (var i = 0; i < providersCache.length; i++) {
      if (providersCache[i].id === activeProviderID) { p = providersCache[i]; break; }
    }
    if (!p) return;
    var m = activeModelOf(p);
    setBaseURL.value = p.base_url || "";
    setModel.value = (m && m.id) || p.model || "";
    setReasoning.value = (m && m.reasoning) || p.reasoning || setReasoning.value || "high";
    setAPIKey.value = "";
    if (m && m.context_limit >= 1000 && setContextLimit) {
      setContextLimit.value = m.context_limit;
    }
  }

  function renderProviderList() {
    if (!providerListEl) return;
    providerListEl.innerHTML = "";
    providersCache.forEach(function (p) {
      var m = activeModelOf(p);
      var li = document.createElement("li");
      var pick = document.createElement("button");
      pick.type = "button";
      pick.className = "profile-pick" + (p.id === activeProviderID ? " active" : "");
      pick.innerHTML = esc(p.name || p.id) +
        '<span class="profile-meta">' + esc((m && (m.name || m.id)) || p.model || p.base_url || "") + "</span>";
      pick.addEventListener("click", function () {
        activeProviderID = p.id;
        syncHiddenProviderFields();
        renderProviderList();
      });
      var edit = document.createElement("button");
      edit.type = "button";
      edit.className = "ghost profile-edit-btn";
      edit.textContent = "Edit";
      edit.addEventListener("click", function () { openProviderEditor(p); });
      li.appendChild(pick);
      li.appendChild(edit);
      providerListEl.appendChild(li);
    });
  }

  function clearModelForm() {
    if (provModelID) provModelID.value = "";
    if (provModelName) provModelName.value = "";
    if (provModelContext) provModelContext.value = "";
    if (provModelReasoning) provModelReasoning.value = "";
    if (provModelPromptPrice) provModelPromptPrice.value = "";
    if (provModelCompletionPrice) provModelCompletionPrice.value = "";
    setModelSenses(null);
  }

  function fillModelForm(m) {
    if (!m) { clearModelForm(); return; }
    if (provModelID) provModelID.value = m.id || "";
    if (provModelName) provModelName.value = m.name || "";
    if (provModelContext) provModelContext.value = m.context_limit ? String(m.context_limit) : "";
    if (provModelReasoning) provModelReasoning.value = m.reasoning || "";
    if (provModelPromptPrice) provModelPromptPrice.value = m.prompt_price || "";
    if (provModelCompletionPrice) provModelCompletionPrice.value = m.completion_price || "";
    setModelSenses(m);
  }

  function closeModelForm() {
    modelFormMode = null;
    modelFormOriginalID = "";
    clearModelForm();
    if (provCatalogStatus) provCatalogStatus.hidden = true;
    if (provModelEditor) provModelEditor.hidden = true;
    if (provModelNew) provModelNew.hidden = false;
  }

  function openModelFormAdd() {
    modelFormMode = "add";
    modelFormOriginalID = "";
    clearModelForm();
    if (provModelEditorTitle) provModelEditorTitle.textContent = "Add model";
    if (provModelEditor) provModelEditor.hidden = false;
    if (provModelNew) provModelNew.hidden = true;
    if (provModelID) provModelID.focus();
  }

  function openModelFormEdit(m) {
    if (!m) return;
    modelFormMode = "edit";
    modelFormOriginalID = m.id;
    fillModelForm(m);
    if (provModelEditorTitle) provModelEditorTitle.textContent = "Edit model";
    if (provModelEditor) provModelEditor.hidden = false;
    if (provModelNew) provModelNew.hidden = true;
    if (provModelID) provModelID.focus();
  }

  function renderProvModelList() {
    if (!provModelList) return;
    provModelList.innerHTML = "";
    editingProviderModels.forEach(function (m) {
      var isDefault = m.id === editingActiveModelID;
      var li = document.createElement("li");
      var label = document.createElement("div");
      label.className = "profile-pick" + (isDefault ? " active" : "");
      var meta = [];
      if (isDefault) meta.push("default");
      if (m.context_limit) meta.push(m.context_limit + " ctx");
      if (m.reasoning) meta.push(m.reasoning);
      if (m.sees_images) meta.push(m.use_images === false ? "images off" : "images");
      if (m.hears_audio) meta.push(m.use_audio === false ? "audio off" : "audio");
      label.innerHTML = esc(m.name || m.id) +
        (meta.length ? '<span class="profile-meta">' + esc(meta.join(" · ")) + "</span>" : "");
      var actions = document.createElement("div");
      actions.className = "profile-row-actions";
      if (!isDefault) {
        var setDef = document.createElement("button");
        setDef.type = "button";
        setDef.className = "ghost profile-edit-btn";
        setDef.textContent = "Default";
        setDef.title = "Use as default model for this provider";
        setDef.addEventListener("click", function () {
          editingActiveModelID = m.id;
          renderProvModelList();
        });
        actions.appendChild(setDef);
      }
      var edit = document.createElement("button");
      edit.type = "button";
      edit.className = "ghost profile-edit-btn";
      edit.textContent = "Edit";
      edit.addEventListener("click", function () { openModelFormEdit(m); });
      actions.appendChild(edit);
      var del = document.createElement("button");
      del.type = "button";
      del.className = "ghost profile-edit-btn danger";
      del.textContent = "Remove";
      del.addEventListener("click", function () {
        if (modelFormMode === "edit" && modelFormOriginalID === m.id) closeModelForm();
        editingProviderModels = editingProviderModels.filter(function (x) { return x.id !== m.id; });
        if (editingActiveModelID === m.id) {
          editingActiveModelID = editingProviderModels[0] ? editingProviderModels[0].id : "";
        }
        renderProvModelList();
      });
      actions.appendChild(del);
      li.appendChild(label);
      li.appendChild(actions);
      provModelList.appendChild(li);
    });
  }

  function saveEditorModel() {
    var id = provModelID ? provModelID.value.trim() : "";
    if (!id) {
      gateErr.hidden = false;
      gateErr.textContent = "Model id required";
      return;
    }
    var ctxRaw = provModelContext ? provModelContext.value.trim() : "";
    var ctx = ctxRaw ? parseInt(ctxRaw, 10) : 0;
    if (ctxRaw && (!isFinite(ctx) || ctx < 0)) {
      gateErr.hidden = false;
      gateErr.textContent = "Invalid context limit";
      return;
    }
    var row = {
      id: id,
      name: provModelName ? provModelName.value.trim() : "",
      context_limit: ctx || 0,
      reasoning: provModelReasoning ? provModelReasoning.value : "",
      prompt_price: provModelPromptPrice ? provModelPromptPrice.value.trim() : "",
      completion_price: provModelCompletionPrice ? provModelCompletionPrice.value.trim() : "",
    };
    if (provModelImagesLabel && !provModelImagesLabel.hidden) {
      row.sees_images = true;
      row.use_images = !!(provModelImages && provModelImages.checked);
    }
    if (provModelAudioLabel && !provModelAudioLabel.hidden) {
      row.hears_audio = true;
      row.use_audio = !!(provModelAudio && provModelAudio.checked);
    }
    if (modelFormMode === "edit") {
      var replaced = false;
      for (var i = 0; i < editingProviderModels.length; i++) {
        if (editingProviderModels[i].id === modelFormOriginalID) {
          // Block rename onto another existing id.
          if (id !== modelFormOriginalID && editingProviderModels.some(function (x) { return x.id === id; })) {
            gateErr.hidden = false;
            gateErr.textContent = "Another model already uses that id";
            return;
          }
          editingProviderModels[i] = row;
          if (editingActiveModelID === modelFormOriginalID) editingActiveModelID = id;
          replaced = true;
          break;
        }
      }
      if (!replaced) editingProviderModels.push(row);
    } else {
      if (editingProviderModels.some(function (x) { return x.id === id; })) {
        gateErr.hidden = false;
        gateErr.textContent = "Model id already in the list";
        return;
      }
      editingProviderModels.push(row);
      if (!editingActiveModelID) editingActiveModelID = id;
    }
    gateErr.hidden = true;
    closeModelForm();
    renderProvModelList();
  }

  function applyCatalogPrefill(entry) {
    if (!entry) return;
    if (provModelID) provModelID.value = entry.id || "";
    if (provModelName && entry.name) provModelName.value = entry.name;
    if (provModelContext && entry.context_limit) provModelContext.value = String(entry.context_limit);
    if (provModelPromptPrice && entry.prompt_price) provModelPromptPrice.value = entry.prompt_price;
    if (provModelCompletionPrice && entry.completion_price) provModelCompletionPrice.value = entry.completion_price;
    setModelSenses(entry.sees_images || entry.hears_audio ? {
      sees_images: !!entry.sees_images,
      hears_audio: !!entry.hears_audio,
      use_images: true,
      use_audio: true,
    } : null);
  }

  function setModelSenses(m) {
    var images = !!(m && m.sees_images);
    var audio = !!(m && m.hears_audio);
    if (provModelImagesLabel) provModelImagesLabel.hidden = !images;
    if (provModelImagesHint) provModelImagesHint.hidden = !images;
    if (provModelAudioLabel) provModelAudioLabel.hidden = !audio;
    if (provModelAudioHint) provModelAudioHint.hidden = !audio;
    if (provModelImages) provModelImages.checked = images && m.use_images !== false;
    if (provModelAudio) provModelAudio.checked = audio && m.use_audio !== false;
  }

  function fillCatalogDatalist(models) {
    catalogCache = models || [];
    if (!provCatalogDatalist) return;
    provCatalogDatalist.innerHTML = "";
    catalogCache.forEach(function (m) {
      var opt = document.createElement("option");
      opt.value = m.id;
      opt.label = m.name ? (m.name + " (" + m.id + ")") : m.id;
      provCatalogDatalist.appendChild(opt);
    });
  }

  function fetchModelCatalog() {
    if (!connected) return;
    if (provCatalogStatus) {
      provCatalogStatus.hidden = false;
      provCatalogStatus.textContent = "Fetching catalog…";
    }
    var base = provEditBaseURL ? provEditBaseURL.value.trim() : "";
    if (!base) {
      if (provCatalogStatus) provCatalogStatus.textContent = "Base URL required";
      return;
    }
    var body = { base_url: base };
    var key = provEditAPIKey ? provEditAPIKey.value.trim() : "";
    if (key) body.api_key = key;
    if (editingProviderID) body.provider_id = editingProviderID;
    fetch(apiURL("/api/settings/model-catalog"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify(body),
    })
      .then(function (r) {
        return r.text().then(function (t) {
          var j = {};
          try { j = JSON.parse(t); } catch (e) {}
          if (!r.ok) throw new Error((j && j.message) || t || ("HTTP " + r.status));
          fillCatalogDatalist(j.models || []);
          if (provCatalogStatus) {
            provCatalogStatus.textContent = (j.count || 0) + " models loaded — pick an id to prefill.";
          }
          var typed = provModelID ? provModelID.value.trim() : "";
          if (typed) {
            for (var i = 0; i < catalogCache.length; i++) {
              if (catalogCache[i].id === typed) {
                applyCatalogPrefill(catalogCache[i]);
                break;
              }
            }
          }
        });
      })
      .catch(function (e) {
        if (provCatalogStatus) {
          provCatalogStatus.hidden = false;
          provCatalogStatus.textContent = String(e);
        }
      });
  }

  function openProviderEditor(p) {
    editingProviderID = p ? p.id : "";
    editingProviderModels = ((p && p.models) || []).map(function (m) {
      return {
        id: m.id,
        name: m.name || "",
        context_limit: m.context_limit || 0,
        reasoning: m.reasoning || "",
        prompt_price: m.prompt_price || "",
        completion_price: m.completion_price || "",
        sees_images: !!m.sees_images,
        hears_audio: !!m.hears_audio,
        use_images: m.use_images !== false,
        use_audio: m.use_audio !== false,
      };
    });
    if (!editingProviderModels.length && p && p.model) {
      editingProviderModels = [{
        id: p.model,
        name: "",
        context_limit: 0,
        reasoning: p.reasoning || "",
        prompt_price: "",
        completion_price: "",
      }];
    }
    editingActiveModelID = (p && p.active_model_id) || (editingProviderModels[0] && editingProviderModels[0].id) || "";
    catalogCache = [];
    fillCatalogDatalist([]);
    if (provCatalogStatus) provCatalogStatus.hidden = true;
    if (providerEditor) providerEditor.hidden = false;
    if (provEditName) provEditName.value = p ? (p.name || "") : "";
    if (provEditBaseURL) provEditBaseURL.value = p ? (p.base_url || "") : "";
    if (provEditKind) provEditKind.value = (p && p.kind) || "auto";
    if (provEditAPIKey) {
      provEditAPIKey.value = "";
      provEditAPIKey.placeholder = p && p.api_key_set ? "Set — leave blank to keep" : "API key";
    }
    if (apiKeyHint) {
      apiKeyHint.textContent = p && p.api_key_set
        ? "Key is set. Leave blank to keep it, or paste a new key."
        : "Required for OpenRouter and most remote APIs.";
    }
    if (provEditDelete) provEditDelete.hidden = !p;
    closeModelForm();
    renderProvModelList();
  }

  function closeProviderEditor() {
    editingProviderID = null;
    editingProviderModels = [];
    editingActiveModelID = "";
    closeModelForm();
    if (providerEditor) providerEditor.hidden = true;
  }

  function saveProviderEditor() {
    if (modelFormMode && provModelEditor && !provModelEditor.hidden) {
      saveEditorModel();
      if (modelFormMode) return;
    }
    var name = provEditName ? provEditName.value.trim() : "";
    var base = provEditBaseURL ? provEditBaseURL.value.trim() : "";
    var kind = provEditKind ? provEditKind.value : "auto";
    var key = provEditAPIKey ? provEditAPIKey.value.trim() : "";
    if (!base) {
      gateErr.hidden = false;
      gateErr.textContent = "Base URL required";
      return;
    }
    if (!editingProviderModels.length) {
      gateErr.hidden = false;
      gateErr.textContent = "Add at least one model";
      return;
    }
    if (!editingActiveModelID || !editingProviderModels.some(function (m) { return m.id === editingActiveModelID; })) {
      editingActiveModelID = editingProviderModels[0].id;
    }
    var activeM = null;
    for (var i = 0; i < editingProviderModels.length; i++) {
      if (editingProviderModels[i].id === editingActiveModelID) { activeM = editingProviderModels[i]; break; }
    }
    var row = {
      name: name || base,
      base_url: base,
      kind: kind || "auto",
      models: editingProviderModels.slice(),
      active_model_id: editingActiveModelID,
      model: activeM ? activeM.id : "",
      reasoning: (activeM && activeM.reasoning) || "",
    };
    if (!editingProviderID) {
      row.id = "provider-" + Date.now();
      if (key) { row.api_key = key; row.api_key_set = true; }
      else { row.api_key_set = false; }
      providersCache.push(row);
      activeProviderID = row.id;
    } else {
      for (var j = 0; j < providersCache.length; j++) {
        if (providersCache[j].id === editingProviderID) {
          row.id = editingProviderID;
          row.api_key_set = providersCache[j].api_key_set;
          if (key) { row.api_key = key; row.api_key_set = true; }
          else if (providersCache[j].api_key) row.api_key = providersCache[j].api_key;
          providersCache[j] = row;
          break;
        }
      }
    }
    closeProviderEditor();
    syncHiddenProviderFields();
    renderProviderList();
  }

  function deleteProviderEditor() {
    if (!editingProviderID) return;
    var current = null;
    for (var i = 0; i < providersCache.length; i++) {
      if (providersCache[i].id === editingProviderID) { current = providersCache[i]; break; }
    }
    var name = (current && current.name) || "this provider";
    var last = providersCache.length <= 1;
    var ask = last
      ? "Remove “" + name + "”? Chat cannot call a model until you add one. Save settings to keep this. The saved API key is removed with it."
      : "Remove “" + name + "”? Save settings to keep this. The saved API key is removed with it.";
    askConfirm(ask, "Remove").then(function (yes) {
      if (!yes) return;
      providersCache = providersCache.filter(function (p) { return p.id !== editingProviderID; });
      if (activeProviderID === editingProviderID) activeProviderID = providersCache[0] ? providersCache[0].id : "";
      closeProviderEditor();
      syncHiddenProviderFields();
      renderProviderList();
    });
  }

  function fillVoiceAndSearch(j) {
    j = j || {};
    if (setVoiceProvider) {
      var current = setVoiceProvider.value;
      setVoiceProvider.innerHTML = '<option value="">Off</option>';
      providersCache.forEach(function (p) {
        var opt = document.createElement("option");
        opt.value = p.id;
        opt.textContent = p.name || p.id;
        setVoiceProvider.appendChild(opt);
      });
      setVoiceProvider.value = j.voice_provider_id || current || "";
      if (setVoiceProvider.value !== (j.voice_provider_id || "")) setVoiceProvider.value = j.voice_provider_id || "";
    }
    fillVoiceModels(j.voice_model || "");
    syncVoiceModelEnabled();
    if (setSearchKind) setSearchKind.value = j.search_kind || "";
    if (setSearchURL) setSearchURL.value = j.search_url || "";
    if (setSearchKey) {
      setSearchKey.value = "";
      setSearchKey.placeholder = j.search_api_key_set ? "Set — leave blank to keep" : "Not set";
    }
    if (searchKeyHint) {
      searchKeyHint.textContent = j.search_api_key_set
        ? "Key is set. Leave blank to keep it, or paste a new key."
        : "Required for Brave and OpenAI.";
    }
    syncSearchFields();
  }

  function fillRealtime(j) {
    j = j || {};
    if (setRealtimeProvider) {
      var current = setRealtimeProvider.value;
      setRealtimeProvider.innerHTML = '<option value="">Off</option>';
      providersCache.forEach(function (p) {
        var opt = document.createElement("option");
        opt.value = p.id;
        opt.textContent = p.name || p.id;
        setRealtimeProvider.appendChild(opt);
      });
      setRealtimeProvider.value = j.realtime_provider_id || current || "";
      if (setRealtimeProvider.value !== (j.realtime_provider_id || "")) setRealtimeProvider.value = j.realtime_provider_id || "";
    }
    if (setRealtimeModel) setRealtimeModel.value = j.realtime_model || "";
    if (setRealtimeVoice) setRealtimeVoice.value = j.realtime_voice || "marin";
    var on = !!(setRealtimeProvider && setRealtimeProvider.value);
    if (setRealtimeModel) setRealtimeModel.disabled = !on;
    if (setRealtimeVoice) setRealtimeVoice.disabled = !on;
    realtimeReady = !!(j.realtime_provider_id && j.realtime_model);
    setState(agentState);
  }

  function fillVoiceModels(selected) {
    if (!setVoiceModel) return;
    setVoiceModel.value = selected || "";
    syncVoiceModelEnabled();
  }

  function syncVoiceModelEnabled() {
    if (!setVoiceModel || !setVoiceProvider) return;
    setVoiceModel.disabled = !setVoiceProvider.value;
  }

  function syncSearchFields() {
    var kind = setSearchKind ? setSearchKind.value : "";
    if (searchURLLabel) searchURLLabel.hidden = kind === "" || kind === "openai";
    if (searchURLHint) searchURLHint.hidden = kind !== "url" && kind !== "brave";
    if (searchKeyLabel) searchKeyLabel.hidden = kind === "";
    if (searchKeyHint) searchKeyHint.hidden = kind === "";
    if (setSearchURL && kind === "brave" && !setSearchURL.placeholder) {
      setSearchURL.placeholder = "https://api.search.brave.com/res/v1/web/search";
    }
    if (kind === "brave" && setSearchURL) setSearchURL.placeholder = "https://api.search.brave.com/res/v1/web/search";
    if (kind === "url" && setSearchURL) setSearchURL.placeholder = "https://search.example/search?q={query}";
  }

  function openProviderSetup() {
    if (providerDialog) providerDialog.hidden = true;
    try { sessionStorage.setItem("keimling.provider_skip", "1"); } catch (e) { /* keep the banner */ }
    showGate({ manage: true, keepApp: true, tab: "models" });
    openProviderEditor(null);
  }

  function applyProviderGate(needs) {
    needsProvider = !!needs;
    if (providerBanner) providerBanner.hidden = !needsProvider;
    if (!providerDialog) return;
    var skipped = false;
    try { skipped = sessionStorage.getItem("keimling.provider_skip") === "1"; } catch (e) { skipped = false; }
    if (!needsProvider) {
      try { sessionStorage.removeItem("keimling.provider_skip"); } catch (e) { /* ignore */ }
      providerDialog.hidden = true;
      return;
    }
    providerDialog.hidden = skipped;
  }

  function fillAgentSettings(j) {
    if (!j) return;
    agentSettingsLoaded = true;
    providersCache = (j.model_providers || []).map(function (p) {
      return {
        id: p.id,
        name: p.name,
        base_url: p.base_url,
        kind: p.kind || "auto",
        model: p.model,
        reasoning: p.reasoning || j.reasoning || "high",
        active_model_id: p.active_model_id || p.model || "",
        models: p.models || [],
        api_key_set: !!p.api_key_set,
      };
    });
    activeProviderID = j.active_model_provider_id || (providersCache[0] && providersCache[0].id) || "";
    setOpenAIAPIKey.value = "";
    if (openaiKeyHint) openaiKeyHint.hidden = true;
    setMode.value = j.mode || "auto";
    fillVoiceAndSearch(j);
    fillRealtime(j);
    setWeb.checked = !!j.web;
    setAutopilot.checked = !!j.autopilot;
    applyAutopilotSettingsFields(j);
    if (setAutopilotInstructionsUndo) setAutopilotInstructionsUndo.hidden = true;
    setMaxSteps.value = j.max_steps != null ? j.max_steps : 200;
    setContextLimit.value = j.context_limit != null ? j.context_limit : 128000;
    if (setVoiceModel && !setVoiceProvider) setVoiceModel.value = j.voice_model || "";
    syncHiddenProviderFields();
    renderProviderList();
  }

  function applyAutopilotSettingsFields(j) {
    autopilotSettingsCustom = !!j.autopilot_instructions_custom;
    autopilotSettingsBaseline = j.autopilot_instructions || "";
    if (setAutopilotInstructions) setAutopilotInstructions.value = autopilotSettingsBaseline;
    if (setAutopilotInstructionsErr) setAutopilotInstructionsErr.hidden = true;
    if (setAutopilotInstructionsStatus) {
      setAutopilotInstructionsStatus.textContent = autopilotSettingsCustom
        ? "Saved global instructions. Threads without their own text use this."
        : "Built-in instructions. Saving replaces them.";
    }
    if (setAutopilotInstructionsReset) setAutopilotInstructionsReset.disabled = !autopilotSettingsCustom;
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
    syncHiddenProviderFields();
    if (setWeb && setSearchKind && agentSettingsLoaded) setWeb.checked = setSearchKind.value !== "";
    var body = {
      mode: setMode.value,
      autopilot: setAutopilot.checked,
      max_steps: parseInt(setMaxSteps.value, 10),
      context_limit: parseInt(setContextLimit.value, 10),
    };
    if (agentSettingsLoaded) {
      body.web = setWeb ? setWeb.checked : false;
      body.voice_model = setVoiceModel ? setVoiceModel.value.trim() : "";
      body.voice_provider_id = setVoiceProvider ? setVoiceProvider.value : "";
      body.realtime_model = setRealtimeModel ? setRealtimeModel.value.trim() : "";
      body.realtime_provider_id = setRealtimeProvider ? setRealtimeProvider.value : "";
      body.realtime_voice = setRealtimeVoice ? setRealtimeVoice.value : "marin";
      body.search_kind = setSearchKind ? setSearchKind.value : "";
      body.search_url = setSearchURL ? setSearchURL.value.trim() : "";
      body.active_model_provider_id = activeProviderID;
      body.model_providers = providersCache.map(function (p) {
        var row = {
          id: p.id,
          name: p.name,
          base_url: p.base_url,
          kind: p.kind || "auto",
          model: p.model,
          reasoning: p.reasoning || "",
          active_model_id: p.active_model_id || p.model || "",
          models: (p.models || []).map(function (m) {
            var model = {
              id: m.id,
              name: m.name || "",
              context_limit: m.context_limit || 0,
              reasoning: m.reasoning || "",
              prompt_price: m.prompt_price || "",
              completion_price: m.completion_price || "",
            };
            if (m.sees_images) {
              model.sees_images = true;
              model.use_images = m.use_images !== false;
            }
            if (m.hears_audio) {
              model.hears_audio = true;
              model.use_audio = m.use_audio !== false;
            }
            return model;
          }),
        };
        if (p.api_key) row.api_key = p.api_key;
        return row;
      });
      var searchKey = setSearchKey ? setSearchKey.value.trim() : "";
      if (searchKey) body.search_api_key = searchKey;
    }
    var openaiKey = setOpenAIAPIKey ? setOpenAIAPIKey.value.trim() : "";
    if (openaiKey) body.openai_api_key = openaiKey;
    if (setAutopilotInstructions) {
      var pilotText = setAutopilotInstructions.value;
      if (!pilotText.trim()) {
        if (setAutopilotInstructionsErr) {
          setAutopilotInstructionsErr.hidden = false;
          setAutopilotInstructionsErr.textContent = "Autopilot instructions cannot be empty. Use Reset to go back to the built-in prompt.";
        }
        return;
      }
      if (autopilotSettingsCustom || pilotText !== autopilotSettingsBaseline) {
        body.autopilot_instructions = pilotText;
      }
    }
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
          // Clear one-shot keys from cache after successful save.
          providersCache.forEach(function (p) { delete p.api_key; });
          fillAgentSettings(j);
          applyProviderGate(!!j.needs_provider);
          if (typeof loadSession === "function") loadSession();
          loadThreadModel();
          modeEl.textContent = j.mode || "—";
          refreshBudget(true);
          closeGate();
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

  function showLogin(error) {
    gate.classList.add("login");
    if (loginPanel) loginPanel.hidden = false;
    gateTitle.textContent = "Keimling";
    gateCopy.textContent = "Enter the connection secret to open this server.";
    gateErr.hidden = !error;
    gateErr.style.color = "";
    gateErr.textContent = error || "";
    settingsTabs.hidden = true;
    gateCancelBtn.hidden = true;
    if (gateCloseBtn) gateCloseBtn.hidden = true;
    gate.hidden = false;
    app.hidden = true;
    if (loginContinue) loginContinue.disabled = false;
    if (loginSecretEl) loginSecretEl.focus();
  }

  function showGate(opts) {
    opts = opts || {};
    gate.classList.remove("login");
    if (loginPanel) loginPanel.hidden = true;
    var prefs = loadPrefs();
    var first = !prefs.seen && !opts.manage;
    gateTitle.textContent = opts.manage ? "Settings" : first ? "Welcome" : "Connect";
    gateCopy.textContent = opts.manage
      ? "My connections (machine), bookmarks on this server, and Models."
      : first
        ? "Pick Currently connected or a saved remote, then Continue. Models live under Settings after connect."
        : "Select current session or a saved connection, then Continue.";
    gateErr.hidden = !opts.error;
    gateErr.style.color = "";
    gateErr.textContent = opts.error || "";
    settingsTabs.hidden = !opts.manage;
    gateManage = !!opts.manage && (!!opts.keepApp || connected);
    gateCancelBtn.hidden = !gateManage;
    if (gateCloseBtn) gateCloseBtn.hidden = !gateManage;
    connectTarget = { source: "current" };
    loadConnectionsFromPrefs().then(function () {
      if (opts.prefillSecret) secretEl.value = opts.prefillSecret;
      else if (prefs.secret && sameOriginLocal() && !secretEl.value) secretEl.value = prefs.secret;
      applyConnectTarget();
      renderAllConnectionLists();
    });
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
    gate.classList.remove("login");
    if (loginPanel) loginPanel.hidden = true;
    gate.hidden = true;
    gateManage = false;
    gateCancelBtn.hidden = true;
    if (gateCloseBtn) gateCloseBtn.hidden = true;
    app.hidden = false;
  }

  function closeGate() {
    if (!connected) return;
    gateErr.hidden = true;
    gateErr.style.color = "";
    showApp();
    msg.focus();
  }

  function fmtSideDur(ms) {
    var s = Math.max(0, Math.floor(ms / 1000));
    var h = Math.floor(s / 3600);
    s = s % 3600;
    var m = Math.floor(s / 60);
    s = s % 60;
    if (h > 0) return h + "h " + m + "m";
    if (m > 0) return m + "m " + s + "s";
    return s + "s";
  }

  function sideWaitingOnModel() {
    return agentState === "running" || agentState === "compacting";
  }

  function normalizeSideRate(rate) {
    if (!(rate > 0) || !isFinite(rate)) {
      sideSpike = 0;
      return 0;
    }
    var recent = 0;
    for (var i = 0; i < sideSamples.length; i++) {
      if (sideSamples[i] > recent) recent = sideSamples[i];
    }
    var limit = Math.max(SIDE_TYPICAL, recent) * 2.5;
    if (rate > limit) {
      var similar = sideSpike > 0 && rate <= sideSpike * 1.5 && sideSpike <= rate * 1.5;
      if (similar) {
        sideSpike = 0;
        return rate;
      }
      sideSpike = rate;
      return recent;
    }
    sideSpike = 0;
    return rate;
  }

  function pushSideSample() {
    if (!sideWaitingOnModel() || !(sideRate > 0)) return;
    var now = Date.now();
    if (now - sideSampleAt < 800) return;
    sideSampleAt = now;
    sideSamples.push(sideRate);
    if (sideSamples.length > SIDE_BAR_KEEP) sideSamples.shift();
  }

  function sideBarWidthPx() {
    if (sideBarWidthCached > 0) return sideBarWidthCached;
    var host = document.getElementById("side-dash") || document.body;
    var probe = document.createElement("span");
    probe.setAttribute("aria-hidden", "true");
    probe.textContent = "MM";
    probe.style.position = "absolute";
    probe.style.visibility = "hidden";
    probe.style.whiteSpace = "nowrap";
    var cs = getComputedStyle(host);
    probe.style.font = cs.font;
    probe.style.letterSpacing = cs.letterSpacing;
    host.appendChild(probe);
    var w = probe.getBoundingClientRect().width;
    probe.remove();
    if (!(w > 4)) w = 18;
    sideBarWidthCached = w;
    return w;
  }

  function sideBarsThatFit(host) {
    var cs = getComputedStyle(host);
    var pad = parseFloat(cs.paddingRight) || 0;
    var gap = parseFloat(cs.columnGap || cs.gap);
    if (!(gap >= 0)) gap = 2;
    var inner = host.clientWidth - pad;
    var bar = sideBarWidthPx();
    if (!(inner > 0) || !(bar > 0)) return 0;
    return Math.max(1, Math.floor((inner + gap) / (bar + gap)));
  }

  function renderSideBars() {
    var host = document.getElementById("side-bars");
    var mark = document.getElementById("side-mark");
    if (!host) return;
    var barW = sideBarWidthPx();
    host.style.setProperty("--side-bar-w", barW.toFixed(1) + "px");
    var meter = host.parentElement;
    if (meter && meter.dataset.watched !== "1" && typeof ResizeObserver === "function") {
      meter.dataset.watched = "1";
      new ResizeObserver(function () { renderSideBars(); }).observe(meter);
    }
    var n = sideBarsThatFit(host);
    if (!n) return;
    var scale = SIDE_TYPICAL;
    for (var i = 0; i < sideSamples.length; i++) {
      if (sideSamples[i] > scale) scale = sideSamples[i];
    }
    var start = Math.max(0, sideSamples.length - n);
    var html = "";
    for (var s = start; s < sideSamples.length; s++) {
      var v = sideSamples[s] || 0;
      var h = Math.max(0, Math.min(100, v / scale * 100));
      html += '<i style="height:' + h.toFixed(1) + '%"></i>';
    }
    host.innerHTML = html;
    if (mark) mark.style.top = ((1 - SIDE_TYPICAL / scale) * 100).toFixed(1) + "%";
  }

  function renderSidePerf() {
    var dash = document.getElementById("side-dash");
    var read = document.getElementById("side-dash-read");
    var waiting = sideWaitingOnModel();
    pushSideSample();
    var rate = waiting ? sideRate : 0;
    var state = "idle";
    var text = "Done";
    if (!waiting) {
      state = "idle";
      text = "Done";
    } else if (agentState === "compacting") {
      state = "thinking";
      text = "Compacting";
    } else if (rate > 0) {
      state = rate >= SIDE_TYPICAL ? "gas" : "flow";
      text = (rate >= 10 ? rate.toFixed(0) : rate.toFixed(1)) + " tok/s";
    } else if (sidePhase === "waiting" || sidePhase === "thinking" || !sideStepResponded) {
      state = "thinking";
      text = "Thinking · " + fmtSideDur(sideThinkStart ? Date.now() - sideThinkStart : 0);
    } else {
      state = "quiet";
      text = "Still working · " + fmtSideDur(sideLastModel ? Date.now() - sideLastModel : 0);
    }
    if (dash) dash.dataset.state = state;
    if (read) read.textContent = state === "idle" ? "" : text;
    var perfMark = document.getElementById("side-perf-mark");
    if (perfMark) {
      var glyph = "✓";
      if (state === "thinking" || state === "quiet") glyph = "◐";
      else if (state === "flow") glyph = "▸";
      else if (state === "gas") glyph = "▲";
      perfMark.textContent = glyph;
      perfMark.dataset.state = state;
    }
    renderSideBars();
  }

  function resetSideSamples() {
    sideSamples = [];
    sideSampleAt = 0;
    sideSpike = 0;
    sideClosed = false;
  }

  function closeSideRun() {
    if (sideClosed) return;
    sideClosed = true;
    var has = false;
    for (var i = 0; i < sideSamples.length; i++) {
      if (sideSamples[i] > 0) { has = true; break; }
    }
    if (!has) return;
    sideSamples.push(0);
    if (sideSamples.length > SIDE_BAR_KEEP) sideSamples.shift();
  }

  function noteSideClear(fromModel) {
    sideRate = 0;
    sidePhase = "idle";
    sideThinkStart = 0;
    if (fromModel) {
      sideLastModel = Date.now();
      sideStepResponded = true;
    }
    renderSidePerf();
  }

  function noteSideProgress(p) {
    p = p || {};
    var phase = p.phase || "waiting";
    var rate = Number(p.tok_per_s) || 0;
    var starting = sidePhase === "idle" || sideStepResponded;
    if (phase === "waiting" || phase === "thinking") {
      if (starting) {
        sideStepResponded = false;
        sideThinkStart = Date.now();
        sideRate = 0;
      } else if (!sideThinkStart) {
        sideThinkStart = Date.now();
      }
      sideRate = normalizeSideRate(rate);
    } else {
      sideStepResponded = true;
      sideThinkStart = 0;
      sideLastModel = Date.now();
      sideRate = normalizeSideRate(rate);
    }
    sidePhase = phase;
    renderSidePerf();
  }

  function clearProgress() {
    noteSideClear(false);
    if (!progressDetail) return;
    progressDetail.hidden = true;
    progressDetail.textContent = "";
    progressDetail.className = "progress-detail";
  }

  function updateProgress(p) {
    p = p || {};
    if (p.clear) {
      noteSideClear(true);
      if (!progressDetail) return;
      progressDetail.hidden = true;
      progressDetail.textContent = "";
      progressDetail.className = "progress-detail";
      return;
    }
    noteSideProgress(p);
    if (!progressDetail) return;
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

  function composerIsStopping() {
    if (currentAgentInstanceID) return !!stoppingAgents[currentAgentInstanceID];
    return !!stoppingThreadID && currentThreadID === stoppingThreadID;
  }

  function glyph(name) {
    var strokes = {
      plus: ["M12 5v14", "M5 12h14"],
      up: ["M12 19V6", "M7 11l5-5 5 5"],
      stop: ["M8 8h8v8H8z"],
      file: ["M7 3.5h7l4.5 4.5V20.5H7z", "M14 3.5v4.5h4.5"],
      screen: ["M4 5h16v11H4z", "M8 20h8", "M12 16v4"],
      mic: ["M9 4.5a3 3 0 0 1 6 0v6a3 3 0 0 1-6 0z", "M6.5 11a5.5 5.5 0 0 0 11 0", "M12 16.5V20"],
      "mic-off": ["M9 4.5a3 3 0 0 1 6 0v6a3 3 0 0 1-6 0z", "M6.5 11a5.5 5.5 0 0 0 11 0", "M12 16.5V20", "M5 5l14 14"],
      cassette: ["M4 8h16v9H4z", "M8 12.5a1.6 1.6 0 1 0 3.2 0a1.6 1.6 0 1 0-3.2 0", "M12.8 12.5a1.6 1.6 0 1 0 3.2 0a1.6 1.6 0 1 0-3.2 0"],
      phone: ["M8 4h2.8l1.2 3.6-2 1.1a11 11 0 0 0 5.3 5.3l1.1-2 3.6 1.2V16a2 2 0 0 1-2 2A13 13 0 0 1 6 6a2 2 0 0 1 2-2z"]
    };
    var turn = name === "hangup" ? 135 : 0;
    strokes = strokes[turn ? "phone" : name] || ["M12 5v14", "M5 12h14"];
    var ns = "http://www.w3.org/2000/svg";
    var svg = document.createElementNS(ns, "svg");
    svg.setAttribute("viewBox", "0 0 24 24");
    svg.setAttribute("aria-hidden", "true");
    svg.setAttribute("focusable", "false");
    var parent = svg;
    if (turn) {
      parent = document.createElementNS(ns, "g");
      parent.setAttribute("transform", "rotate(" + turn + " 12 12)");
      svg.appendChild(parent);
    }
    strokes.forEach(function (d) {
      var p = document.createElementNS(ns, "path");
      p.setAttribute("d", d);
      p.setAttribute("fill", "none");
      p.setAttribute("stroke", "currentColor");
      p.setAttribute("stroke-width", "2");
      p.setAttribute("stroke-linecap", "round");
      p.setAttribute("stroke-linejoin", "round");
      parent.appendChild(p);
    });
    return svg;
  }

  function mountGlyphs(root) {
    (root || document).querySelectorAll("[data-icon]").forEach(function (el) {
      if (el.querySelector("svg")) return;
      var icon = glyph(el.getAttribute("data-icon"));
      el.insertBefore(icon, el.firstChild);
    });
  }

  function setGlyph(btn, name) {
    if (!btn) return;
    if (btn.getAttribute("data-icon") === name && btn.querySelector("svg")) return;
    btn.setAttribute("data-icon", name);
    var old = btn.querySelector("svg");
    var next = glyph(name);
    if (old) btn.replaceChild(next, old);
    else btn.insertBefore(next, btn.firstChild);
  }

  function setControlLabel(btn, label, title) {
    if (!btn) return;
    var cap = btn.querySelector(".btn-caption");
    if (cap) cap.textContent = label;
    btn.setAttribute("aria-label", label);
    btn.title = title || label;
    btn.classList.remove("show-label");
  }

  function setComposerMore(open) {
    if (!composerMoreMenu || !composerMoreBtn) return;
    composerMoreMenu.hidden = !open;
    composerMoreBtn.setAttribute("aria-expanded", open ? "true" : "false");
  }

  function placeComposerControls() {
    var col = document.querySelector(".composer-send-col");
    if (!composerMoreMenu || !col || !send) return;
    function park(btn, where) {
      if (!btn) return;
      btn.classList.toggle("icon-btn", where !== "menu");
      if (where === "menu") composerMoreMenu.appendChild(btn);
      else col.insertBefore(btn, send);
    }
    park(attachFileBtn, "menu");
    if (typeof screenBtn !== "undefined") park(screenBtn, "menu");
    park(voiceBtn, "col");
    park(callBtn, "col");
    if (callMuteBtn) park(callMuteBtn, "col");
    if (stopBtn) park(stopBtn, "col");
  }

  function fitComposer() {
    if (!msg) return;
    var style = window.getComputedStyle(msg);
    var line = parseFloat(style.lineHeight);
    if (!isFinite(line) || line <= 0) line = (parseFloat(style.fontSize) || 16) * 1.35;
    var pad = (parseFloat(style.paddingTop) || 0) + (parseFloat(style.paddingBottom) || 0);
    var max = Math.ceil(line * 7.5 + pad);
    msg.style.height = "auto";
    var need = msg.scrollHeight;
    var next = need > max ? max : need;
    var min = Math.ceil(line + pad);
    if (next < min) next = min;
    msg.style.height = next + "px";
    msg.style.overflowY = need > max + 1 ? "auto" : "hidden";
  }

  function setState(s) {
    var prev = agentState;
    agentState = s || "idle";
    var wasRun = prev === "running" || prev === "compacting" || prev === "awaiting_approval";
    if (wasRun && agentState === "idle") closeSideRun();
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
    // Allow typing/voice/sending while busy → server queues or steers (CLI-like).
    // While recording, Send/Queue = stop mic + transcribe + send/queue.
    var queueHere = busyish || backgroundBusy;
    send.disabled = !connected || needsProvider;
    msg.disabled = !connected || needsProvider;
    if (attachFileBtn) attachFileBtn.disabled = !connected || needsProvider;
    if (msg) msg.placeholder = needsProvider ? "Add a model provider before sending" : "Message Keimling…";
    if (modelEl) modelEl.disabled = !connected || needsProvider;
    voiceBtn.disabled = !connected || !voiceOK || callLive;
    voiceBtn.hidden = !voiceOK;
    if (callBtn) {
      callBtn.hidden = !realtimeReady;
      callBtn.disabled = !connected || !realtimeReady || recording;
      callBtn.classList.toggle("live", callLive);
      setControlLabel(callBtn, callLive ? "Hang up" : "Call", callLive ? "Hang up. The transcript stays in the chat." : "Live voice call");
      setGlyph(callBtn, callLive ? "hangup" : "phone");
    }
    if (screenBtn) screenBtn.disabled = !connected || screenBusy;
    if (callMuteBtn) {
      callMuteBtn.hidden = !callLive;
      setControlLabel(callMuteBtn, callMuted ? "Unmute" : "Mute", callMuted ? "Turn the microphone back on" : "Mute the microphone");
      setGlyph(callMuteBtn, callMuted ? "mic-off" : "mic");
    }
    newChatBtn.disabled = !connected || callLive;
    newChatBtn.title = callLive ? "Hang up the call first" : "New thread";
    if (contextCompactBtn) contextCompactBtn.disabled = !connected || busyish;
    if (send) {
      if (recording && queueHere) {
        setControlLabel(send, "Queue", "Stop recording, transcribe, and queue");
      } else if (recording) {
        setControlLabel(send, "Send", "Stop recording, transcribe, and send");
      } else if (queueHere) {
        setControlLabel(send, "Queue", backgroundBusy
          ? "Queue in this thread. It runs when this thread is free."
          : "Queue for when the agent is idle (or steer the current run)");
      } else {
        setControlLabel(send, "Send", "Send message");
      }
    }
    if (stopBtn) {
      var hereStopping = composerIsStopping();
      var workspaceStopping = !currentAgentInstanceID && !!stoppingThreadID && (busyish || backgroundBusy);
      var stopVisible = connected && (busyish || backgroundBusy || callLive);
      stopBtn.hidden = !stopVisible;
      stopBtn.disabled = !connected || !stopVisible || hereStopping || (workspaceStopping && !hereStopping);
      setControlLabel(stopBtn, hereStopping ? "Stopping…" : "Interrupt", hereStopping ? "This one is stopping" : (callLive && !busyish ? "Stop the current spoken reply" : "Interrupt the running agent"));
    }
    if (voiceBtn && recording) {
      setControlLabel(voiceBtn, "Done", busyish
        ? "Finish recording into the composer (Queue to enqueue)"
        : "Finish recording (Space = draft · Enter = send)");
    } else if (voiceBtn && !recording) {
      setControlLabel(voiceBtn, "Record", busyish
        ? "Record a message to queue or steer"
        : "Record (Space/Enter on empty · F1). While recording: Space = draft, Enter = send");
    }
    placeComposerControls();
    if (!waiting) hideApproval();
    if (currentAgentInstanceID && agentsCache.length) {
      var busyNow = agentState === "running" || agentState === "compacting";
      for (var ai = 0; ai < agentsCache.length; ai++) {
        if (agentsCache[ai].id === currentAgentInstanceID) {
          agentsCache[ai].busy = busyNow;
          agentsCache[ai].running = busyNow;
        }
      }
    }
    if (agentListEl) renderAgentList();
    if (threadListEl && currentThreadID) {
      threadsListState = agentState;
      renderThreadListFromCache();
    }
    if (workspaceList) renderWorkspaceList(workspaceSearch ? workspaceSearch.value : "");
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

  function closePaperMenu() {}

  function setWorkspaceMenuOpen(open) {
    if (!workspaceMenu || !workspaceToggle) return;
    workspaceMenu.hidden = !open;
    workspaceToggle.setAttribute("aria-expanded", open ? "true" : "false");
  }

  function workspaceNameShared(ws) {
    var name = ws.name || ws.id;
    var n = 0;
    workspacesCache.forEach(function (other) {
      if ((other.name || other.id) === name) n++;
    });
    return n > 1;
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
      var label = esc(ws.name || ws.id);
      if (workspaceNameShared(ws)) label += '<span class="ws-path">' + esc(ws.root || "") + "</span>";
      btn.innerHTML = label;
      var here = ws.id === currentWorkspaceID && !currentAgentInstanceID;
      if (callLive && !here) {
        btn.disabled = true;
        btn.title = "Hang up the call first";
      }
      btn.addEventListener("click", function () {
        switchWorkspace(ws.id);
      });
      li.appendChild(btn);
      workspaceList.appendChild(li);
    });
  }

  function renderWorkspaceManage() {
    var list = document.getElementById("workspace-manage-list");
    if (!list) return;
    list.innerHTML = "";
    workspacesCache.forEach(function (ws) {
      var li = document.createElement("li");
      li.className = "agent-manage-row";
      var name = document.createElement("div");
      name.className = "agent-manage-name";
      name.appendChild(document.createTextNode(ws.name || ws.id));
      var small = document.createElement("small");
      small.textContent = ws.root || "";
      name.appendChild(small);
      var actions = document.createElement("div");
      actions.className = "agent-manage-actions";
      if (ws.id === currentWorkspaceID) {
        var current = document.createElement("span");
        current.className = "meta";
        current.textContent = "Current";
        current.title = "Switch to another workspace before removing this one";
        actions.appendChild(current);
      } else if (!ws.builtin) {
        var remove = document.createElement("button");
        remove.type = "button";
        remove.className = "ghost";
        remove.textContent = "Remove";
        remove.addEventListener("click", function () { hideWorkspace(ws); });
        actions.appendChild(remove);
      }
      var edit = document.createElement("button");
      edit.type = "button";
      edit.className = "ghost";
      edit.textContent = "Edit";
      edit.addEventListener("click", function () { beginWorkspaceRename(name, actions, ws); });
      actions.appendChild(edit);
      li.appendChild(name);
      li.appendChild(actions);
      list.appendChild(li);
    });
    var undo = document.getElementById("workspace-undo");
    if (!undo) return;
    undo.hidden = !workspaceUndo;
    undo.textContent = "";
    if (!workspaceUndo) return;
    undo.appendChild(document.createTextNode("Removed “" + (workspaceUndo.name || "workspace") + "” from the list. "));
    var back = document.createElement("button");
    back.type = "button";
    back.className = "ghost";
    back.textContent = "Undo";
    back.addEventListener("click", function () {
      var remembered = workspaceUndo;
      workspaceUndo = null;
      fetch(apiURL("/api/workspaces/restore"), {
        method: "POST",
        headers: headers(),
        body: JSON.stringify({ id: remembered.id }),
      })
        .then(readAPIResult)
        .then(function () { return loadWorkspaces(); })
        .catch(function (err) {
          workspaceUndo = remembered;
          appendLive("\n[workspace] " + err + "\n");
        });
    });
    undo.appendChild(back);
  }

  function beginWorkspaceRename(nameEl, actions, ws) {
    var input = document.createElement("input");
    input.type = "text";
    input.value = ws.name || "";
    input.setAttribute("aria-label", "Workspace name");
    nameEl.textContent = "";
    nameEl.appendChild(input);
    var small = document.createElement("small");
    small.textContent = ws.root || "";
    nameEl.appendChild(small);
    actions.textContent = "";
    var cancel = document.createElement("button");
    cancel.type = "button";
    cancel.className = "ghost";
    cancel.textContent = "Cancel";
    var save = document.createElement("button");
    save.type = "button";
    save.textContent = "Save";
    cancel.addEventListener("click", function () { renderWorkspaceManage(); });
    save.addEventListener("click", function () {
      var next = input.value.trim();
      if (!next) return;
      fetch(apiURL("/api/workspaces/rename"), {
        method: "POST",
        headers: headers(),
        body: JSON.stringify({ id: ws.id, name: next }),
      })
        .then(readAPIResult)
        .then(function () { return loadWorkspaces(); })
        .catch(function (err) {
          appendLive("\n[workspace] " + err + "\n");
        });
    });
    actions.appendChild(cancel);
    actions.appendChild(save);
    input.focus();
    input.addEventListener("keydown", function (e) {
      if (e.key === "Enter") {
        e.preventDefault();
        save.click();
      } else if (e.key === "Escape") {
        e.preventDefault();
        e.stopPropagation();
        cancel.click();
      }
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
          if (workspaceToggle) workspaceToggle.title = (cur && cur.name) || "";
          renderWorkspaceList(workspaceSearch && workspaceSearch.value);
          if (workspaceDialog && workspaceDialog.open) renderWorkspaceManage();
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
        return Promise.all([loadWorkspaces(), loadThreads(), loadAgents()]);
      })
      .then(function () {
        if (msg) msg.focus();
      });
  }

  function afterAgentAttach() {
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
        return Promise.all([loadWorkspaces(), loadThreads(), loadAgents()]);
      })
      .then(function () {
        if (msg) msg.focus();
      });
  }

  function switchWorkspace(id, force) {
    if (!id || (!force && id === currentWorkspaceID && !currentAgentInstanceID)) {
      setWorkspaceMenuOpen(false);
      return;
    }
    fetch(apiURL("/api/workspaces/switch"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: id }),
    })
      .then(readAPIResult)
      .then(function (j) {
        currentAgentInstanceID = "";
        return afterWorkspaceAttach(j);
      })
      .catch(function (err) {
        appendLive("\n[workspace] " + err + "\n");
      });
  }

  function hideWorkspace(ws) {
    var name = (ws && (ws.name || ws.id)) || "this workspace";
    askConfirm("Remove “" + name + "” from the list?\n\nNo files are deleted. Add the folder again to put it back.", "Remove").then(function (yes) {
      if (!yes) return;
    fetch(apiURL("/api/workspaces/hide"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: ws.id }),
    })
      .then(readAPIResult)
      .then(function () {
        workspaceUndo = { id: ws.id, name: ws.name || name };
        return loadWorkspaces();
      })
      .catch(function (err) {
        appendLive("\n[workspace] " + err + "\n");
      });
    });
  }

  function openWorkspacePath(path, name) {
    path = String(path || "").trim();
    name = String(name || "").trim();
    if (!path) return;
    fetch(apiURL("/api/workspaces/open"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ path: path, name: name }),
    })
      .then(readAPIResult)
      .then(function (j) {
        if (workspacePath) workspacePath.value = "";
        if (workspaceNewName) workspaceNewName.value = "";
        if (workspaceDialog && workspaceDialog.open) workspaceDialog.close();
        return afterWorkspaceAttach(j);
      })
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
      var isActive = !currentAgentInstanceID && th.id === currentThreadID;
      var showRun = !currentAgentInstanceID && runningThreadID && th.id === runningThreadID;
      var showStop = stoppingThreadID && th.id === stoppingThreadID;
      row.className = "thread-item" + (isActive ? " active" : "");
      row.title = th.id;

      var open = document.createElement("button");
      open.type = "button";
      open.className = "thread-open";
      var st = document.createElement("span");
      st.className = "thread-status" + (showStop ? " stopping" : (showRun ? " running" : ""));
      if (showStop) st.title = "Stopping";
      var title = document.createElement("span");
      title.className = "thread-title";
      title.textContent = th.title || th.id;
      open.appendChild(st);
      open.appendChild(title);
      if (callLive && !isActive) {
        open.disabled = true;
        open.title = "Hang up the call first";
      }
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
    if (!currentThreadID && payload.current) currentThreadID = payload.current;
    threadsSearchActive = !!(payload.query && String(payload.query).trim());
    renderThreadListFromCache();
  }

  function loadThreads(query) {
    if (!connected) return Promise.resolve();
    var q = query == null
      ? (threadSearchEl ? String(threadSearchEl.value || "").trim() : "")
      : String(query || "").trim();
    var url = apiURL("/api/threads");
    var params = [];
    if (currentWorkspaceID) params.push("workspace=" + encodeURIComponent(currentWorkspaceID));
    if (q) params.push("q=" + encodeURIComponent(q));
    if (params.length) url += "?" + params.join("&");
    return fetch(url, { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          // While an agent is open, keep workspace threads listed but none marked active.
          if (currentAgentInstanceID && j && j.threads) {
            j.current = "";
            j.threads.forEach(function (th) { th.active = false; });
          }
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

  function eventsForThisView() {
    if (currentAgentInstanceID) return true;
    if (!runningThreadID || !currentThreadID) return true;
    return runningThreadID === currentThreadID;
  }

  function switchThread(id) {
    if (!id) return;
    if (!currentAgentInstanceID && id === currentThreadID) return;
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
        currentAgentInstanceID = "";
        currentThreadID = id;
        if (id !== "main") workspaceThreadID = id;
        if (es && typeof es.close === "function") {
          try { es.close(); } catch (e) {}
          es = null;
        }
        messages = [];
        chapters = [];
        activeChapter = null;
        clearLive();
        hideApproval();
        return loadSession();
      })
      .then(function () {
        connectEvents();
        return Promise.all([loadThreads(), loadAgents()]);
      })
      .catch(function (err) {
        appendLive("\n[threads] " + String((err && err.message) || err || "") + "\n");
      });
  }

  function deleteThread(id, label) {
    if (!id) return;
    var name = label || id;
    askConfirm("Delete thread “" + name + "” permanently?\n\nThis cannot be undone. The conversation and its local data will be removed.", "Delete").then(function (yes) {
      if (!yes) return;
    fetch(apiURL("/api/threads/delete"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: id }),
    })
      .then(readAPIResult)
      .then(function (j) {
        currentAgentInstanceID = "";
        currentThreadID = j.thread_id || currentThreadID;
        if (es && typeof es.close === "function") {
          try { es.close(); } catch (e) {}
          es = null;
        }
        return loadSession();
      })
      .then(function () {
        connectEvents();
        return Promise.all([loadThreads(), loadAgents()]);
      })
      .catch(function (err) {
        appendLive("\n[threads] " + err + "\n");
      });
    });
  }

  function renderAgentList() {
    if (!agentListEl) return;
    agentListEl.innerHTML = "";
    if (!agentsCache.length) {
      var empty = document.createElement("p");
      empty.className = "hint";
      empty.textContent = "No agents for this workspace";
      agentListEl.appendChild(empty);
      return;
    }
    agentsCache.forEach(function (ag) {
      var row = document.createElement("div");
      var rowStopping = !!stoppingAgents[ag.id];
      row.className = "thread-item" + (ag.id === currentAgentInstanceID || ag.attached ? " active" : "") + (rowStopping ? " stopping" : "");
      row.title = ag.id;

      var open = document.createElement("button");
      open.type = "button";
      open.className = "thread-open";
      var st = document.createElement("span");
      var busy = !!(ag.busy || ag.running);
      st.className = "thread-status" + (busy ? " running" : "");
      var title = document.createElement("span");
      title.className = "thread-title";
      title.textContent = ag.name || ag.id;
      open.appendChild(st);
      open.appendChild(title);
      if (callLive && ag.id !== currentAgentInstanceID) {
        open.disabled = true;
        open.title = "Hang up the call first";
      }
      open.addEventListener("click", function () {
        attachAgent(ag.id);
      });

      var actions = document.createElement("div");
      actions.className = "agent-actions";
      var stop = document.createElement("button");
      stop.type = "button";
      stop.textContent = rowStopping ? "Stopping…" : "Stop";
      stop.title = rowStopping ? "This agent is stopping" : "Stop current run";
      stop.disabled = rowStopping;
      stop.addEventListener("click", function (e) {
        e.preventDefault();
        e.stopPropagation();
        stopAgent(ag.id);
      });
      actions.appendChild(stop);
      var reset = document.createElement("button");
      reset.type = "button";
      reset.textContent = "Reset";
      reset.title = "Clear this conversation. Contacts, tools, and watches stay.";
      reset.addEventListener("click", function (e) {
        e.preventDefault();
        e.stopPropagation();
        resetAgent(ag.id, ag.name || ag.id);
      });
      actions.appendChild(reset);

      row.appendChild(open);
      row.appendChild(actions);
      agentListEl.appendChild(row);
    });
  }

  function loadAgents(opts) {
    opts = opts || {};
    if (!connected) return Promise.resolve();
    if (opts.poll && agentsPollInFlight) return Promise.resolve();
    if (opts.poll) agentsPollInFlight = true;
    var url = apiURL("/api/agents/instances");
    if (currentWorkspaceID) url += "?workspace=" + encodeURIComponent(currentWorkspaceID);
    return fetch(url, { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          agentsCache = j.instances || [];
          agentsCache.forEach(function (ag) {
            if (ag.busy || ag.running) return;
            delete stoppingAgents[ag.id];
          });
          if (j.attached) currentAgentInstanceID = j.attached;
          else if (!j.attached && currentAgentInstanceID) {
            // Host reports no agent attached (e.g. after thread switch).
            currentAgentInstanceID = "";
          }
          renderAgentList();
          setState(agentState);
        });
      })
      .catch(function (err) {
        if (opts.poll) return; // quiet background refresh
        if (String(err).indexOf("404") >= 0 || String(err).indexOf("Not Found") >= 0) return;
        appendLive("\n[agents] " + err + "\n");
      })
      .finally(function () {
        if (opts.poll) agentsPollInFlight = false;
      });
  }

  function startAgentsPoll() {
    stopAgentsPoll();
    agentsPollTimer = setInterval(function () {
      if (!connected) return;
      loadAgents({ poll: true });
      if (sideTab === "agents") loadAgentBoard();
    }, 2000);
  }

  function stopAgentsPoll() {
    if (agentsPollTimer) {
      clearInterval(agentsPollTimer);
      agentsPollTimer = null;
    }
  }

  function attachAgent(id) {
    if (!id || id === currentAgentInstanceID) return;
    fetch(apiURL("/api/agents/instances/attach"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: id }),
    })
      .then(readAPIResult)
      .then(function () {
        currentAgentInstanceID = id;
        return afterAgentAttach();
      })
      .catch(function (err) {
        appendLive("\n[agents] " + err + "\n");
      });
  }

  var agentAccessTitle = document.getElementById("agent-access-title");
  var agentAccessWorkspaces = document.getElementById("agent-access-workspaces");
  var agentAccessTools = document.getElementById("agent-access-tools");
  var agentAccessPeers = document.getElementById("agent-access-peers");
  var agentAccessFollow = document.getElementById("agent-access-follow");
  var agentAccessStatus = document.getElementById("agent-access-status");
  var agentDefAccess = document.getElementById("agent-def-access");
  var agentAccessID = "";

  function markAgentAccessUnsaved() {
    if (!agentAccessStatus) return;
    agentAccessStatus.hidden = true;
    agentAccessStatus.textContent = "";
  }

  function applyAccessFollow() {
    var follow = !!(agentAccessFollow && agentAccessFollow.checked);
    if (!agentAccessTools) return;
    agentAccessTools.querySelectorAll("input[type=checkbox]").forEach(function (box) {
      var allowed = box.dataset.allowed !== "0";
      box.disabled = follow || !allowed;
    });
  }

  function fillAgentAccessList(el, rows) {
    if (!el) return;
    el.innerHTML = "";
    rows.forEach(function (row) {
      var label = document.createElement("label");
      label.className = "agent-access-row";
      var box = document.createElement("input");
      box.type = "checkbox";
      box.checked = !!row.granted;
      box.disabled = row.allowed === false;
      box.dataset.accessId = row.id || row.name;
      if (row.allowed === false) box.dataset.allowed = "0";
      box.addEventListener("change", markAgentAccessUnsaved);
      var text = document.createElement("span");
      text.textContent = row.label || row.name || row.id;
      label.appendChild(box);
      label.appendChild(text);
      el.appendChild(label);
    });
  }

  function checkedAccess(el, onlyEnabled) {
    var ids = [];
    if (!el) return ids;
    el.querySelectorAll("input[type=checkbox]").forEach(function (box) {
      if (onlyEnabled && box.disabled) return;
      if (box.checked) ids.push(box.dataset.accessId);
    });
    return ids;
  }

  function openAgentAccess(id, name) {
    if (!id) return;
    agentAccessID = id;
    if (agentAccessPeers) agentAccessPeers.dataset.ready = "";
    if (agentAccessTitle) agentAccessTitle.textContent = name || "Access";
    showAgentDefErr("");
    markAgentAccessUnsaved();
    var accessModel = document.getElementById("agent-access-model");
    if (accessModel) {
      accessModel.value = "";
      accessModel.dataset.ready = "";
    }
    setAgentDefPane("access");
    if (agentDefDialog && typeof agentDefDialog.showModal === "function" && !agentDefDialog.open) {
      agentDefDialog.showModal();
    }
    Promise.all([
      fetch(apiURL("/api/agents/instances/access?id=" + encodeURIComponent(id)), { headers: headers() })
        .then(function (r) {
          return r.json().then(function (j) {
            if (!r.ok) throw new Error((j && (j.error || j)) || r.status);
            return j;
          });
        }),
      loadMcpServers(),
    ])
      .then(function (result) {
        var j = result[0];
        if (agentAccessTitle) agentAccessTitle.textContent = (j.name || name || "Instance") + " — workspaces, tools, and contacts";
        fillAgentAccessList(agentAccessWorkspaces, j.workspaces || []);
        var peerRows = j.peers || [];
        if (!peerRows.length && agentAccessPeers) {
          agentAccessPeers.innerHTML = "";
          var empty = document.createElement("p");
          empty.className = "hint";
          empty.textContent = "No other agents.";
          agentAccessPeers.appendChild(empty);
          agentAccessPeers.dataset.ready = "1";
        } else {
          fillAgentAccessList(agentAccessPeers, peerRows.map(function (p) {
            return { id: p.id, label: p.name || p.id, granted: !!p.granted };
          }));
          if (agentAccessPeers) agentAccessPeers.dataset.ready = "1";
        }
        if (j.groups) agentToolGroups = j.groups;
        var states = {};
        (j.tools || []).forEach(function (t) { states[t.name] = t; });
        renderToolMatrix(agentAccessTools, { states: states, lock: !!j.follows, mcp: j.mcp || null });
        if (agentAccessFollow) agentAccessFollow.checked = !!j.follows;
        applyAccessFollow();
        var accessModel = document.getElementById("agent-access-model");
        var accessModelHint = document.getElementById("agent-access-model-hint");
        if (accessModel) {
          accessModel.value = j.model || "";
          accessModel.dataset.ready = "1";
          accessModel.oninput = markAgentAccessUnsaved;
        }
        if (accessModelHint) {
          var defModel = j.definition_model || "";
          accessModelHint.textContent = defModel
            ? "Empty uses the definition (" + defModel + "). A model here applies only to this instance."
            : "Empty uses the definition, which follows Settings. A model here applies only to this instance.";
        }
        loadModelSuggestions();
      })
      .catch(function (err) {
        showAgentDefErr(String(err.message || err));
      });
  }

  function saveAgentAccess() {
    if (!agentAccessID) return;
    var workspaces = checkedAccess(agentAccessWorkspaces, false);
    var follow = !!(agentAccessFollow && agentAccessFollow.checked);
    var tools = follow ? null : checkedToolNames(agentAccessTools);
    var body = { id: agentAccessID, workspaces: workspaces, tools: tools };
    var accessModel = document.getElementById("agent-access-model");
    if (accessModel && accessModel.dataset.ready === "1") {
      body.model = accessModel.value.trim();
    }
    if (agentAccessPeers && agentAccessPeers.dataset.ready === "1") {
      body.peers = checkedAccess(agentAccessPeers, false);
    }
    showAgentDefErr("");
    if (!workspaces.length) {
      showAgentDefErr("Choose at least one workspace.");
      return;
    }
    var saveBtn = document.getElementById("agent-access-save");
    if (saveBtn) saveBtn.disabled = true;
    fetch(apiURL("/api/agents/instances/access"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify(body),
    })
      .then(readAPIResult)
      .then(function () {
        if (agentAccessStatus) {
          agentAccessStatus.hidden = false;
          var modelNote = "";
          if (accessModel && accessModel.dataset.ready === "1") {
            modelNote = accessModel.value.trim()
              ? " Model: " + accessModel.value.trim() + "."
              : " Model follows the definition.";
          }
          agentAccessStatus.textContent = (follow ? "Saved. This instance uses the definition’s tools." : "Saved. This instance uses a narrower tool set.") + modelNote;
        }
        return Promise.all([loadAgents(), loadAllInstances()]).then(function () {
          if (currentAgentInstanceID === agentAccessID) return loadSession();
        });
      })
      .catch(function (err) {
        showAgentDefErr(String(err.message || err));
      })
      .finally(function () {
        if (saveBtn) saveBtn.disabled = false;
      });
  }

  function stopAgent(id) {
    if (!id || stoppingAgents[id]) return;
    stoppingAgents[id] = true;
    renderAgentList();
    setState(agentState);
    fetch(apiURL("/api/agents/instances/stop"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: id }),
    })
      .then(readAPIResult)
      .then(function (j) {
        if (j && j.running === false) {
          delete stoppingAgents[id];
        }
        return loadAgents();
      })
      .catch(function (err) {
        delete stoppingAgents[id];
        renderAgentList();
        setState(agentState);
        appendLive("\n[agents] " + err + "\n");
      });
  }

  function resetAgent(id, label) {
    if (!id) return;
    var name = label || id;
    askConfirm("Reset “" + name + "” to a fresh context?\n\nThe conversation and charter are cleared. The old transcript is kept as a backup file. Contacts, tools, and watches stay.", "Reset").then(function (yes) {
      if (!yes) return;
    fetch(apiURL("/api/agents/instances/reset"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: id }),
    })
      .then(readAPIResult)
      .then(function () {
        delete stoppingAgents[id];
        if (currentAgentInstanceID === id) return loadSession();
      })
      .then(function () { return loadAgents(); })
      .catch(function (err) {
        appendLive("\n[agents] " + err + "\n");
      });
    });
  }

  function deleteAgent(id, label) {
    if (!id) return;
    var name = label || id;
    askConfirm("Delete agent “" + name + "” permanently?\n\nThis removes the instance, its chat, and its watches. The definition stays.", "Delete").then(function (yes) {
      if (!yes) return;
    fetch(apiURL("/api/agents/instances/delete"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: id }),
    })
      .then(readAPIResult)
      .then(function () {
        if (currentAgentInstanceID === id) {
          currentAgentInstanceID = "";
          if (currentWorkspaceID) return switchWorkspace(currentWorkspaceID, true);
        }
        return Promise.all([loadAgents(), loadAllInstances()]);
      })
      .catch(function (err) {
        showAgentDefErr(String(err.message || err));
      });
    });
  }

  function showAgentDefErr(msg) {
    if (!agentDefErr) return;
    if (!msg) {
      agentDefErr.hidden = true;
      agentDefErr.textContent = "";
      return;
    }
    agentDefErr.hidden = false;
    agentDefErr.textContent = msg;
  }

  function setAgentDefPane(name) {
    if (agentDefPick) agentDefPick.hidden = name !== "pick";
    if (agentDefCreate) agentDefCreate.hidden = name !== "create";
    if (agentDefEdit) agentDefEdit.hidden = name !== "edit";
    if (agentDefInstance) agentDefInstance.hidden = name !== "instance";
    if (agentDefAccess) agentDefAccess.hidden = name !== "access";
    showAgentDefErr("");
  }

  function setManageTab(name) {
    manageTab = name || "definitions";
    var panes = {
      definitions: document.getElementById("agent-pane-definitions"),
      instances: document.getElementById("agent-pane-instances"),
      teams: document.getElementById("agent-pane-teams"),
    };
    Object.keys(panes).forEach(function (key) {
      if (panes[key]) panes[key].hidden = key !== manageTab;
      var tab = document.getElementById("agent-tab-" + key);
      if (tab) tab.setAttribute("aria-selected", key === manageTab ? "true" : "false");
    });
  }

  function manageAction(label, className, onClick) {
    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = className || "ghost";
    btn.textContent = label;
    btn.addEventListener("click", function (e) {
      e.preventDefault();
      onClick();
    });
    return btn;
  }

  function toolLabel(name) {
    for (var i = 0; i < agentToolCatalog.length; i++) {
      if (agentToolCatalog[i].name === name) return agentToolCatalog[i].label || name;
    }
    return name;
  }

  function capVisible(names, states) {
    return (names || []).some(function (name) {
      var st = states[name];
      return st && st.allowed !== false;
    });
  }

  function toolCapCell(names, granted, states, lock) {
    var td = document.createElement("td");
    var usable = [];
    var labels = [];
    var onNames = [];
    (names || []).forEach(function (name) {
      var st = states ? states[name] : null;
      if (states && (!st || st.allowed === false)) return;
      usable.push(name);
      labels.push((st && st.label) || toolLabel(name));
      var on = states ? !!(st && st.granted) : (granted == null || !!granted[name]);
      if (on) onNames.push(name);
    });
    if (!usable.length) {
      td.className = "agent-cap-empty";
      td.textContent = "—";
      return td;
    }
    var label = document.createElement("label");
    label.className = "agent-cap";
    var box = document.createElement("input");
    box.type = "checkbox";
    box.dataset.toolNames = usable.join(",");
    var all = onNames.length === usable.length;
    box.checked = all;
    box.indeterminate = !all && onNames.length > 0;
    if (box.indeterminate) box.dataset.partialNames = onNames.join(",");
    if (lock) box.disabled = true;
    box.addEventListener("change", function () {
      box.indeterminate = false;
      box.removeAttribute("data-partial-names");
      markAgentAccessUnsaved();
    });
    var text = document.createElement("span");
    var mixed = onNames.length > 0 && onNames.length < usable.length;
    usable.forEach(function (name, i) {
      if (i) text.appendChild(document.createTextNode(", "));
      var bit = document.createElement("span");
      bit.textContent = labels[i];
      if (mixed && onNames.indexOf(name) < 0) bit.className = "agent-cap-off";
      text.appendChild(bit);
    });
    label.appendChild(box);
    label.appendChild(text);
    td.appendChild(label);
    return td;
  }

  function renderToolMatrix(el, opts) {
    if (!el) return;
    opts = opts || {};
    el.innerHTML = "";
    var groups = agentToolGroups;
    if (!groups.length && (agentToolCatalog || []).length) {
      groups = agentToolCatalog.map(function (tool) {
        return { id: tool.name, label: tool.label || tool.name, read: [], write: [tool.name] };
      });
    }
    var granted = null;
    if (Array.isArray(opts.granted)) {
      granted = {};
      opts.granted.forEach(function (name) { granted[name] = true; });
    }
    var table = document.createElement("table");
    table.className = "agent-tool-table";
    var head = document.createElement("thead");
    var hr = document.createElement("tr");
    ["Topic", "Read", "Write"].forEach(function (title) {
      var th = document.createElement("th");
      th.scope = "col";
      th.textContent = title;
      hr.appendChild(th);
    });
    head.appendChild(hr);
    var body = document.createElement("tbody");
    groups.forEach(function (g) {
      if (opts.states && !capVisible(g.read, opts.states) && !capVisible(g.write, opts.states)) return;
      var tr = document.createElement("tr");
      var topic = document.createElement("th");
      topic.scope = "row";
      topic.textContent = g.label || g.id;
      tr.appendChild(topic);
      tr.appendChild(toolCapCell(g.read, granted, opts.states, opts.lock));
      tr.appendChild(toolCapCell(g.write, granted, opts.states, opts.lock));
      body.appendChild(tr);
    });
    table.appendChild(head);
    table.appendChild(body);
    el.appendChild(table);
    renderMCPServers(el, opts);
  }

  function mcpShortTool(exposed, grant) {
    var prefix = (grant || "") + "__";
    if (grant && exposed.indexOf(prefix) === 0) return exposed.slice(prefix.length);
    var parts = String(exposed || "").split("__");
    return parts.length >= 3 ? parts.slice(2).join("__") : exposed;
  }

  function serverChoiceOn(grant, opts) {
    if (opts.mcp) {
      if (opts.mcp.granted_all) return true;
      return (opts.mcp.granted || []).indexOf(grant) >= 0;
    }
    var names = opts.granted;
    if (!Array.isArray(names)) return true;
    if (names.indexOf("mcp") >= 0) return true;
    return names.indexOf(grant) >= 0;
  }

  function serverChoiceAllowed(grant, opts) {
    if (!opts.mcp) return true;
    if (opts.mcp.allowed_all) return true;
    return (opts.mcp.allowed || []).indexOf(grant) >= 0;
  }

  function renderMCPServers(el, opts) {
    var block = document.createElement("section");
    block.className = "agent-mcp-block";
    var title = document.createElement("h3");
    title.className = "settings-section-title";
    title.textContent = "MCP servers";
    block.appendChild(title);
    var hint = document.createElement("p");
    hint.className = "hint";
    hint.textContent = "A server is on or off as a whole. The names are that server’s tools.";
    block.appendChild(hint);
    var known = mcpServersCache || [];
    var servers = known.filter(function (s) {
      return serverChoiceAllowed(s.grant || "", opts);
    });
    if (!servers.length) {
      var empty = document.createElement("p");
      empty.className = "hint";
      empty.textContent = known.length ? "This definition has no MCP servers." : "No MCP servers.";
      block.appendChild(empty);
      el.appendChild(block);
      return;
    }
    var table = document.createElement("table");
    table.className = "agent-tool-table";
    var head = document.createElement("thead");
    var hr = document.createElement("tr");
    ["Server", "Tools"].forEach(function (label) {
      var th = document.createElement("th");
      th.scope = "col";
      th.textContent = label;
      hr.appendChild(th);
    });
    head.appendChild(hr);
    var body = document.createElement("tbody");
    servers.forEach(function (s) {
      var grant = s.grant || "";
      var tr = document.createElement("tr");
      var server = document.createElement("th");
      server.scope = "row";
      var label = document.createElement("label");
      label.className = "agent-cap";
      var box = document.createElement("input");
      box.type = "checkbox";
      box.dataset.toolNames = grant;
      box.checked = serverChoiceOn(grant, opts);
      if (opts.lock) box.disabled = true;
      box.addEventListener("change", markAgentAccessUnsaved);
      var name = document.createElement("span");
      name.textContent = s.name || grant;
      label.appendChild(box);
      label.appendChild(name);
      server.appendChild(label);
      var tools = document.createElement("td");
      var list = (s.tools || []).map(function (tool) { return mcpShortTool(tool, grant); });
      tools.textContent = list.length ? list.join(", ") : (s.error ? "Not connected" : "No tools yet");
      if (!list.length) tools.className = "agent-cap-empty";
      tr.appendChild(server);
      tr.appendChild(tools);
      body.appendChild(tr);
    });
    table.appendChild(head);
    table.appendChild(body);
    block.appendChild(table);
    el.appendChild(block);
  }

  function checkedToolNames(el) {
    var names = [];
    var seen = {};
    function add(name) {
      if (!name || seen[name]) return;
      seen[name] = true;
      names.push(name);
    }
    if (!el) return names;
    el.querySelectorAll("input[type=checkbox][data-tool-names]").forEach(function (box) {
      if (box.disabled) return;
      if (box.indeterminate) {
        (box.dataset.partialNames || "").split(",").forEach(add);
        return;
      }
      if (box.checked) (box.dataset.toolNames || "").split(",").forEach(add);
    });
    return names;
  }

  function renderDefinitionList() {
    if (!agentDefList) return;
    agentDefList.innerHTML = "";
    if (!definitionsCache.length) {
      var empty = document.createElement("li");
      empty.className = "hint";
      empty.textContent = "No definitions yet.";
      agentDefList.appendChild(empty);
      return;
    }
    definitionsCache.forEach(function (def) {
      var li = document.createElement("li");
      var row = document.createElement("div");
      row.className = "agent-manage-row";
      var name = document.createElement("div");
      name.className = "agent-manage-name";
      var title = document.createElement("span");
      title.textContent = def.name || def.id;
      var meta = document.createElement("small");
      meta.textContent = def.tools ? (def.tools.length + " tools") : "All tools";
      name.appendChild(title);
      name.appendChild(meta);
      var actions = document.createElement("div");
      actions.className = "agent-manage-actions";
      actions.appendChild(manageAction("Edit", "ghost", function () { openEditDefinition(def); }));
      actions.appendChild(manageAction("New instance", "ghost", function () { openNewInstance(def.id); }));
      actions.appendChild(manageAction("Delete", "ghost danger", function () { deleteDefinition(def); }));
      row.appendChild(name);
      row.appendChild(actions);
      li.appendChild(row);
      agentDefList.appendChild(li);
    });
  }

  function renderTeamList() {
    if (!agentTeamList) return;
    agentTeamList.innerHTML = "";
    if (!teamsCache.length) {
      var empty = document.createElement("li");
      empty.className = "hint";
      empty.textContent = "No teams yet.";
      agentTeamList.appendChild(empty);
    } else {
      teamsCache.forEach(function (team) {
        var li = document.createElement("li");
        var row = document.createElement("div");
        row.className = "agent-manage-row";
        var name = document.createElement("div");
        name.className = "agent-manage-name";
        var title = document.createElement("span");
        title.textContent = team.name || team.id;
        var meta = document.createElement("small");
        meta.textContent = (team.definition_ids || []).length + " definitions";
        name.appendChild(title);
        name.appendChild(meta);
        var actions = document.createElement("div");
        actions.className = "agent-manage-actions";
        actions.appendChild(manageAction("Spawn", "ghost", function () { spawnTeam(team.id); }));
        actions.appendChild(manageAction("Delete", "ghost danger", function () {
          askConfirm("Delete team “" + (team.name || team.id) + "”? Instances already spawned stay.", "Delete").then(function (yes) {
            if (!yes) return;
            fetch(apiURL("/api/agents/teams/delete"), {
              method: "POST",
              headers: headers(),
              body: JSON.stringify({ id: team.id }),
            })
              .then(readAPIResult)
              .then(function () { return loadTeams(); })
              .catch(function (err) { showAgentDefErr(String(err)); });
          });
        }));
        row.appendChild(name);
        row.appendChild(actions);
        li.appendChild(row);
        agentTeamList.appendChild(li);
      });
    }
    var picks = document.getElementById("agent-team-defs");
    if (!picks) return;
    picks.innerHTML = "";
    definitionsCache.forEach(function (def) {
      var label = document.createElement("label");
      label.className = "agent-access-row";
      var box = document.createElement("input");
      box.type = "checkbox";
      box.dataset.accessId = def.id;
      box.checked = !!teamSelectIDs[def.id];
      box.addEventListener("change", function () {
        if (box.checked) teamSelectIDs[def.id] = true;
        else delete teamSelectIDs[def.id];
      });
      var text = document.createElement("span");
      text.textContent = def.name || def.id;
      label.appendChild(box);
      label.appendChild(text);
      picks.appendChild(label);
    });
  }

  function workspaceDisplayName(id) {
    for (var i = 0; i < workspacesCache.length; i++) {
      if (workspacesCache[i].id === id) return workspacesCache[i].name || id;
    }
    return id;
  }

  function renderAllInstances() {
    if (!agentInstanceList) return;
    agentInstanceList.innerHTML = "";
    if (!allInstancesCache.length) {
      var empty = document.createElement("li");
      empty.className = "hint";
      empty.textContent = "No instances yet.";
      agentInstanceList.appendChild(empty);
      return;
    }
    allInstancesCache.forEach(function (ag) {
      var li = document.createElement("li");
      var row = document.createElement("div");
      row.className = "agent-manage-row";
      var name = document.createElement("div");
      name.className = "agent-manage-name";
      var title = document.createElement("span");
      title.textContent = ag.name || ag.id;
      var defName = ag.recipe_id;
      definitionsCache.forEach(function (def) {
        if (def.id === ag.recipe_id) defName = def.name || def.id;
      });
      var places = (ag.workspaces || []).map(workspaceDisplayName).join(", ");
      var meta = document.createElement("small");
      meta.textContent = defName + (places ? " · " + places : "") + (ag.tools ? " · narrower tools" : " · definition’s tools");
      name.appendChild(title);
      name.appendChild(meta);
      var actions = document.createElement("div");
      actions.className = "agent-manage-actions";
      actions.appendChild(manageAction("Access", "ghost", function () { openAgentAccess(ag.id, ag.name || ag.id); }));
      actions.appendChild(manageAction("Delete", "ghost danger", function () { deleteAgent(ag.id, ag.name || ag.id); }));
      row.appendChild(name);
      row.appendChild(actions);
      li.appendChild(row);
      agentInstanceList.appendChild(li);
    });
  }

  function loadAllInstances() {
    return fetch(apiURL("/api/agents/instances?all=1"), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          allInstancesCache = j.instances || [];
          renderAllInstances();
        });
      });
  }

  function fillInstanceDefinitionSelect(selected) {
    if (!agentInstanceDefinition) return;
    agentInstanceDefinition.innerHTML = "";
    definitionsCache.forEach(function (def) {
      var opt = document.createElement("option");
      opt.value = def.id;
      opt.textContent = def.name || def.id;
      if (def.id === selected) opt.selected = true;
      agentInstanceDefinition.appendChild(opt);
    });
  }

  function openNewInstance(definitionID) {
    selectedDefinitionID = definitionID || (definitionsCache[0] && definitionsCache[0].id) || "";
    fillInstanceDefinitionSelect(selectedDefinitionID);
    var def = null;
    definitionsCache.forEach(function (d) { if (d.id === selectedDefinitionID) def = d; });
    if (agentInstanceName) agentInstanceName.value = def ? (def.name || "Agent") : "";
    setAgentDefPane("instance");
  }

  function loadMcpServers() {
    return fetch(apiURL("/api/mcp"), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          mcpServersCache = j.servers || [];
        });
      })
      .catch(function () {});
  }

  function loadDefinitions() {
    return fetch(apiURL("/api/agents/recipes"), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          definitionsCache = j.recipes || [];
          if (j.review_prompt_factory) reviewPromptFactory = j.review_prompt_factory;
          agentToolCatalog = j.tools || agentToolCatalog;
          if (j.groups) agentToolGroups = j.groups;
          renderDefinitionList();
          renderTeamList();
        });
      });
  }

  function loadTeams() {
    return fetch(apiURL("/api/agents/teams"), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          teamsCache = j.teams || [];
          renderTeamList();
        });
      });
  }

  function openAgentDialog() {
    if (!connected || !currentWorkspaceID) return;
    selectedDefinitionID = "";
    editingDefinitionID = "";
    teamSelectIDs = {};
    setManageTab(manageTab || "definitions");
    setAgentDefPane("pick");
    if (agentDefDialog && typeof agentDefDialog.showModal === "function") {
      agentDefDialog.showModal();
    }
    Promise.all([loadWorkspaces(), loadDefinitions(), loadTeams(), loadAllInstances()]).catch(function (err) {
      showAgentDefErr(String(err));
    });
  }

  function createAgent() {
    openAgentDialog();
  }

  function loadModelSuggestions() {
    var list = document.getElementById("agent-model-suggestions");
    if (!list || !connected) return Promise.resolve();
    return fetch(apiURL("/api/threads/model"), { headers: headers() })
      .then(function (r) { return r.json().then(function (j) { return r.ok ? j : null; }); })
      .then(function (j) {
        if (!j) return;
        list.innerHTML = "";
        var seen = {};
        function add(id, label) {
          if (!id || seen[id]) return;
          seen[id] = true;
          var opt = document.createElement("option");
          opt.value = id;
          if (label && label !== id) opt.label = label;
          list.appendChild(opt);
        }
        (j.providers || []).forEach(function (p) {
          (p.models || []).forEach(function (m) {
            add(m.id, (p.name ? p.name + " · " : "") + (m.name || m.id));
          });
        });
        add(j.settings_model_id, "Settings");
        add(j.model_id, "Current");
      })
      .catch(function () {});
  }

  function syncAgentApprovalUI(modeEl, hintEl, wrapEl) {
    var mode = modeEl ? modeEl.value : "auto";
    if (hintEl) hintEl.hidden = mode !== "ask";
    if (wrapEl) wrapEl.hidden = mode !== "auto";
  }
  function reviewPromptPayload(text) {
    var value = String(text || "");
    if (!value.trim()) {
      if (!reviewPromptFactory) return { reset: true };
      return { error: "Review prompt cannot be empty. Use Reset to follow the built-in prompt again." };
    }
    if (value.trim() === reviewPromptFactory.trim()) return { reset: true };
    return { prompt: value };
  }
  function openEditDefinition(def) {
    editingDefinitionID = def.id;
    if (agentDefEditName) agentDefEditName.value = def.name || "";
    if (agentDefEditInstructions) agentDefEditInstructions.value = def.instructions || "";
    var editModel = document.getElementById("agent-def-edit-model");
    if (editModel) editModel.value = def.model || "";
    var editMode = document.getElementById("agent-def-edit-mode");
    if (editMode) editMode.value = def.mode || "auto";
    var editReview = document.getElementById("agent-def-edit-review");
    if (editReview) editReview.value = def.review_prompt || reviewPromptFactory;
    syncAgentApprovalUI(editMode, document.getElementById("agent-def-edit-ask-hint"), document.getElementById("agent-def-edit-review-wrap"));
    setAgentDefPane("edit");
    loadModelSuggestions();
    loadMcpServers().then(function () {
      if (editingDefinitionID !== def.id) return;
      renderToolMatrix(document.getElementById("agent-def-edit-tools"), { granted: def.tools == null ? null : def.tools });
    });
  }

  function saveNewDefinition() {
    var name = agentDefName ? String(agentDefName.value || "").trim() : "";
    var instructions = agentDefInstructions ? String(agentDefInstructions.value || "").trim() : "";
    if (!name) {
      showAgentDefErr("Definition name required");
      return;
    }
    if (!agentToolCatalog.length) {
      showAgentDefErr("Tools are still loading");
      return;
    }
    var tools = checkedToolNames(document.getElementById("agent-def-create-tools"));
    var createModel = document.getElementById("agent-def-model");
    var model = createModel ? String(createModel.value || "").trim() : "";
    var createMode = document.getElementById("agent-def-mode");
    var mode = createMode ? createMode.value : "auto";
    var body = { name: name, instructions: instructions, model: model, tools: tools, mode: mode };
    if (mode === "auto") {
      var prompt = reviewPromptPayload(document.getElementById("agent-def-review") && document.getElementById("agent-def-review").value);
      if (prompt.error) {
        showAgentDefErr(prompt.error);
        return;
      }
      if (prompt.reset) body.review_prompt_reset = true;
      else body.review_prompt = prompt.prompt;
    }
    fetch(apiURL("/api/agents/recipes"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify(body),
    })
      .then(readAPIResult)
      .then(function () {
        if (agentDefName) agentDefName.value = "";
        if (agentDefInstructions) agentDefInstructions.value = "";
        if (createModel) createModel.value = "";
        setManageTab("definitions");
        setAgentDefPane("pick");
        return loadDefinitions();
      })
      .catch(function (err) {
        showAgentDefErr(String(err));
      });
  }

  function saveEditDefinition() {
    if (!editingDefinitionID) return;
    if (!agentToolCatalog.length) {
      showAgentDefErr("Tools are still loading");
      return;
    }
    var name = agentDefEditName ? String(agentDefEditName.value || "").trim() : "";
    var instructions = agentDefEditInstructions ? String(agentDefEditInstructions.value || "") : "";
    var editModel = document.getElementById("agent-def-edit-model");
    var model = editModel ? String(editModel.value || "").trim() : "";
    var tools = checkedToolNames(document.getElementById("agent-def-edit-tools"));
    var editMode = document.getElementById("agent-def-edit-mode");
    var mode = editMode ? editMode.value : "auto";
    var body = { id: editingDefinitionID, name: name, instructions: instructions, model: model, tools: tools, mode: mode };
    if (mode === "auto") {
      var prompt = reviewPromptPayload(document.getElementById("agent-def-edit-review") && document.getElementById("agent-def-edit-review").value);
      if (prompt.error) {
        showAgentDefErr(prompt.error);
        return;
      }
      if (prompt.reset) body.review_prompt_reset = true;
      else body.review_prompt = prompt.prompt;
    }
    fetch(apiURL("/api/agents/recipes/update"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify(body),
    })
      .then(readAPIResult)
      .then(function () {
        editingDefinitionID = "";
        setManageTab("definitions");
        setAgentDefPane("pick");
        return loadDefinitions().then(function () {
          if (currentAgentInstanceID) return loadSession();
        });
      })
      .catch(function (err) {
        showAgentDefErr(String(err));
      });
  }

  function deleteDefinition(def) {
    if (!def || !def.id) return;
    var name = def.name || def.id;
    askConfirm("Delete definition “" + name + "”?\n\nInstances using it must be deleted first.", "Delete").then(function (yes) {
      if (!yes) return;
      fetch(apiURL("/api/agents/recipes/delete"), {
        method: "POST",
        headers: headers(),
        body: JSON.stringify({ id: def.id }),
      })
        .then(readAPIResult)
        .then(function () {
          if (editingDefinitionID === def.id) {
            editingDefinitionID = "";
            setAgentDefPane("pick");
          }
          return loadDefinitions();
        })
        .catch(function (err) {
          showAgentDefErr(String(err));
        });
    });
  }

  function createInstanceFromDefinition() {
    selectedDefinitionID = agentInstanceDefinition ? agentInstanceDefinition.value : selectedDefinitionID;
    if (!selectedDefinitionID || !currentWorkspaceID) {
      showAgentDefErr("Pick a definition first");
      return;
    }
    var name = agentInstanceName ? String(agentInstanceName.value || "").trim() : "";
    if (!name) {
      showAgentDefErr("Instance name required");
      return;
    }
    fetch(apiURL("/api/agents/instances"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({
        recipe_id: selectedDefinitionID,
        name: name,
        workspace_id: currentWorkspaceID,
      }),
    })
      .then(readAPIResult)
      .then(function (j) {
        if (agentDefDialog && agentDefDialog.open) agentDefDialog.close();
        currentAgentInstanceID = (j.instance && j.instance.id) || "";
        return afterAgentAttach();
      })
      .catch(function (err) {
        showAgentDefErr(String(err));
      });
  }

  function createTeamFromSelection() {
    var ids = checkedAccess(document.getElementById("agent-team-defs"), false);
    if (!ids.length) {
      showAgentDefErr("Choose at least one definition.");
      return;
    }
    var nameEl = document.getElementById("agent-team-name");
    var name = nameEl ? String(nameEl.value || "").trim() : "";
    if (!name) {
      showAgentDefErr("Team name required");
      return;
    }
    fetch(apiURL("/api/agents/teams"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ name: name, definition_ids: ids }),
    })
      .then(readAPIResult)
      .then(function () {
        teamSelectIDs = {};
        if (nameEl) nameEl.value = "";
        return loadTeams();
      })
      .then(function () { return loadDefinitions(); })
      .catch(function (err) {
        showAgentDefErr(String(err));
      });
  }

  function spawnTeam(id) {
    if (!id || !currentWorkspaceID) return;
    askConfirm("Spawn team instances in this workspace?", "Spawn").then(function (yes) {
      if (!yes) return;
    fetch(apiURL("/api/agents/teams/spawn"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: id, workspace_id: currentWorkspaceID }),
    })
      .then(readAPIResult)
      .then(function (j) {
        if (agentDefDialog && agentDefDialog.open) agentDefDialog.close();
        var first = (j.instances && j.instances[0]) || null;
        currentAgentInstanceID = first ? first.id : "";
        return afterAgentAttach();
      })
      .catch(function (err) {
        showAgentDefErr(String(err));
      });
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
    if (m.pending) wrap.className += " pending";
    var filtered = !!m.filtered && !!(m.raw && m.raw !== m.result);
    // Unfiltered: single output lives in "raw". Filtered: raw + model result.
    var rawBody = filtered ? (m.raw || "") : (m.raw || m.result || "");
    var filteredBody = filtered ? (m.result || "") : "";

    var status = m.status || (m.pending ? "pending" : (m.ok === false ? "failed" : "ran"));
    if (status === "failed") wrap.className += " failed";
    if (status === "stopped") wrap.className += " stopped";
    var ranLabel = status === "failed" ? "Failed" : status === "stopped" ? "Stopped" : "Ran";
    var summary = document.createElement("summary");
    summary.innerHTML =
      '<span class="tool-line-ran">' + ranLabel + "</span>" +
      '<span class="tool-line-name">' + esc(name) + "</span>" +
      '<code class="tool-line-cmd">' + esc(toolFirstLine(cmd)) + "</code>" +
      (filtered ? '<span class="tool-line-funnel" title="Output filtered for model">▽</span>' : "") +
      (m.pending ? '<span class="tool-line-pending">…</span>' : "");
    wrap.appendChild(summary);

    var stack = document.createElement("div");
    stack.className = "tool-stack";

    var cmdBox = document.createElement("div");
    cmdBox.className = "tool-box tool-box-cmd";
    cmdBox.innerHTML =
      '<div class="tool-box-label">Command</div>' +
      '<pre class="tool-body">' + esc(cmd) + "</pre>";
    stack.appendChild(cmdBox);

    if (m.pending && !rawBody && !filteredBody) {
      var pend = document.createElement("p");
      pend.className = "tool-pending-hint";
      pend.textContent = "Running…";
      stack.appendChild(pend);
    } else {
      if (rawBody) {
        var rawBox = document.createElement("div");
        rawBox.className = "tool-box tool-box-raw";
        rawBox.innerHTML =
          '<div class="tool-box-label">Raw output</div>' +
          '<pre class="tool-body">' + window.ansiToHtml(rawBody) + "</pre>";
        stack.appendChild(rawBox);
      }
      if (filtered && filteredBody) {
        var filtBox = document.createElement("div");
        filtBox.className = "tool-box tool-box-filtered";
        filtBox.innerHTML =
          '<div class="tool-box-label">Filtered <span class="tool-line-funnel">▽</span></div>' +
          '<pre class="tool-body">' + window.ansiToHtml(filteredBody) + "</pre>";
        stack.appendChild(filtBox);
      }
    }
    wrap.appendChild(stack);
    return wrap;
  }

  function highlightFences(html) {
    if (typeof hljs === "undefined" || !hljs.highlightElement) return html;
    var wrap = document.createElement("div");
    wrap.innerHTML = html;
    wrap.querySelectorAll("pre code").forEach(function (block) {
      try {
        hljs.highlightElement(block);
      } catch (e) {}
    });
    return wrap.innerHTML;
  }

  function renderAssistantHTML(text) {
    var raw = text || "";
    try {
      if (typeof marked !== "undefined" && marked.parse) {
        var options = { async: false, gfm: true, breaks: true };
        if (typeof marked.Renderer === "function") {
          var renderer = new marked.Renderer();
          renderer.html = function (token) {
            var chunk = "";
            if (token && typeof token === "object") chunk = token.text || "";
            else if (typeof token === "string") chunk = token;
            return esc(chunk);
          };
          options.renderer = renderer;
        }
        var html = marked.parse(raw, options);
        if (typeof DOMPurify !== "undefined" && DOMPurify.sanitize) {
          html = DOMPurify.sanitize(html, {
            USE_PROFILES: { html: true },
            FORBID_TAGS: ["style", "link", "meta", "iframe", "object", "embed", "form", "base", "script"],
            FORBID_ATTR: ["style"],
          });
        }
        return highlightFences(html);
      }
    } catch (e) {}
    return window.ansiToHtml(raw);
  }

  function ensureDomKey(m) {
    if (!m.domKey) {
      transcriptSeq += 1;
      m.domKey = "m" + transcriptSeq;
    }
    return m.domKey;
  }

  function messageSig(m) {
    if (!m) return "";
    if (m.role === "tool_group") {
      return ["tool", m.name || "", m.command || "", m.pending ? "1" : "0", m.status || "", m.result || "", m.raw || "", m.filtered ? "1" : "0"].join("\u0001");
    }
    return [m.role || "", m.text || "", m.name || "", (m.files || []).join(","), m.streaming ? "1" : "0", m.collapsed ? "1" : "0", m.line || 0, m.branch_line || 0, m.branch_index || 0, m.branch_count || 0].join("\u0001");
  }

  function stampMessageEl(el, m, role) {
    if (!el) return;
    el.dataset.domKey = ensureDomKey(m);
    el.dataset.sig = messageSig(m);
    el.dataset.msgRole = role || m.role || "";
    if (el.tagName === "DETAILS" && heldOpen && heldOpen[el.dataset.domKey]) el.open = true;
  }

  function openMapFrom(el) {
    var open = {};
    if (!el) return open;
    function take(d) {
      if (!d || !d.open) return;
      var key = d.dataset.domKey || d.dataset.foldKey;
      if (key) open[key] = true;
    }
    take(el);
    el.querySelectorAll("details").forEach(take);
    return open;
  }

  function appendMessageNode(parent, m, prevRole) {
    if (m.role === "tool_group") {
      var card = toolCard(m);
      stampMessageEl(card, m, m.role);
      parent.appendChild(card);
      return m.role;
    }
    var el = document.createElement("article");
    var role = m.role || "assistant";
    if (role === "system") {
      el.className = "msg system";
      if (m.collapsed) {
        var first = String(m.text || "").split("\n")[0];
        if (first.length > 160) first = first.slice(0, 157) + "…";
        el.innerHTML = '<details class="notice"><summary>' + esc(first) + '</summary><div class="body">' + esc(m.text || "") + "</div></details>";
        var notice = el.querySelector("details");
        if (notice) {
          notice.dataset.domKey = ensureDomKey(m) + ":notice";
          if (heldOpen && heldOpen[notice.dataset.domKey]) notice.open = true;
        }
      } else {
        el.innerHTML = '<div class="body">' + esc(m.text || "") + "</div>";
      }
      stampMessageEl(el, m, role);
      parent.appendChild(el);
      return role;
    }
    if (role === "watch") {
      el.className = "msg watch";
      el.innerHTML = '<p class="who">Watch</p><div class="body">' + esc(m.text || "") + "</div>";
      stampMessageEl(el, m, role);
      parent.appendChild(el);
      return role;
    }
    if (role === "peer") {
      el.className = "msg peer";
      var peerWho = m.name || "Agent";
      var peerFiles = "";
      if (m.files && m.files.length) {
        peerFiles = '<ul class="msg-files">' + m.files.map(function (name) {
          return "<li>" + esc(name) + "</li>";
        }).join("") + "</ul>";
      }
      el.innerHTML = '<p class="who">' + esc(peerWho) + '</p><div class="body">' + esc(m.text || "") + peerFiles + "</div>";
      stampMessageEl(el, m, role);
      parent.appendChild(el);
      return role;
    }
    // One monologue label for the whole assistant turn (text + tools + more text).
    var continuation = role === "assistant" && (prevRole === "assistant" || prevRole === "tool_group");
    el.className = "msg " + role + (continuation ? " continuation" : "") + (m.streaming ? " streaming" : "");
    var who = role === "user" ? "You" : role === "system" ? "System" : "Keimling";
    var body;
    if (role === "user") {
      body = '<span class="msg-text">' + esc(m.text || "") + "</span>";
      if (m.files && m.files.length) {
        body += '<ul class="msg-files">' + m.files.map(function (name) {
          return "<li>" + esc(name) + "</li>";
        }).join("") + "</ul>";
      }
    } else {
      body = role === "assistant" ? renderAssistantHTML(m.text) : esc(m.text || "");
    }
    if (continuation) {
      el.innerHTML = '<div class="body">' + body + "</div>";
    } else {
      el.innerHTML = '<p class="who">' + esc(who) + '</p><div class="body">' + body + "</div>";
    }
    if (role === "user" && m.line) attachBranchControls(el, m);
    stampMessageEl(el, m, role);
    parent.appendChild(el);
    return role;
  }

  function agentBusy() {
    return agentState === "running" || agentState === "compacting" || agentState === "awaiting_approval";
  }

  function postJSON(url, body) {
    return fetch(apiURL(url), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify(body),
    }).then(function (r) {
      return r.text().then(function (t) {
        var j = {};
        try { j = t ? JSON.parse(t) : {}; } catch (e) {}
        if (!r.ok) throw new Error((j && (j.error || j.message)) || t || r.status);
        return j;
      });
    });
  }

  function attachBranchControls(el, m) {
    var body = el.querySelector(".body");
    if (!body) return;
    var source = m.branch_line || m.line;
    var count = m.branch_count || 1;
    var index = m.branch_index || 0;
    var nav = document.createElement("div");
    nav.className = "branch-nav";
    if (count > 1) {
      var prev = document.createElement("button");
      prev.type = "button";
      prev.className = "ghost";
      prev.textContent = "‹";
      prev.setAttribute("aria-label", "Previous continuation");
      prev.disabled = index <= 0;
      prev.addEventListener("click", function () { selectContinuation(source, index - 1); });
      var pos = document.createElement("span");
      pos.className = "branch-pos";
      pos.textContent = (index + 1) + " of " + count;
      var next = document.createElement("button");
      next.type = "button";
      next.className = "ghost";
      next.textContent = "›";
      next.setAttribute("aria-label", "Next continuation");
      next.disabled = index >= count - 1;
      next.addEventListener("click", function () { selectContinuation(source, index + 1); });
      nav.appendChild(prev);
      nav.appendChild(pos);
      nav.appendChild(next);
    }
    var edit = document.createElement("button");
    edit.type = "button";
    edit.className = "ghost";
    edit.textContent = "Edit";
    edit.setAttribute("aria-label", "Edit message");
    edit.disabled = agentBusy();
    edit.addEventListener("click", function () { openBranchEditor(el, m, source); });
    nav.appendChild(edit);
    body.appendChild(nav);
  }

  function openBranchEditor(el, m, source) {
    if (agentBusy()) {
      appendLive("\n[branch] The agent is still working.\n");
      return;
    }
    var body = el.querySelector(".body");
    if (!body || body.querySelector(".branch-edit")) return;
    var textEl = body.querySelector(".msg-text");
    var files = body.querySelector(".msg-files");
    var nav = body.querySelector(".branch-nav");
    if (textEl) textEl.hidden = true;
    if (files) files.hidden = true;
    if (nav) nav.hidden = true;
    var box = document.createElement("div");
    box.className = "branch-edit";
    var ta = document.createElement("textarea");
    ta.value = m.text || "";
    ta.setAttribute("aria-label", "Edit message");
    var row = document.createElement("div");
    row.className = "row";
    var cancel = document.createElement("button");
    cancel.type = "button";
    cancel.className = "ghost";
    cancel.textContent = "Cancel";
    var send = document.createElement("button");
    send.type = "button";
    send.textContent = "Send";
    function closeEditor() {
      box.remove();
      if (textEl) textEl.hidden = false;
      if (files) files.hidden = false;
      if (nav) nav.hidden = false;
    }
    cancel.addEventListener("click", closeEditor);
    send.addEventListener("click", function () {
      var text = ta.value.trim();
      if (!text) return;
      send.disabled = true;
      cancel.disabled = true;
      sendEditedMessage(source, text).catch(function (err) {
        send.disabled = false;
        cancel.disabled = false;
        appendLive("\n[branch] " + ((err && err.message) || err) + "\n");
      });
    });
    ta.addEventListener("keydown", function (e) {
      if (e.key === "Escape") {
        e.preventDefault();
        closeEditor();
      } else if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        send.click();
      }
    });
    row.appendChild(cancel);
    row.appendChild(send);
    box.appendChild(ta);
    box.appendChild(row);
    body.appendChild(box);
    ta.focus();
    ta.setSelectionRange(ta.value.length, ta.value.length);
  }

  function sendEditedMessage(line, text) {
    if (agentBusy()) return Promise.reject(new Error("The agent is still working."));
    return postJSON("/api/branch", { line: line, text: text }).then(function () {
      if (agentState === "idle") resetSideSamples();
      setState("running");
      return loadSession();
    }).then(function () { return refreshSideContext(); });
  }

  function selectContinuation(line, index) {
    if (agentBusy()) {
      appendLive("\n[branch] The agent is still working.\n");
      return;
    }
    postJSON("/api/branch/select", { line: line, index: index })
      .then(function () { return loadSession(); })
      .then(function () { return refreshSideContext(); })
      .catch(function (err) {
        appendLive("\n[branch] " + ((err && err.message) || err) + "\n");
      });
  }

  function appendMessageList(parent, list) {
    var prev = "";
    (list || []).forEach(function (m) {
      prev = appendMessageNode(parent, m, prev) || prev;
    });
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

    appendChapterItems(wrap, items);
    return wrap;
  }

  function chapterItemLoud(m) {
    if (!m) return false;
    if (m.role === "tool_group" && (m.status === "failed" || m.status === "stopped")) return true;
    if (m.role === "peer") return true;
    if (m.role === "system" && !m.collapsed) return true;
    return false;
  }

  function appendChapterItems(parent, items) {
    var prev = "";
    var buf = [];
    function flush() {
      if (!buf.length) return;
      if (buf.length <= 2) {
        buf.forEach(function (m) { prev = appendMessageNode(parent, m, prev) || prev; });
      } else {
        prev = appendMessageNode(parent, buf[0], prev) || prev;
        var mid = buf.slice(1, -1);
        var fold = document.createElement("details");
        fold.className = "chapter-fold";
        fold.dataset.foldKey = "fold:" + ensureDomKey(mid[0]) + ":" + ensureDomKey(mid[mid.length - 1]);
        if (heldOpen && heldOpen[fold.dataset.foldKey]) fold.open = true;
        var label = mid.length === 1 ? "1 message in between" : mid.length + " messages in between";
        fold.innerHTML = "<summary>" + esc(label) + "</summary>";
        var inner = document.createElement("div");
        inner.className = "chapter-fold-body";
        appendMessageList(inner, mid);
        fold.appendChild(inner);
        parent.appendChild(fold);
        prev = appendMessageNode(parent, buf[buf.length - 1], prev) || prev;
      }
      buf = [];
    }
    items.forEach(function (m) {
      if (chapterItemLoud(m)) {
        flush();
        prev = appendMessageNode(parent, m, prev) || prev;
      } else {
        buf.push(m);
      }
    });
    flush();
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

  function chapterBlockSig() {
    return chapters.map(function (ch) {
      return [ch.id, ch.start, ch.end, ch.title, ch.status, ch.memory, ch.summary, ch.intro, ch.conclusion].join("\u0001");
    }).join("\u0002");
  }

  // Keep existing nodes when only a later message arrives, so an opened tool or fold stays open.
  function syncMessageList(parent, list) {
    var kids = [];
    for (var i = 0; i < parent.children.length; i++) {
      var ch = parent.children[i];
      if (!ch.dataset || !ch.dataset.domKey) return false;
      kids.push(ch);
    }
    if (kids.length > list.length) return false;
    for (var n = 0; n < kids.length; n++) {
      if (kids[n].dataset.domKey !== ensureDomKey(list[n])) return false;
    }
    var prev = "";
    for (var k = 0; k < kids.length; k++) {
      var sig = messageSig(list[k]);
      if (kids[k].dataset.sig !== sig) {
        var holder = document.createElement("div");
        var prevHeld = heldOpen;
        heldOpen = openMapFrom(kids[k]);
        prev = appendMessageNode(holder, list[k], prev) || prev;
        heldOpen = prevHeld;
        var built = holder.firstElementChild;
        if (!built) return false;
        kids[k].replaceWith(built);
        kids[k] = built;
      } else {
        prev = kids[k].dataset.msgRole || prev;
      }
    }
    for (var j = kids.length; j < list.length; j++) {
      prev = appendMessageNode(parent, list[j], prev) || prev;
    }
    return true;
  }

  function fillMessageList(parent, list) {
    var prevHeld = heldOpen;
    heldOpen = openMapFrom(parent);
    parent.innerHTML = "";
    appendMessageList(parent, list);
    heldOpen = prevHeld;
  }

  function renderThread() {
    var stick = nearBottom();
    if (thread.dataset.rebuild === "1") {
      thread.innerHTML = "";
      delete thread.dataset.rebuild;
    }
    if (!messages.length) {
      thread.innerHTML = "";
      var hint = document.createElement("div");
      hint.className = "msg system";
      hint.innerHTML = '<p class="who">Keimling</p><p class="body">Send a message to continue this session.</p>';
      thread.appendChild(hint);
      stickScrollIfNearBottom(stick);
      return;
    }

    // Single chapter: full transcript for that range.
    if (activeChapter) {
      var prevHeld = heldOpen;
      heldOpen = openMapFrom(thread);
      thread.innerHTML = "";
      var head = document.createElement("div");
      head.className = "msg system";
      var sum = activeChapter.summary ? "\n\n" + activeChapter.summary : "";
      head.innerHTML = '<p class="who">Chapter</p><div class="body">' + esc(activeChapter.title + sum) + "</div>";
      thread.appendChild(head);
      appendMessageList(thread, messages.filter(inChapter));
      heldOpen = prevHeld;
      stickScrollIfNearBottom(stick);
      return;
    }

    // Chapters stay put; only the live tail grows, so an opened fold is not rebuilt.
    if (chapters.length) {
      var sig = chapterBlockSig();
      var block = thread.querySelector(":scope > .chapter-block");
      var tail = thread.querySelector(":scope > .thread-tail");
      if (!block || block.dataset.sig !== sig || !tail) {
        var chapterOpen = openMapFrom(thread);
        var prevOpen = heldOpen;
        heldOpen = chapterOpen;
        thread.innerHTML = "";
        block = document.createElement("div");
        block.className = "chapter-block";
        block.dataset.sig = sig;
        chapters.forEach(function (ch) {
          block.appendChild(renderChapterOverview(ch));
        });
        thread.appendChild(block);
        tail = document.createElement("div");
        tail.className = "thread-tail";
        thread.appendChild(tail);
        appendMessageList(tail, messagesOutsideChapters());
        heldOpen = prevOpen;
      } else if (!syncMessageList(tail, messagesOutsideChapters())) {
        fillMessageList(tail, messagesOutsideChapters());
      }
      stickScrollIfNearBottom(stick);
      return;
    }

    if (thread.querySelector(".chapter-block, .thread-tail")) thread.innerHTML = "";
    if (!syncMessageList(thread, messages)) fillMessageList(thread, messages);
    stickScrollIfNearBottom(stick);
  }

  function appendLive(text) {
    // Client-side diagnostics only — not part of the transcript.
    // A dropped event stream stays until events flow again.
    if (!uiFlash) return;
    var raw = String(text || "");
    uiFlash.hidden = false;
    uiFlash.textContent = raw.replace(/^\s+|\s+$/g, "").slice(0, 240);
    if (/^\[events\]/.test(raw.replace(/^\s+/, ""))) uiFlash.dataset.sticky = "1";
    else delete uiFlash.dataset.sticky;
  }

  function clearLive() {
    if (!uiFlash || uiFlash.dataset.sticky === "1") return;
    uiFlash.hidden = true;
    uiFlash.textContent = "";
  }

  function paintTranscript() {
    renderThread();
  }

  // Patch only the open streaming bubble — full rebuild per token freezes the UI.
  function patchStreamingBubble() {
    var last = messages[messages.length - 1];
    if (!last || last.role !== "assistant" || !last.streaming) {
      paintTranscript();
      return;
    }
    var nodes = thread.querySelectorAll(".msg.assistant.streaming");
    var el = nodes.length ? nodes[nodes.length - 1] : null;
    if (!el) {
      paintTranscript();
      return;
    }
    var body = el.querySelector(".body");
    if (!body) {
      paintTranscript();
      return;
    }
    var stick = nearBottom();
    // Lightweight during stream: plain text; markdown applied on replace/final paint.
    body.textContent = last.text || "";
    stickScrollIfNearBottom(stick);
  }

  function handleStructuredMessage(payload) {
    if (payload && payload.thread_id) {
      if (payload.thread_id !== currentThreadID) return;
    } else if (!eventsForThisView()) return;
    var text = payload.text || "";
    var role = payload.role || "assistant";
    if (payload.delta) {
      var last = messages[messages.length - 1];
      if (last && last.role === "assistant" && last.streaming) {
        last.text = (last.text || "") + text;
        patchStreamingBubble();
      } else {
        messages.push({ role: "assistant", text: text, streaming: true });
        paintTranscript();
      }
      return;
    }
    if (payload.replace) {
      var stream = messages[messages.length - 1];
      if (stream && stream.role === "assistant" && stream.streaming) {
        stream.text = text;
        delete stream.streaming;
      } else if (String(text).trim()) {
        messages.push({ role: role, text: text });
      }
      paintTranscript();
      return;
    }
    // Drop an unfinished stream bubble if a full message arrives.
    if (messages.length && messages[messages.length - 1].streaming) {
      messages.pop();
    }
    if (!String(text).trim()) return;
    if (role === "system" && String(text).trim() === "Stopped") {
      messages.forEach(function (m) {
        if (m.role === "tool_group" && m.pending) {
          m.pending = false;
          m.status = "stopped";
          if (!m.result) m.result = "Stopped";
        }
      });
    }
    messages.push({ role: role, text: text, name: payload.name || "", collapsed: !!payload.collapsed });
    paintTranscript();
  }

  function handleStructuredTool(payload) {
    if (!eventsForThisView()) return;
    if (payload.phase === "start") {
      // Finalize any open stream so tool sits in the monologue correctly.
      var open = messages[messages.length - 1];
      if (open && open.streaming) delete open.streaming;
      messages.push({
        role: "tool_group",
        name: payload.name || "tool",
        command: payload.command || payload.name || "tool",
        result: "",
        raw: "",
        pending: true,
      });
      // Collapsed by default (no approval) — avoids expand/collapse scroll jump.
      paintTranscript();
      return;
    }
    if (payload.phase !== "result") return;
    if (payload.name === "update_plan") refreshSideContext();
    var resultText = payload.result || "";
    var status = "ran";
    if (String(resultText).indexOf("INTERRUPTED:") === 0) status = "stopped";
    else if (payload.ok === false) status = "failed";
    var resultMsg = {
      role: "tool_group",
      name: payload.name || "tool",
      command: payload.command || payload.name || "tool",
      result: resultText,
      raw: payload.filtered ? (payload.raw || "") : "",
      filtered: !!payload.filtered,
      pending: false,
      ok: payload.ok,
      status: status,
    };
    for (var i = messages.length - 1; i >= 0; i--) {
      if (messages[i].role === "tool_group" && messages[i].pending && messages[i].name === resultMsg.name) {
        messages[i] = resultMsg;
        paintTranscript();
        return;
      }
    }
    messages.push(resultMsg);
    paintTranscript();
  }

  function formatUSD(n) {
    if (n == null || !isFinite(n)) return null;
    if (Math.abs(n) >= 1) return "$" + n.toFixed(2);
    if (Math.abs(n) >= 0.01) return "$" + n.toFixed(3);
    return "$" + n.toFixed(4);
  }

  function renderBudgetLine(budget) {
    if (!budgetLineEl) return;
    budget = budget || {};
    var parts = [];
    var sess = budget.session || {};
    if (sess.priced && sess.estimate_usd != null) {
      parts.push("session " + formatUSD(sess.estimate_usd));
    } else if ((sess.input_tokens || 0) + (sess.output_tokens || 0) > 0) {
      parts.push("session " + ((sess.input_tokens || 0) + (sess.output_tokens || 0)) + " tok");
    }
    var prov = budget.provider || {};
    if (prov.supported) {
      if (prov.error) {
        parts.push("key budget ?");
      } else if (prov.remaining_usd != null) {
        // OpenRouter: limit_remaining = remaining per-key cap, not account credits.
        var kind = prov.kind || "";
        parts.push((kind === "openrouter" ? "key left " : "left ") + formatUSD(prov.remaining_usd));
      }
    }
    budgetLineEl.textContent = parts.length ? parts.join(" · ") : "—";
    var tip = [];
    if (sess.priced && sess.estimate_usd != null) tip.push("Session estimate from token usage × model prices");
    if (prov.supported && !prov.error && prov.remaining_usd != null) {
      tip.push(prov.kind === "openrouter"
        ? "OpenRouter key limit_remaining (per-key cap), not account credit balance"
        : "Provider remaining budget");
    }
    tip.push("Click to refresh");
    budgetLineEl.title = tip.join(". ");
  }

  function refreshBudget(force) {
    if (!connected) return Promise.resolve();
    var q = force ? "?refresh=1" : "";
    return fetch(apiURL("/api/settings/budget" + q), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) return;
          renderBudgetLine(j);
        });
      })
      .catch(function () {});
  }

  function closeModelMenu() {
    if (!modelMenu) return;
    modelMenu.hidden = true;
    if (modelEl) modelEl.setAttribute("aria-expanded", "false");
  }

  function agentModelTitle(label, source, unresolved, saved) {
    if (unresolved) {
      return "Saved model “" + (saved || "this id") + "” is not in Settings. This chat is using " + label + " until that id is fixed on the definition or this instance.";
    }
    if (source === "instance") {
      return label + ", set on this instance. Change it in the instance editor. This chat cannot change it.";
    }
    if (source === "definition") {
      return label + ", from the definition. This chat cannot change it.";
    }
    return label + ", from Settings. The definition has no model. This chat cannot change it.";
  }

  function applyModelChrome(j) {
    if (!modelEl || !j) return;
    if (needsProvider) {
      modelEl.textContent = "—";
      modelEl.title = "No model provider";
      modelEl.disabled = true;
      return;
    }
    var label = j.label || j.model || j.model_id || "—";
    modelEl.textContent = label;
    var locked = !!(j.locked || j.chat_model_locked);
    var source = j.source || j.chat_model_source || "";
    var unresolved = !!(j.unresolved || j.chat_model_unresolved);
    var custom = !!(j.custom || j.chat_model_custom);
    if (locked) {
      closeModelMenu();
      modelEl.dataset.locked = "1";
      delete modelEl.dataset.custom;
      modelEl.removeAttribute("aria-haspopup");
      var titleName = label;
      var next = j.next_label || j.chat_model_next || "";
      if (next && next !== label) titleName = next;
      var title = agentModelTitle(titleName, source, unresolved, j.saved_model_id || j.chat_model_saved || "");
      if (next && next !== label) title = "This run uses " + label + ". " + title;
      modelEl.title = title;
      return;
    }
    delete modelEl.dataset.locked;
    modelEl.setAttribute("aria-haspopup", "menu");
    var next = j.next_label || j.chat_model_next || "";
    if (next && next !== label) {
      modelEl.title = "This run uses " + label + ". The next message uses " + next + ".";
    } else if (custom) {
      modelEl.title = "Kept on this thread (" + label + "). Pick the Settings model to follow Settings again.";
    } else {
      modelEl.title = "Follows Settings (" + label + "). Another model stays on this thread.";
    }
    if (custom) {
      modelEl.dataset.custom = "1";
    } else {
      delete modelEl.dataset.custom;
    }
  }

  function applyThreadModelView(j) {
    applyModelChrome(j);
  }

  function loadThreadModel() {
    if (!connected || !modelEl) return Promise.resolve();
    return fetch(apiURL("/api/threads/model"), { headers: headers() })
      .then(readJSONResponse)
      .then(function (j) {
        applyThreadModelView(j);
        return j;
      });
  }

  function openModelMenu() {
    if (!connected || (modelEl && modelEl.dataset.locked === "1")) return;
    loadThreadModel().then(function (j) {
      if (!j || !modelMenu || !modelEl) return;
      modelMenu.textContent = "";
      var note = document.createElement("p");
      note.className = "model-menu-note";
      var busy = agentState === "running" || agentState === "compacting" || agentState === "awaiting_approval";
      note.textContent = (j.custom
        ? "Kept on this thread. The Settings model follows Settings again."
        : "Follows Settings. Another model stays on this thread.") + (busy ? " This run keeps its model." : "");
      modelMenu.appendChild(note);
      var any = false;
      (j.providers || []).forEach(function (p) {
        var models = p.models || [];
        if (!models.length) return;
        any = true;
        var head = document.createElement("div");
        head.className = "model-menu-provider";
        head.textContent = p.name || p.id;
        modelMenu.appendChild(head);
        models.forEach(function (m) {
          var btn = document.createElement("button");
          btn.type = "button";
          btn.setAttribute("role", "menuitemradio");
          var selected = p.id === j.provider_id && m.id === j.model_id;
          btn.setAttribute("aria-checked", selected ? "true" : "false");
          btn.appendChild(document.createTextNode(m.name || m.id));
          if (p.id === j.settings_provider_id && m.id === j.settings_model_id) {
            var tag = document.createElement("span");
            tag.className = "model-menu-settings";
            tag.textContent = "Settings";
            btn.appendChild(tag);
          }
          btn.addEventListener("click", function (ev) {
            ev.stopPropagation();
            chooseThreadModel(p.id, m.id);
          });
          modelMenu.appendChild(btn);
        });
      });
      if (!any) {
        var empty = document.createElement("p");
        empty.className = "model-menu-note";
        empty.textContent = "No models saved in Settings.";
        modelMenu.appendChild(empty);
      }
      document.body.appendChild(modelMenu);
      modelMenu.hidden = false;
      var rect = modelEl.getBoundingClientRect();
      var height = modelMenu.offsetHeight;
      var top = rect.bottom + 6;
      if (top + height > window.innerHeight - 8 && rect.top - height - 6 >= 8) {
        top = rect.top - height - 6;
      }
      modelMenu.style.top = Math.max(8, top) + "px";
      modelMenu.style.bottom = "auto";
      modelMenu.style.left = Math.max(8, Math.min(rect.left, window.innerWidth - modelMenu.offsetWidth - 8)) + "px";
      modelEl.setAttribute("aria-expanded", "true");
    }).catch(function (e) {
      appendLive(e && e.message ? e.message : String(e));
    });
  }

  var modelSave = Promise.resolve();

  function chooseThreadModel(providerID, modelID) {
    closeModelMenu();
    modelSave = fetch(apiURL("/api/threads/model"), {
      method: "PUT",
      headers: headers(),
      body: JSON.stringify({ provider_id: providerID, model_id: modelID }),
    })
      .then(readJSONResponse)
      .then(function (j) {
        applyThreadModelView(j);
      })
      .catch(function (e) {
        appendLive(e && e.message ? e.message : String(e));
        modelSave = Promise.resolve();
        throw e;
      });
  }

  function applySession(j) {
    applyProviderGate(!!j.needs_provider);
    modelEl.textContent = needsProvider ? "—" : (j.model || "—");
    applyModelChrome(j);
    modeEl.textContent = j.mode || "—";
    messages = j.messages || [];
    chapters = j.chapters || [];
    if (thread) thread.dataset.rebuild = "1";
    voiceOK = !!j.voice;
    voiceBtn.hidden = !voiceOK;
    realtimeReady = !!j.realtime_ready;
    if (j.realtime_call && !callLive) {
      fetch(apiURL("/api/realtime/end"), { method: "POST", headers: headers() }).catch(function () {});
    }
    if (j.workspace_id) currentWorkspaceID = j.workspace_id;
    if (j.background_busy || j.busy) runningThreadID = j.running_thread || j.thread_id || runningThreadID;
    else if (!j.busy) runningThreadID = "";
    backgroundBusy = !!j.background_busy;
    if (j.thread_id) currentThreadID = j.thread_id;
    if (j.thread_id && !j.agent_instance_id && j.thread_id !== "main") workspaceThreadID = j.thread_id;
    if (!j.agent_instance_id && !j.busy && !j.background_busy) stoppingThreadID = "";
    if (j.agent_instance_id && !j.busy) delete stoppingAgents[j.agent_instance_id];
    // Prefer host attach state; session still reveals agent worker identity.
    if (j.agent_instance_id) currentAgentInstanceID = j.agent_instance_id;
    else if (!j.agent_instance_id) currentAgentInstanceID = "";
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
    renderBudgetLine(j.budget);
    renderChapters();
    renderThread();
    var next = j.busy ? "running" : (j.state || "idle");
    // The run can already be idle while this snapshot still says busy.
    if (next === "running" && sideClosed) next = "idle";
    setState(next);
    if (!j.busy) clearLive();
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
            if (uiFlash && uiFlash.dataset.sticky === "1") {
              delete uiFlash.dataset.sticky;
              clearLive();
            }
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
              // Agent "line" events are for CLI attach; GUI uses structured message/tool.
              if (ev === "message") handleStructuredMessage(payload);
              if (ev === "realtime_inject") speakText(payload && payload.text);
              if (ev === "tool") handleStructuredTool(payload);
              if (ev === "approval" && eventsForThisView()) showApproval(payload);
              if (ev === "progress") updateProgress(payload);
              if (ev === "artifact") {
                showArtifact(payload);
                loadArtifactsList();
              }
              if (ev === "compaction") updateCompactionProgress(payload);
              if (ev === "thread") updateThreadTitle(payload);
              if (ev === "status" && payload.state) {
                if (payload.thread_id) runningThreadID = payload.thread_id;
                var otherView = !currentAgentInstanceID && runningThreadID && currentThreadID && runningThreadID !== currentThreadID;
                if (payload.state === "idle") {
                  if (currentAgentInstanceID) delete stoppingAgents[currentAgentInstanceID];
                  if (!currentAgentInstanceID && (!payload.thread_id || payload.thread_id === stoppingThreadID)) stoppingThreadID = "";
                  runningThreadID = "";
                  backgroundBusy = false;
                  setState("idle");
                  loadSession()
                    .then(function () {
                      clearLive();
                      return refreshBudget(false);
                    })
                    .catch(function () {});
                  refreshSideContext();
                } else if (otherView && (payload.state === "running" || payload.state === "compacting" || payload.state === "awaiting_approval")) {
                  backgroundBusy = true;
                  setState("idle");
                } else {
                  backgroundBusy = false;
                  if (payload.state === "running" && agentState === "idle") resetSideSamples();
                  setState(payload.state);
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
    if (/unauthorized|\b401\b|\b403\b|Unexpected token 'u'/i.test(s)) {
      return "That secret was not accepted.";
    }
    return s;
  }

  function unlockServer() {
    var sec = loginSecretEl ? loginSecretEl.value.trim() : "";
    if (!sec) {
      showLogin("Connection secret required.");
      return;
    }
    secretEl.value = sec;
    apiBase = "";
    mode = "local";
    if (loginContinue) loginContinue.disabled = true;
    gateErr.hidden = true;
    fetch(apiURL("/api/health"), { headers: { Authorization: "Bearer " + sec } })
      .then(function (r) {
        if (r.status === 401 || r.status === 403) throw new Error("That secret was not accepted.");
        if (!r.ok) return r.text().then(function (t) {
          throw new Error((t || "").trim() || ("HTTP " + r.status));
        });
        storePrefs();
        openSession({ auto: true, stay: true, unlock: true });
      })
      .catch(function (e) {
        showLogin(formatConnectError(e));
      });
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
    // Adopt into machine book on actual connect; also mirror legacy gui-prefs connect.
    return adoptServerConnectionToClient({
      name: "",
      url: connect,
      access_secret: sec,
    }).then(function () {
      return saveGuiPrefs({ connect: connect, access_secret: sec }).catch(function () {});
    });
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
        if (opts.unlock) {
          showLogin("Connection secret required.");
          return;
        }
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
      // "This window" while still on a remote origin must leave that host,
      // unless this page is the server being unlocked.
      if (!sameOriginLocal() && !opts.stay) {
        switchAwayFromRemotePage();
        return;
      }
      apiBase = "";
      if (!secret()) {
        if (opts.unlock) {
          showLogin("Connection secret required.");
          return;
        }
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
    var adopt = Promise.resolve();
    if (!opts.auto && connectTarget && connectTarget.source === "server") {
      var sc = findIn(serverConnectionsCache, connectTarget.id);
      if (sc) adopt = adoptServerConnectionToClient(sc);
    }
    (clearRemote
      ? saveGuiPrefs({ active_connection_id: "local", connect: "", access_secret: "" }).then(function () {
          connectTarget = { source: "current" };
          renderAllConnectionLists();
        }).catch(function () {})
      : Promise.resolve())
      .then(function () { return adopt; })
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
        loadAgents();
        startAgentsPoll();
        msg.focus();
      })
      .catch(function (e) {
        connected = false;
        stopAgentsPoll();
        setState(agentState);
        if (opts.unlock) {
          var msg = formatConnectError(e);
          if (msg === "That secret was not accepted.") {
            try { localStorage.removeItem(STORAGE_SECRET); } catch (err) {}
            location.replace("/");
            return;
          }
          showLogin(msg);
          return;
        }
        showGate({ error: formatConnectError(e), manage: true, keepApp: false });
      })
      .finally(function () {
        connectBtn.disabled = false;
      });
  }

  if (choiceLocal) choiceLocal.addEventListener("click", function () { setChoice("local"); });
  if (choiceRemote) choiceRemote.addEventListener("click", function () { setChoice("remote"); });
  connectBtn.addEventListener("click", function () {
    applyConnectTarget();
    if (connectTarget && connectTarget.source === "local") {
      switchAwayFromRemotePage();
      return;
    }
    openSession();
  });
  if (secretEl) secretEl.addEventListener("keydown", function (e) {
    if (e.key === "Enter") openSession();
  });
  if (connectURLEl) connectURLEl.addEventListener("keydown", function (e) {
    if (e.key === "Enter") openSession();
  });
  if (connectionClientNewBtn) connectionClientNewBtn.addEventListener("click", function () {
    openConnectionEditor("client", null);
  });
  if (connectionServerNewBtn) connectionServerNewBtn.addEventListener("click", function () {
    openConnectionEditor("server", null);
  });
  if (connectionUseLocalBtn) connectionUseLocalBtn.addEventListener("click", function () {
    selectConnectTarget("local");
  });
  if (connEditCancel) connEditCancel.addEventListener("click", closeConnectionEditor);
  if (connEditSave) connEditSave.addEventListener("click", saveConnectionEditor);
  if (connEditDelete) connEditDelete.addEventListener("click", deleteConnectionEditor);
  if (vaultUnlockBtn) vaultUnlockBtn.addEventListener("click", function () { unlockVault(false); });
  if (vaultCreateBtn) vaultCreateBtn.addEventListener("click", function () { unlockVault(true); });
  if (credentialNewBtn) credentialNewBtn.addEventListener("click", function () { openCredentialEditor(null); });
  if (credEditCancel) credEditCancel.addEventListener("click", closeCredentialEditor);
  if (credEditSave) credEditSave.addEventListener("click", saveCredentialEditor);
  if (credEditDelete) credEditDelete.addEventListener("click", deleteCredentialEditor);
  if (providerNewBtn) providerNewBtn.addEventListener("click", function () {
    openProviderEditor(null);
  });
  if (setVoiceProvider) setVoiceProvider.addEventListener("change", syncVoiceModelEnabled);
  if (setSearchKind) setSearchKind.addEventListener("change", syncSearchFields);
  var providerDialogAdd = document.getElementById("provider-dialog-add");
  var providerDialogSkip = document.getElementById("provider-dialog-skip");
  var providerBannerAdd = document.getElementById("provider-banner-add");
  if (providerDialogAdd) providerDialogAdd.addEventListener("click", openProviderSetup);
  if (providerBannerAdd) providerBannerAdd.addEventListener("click", openProviderSetup);
  if (providerDialogSkip) providerDialogSkip.addEventListener("click", function () {
    try { sessionStorage.setItem("keimling.provider_skip", "1"); } catch (e) { /* banner stays */ }
    if (providerDialog) providerDialog.hidden = true;
  });
  if (provEditCancel) provEditCancel.addEventListener("click", closeProviderEditor);
  if (provEditSave) provEditSave.addEventListener("click", saveProviderEditor);
  if (provEditDelete) provEditDelete.addEventListener("click", deleteProviderEditor);
  if (provModelNew) provModelNew.addEventListener("click", openModelFormAdd);
  if (provModelSave) provModelSave.addEventListener("click", saveEditorModel);
  if (provModelID) provModelID.addEventListener("change", function () {
    if (!catalogCache.length) return;
    var id = provModelID.value.trim();
    var found = null;
    for (var i = 0; i < catalogCache.length; i++) {
      if (catalogCache[i].id === id) { found = catalogCache[i]; break; }
    }
    if (!found) setModelSenses(null);
    else setModelSenses({
      sees_images: !!found.sees_images,
      hears_audio: !!found.hears_audio,
      use_images: true,
      use_audio: true,
    });
  });
  if (provModelCancel) provModelCancel.addEventListener("click", closeModelForm);
  if (provCatalogFetch) provCatalogFetch.addEventListener("click", fetchModelCatalog);
  if (budgetLineEl) {
    budgetLineEl.style.cursor = "pointer";
    budgetLineEl.addEventListener("click", function () { refreshBudget(true); });
  }
  if (provModelID) {
    provModelID.addEventListener("change", function () {
      var id = provModelID.value.trim();
      for (var i = 0; i < catalogCache.length; i++) {
        if (catalogCache[i].id === id) {
          applyCatalogPrefill(catalogCache[i]);
          break;
        }
      }
    });
  }
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
    var raw = (data && data.raw) || {};
    var note = raw.note || "";
    if (raw.measured_input_tokens) note += " Last measured input: " + raw.measured_input_tokens + " tokens.";
    if (raw.est_tokens) note += " Shown parts estimate about " + raw.est_tokens + " tokens.";
    var noteEl = document.getElementById("ctx-raw-note");
    if (noteEl) noteEl.textContent = note;
    var rawEl = document.getElementById("ctx-raw");
    if (rawEl) {
      var chunks = [];
      (raw.parts || []).forEach(function (p) {
        chunks.push("## " + (p.name || "part") + " — " + (p.chars || 0) + " chars, ~" + (p.est_tokens || 0) + " tokens\n" + (p.text || ""));
      });
      rawEl.textContent = chunks.length ? chunks.join("\n\n") : "—";
    }
    var pct = typeof b.used_pct === "number" ? b.used_pct : 0;
    var label = "used " + (b.used || 0) + " / " + (b.limit || 0) +
      " (" + pct.toFixed(1) + "%) · remaining " + (b.remaining || 0) +
      " · " + (b.source || "estimate");
    document.getElementById("budget-label").textContent = label;
    var fill = document.getElementById("budget-fill");
    fill.style.width = Math.min(100, Math.max(0, pct)) + "%";
    fill.classList.toggle("warn", !!b.warn);
    renderSidePlan(c);
  }
  function planStatusLabel(status) {
    if (status === "in_progress") return "In progress";
    if (status === "done") return "Done";
    if (status === "pending") return "Pending";
    return status || "Pending";
  }
  function planStatusMark(status) {
    if (status === "done") return "✓";
    if (status === "in_progress") return "◐";
    return "○";
  }
  function renderSidePlan(charter) {
    var list = document.getElementById("side-plan");
    var empty = document.getElementById("side-plan-empty");
    if (!list) return;
    var steps = (charter && charter.plan) || [];
    list.innerHTML = "";
    if (empty) empty.hidden = steps.length > 0;
    steps.forEach(function (step, i) {
      var li = document.createElement("li");
      var status = step.status || "pending";
      li.dataset.status = status;
      if (charter && charter.position === i + 1) li.setAttribute("aria-current", "step");
      var st = document.createElement("span");
      st.className = "side-plan-status";
      st.textContent = planStatusMark(status);
      st.title = planStatusLabel(status);
      st.setAttribute("aria-label", planStatusLabel(status));
      var tx = document.createElement("span");
      tx.textContent = step.step || "";
      li.appendChild(st);
      li.appendChild(tx);
      list.appendChild(li);
    });
  }
  function refreshSideContext(opts) {
    if (!connected) return Promise.resolve();
    return fetch(apiURL("/api/context"), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          return j;
        });
      })
      .then(function (j) { renderContextDialog(j); })
      .catch(function (err) {
        if (opts && opts.report) appendLive("\n[context] " + err + "\n");
      });
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
    selectSidebar("context");
  }

  function threadAutopilotStatusText(j) {
    if (j.custom) return "Saved for this thread. It stays when the global text changes.";
    if (j.source === "factory") return "Following the built-in instructions. Saving stores a copy for this thread.";
    return "Following the global instructions. Saving stores a copy for this thread.";
  }

  function applyThreadAutopilot(j) {
    threadAutopilotCustom = !!j.custom;
    threadAutopilotSaved = j.custom ? (j.instructions || "") : "";
    if (autopilotInstructionsEl) autopilotInstructionsEl.value = j.instructions || "";
    if (autopilotInstructionsErr) autopilotInstructionsErr.hidden = true;
    if (autopilotInstructionsStatus) autopilotInstructionsStatus.textContent = threadAutopilotStatusText(j);
    if (autopilotInstructionsReset) autopilotInstructionsReset.disabled = !threadAutopilotCustom;
  }

  function readJSONResponse(r) {
    return r.text().then(function (t) {
      var j = {};
      try { j = JSON.parse(t); } catch (e) {}
      if (!r.ok) throw new Error((j && (j.message || j.error)) || t || ("HTTP " + r.status));
      return j;
    });
  }

  function loadThreadAutopilot() {
    if (!connected) return Promise.resolve();
    return fetch(apiURL("/api/threads/autopilot"), { headers: headers() })
      .then(readJSONResponse)
      .then(function (j) {
        applyThreadAutopilot(j);
        if (autopilotInstructionsUndo) autopilotInstructionsUndo.hidden = true;
      })
      .catch(function (err) {
        if (autopilotInstructionsErr) {
          autopilotInstructionsErr.hidden = false;
          autopilotInstructionsErr.textContent = String(err.message || err);
        }
      });
  }

  function openAutopilotDialog() {
    selectSidebar("instructions");
  }
  var autopilotForm = document.getElementById("autopilot-form");
  if (autopilotForm) {
    autopilotForm.addEventListener("submit", function (e) { e.preventDefault(); });
  }
  if (autopilotInstructionsSave) {
    autopilotInstructionsSave.addEventListener("click", function () {
      var text = autopilotInstructionsEl ? autopilotInstructionsEl.value : "";
      if (!text.trim()) {
        if (autopilotInstructionsErr) {
          autopilotInstructionsErr.hidden = false;
          autopilotInstructionsErr.textContent = "Autopilot instructions cannot be empty. Use Reset to follow the global prompt again.";
        }
        return;
      }
      autopilotInstructionsSave.disabled = true;
      fetch(apiURL("/api/threads/autopilot"), {
        method: "PUT",
        headers: headers(),
        body: JSON.stringify({ instructions: text }),
      })
        .then(readJSONResponse)
        .then(function (j) {
          applyThreadAutopilot(j);
          if (autopilotInstructionsUndo) autopilotInstructionsUndo.hidden = true;
          if (autopilotInstructionsStatus) autopilotInstructionsStatus.textContent = "Saved for this thread.";
        })
        .catch(function (err) {
          if (autopilotInstructionsErr) {
            autopilotInstructionsErr.hidden = false;
            autopilotInstructionsErr.textContent = String(err.message || err);
          }
        })
        .finally(function () { autopilotInstructionsSave.disabled = false; });
    });
  }
  if (autopilotInstructionsReset) {
    autopilotInstructionsReset.addEventListener("click", function () {
      if (!threadAutopilotCustom) return;
      var dirty = autopilotInstructionsEl && autopilotInstructionsEl.value !== threadAutopilotSaved;
      var msg = "Drop this thread's instructions and follow the global prompt again?";
      if (dirty) msg += " Unsaved edits in this box are discarded.";
      askConfirm(msg, "Reset").then(function (yes) {
        if (!yes) return;
      threadAutopilotUndoText = threadAutopilotSaved;
      fetch(apiURL("/api/threads/autopilot"), {
        method: "PUT",
        headers: headers(),
        body: JSON.stringify({ reset: true }),
      })
        .then(readJSONResponse)
        .then(function (j) {
          applyThreadAutopilot(j);
          if (autopilotInstructionsUndo) autopilotInstructionsUndo.hidden = false;
          if (autopilotInstructionsStatus) autopilotInstructionsStatus.textContent = "Reset. This thread follows the global prompt again.";
        })
        .catch(function (err) {
          if (autopilotInstructionsErr) {
            autopilotInstructionsErr.hidden = false;
            autopilotInstructionsErr.textContent = String(err.message || err);
          }
        });
      });
    });
  }
  if (autopilotInstructionsUndo) {
    autopilotInstructionsUndo.addEventListener("click", function () {
      if (!threadAutopilotUndoText.trim()) return;
      fetch(apiURL("/api/threads/autopilot"), {
        method: "PUT",
        headers: headers(),
        body: JSON.stringify({ instructions: threadAutopilotUndoText }),
      })
        .then(readJSONResponse)
        .then(function (j) {
          applyThreadAutopilot(j);
          autopilotInstructionsUndo.hidden = true;
          if (autopilotInstructionsStatus) autopilotInstructionsStatus.textContent = "Restored this thread's instructions.";
        })
        .catch(function (err) {
          if (autopilotInstructionsErr) {
            autopilotInstructionsErr.hidden = false;
            autopilotInstructionsErr.textContent = String(err.message || err);
          }
        });
    });
  }
  if (setAutopilotInstructionsReset) {
    setAutopilotInstructionsReset.addEventListener("click", function () {
      if (!autopilotSettingsCustom) return;
      var dirty = setAutopilotInstructions && setAutopilotInstructions.value !== autopilotSettingsBaseline;
      var msg = "Reset global autopilot instructions to the built-in prompt? Threads that saved their own text keep it.";
      if (dirty) msg += " Unsaved edits in this box are discarded.";
      askConfirm(msg, "Reset").then(function (yes) {
        if (!yes) return;
      autopilotSettingsUndoText = autopilotSettingsBaseline;
      fetch(apiURL("/api/settings"), {
        method: "PUT",
        headers: headers(),
        body: JSON.stringify({ autopilot_instructions_reset: true }),
      })
        .then(readJSONResponse)
        .then(function (j) {
          applyAutopilotSettingsFields(j);
          if (setAutopilotInstructionsUndo) setAutopilotInstructionsUndo.hidden = false;
          if (setAutopilotInstructionsStatus) setAutopilotInstructionsStatus.textContent = "Reset to the built-in prompt.";
        })
        .catch(function (err) {
          if (setAutopilotInstructionsErr) {
            setAutopilotInstructionsErr.hidden = false;
            setAutopilotInstructionsErr.textContent = String(err.message || err);
          }
        });
      });
    });
  }
  if (setAutopilotInstructionsUndo) {
    setAutopilotInstructionsUndo.addEventListener("click", function () {
      if (!autopilotSettingsUndoText.trim()) return;
      fetch(apiURL("/api/settings"), {
        method: "PUT",
        headers: headers(),
        body: JSON.stringify({ autopilot_instructions: autopilotSettingsUndoText }),
      })
        .then(readJSONResponse)
        .then(function (j) {
          applyAutopilotSettingsFields(j);
          setAutopilotInstructionsUndo.hidden = true;
          if (setAutopilotInstructionsStatus) setAutopilotInstructionsStatus.textContent = "Restored the global instructions.";
        })
        .catch(function (err) {
          if (setAutopilotInstructionsErr) {
            setAutopilotInstructionsErr.hidden = false;
            setAutopilotInstructionsErr.textContent = String(err.message || err);
          }
        });
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
      var head = document.createElement("div");
      head.className = "skill-head";
      var box = document.createElement("input");
      box.type = "checkbox";
      box.checked = !!sk.enabled;
      box.title = "Include this skill in the prompt";
      box.setAttribute("aria-label", "Include " + (sk.name || "skill") + " in the prompt");
      box.addEventListener("change", function () {
        var want = box.checked;
        showSkillsErr("");
        box.disabled = true;
        fetch(apiURL("/api/skills/enabled"), {
          method: "POST",
          headers: headers(),
          body: JSON.stringify({ name: sk.name, enabled: want }),
        })
          .then(function (r) {
            return r.json().then(function (j) {
              if (!r.ok) throw new Error((j && (j.error || j.message)) || r.status);
              return j;
            });
          })
          .catch(function (e) {
            box.checked = !want;
            showSkillsErr(e.message || e);
          })
          .finally(function () { box.disabled = false; });
      });
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
      if (sk.description) desc.title = sk.description;
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
          askConfirm("Remove skill “" + sk.name + "”?", "Remove").then(function (yes) {
            if (!yes) return;
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
        });
        actions.appendChild(rmBtn);
      }
      head.appendChild(box);
      head.appendChild(title);
      head.appendChild(meta);
      li.appendChild(head);
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
        var auto = document.getElementById("skills-auto-new");
        if (auto) auto.checked = !!j.auto_enable_new;
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
  var skillsAutoNew = document.getElementById("skills-auto-new");
  if (skillsAutoNew) {
    skillsAutoNew.addEventListener("change", function () {
      var want = skillsAutoNew.checked;
      showSkillsErr("");
      skillsAutoNew.disabled = true;
      fetch(apiURL("/api/skills/policy"), {
        method: "POST",
        headers: headers(),
        body: JSON.stringify({ auto_enable_new: want }),
      })
        .then(function (r) {
          return r.json().then(function (j) {
            if (!r.ok) throw new Error((j && (j.error || j.message)) || r.status);
            return j;
          });
        })
        .catch(function (e) {
          skillsAutoNew.checked = !want;
          showSkillsErr(e.message || e);
        })
        .finally(function () { skillsAutoNew.disabled = false; });
    });
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
  function mcpAddPair(list, namePh, valuePh) {
    if (!list) return;
    var row = document.createElement("div");
    row.className = "mcp-pair";
    var name = document.createElement("input");
    name.type = "text";
    name.placeholder = namePh;
    name.autocomplete = "off";
    name.spellcheck = false;
    var value = document.createElement("input");
    value.type = "password";
    value.placeholder = valuePh;
    value.autocomplete = "off";
    value.spellcheck = false;
    row.appendChild(name);
    row.appendChild(value);
    list.appendChild(row);
  }
  function mcpReadPairs(list) {
    var out = {};
    if (!list) return out;
    list.querySelectorAll(".mcp-pair").forEach(function (row) {
      var inputs = row.querySelectorAll("input");
      var k = (inputs[0] && inputs[0].value || "").trim();
      var v = inputs[1] ? inputs[1].value : "";
      if (k && v) out[k] = v;
    });
    return out;
  }
  function syncMcpAuthUI() {
    var auth = (mcpAuth && mcpAuth.value) || "none";
    if (mcpTokenRow) mcpTokenRow.hidden = auth !== "bearer";
    if (mcpHeaderRow) mcpHeaderRow.hidden = auth !== "headers";
    if (mcpAddBtn) mcpAddBtn.textContent = auth === "oauth" ? "Sign in" : "Add";
  }
  function syncMcpKindUI() {
    var remote = mcpKind && mcpKind.value === "url";
    if (mcpStdioRow) mcpStdioRow.hidden = remote;
    if (mcpEnvRow) mcpEnvRow.hidden = remote;
    if (mcpUrlRow) mcpUrlRow.hidden = !remote;
    if (mcpAuthRow) mcpAuthRow.hidden = !remote;
    if (!remote && mcpAuth) mcpAuth.value = "none";
    syncMcpAuthUI();
  }
  function mcpSignIn(name, url) {
    return fetch(apiURL("/api/mcp/sign-in"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ name: name, url: url, origin: location.origin }),
    }).then(function (r) {
      return r.text().then(function (t) {
        if (!r.ok) throw new Error(t || r.status);
        var j = {};
        try { j = JSON.parse(t); } catch (e) {}
        return j;
      });
    }).then(function (j) {
      if (j.authorize_url && !j.opened) window.open(j.authorize_url, "_blank", "noopener");
      showMcpErr("Waiting for the browser sign-in…");
      var left = 60;
      return new Promise(function (resolve, reject) {
        var timer = setInterval(function () {
          left--;
          loadMcpList().then(function () {
            var mine = (mcpServersCache || []).filter(function (s) { return s.name === name; })[0];
            if (mine && mine.oauth === "signed_in") {
              clearInterval(timer);
              showMcpErr("");
              resolve(mine);
            } else if (left <= 0) {
              clearInterval(timer);
              reject(new Error("Sign-in did not finish."));
            }
          }).catch(function () {});
        }, 1000);
      });
    });
  }
  function mcpNeedsSignIn(s) {
    if (!s || !s.url) return false;
    if (s.oauth === "expired") return true;
    if (s.oauth === "signed_in" && s.status === "connected") return false;
    if (s.auth === "oauth" && s.oauth !== "signed_in") return true;
    return /authorization required|sign-in|oauth/i.test(s.error || "");
  }
  function showSideMcpStatus(msg) {
    sideMcpStatus = msg || "";
    var el = document.getElementById("side-mcp-status");
    if (!el) return;
    el.hidden = !sideMcpStatus;
    el.textContent = sideMcpStatus;
  }
  function renderSideMcp(servers) {
    var list = document.getElementById("side-mcp");
    var empty = document.getElementById("side-mcp-empty");
    if (!list) return;
    list.innerHTML = "";
    servers = servers || [];
    if (empty) empty.hidden = servers.length > 0;
    var statusEl = document.getElementById("side-mcp-status");
    if (statusEl) {
      statusEl.hidden = !sideMcpStatus;
      statusEl.textContent = sideMcpStatus;
    }
    servers.forEach(function (s) {
      var li = document.createElement("li");
      var name = document.createElement("div");
      name.className = "side-mcp-name";
      name.textContent = s.name || "";
      var meta = document.createElement("div");
      meta.className = "side-mcp-meta";
      var needs = mcpNeedsSignIn(s);
      var usable = s.status === "connected" && !s.error && !needs;
      if (s.status === "disabled") meta.textContent = "Off";
      else if (needs) meta.textContent = "Needs sign-in";
      else if (usable) meta.textContent = "Usable";
      else meta.textContent = "Not usable";
      li.appendChild(name);
      li.appendChild(meta);
      if (!usable && s.status !== "disabled" && s.error && !needs) {
        var err = document.createElement("div");
        err.className = "side-mcp-err";
        err.textContent = /authorization required/i.test(s.error) ? "The server requires a bearer token or sign-in." : s.error;
        li.appendChild(err);
      }
      if (needs && s.status !== "disabled") {
        var sign = document.createElement("button");
        sign.type = "button";
        sign.className = "ghost";
        sign.textContent = "Sign in";
        sign.addEventListener("click", function () {
          sign.disabled = true;
          showSideMcpStatus("Waiting for the browser sign-in…");
          mcpSignIn(s.name, s.url).then(function () {
            showSideMcpStatus("");
          }).catch(function (e) {
            showSideMcpStatus(String(e.message || e));
          }).finally(function () { sign.disabled = false; });
        });
        li.appendChild(sign);
      }
      list.appendChild(li);
    });
  }
  function renderMcpServers(servers) {
    renderSideMcp(servers);
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
      var errText = s.error || "";
      if (/authorization required/i.test(errText)) errText = "The server requires a bearer token or sign-in.";
      var authNote = "";
      if (s.oauth === "signed_in") authNote = " · signed in";
      else if (s.oauth === "expired") authNote = " · sign-in expired";
      else if (s.auth === "bearer" || s.auth === "headers" || s.auth === "env") authNote = " · secret set";
      var from = "";
      if (s.source === "cursor") from = " · .cursor/mcp.json";
      else if (s.source === "project") from = " · .keimling/mcp.json";
      else if (s.source === "config") from = " · config.json";
      else if (s.source === "plugin") from = " · plugin " + (s.source_name || "");
      meta.textContent = (s.transport || "") + " · " + (s.status || "") + from + authNote +
        (errText ? " — " + errText : "") +
        (s.tools && s.tools.length ? " · " + s.tools.length + " tools" : "");
      var tools = document.createElement("div");
      tools.className = "mcp-tools";
      var allowed = {};
      (s.allow || []).forEach(function (name) { allowed[name] = true; });
      (s.tools || []).forEach(function (toolName) {
        var row = document.createElement("label");
        row.className = "check";
        var box = document.createElement("input");
        box.type = "checkbox";
        box.checked = !!allowed[toolName];
        if (s.source === "plugin") {
          box.disabled = true;
          box.title = "This server comes from a plugin";
        }
        box.addEventListener("change", function () {
          if (s.source === "plugin") return;
          box.disabled = true;
          fetch(apiURL("/api/mcp/allow"), {
            method: "POST",
            headers: headers(),
            body: JSON.stringify({ name: s.name, tool: toolName, allow: box.checked }),
          })
            .then(function (r) {
              return r.text().then(function (t) {
                if (!r.ok) throw new Error(t || r.status);
              });
            })
            .then(function () { return loadMcpList(); })
            .catch(function (e) {
              box.checked = !box.checked;
              showMcpErr(e.message || e);
            })
            .finally(function () { box.disabled = false; });
        });
        row.appendChild(box);
        row.appendChild(document.createTextNode(" Always allow " + toolName));
        tools.appendChild(row);
      });
      if (s.source !== "plugin") {
      var rm = document.createElement("button");
      rm.type = "button";
      rm.textContent = "Remove";
      rm.addEventListener("click", function (ev) {
        ev.preventDefault();
        ev.stopPropagation();
        askConfirm("Remove MCP server " + (s.name || "") + "?", "Remove").then(function (yes) {
          if (!yes) return;
          showMcpErr("");
          rm.disabled = true;
          fetch(apiURL("/api/mcp/remove"), {
            method: "POST",
            headers: headers(),
            body: JSON.stringify({ name: s.name }),
          })
            .then(function (r) {
              return r.text().then(function (t) {
                var j = {};
                try { j = t ? JSON.parse(t) : {}; } catch (e) {}
                if (!r.ok) throw new Error((j && (j.error || j.message)) || t || r.status);
                return j;
              });
            })
            .then(function (j) {
              return loadMcpList().then(function () {
                if (j && j.warning) showMcpErr(j.warning);
              });
            })
            .catch(function (e) { showMcpErr(e.message || e); })
            .finally(function () { rm.disabled = false; });
        });
      });
      }
      li.appendChild(title);
      li.appendChild(meta);
      if (s.tools && s.tools.length) {
        var allowHint = document.createElement("p");
        allowHint.className = "hint";
        allowHint.textContent = "Always allow runs that tool in ask and auto without a question. Uncheck to ask again.";
        li.appendChild(allowHint);
        li.appendChild(tools);
      }
      if (s.url) {
        var sign = document.createElement("button");
        sign.type = "button";
        sign.className = "ghost";
        sign.textContent = s.oauth === "signed_in" ? "Sign in again" : "Sign in";
        sign.addEventListener("click", function () {
          showMcpErr("");
          sign.disabled = true;
          mcpSignIn(s.name, s.url).catch(function (e) { showMcpErr(e.message || e); })
            .finally(function () { sign.disabled = false; });
        });
        li.appendChild(sign);
      }
      if (rm) li.appendChild(rm);
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
        mcpServersCache = j.servers || [];
        renderMcpServers(mcpServersCache);
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
  var sideMcpManage = document.getElementById("side-mcp-manage");
  if (sideMcpManage) sideMcpManage.addEventListener("click", openMcpDialog);

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
    var savedOpen = false;
    try {
      var savedW = parseInt(localStorage.getItem(STORAGE_ARTIFACT_WIDTH) || "", 10);
      if (savedW > 0) applyArtifactWidth(savedW);
      var savedMode = localStorage.getItem(STORAGE_ARTIFACT_MODE);
      if (savedMode === "overlay" || savedMode === "dock") artifactViewMode = savedMode;
      var savedTab = localStorage.getItem(STORAGE_SIDEBAR_TAB);
      if (savedTab) sideTab = savedTab;
      savedOpen = localStorage.getItem(STORAGE_SIDEBAR_OPEN) === "1";
    } catch (e) {}
    syncArtifactModeUI();
    selectSidebar(sideTab, { keepClosed: true });
    if (savedOpen) openArtifactPane();
  }
  function saveArtifactWidth(px) {
    try { localStorage.setItem(STORAGE_ARTIFACT_WIDTH, String(px)); } catch (e) {}
  }
  function persistSidebarLayout() {
    var open = !!(artifactPane && !artifactPane.hidden);
    try {
      localStorage.setItem(STORAGE_SIDEBAR_OPEN, open ? "1" : "0");
      localStorage.setItem(STORAGE_SIDEBAR_TAB, sideTab);
    } catch (e) {}
    saveGuiPrefs({ sidebar_open: open, sidebar_tab: sideTab }).catch(function () {});
  }
  function persistSidebarWidth(px) {
    saveArtifactWidth(px);
    saveGuiPrefs({ sidebar_width: px }).catch(function () {});
  }
  function applySidebarFromGuiPrefs(p) {
    if (!p) return;
    if (typeof p.sidebar_width === "number" && p.sidebar_width > 0) {
      saveArtifactWidth(applyArtifactWidth(p.sidebar_width));
    }
    var known = { now: 1, agents: 1, artifacts: 1, context: 1, connections: 1, instructions: 1 };
    if (known[p.sidebar_tab]) sideTab = p.sidebar_tab;
    if (p.sidebar_open === true) {
      selectSidebar(sideTab);
      return;
    }
    if (known[p.sidebar_tab]) selectSidebar(sideTab, { keepClosed: true });
    if (p.sidebar_open === false) closeArtifactPane({ silent: true });
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
    if (sidebarToggle) sidebarToggle.setAttribute("aria-pressed", "true");
  }
  function syncSidebarOpen() {
    if (!artifactOpenExt) return;
    artifactOpenExt.hidden = !(sideTab === "artifacts" && artifactSrc(currentArtifact));
  }
  function selectSidebar(tab, opts) {
    opts = opts || {};
    var known = { now: 1, agents: 1, artifacts: 1, context: 1, connections: 1, instructions: 1 };
    if (!known[tab]) tab = "now";
    sideTab = tab;
    try { localStorage.setItem(STORAGE_SIDEBAR_TAB, tab); } catch (e) {}
    var panels = artifactPane ? artifactPane.querySelectorAll(".side-panel") : [];
    panels.forEach(function (el) {
      el.hidden = el.getAttribute("data-side") !== tab;
    });
    var tabs = document.getElementById("sidebar-tabs");
    if (tabs) {
      tabs.querySelectorAll("button").forEach(function (btn) {
        btn.setAttribute("aria-selected", btn.getAttribute("data-side") === tab ? "true" : "false");
      });
    }
    syncSidebarOpen();
    if (opts.keepClosed) return;
    openArtifactPane();
    persistSidebarLayout();
    if (opts.skipFetch || !connected) return;
    if (tab === "agents") loadAgentBoard();
    if (tab === "now") renderSidePerf();
    if (tab === "now" || tab === "context") refreshSideContext({ report: tab === "context" });
    if (tab === "artifacts") {
      loadArtifactsList().then(function (items) {
        if (!currentArtifact && items && items.length) showArtifact(sortArtifacts(items)[0]);
      }).catch(function () {});
    }
    if (tab === "connections") loadMcpList().catch(function () {});
    if (tab === "instructions") loadThreadAutopilot();
  }
  function openSidebar() {
    selectSidebar(sideTab);
  }
  function sortArtifacts(items) {
    return (items || []).slice().sort(function (a, b) {
      return String(b.created || "").localeCompare(String(a.created || ""));
    });
  }
  function artifactKey(art) {
    if (!art) return "";
    return String(art.token || art.id || art.url || art.path || "");
  }
  function setArtifactEmpty(on) {
    if (artifactEmpty) artifactEmpty.hidden = !on;
    if (artifactFrame) artifactFrame.hidden = !!on;
    if (artifactPaper) artifactPaper.classList.toggle("is-empty", !!on);
    if (on && artifactFrame) artifactFrame.removeAttribute("src");
    syncSidebarOpen();
  }
  function renderArtifactHistory(items) {
    if (!artifactHistory) return;
    var list = sortArtifacts(items);
    artifactHistory.innerHTML = "";
    artifactHistory.hidden = !list.length;
    var currentKey = artifactKey(currentArtifact);
    list.forEach(function (art) {
      var li = document.createElement("li");
      var btn = document.createElement("button");
      btn.type = "button";
      btn.setAttribute("data-artifact", artifactKey(art));
      var name = art.title || art.path || "Artifact";
      btn.appendChild(document.createTextNode(name));
      if (art.path && art.path !== name) {
        var meta = document.createElement("small");
        meta.textContent = art.path;
        btn.appendChild(meta);
      }
      if (artifactKey(art) && artifactKey(art) === currentKey) btn.setAttribute("aria-current", "true");
      btn.addEventListener("click", function () { showArtifact(art); });
      li.appendChild(btn);
      artifactHistory.appendChild(li);
    });
  }
  function showArtifact(art) {
    if (!art || !artifactPane || !artifactFrame) return;
    currentArtifact = art;
    var src = artifactSrc(art);
    if (src) artifactFrame.src = src;
    setArtifactEmpty(!src);
    var key = artifactKey(art);
    if (artifactHistory) {
      artifactHistory.querySelectorAll("button").forEach(function (btn) {
        if (key && btn.getAttribute("data-artifact") === key) btn.setAttribute("aria-current", "true");
        else btn.removeAttribute("aria-current");
      });
    }
    selectSidebar("artifacts", { skipFetch: true });
  }
  function closeArtifactPane(opts) {
    var fs = document.fullscreenElement || document.webkitFullscreenElement;
    if (fs && artifactPane && artifactPane.contains(fs)) {
      var exit = document.exitFullscreen || document.webkitExitFullscreen;
      if (exit) exit.call(document);
    }
    if (artifactPane) artifactPane.hidden = true;
    if (artifactBackdrop) artifactBackdrop.hidden = true;
    if (shellEl) {
      shellEl.classList.remove("artifact-open");
      shellEl.classList.remove("artifact-overlay");
      shellEl.classList.remove("artifact-resizing");
    }
    if (sidebarToggle) sidebarToggle.setAttribute("aria-pressed", "false");
    if (!opts || !opts.silent) persistSidebarLayout();
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
  var sidebarTabs = document.getElementById("sidebar-tabs");
  if (sidebarTabs) {
    sidebarTabs.addEventListener("click", function (e) {
      var btn = e.target.closest && e.target.closest("button[data-side]");
      if (!btn) return;
      selectSidebar(btn.getAttribute("data-side"));
    });
  }
  setInterval(function () {
    if (!artifactPane || artifactPane.hidden || sideTab !== "now") return;
    renderSidePerf();
  }, 1000);
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
      var el = sideTab === "artifacts"
        ? (artifactPaper || artifactFrame)
        : (artifactPane && artifactPane.querySelector(".side-panel:not([hidden])"));
      if (!el) return;
      var req = el.requestFullscreen || el.webkitRequestFullscreen || el.msRequestFullscreen;
      if (!req) {
        appendLive("\n[sidebar] Fullscreen is unavailable\n");
        return;
      }
      Promise.resolve(req.call(el)).catch(function (err) {
        appendLive("\n[sidebar] Fullscreen: " + (err && err.message ? err.message : err) + "\n");
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
        var w = parseInt(shellEl.style.getPropertyValue("--artifact-width"), 10);
        if (w > 0) persistSidebarWidth(w);
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
        persistSidebarWidth(applyArtifactWidth(cur + step));
      } else if (e.key === "ArrowRight") {
        e.preventDefault();
        persistSidebarWidth(applyArtifactWidth(cur - step));
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
  if (mcpKind) {
    mcpKind.addEventListener("change", syncMcpKindUI);
  }
  if (mcpAuth) mcpAuth.addEventListener("change", syncMcpAuthUI);
  if (mcpEnvAdd) mcpEnvAdd.addEventListener("click", function () { mcpAddPair(mcpEnvList, "NAME", "value"); });
  if (mcpHeaderAdd) mcpHeaderAdd.addEventListener("click", function () { mcpAddPair(mcpHeaderList, "Header", "value"); });
  mcpAddPair(mcpEnvList, "NAME", "value");
  mcpAddPair(mcpHeaderList, "Header", "value");
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
        body.env = mcpReadPairs(mcpEnvList);
      } else {
        var url = (mcpUrl && mcpUrl.value || "").trim();
        if (!url) {
          showMcpErr("URL required.");
          return;
        }
        body.url = url;
        var auth = (mcpAuth && mcpAuth.value) || "none";
        body.auth = auth;
        if (auth === "oauth") {
          mcpAddBtn.disabled = true;
          mcpSignIn(name, url).then(function () {
            if (mcpName) mcpName.value = "";
            if (mcpUrl) mcpUrl.value = "";
            if (mcpToken) mcpToken.value = "";
          }).catch(function (e) { showMcpErr(e.message || e); })
            .finally(function () { mcpAddBtn.disabled = false; });
          return;
        }
        if (auth === "bearer") {
          var token = (mcpToken && mcpToken.value || "").trim();
          if (token) body.bearer = token;
        }
        if (auth === "headers") body.headers = mcpReadPairs(mcpHeaderList);
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
            var j = {};
            try { j = JSON.parse(t); } catch (e) {}
            return j;
          });
        })
        .then(function (j) {
          var mine = (j.servers || []).filter(function (s) { return s.name === name; })[0];
          renderMcpServers(j.servers || []);
          if (mine && mine.status === "error") {
            showMcpErr(/authorization required/i.test(mine.error || "") ? "The server requires a bearer token or sign-in." : (mine.error || "Could not connect."));
            return;
          }
          if (mcpName) mcpName.value = "";
          if (mcpCommand) mcpCommand.value = "";
          if (mcpUrl) mcpUrl.value = "";
          if (mcpToken) mcpToken.value = "";
        })
        .catch(function (e) { showMcpErr(e.message || e); })
        .finally(function () { mcpAddBtn.disabled = false; });
    });
  }

  gateCancelBtn.addEventListener("click", closeGate);
  if (gateCloseBtn) gateCloseBtn.addEventListener("click", closeGate);
  document.addEventListener("keydown", function (e) {
    if (e.key !== "Escape") return;
    if (gate.hidden || !gateManage) return;
    e.preventDefault();
    closeGate();
  });

  function persistThemes() {
    applyThemeRecord(themeActive());
    applyColorScheme();
    renderThemeList();
    saveGuiPrefs({ themes: themeState.themes, active_theme: themeState.active || "keimling" }).catch(function () {});
  }

  function applyColorScheme() {
    var scheme = themeState.scheme || "auto";
    if (scheme !== "light" && scheme !== "dark") scheme = "auto";
    document.documentElement.setAttribute("data-scheme", scheme);
    var light = document.getElementById("hl-light");
    var dark = document.getElementById("hl-dark");
    if (scheme === "auto") {
      if (light) { light.media = "(prefers-color-scheme: light)"; light.disabled = false; }
      if (dark) { dark.media = "(prefers-color-scheme: dark)"; dark.disabled = false; }
    } else {
      if (light) { light.media = "all"; light.disabled = scheme === "dark"; }
      if (dark) { dark.media = "all"; dark.disabled = scheme === "light"; }
    }
    document.querySelectorAll(".scheme-switch button").forEach(function (btn) {
      var on = btn.getAttribute("data-scheme") === scheme;
      btn.setAttribute("aria-pressed", on ? "true" : "false");
    });
  }

  document.querySelectorAll(".scheme-switch button").forEach(function (btn) {
    btn.addEventListener("click", function () {
      var scheme = btn.getAttribute("data-scheme") || "auto";
      themeState.scheme = scheme;
      applyColorScheme();
      saveGuiPrefs({ color_scheme: scheme }).catch(function () {});
    });
  });

  function renderThemeList() {
    var list = document.getElementById("theme-list");
    if (!list) return;
    list.innerHTML = "";
    themeCatalog().forEach(function (theme) {
      var li = document.createElement("li");
      li.className = "theme-row";
      var swatches = document.createElement("span");
      swatches.className = "theme-swatches";
      ["bg", "surface", "accent"].forEach(function (role) {
        var dot = document.createElement("i");
        dot.style.background = (theme.dark && theme.dark[role]) || "#000";
        swatches.appendChild(dot);
      });
      var name = document.createElement("span");
      name.textContent = theme.name || theme.id;
      if (theme.id === (themeState.active || "keimling")) name.textContent += " · in use";
      var use = document.createElement("button");
      use.type = "button";
      use.className = "ghost";
      use.textContent = "Use";
      use.addEventListener("click", function () {
        themeState.active = theme.id;
        persistThemes();
      });
      var edit = document.createElement("button");
      edit.type = "button";
      edit.className = "ghost";
      edit.textContent = "Edit";
      edit.addEventListener("click", function () { openThemeEditor(theme.id); });
      var del = document.createElement("button");
      del.type = "button";
      del.className = "ghost danger";
      del.textContent = "Delete";
      del.hidden = theme.id === "keimling";
      del.addEventListener("click", function () {
        askConfirm("Delete theme “" + (theme.name || theme.id) + "”?", "Delete").then(function (yes) {
          if (!yes) return;
          themeState.themes = (themeState.themes || []).filter(function (t) { return t.id !== theme.id; });
          if (themeState.active === theme.id) themeState.active = "keimling";
          persistThemes();
        });
      });
      li.appendChild(swatches);
      li.appendChild(name);
      li.appendChild(use);
      li.appendChild(edit);
      li.appendChild(del);
      list.appendChild(li);
    });
  }

  function openThemeEditor(id) {
    var editor = document.getElementById("theme-editor");
    var cols = document.getElementById("theme-cols");
    var nameEl = document.getElementById("theme-name");
    var reset = document.getElementById("theme-reset");
    if (!editor || !cols) return;
    var theme = themeCopy(themeCatalog().filter(function (t) { return t.id === id; })[0] || THEME_KEIMLING);
    themeEditID = theme.id;
    if (nameEl) nameEl.value = theme.name || "";
    var resetHint = document.getElementById("theme-reset-hint");
    if (reset) reset.hidden = theme.id !== "keimling";
    if (resetHint) resetHint.hidden = theme.id !== "keimling";
    cols.innerHTML = "";
    ["light", "dark"].forEach(function (scheme) {
      var col = document.createElement("div");
      var h = document.createElement("h3");
      h.textContent = scheme === "light" ? "Light" : "Dark";
      col.appendChild(h);
      THEME_ROLES.forEach(function (role) {
        var label = document.createElement("label");
        label.className = "color-field";
        label.appendChild(document.createTextNode(role[1]));
        var input = document.createElement("input");
        input.type = "color";
        input.dataset.scheme = scheme;
        input.dataset.role = role[0];
        input.value = (theme[scheme] && theme[scheme][role[0]]) || THEME_KEIMLING[scheme][role[0]];
        label.appendChild(input);
        col.appendChild(label);
      });
      cols.appendChild(col);
    });
    editor.hidden = false;
    paintThemePreview();
  }

  function paintThemePreview() {
    var host = document.getElementById("theme-preview");
    if (!host) return;
    host.hidden = false;
    host.innerHTML = "";
    ["light", "dark"].forEach(function (scheme) {
      var pal = {};
      THEME_ROLES.forEach(function (role) {
        var input = document.querySelector('#theme-cols input[data-scheme="' + scheme + '"][data-role="' + role[0] + '"]');
        pal[role[0]] = (input && normalizeBgHex(input.value)) || THEME_KEIMLING[scheme][role[0]];
      });
      var ink = scheme === "dark" ? "#0c1a14" : "#f4fffa";
      var card = document.createElement("div");
      card.className = "theme-sample";
      card.style.background = pal.bg;
      card.style.color = pal.fg;
      card.style.borderColor = pal.line;
      var title = document.createElement("strong");
      title.textContent = scheme === "light" ? "Light" : "Dark";
      var sheet = document.createElement("div");
      sheet.className = "theme-sample-sheet";
      sheet.style.background = pal.surface;
      sheet.style.color = pal.fg;
      sheet.style.borderColor = pal.line;
      var row = document.createElement("div");
      row.className = "theme-sample-row";
      row.style.background = pal.surface2;
      row.style.color = pal.fg;
      row.style.borderColor = pal.line;
      row.textContent = "Raised";
      var text = document.createElement("p");
      text.style.color = pal.fg;
      text.appendChild(document.createTextNode("Text "));
      var muted = document.createElement("span");
      muted.style.color = pal.muted;
      muted.textContent = "Muted";
      text.appendChild(muted);
      var bubble = document.createElement("div");
      bubble.className = "theme-sample-bubble";
      bubble.style.background = pal.bubble;
      bubble.style.color = pal.fg;
      bubble.style.borderColor = pal.line;
      bubble.textContent = "Bubble";
      var actions = document.createElement("div");
      actions.className = "theme-sample-actions";
      var btn = document.createElement("button");
      btn.type = "button";
      btn.textContent = "Button";
      btn.style.background = pal.accent;
      btn.style.color = ink;
      var danger = document.createElement("span");
      danger.style.color = pal.danger;
      danger.textContent = "Danger";
      actions.appendChild(btn);
      actions.appendChild(danger);
      sheet.appendChild(row);
      sheet.appendChild(text);
      sheet.appendChild(bubble);
      sheet.appendChild(actions);
      card.appendChild(title);
      card.appendChild(sheet);
      host.appendChild(card);
    });
  }

  function readThemeEditor() {
    var theme = themeCopy(THEME_KEIMLING);
    var existing = themeCatalog().filter(function (t) { return t.id === themeEditID; })[0];
    if (existing) theme = themeCopy(existing);
    var nameEl = document.getElementById("theme-name");
    theme.name = (nameEl && nameEl.value.trim()) || theme.name || theme.id;
    document.querySelectorAll("#theme-cols input[type=color]").forEach(function (input) {
      var hex = normalizeBgHex(input.value);
      if (!hex || !theme[input.dataset.scheme]) return;
      theme[input.dataset.scheme][input.dataset.role] = hex;
    });
    return theme;
  }

  var themeNewBtn = document.getElementById("theme-new");
  var themeSaveBtn = document.getElementById("theme-save");
  var themeCancelBtn = document.getElementById("theme-cancel");
  var themeCols = document.getElementById("theme-cols");
  if (themeCols) themeCols.addEventListener("input", paintThemePreview);
  var themeResetBtn = document.getElementById("theme-reset");
  if (themeNewBtn) {
    themeNewBtn.addEventListener("click", function () {
      var copy = themeCopy(themeActive());
      copy.id = "theme-" + Math.random().toString(36).slice(2, 8);
      copy.name = "New theme";
      copy.builtin = false;
      var stored = (themeState.themes || []).filter(function (t) { return t.id !== copy.id; });
      stored.push(copy);
      themeState.themes = stored;
      themeState.active = copy.id;
      persistThemes();
      openThemeEditor(copy.id);
    });
  }
  if (themeCancelBtn) {
    themeCancelBtn.addEventListener("click", function () {
      var editor = document.getElementById("theme-editor");
      if (editor) editor.hidden = true;
    });
  }
  if (themeResetBtn) {
    themeResetBtn.addEventListener("click", function () {
      themeState.themes = (themeState.themes || []).filter(function (t) { return t.id !== "keimling"; });
      themeState.active = "keimling";
      persistThemes();
      openThemeEditor("keimling");
    });
  }
  if (themeSaveBtn) {
    themeSaveBtn.addEventListener("click", function () {
      var theme = readThemeEditor();
      var next = (themeState.themes || []).filter(function (t) { return t.id !== theme.id; });
      if (theme.id === "keimling") {
        var same = JSON.stringify(theme.light) === JSON.stringify(THEME_KEIMLING.light) && JSON.stringify(theme.dark) === JSON.stringify(THEME_KEIMLING.dark) && theme.name === THEME_KEIMLING.name;
        if (!same) next.push(theme);
      } else {
        next.push(theme);
      }
      themeState.themes = next;
      themeState.active = theme.id;
      persistThemes();
      var editor = document.getElementById("theme-editor");
      if (editor) editor.hidden = true;
    });
  }

  function loadAgentBoard() {
    if (!connected) return Promise.resolve();
    return fetch(apiURL("/api/agents/instances?all=1"), { headers: headers() })
      .then(function (r) { return r.json().then(function (j) { if (!r.ok) throw new Error((j && j.error) || r.status); return j; }); })
      .then(function (j) { renderAgentBoard((j && j.instances) || []); })
      .catch(function () { renderAgentBoard([]); });
  }

  function agentPhaseLabel(ag) {
    if (ag.call && ag.phase === "running") return "On a call";
    if (ag.call) return "On a call";
    if (ag.phase === "running") return "Running";
    if (ag.phase === "paused") return "Paused";
    if (ag.phase === "waiting") {
      if (ag.waiting === "alarm") return "Waiting for an alarm";
      if (ag.waiting === "watch") return "Waiting for a watch";
      return "Waiting for a message";
    }
    return "Idle";
  }

  function renderAgentBoard(list) {
    var board = document.getElementById("agent-board");
    if (!board) return;
    board.innerHTML = "";
    if (!list.length) {
      var empty = document.createElement("p");
      empty.className = "hint";
      empty.textContent = "No agents yet.";
      board.appendChild(empty);
      return;
    }
    list.forEach(function (ag) {
      var row = document.createElement("div");
      row.className = "agent-board-row";
      var open = document.createElement("button");
      open.type = "button";
      open.className = "agent-board-open";
      var name = document.createElement("span");
      name.className = "agent-board-name";
      name.textContent = ag.name || ag.id;
      var meta = document.createElement("span");
      meta.className = "agent-board-meta";
      meta.textContent = ag.recipe_name || ag.recipe_id || "Definition";
      open.appendChild(name);
      open.appendChild(meta);
      open.addEventListener("click", function () { attachAgent(ag.id); });
      var phase = document.createElement("span");
      phase.className = "agent-phase" + (ag.phase ? " " + ag.phase : "");
      phase.textContent = agentPhaseLabel(ag);
      row.appendChild(open);
      row.appendChild(phase);
      board.appendChild(row);
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
  if (workspaceManageBtn) {
    workspaceManageBtn.addEventListener("click", function () {
      setWorkspaceMenuOpen(false);
      renderWorkspaceManage();
      if (workspaceDialog && workspaceDialog.showModal) workspaceDialog.showModal();
    });
  }
  if (workspaceDialog) {
    workspaceDialog.addEventListener("cancel", function (e) {
      if (workspaceDialog.querySelector(".agent-manage-name input")) {
        e.preventDefault();
        renderWorkspaceManage();
      }
    });
  }
  if (workspaceOpenBtn) {
    workspaceOpenBtn.addEventListener("click", function () {
      openWorkspacePath(workspacePath && workspacePath.value, workspaceNewName && workspaceNewName.value);
    });
  }
  if (workspacePath) {
    workspacePath.addEventListener("keydown", function (e) {
      if (e.key === "Enter") {
        e.preventDefault();
        openWorkspacePath(workspacePath.value, workspaceNewName && workspaceNewName.value);
      }
    });
  }
  if (workspaceNewName) {
    workspaceNewName.addEventListener("keydown", function (e) {
      if (e.key === "Enter") {
        e.preventDefault();
        openWorkspacePath(workspacePath && workspacePath.value, workspaceNewName.value);
      }
    });
  }
  mountGlyphs();
  if (composerMoreBtn) {
    composerMoreBtn.addEventListener("click", function (e) {
      e.preventDefault();
      e.stopPropagation();
      setComposerMore(composerMoreMenu && composerMoreMenu.hidden);
    });
  }
  if (composerMoreMenu) {
    composerMoreMenu.addEventListener("click", function (e) {
      if (e.target.closest("button")) setComposerMore(false);
    });
  }
  document.addEventListener("click", function (e) {
    if (!composerMoreMenu || composerMoreMenu.hidden) return;
    if (e.target.closest && e.target.closest(".composer-more")) return;
    setComposerMore(false);
  });
  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") setComposerMore(false);
  });
  document.addEventListener("copy", function (e) {
    if (!thread || !e.clipboardData) return;
    var sel = window.getSelection();
    if (!sel || sel.isCollapsed || sel.rangeCount !== 1) return;
    var range = sel.getRangeAt(0);
    var box = range.commonAncestorContainer;
    var el = box.nodeType === 1 ? box : box.parentElement;
    if (!el || !thread.contains(el)) return;
    var frag = range.cloneContents();
    var badges = frag.querySelectorAll(".chapter-overview-badges");
    if (!badges.length) return;
    badges.forEach(function (n) { n.remove(); });
    e.clipboardData.setData("text/plain", frag.textContent || "");
    e.preventDefault();
  });
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
  if (sidebarToggle) {
    sidebarToggle.addEventListener("click", function () {
      closeChaptersFlyout();
      closeModelMenu();
      if (artifactPane && !artifactPane.hidden) closeArtifactPane();
      else openSidebar();
    });
  }
  if (modelEl) {
    modelEl.addEventListener("click", function (e) {
      e.stopPropagation();
      if (!connected || modelEl.dataset.locked === "1") return;
      if (modelMenu && !modelMenu.hidden) {
        closeModelMenu();
        return;
      }
      closePaperMenu();
      closeChaptersFlyout();
      openModelMenu();
    });
  }
  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") closeModelMenu();
  });
  document.addEventListener("click", function (e) {
    if (modelMenu && !modelMenu.hidden && !modelMenu.contains(e.target) && e.target !== modelEl) {
      closeModelMenu();
    }
    if (chaptersFlyout && !chaptersFlyout.hidden) {
      var left = chaptersToggle && chaptersToggle.parentElement;
      if (!(left && left.contains(e.target))) closeChaptersFlyout();
    }
  });

  var pluginAdd = document.getElementById("plugin-add");
  if (pluginAdd) {
    pluginAdd.addEventListener("click", function () {
      var input = document.getElementById("plugin-location");
      var location = input ? String(input.value || "").trim() : "";
      if (!location) {
        showPluginErr("Git URL or directory required");
        return;
      }
      showPluginErr("");
      fetch(apiURL("/api/plugins/add"), { method: "POST", headers: headers(), body: JSON.stringify({ location: location }) })
        .then(readAPIResult)
        .then(function () {
          if (input) input.value = "";
          loadPlugins();
        })
        .catch(function (err) { showPluginErr(String(err.message || err)); });
    });
  }

  settingsTabs.querySelectorAll(".settings-tab").forEach(function (btn) {
    btn.addEventListener("click", function () {
      setSettingsTab(btn.getAttribute("data-tab"));
      if ((btn.getAttribute("data-tab") === "models" || btn.getAttribute("data-tab") === "agent") && connected) loadAgentSettings();
      if (btn.getAttribute("data-tab") === "secrets" && connected) loadCredentialsUI();
      if (btn.getAttribute("data-tab") === "plugins" && connected) loadPlugins();
    });
  });
  saveSettingsBtn.addEventListener("click", saveAgentSettings);

  if (newAgentBtn) {
    newAgentBtn.addEventListener("click", function () {
      createAgent();
    });
  }
  var agentAccessSave = document.getElementById("agent-access-save");
  if (agentAccessSave) {
    agentAccessSave.addEventListener("click", function (e) {
      e.preventDefault();
      saveAgentAccess();
    });
  }
  var agentDefNewBtn = document.getElementById("agent-def-new-btn");
  var agentDefCreateBack = document.getElementById("agent-def-create-back");
  var agentDefCreateSave = document.getElementById("agent-def-create-save");
  var agentDefEditBack = document.getElementById("agent-def-edit-back");
  var agentDefEditSave = document.getElementById("agent-def-edit-save");
  var agentDefInstanceBack = document.getElementById("agent-def-instance-back");
  var agentDefInstanceCreate = document.getElementById("agent-def-instance-create");
  var agentTeamNewBtn = document.getElementById("agent-team-new-btn");
  if (agentDefNewBtn) {
    agentDefNewBtn.addEventListener("click", function () {
      setAgentDefPane("create");
      loadModelSuggestions();
      loadMcpServers().then(function () {
        renderToolMatrix(document.getElementById("agent-def-create-tools"), { granted: null });
        var createMode = document.getElementById("agent-def-mode");
        if (createMode) createMode.value = "auto";
        var createReview = document.getElementById("agent-def-review");
        if (createReview) createReview.value = reviewPromptFactory;
        syncAgentApprovalUI(createMode, document.getElementById("agent-def-ask-hint"), document.getElementById("agent-def-review-wrap"));
      });
      if (agentDefName) agentDefName.focus();
    });
  }
  if (agentDefCreateBack) {
    agentDefCreateBack.addEventListener("click", function () {
      setAgentDefPane("pick");
      loadDefinitions().catch(function () {});
    });
  }
  if (agentDefCreateSave) {
    agentDefCreateSave.addEventListener("click", function () {
      saveNewDefinition();
    });
  }
  if (agentDefEditBack) {
    agentDefEditBack.addEventListener("click", function () {
      editingDefinitionID = "";
      setManageTab("definitions");
      setAgentDefPane("pick");
    });
  }
  if (agentDefEditSave) {
    agentDefEditSave.addEventListener("click", function () {
      saveEditDefinition();
    });
  }
  ["agent-def-mode", "agent-def-edit-mode"].forEach(function (id) {
    var el = document.getElementById(id);
    if (!el) return;
    el.addEventListener("change", function () {
      var edit = id.indexOf("edit") !== -1;
      syncAgentApprovalUI(el, document.getElementById(edit ? "agent-def-edit-ask-hint" : "agent-def-ask-hint"), document.getElementById(edit ? "agent-def-edit-review-wrap" : "agent-def-review-wrap"));
    });
  });
  var agentDefReviewReset = document.getElementById("agent-def-edit-review-reset");
  if (agentDefReviewReset) {
    agentDefReviewReset.addEventListener("click", function () {
      if (!editingDefinitionID) return;
      var box = document.getElementById("agent-def-edit-review");
      var dirty = box && reviewPromptFactory && box.value.trim() !== reviewPromptFactory.trim();
      var msg = "Drop this definition's review prompt and use the built-in one again?";
      if (dirty) msg += " Unsaved edits in this box are discarded.";
      askConfirm(msg, "Reset").then(function (yes) {
        if (!yes) return;
      var modeEl = document.getElementById("agent-def-edit-mode");
      agentDefReviewReset.disabled = true;
      fetch(apiURL("/api/agents/recipes/update"), {
        method: "POST",
        headers: headers(),
        body: JSON.stringify({ id: editingDefinitionID, mode: modeEl ? modeEl.value : "auto", review_prompt_reset: true }),
      })
        .then(readAPIResult)
        .then(function (rec) {
          if (box) box.value = reviewPromptFactory;
          var found = (definitionsCache || []).filter(function (d) { return d.id === editingDefinitionID; })[0];
          if (found) found.review_prompt = "";
          if (rec && rec.mode && modeEl) modeEl.value = rec.mode;
          showAgentDefErr("");
        })
        .catch(function (err) { showAgentDefErr(String(err)); })
        .finally(function () { agentDefReviewReset.disabled = false; });
      });
    });
  }
  if (agentDefInstanceBack) {
    agentDefInstanceBack.addEventListener("click", function () {
      selectedDefinitionID = "";
      setManageTab("instances");
      setAgentDefPane("pick");
    });
  }
  if (agentDefInstanceCreate) {
    agentDefInstanceCreate.addEventListener("click", function () {
      createInstanceFromDefinition();
    });
  }
  if (agentTeamNewBtn) {
    agentTeamNewBtn.addEventListener("click", function () {
      createTeamFromSelection();
    });
  }
  ["definitions", "instances", "teams"].forEach(function (name) {
    var tab = document.getElementById("agent-tab-" + name);
    if (!tab) return;
    tab.addEventListener("click", function () {
      setManageTab(name);
      setAgentDefPane("pick");
    });
  });
  var agentInstanceNewBtn = document.getElementById("agent-instance-new-btn");
  if (agentInstanceNewBtn) {
    agentInstanceNewBtn.addEventListener("click", function () {
      openNewInstance("");
    });
  }
  var agentAccessBack = document.getElementById("agent-access-back");
  if (agentAccessBack) {
    agentAccessBack.addEventListener("click", function () {
      setManageTab("instances");
      setAgentDefPane("pick");
      loadAllInstances().catch(function () {});
    });
  }
  if (agentAccessFollow) {
    agentAccessFollow.addEventListener("change", function () {
      markAgentAccessUnsaved();
      applyAccessFollow();
    });
  }

  newChatBtn.addEventListener("click", function () {
    if (!connected || callLive) return;
    newChatBtn.disabled = true;
    fetch(apiURL("/api/threads"), { method: "POST", headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          return j;
        });
      })
      .then(function (j) {
        currentAgentInstanceID = "";
        if (j && j.thread_id) {
          currentThreadID = j.thread_id;
          if (j.thread_id !== "main") workspaceThreadID = j.thread_id;
        }
        if (es && typeof es.close === "function") {
          try { es.close(); } catch (e) {}
          es = null;
        }
        messages = [];
        chapters = [];
        activeChapter = null;
        clearLive();
        hideApproval();
        return loadSession();
      })
      .then(function () {
        connectEvents();
        renderChapters();
        renderThread();
        return Promise.all([loadThreads(), loadAgents()]);
      })
      .then(function () {
        msg.focus();
      })
      .catch(function (err) {
        appendLive("\n[thread] " + err + "\n");
      })
      .finally(function () {
        newChatBtn.disabled = !connected || callLive;
      });
  });

  var mentionMenu = document.getElementById("mention-menu");
  var mentionPicks = [];
  var mentionItems = [];
  var mentionIndex = 0;
  var mentionDismissed = null;
  var mentionLoading = false;
  var fileMentionCache = { q: "", files: [] };
  var fileMentionToken = 0;

  function closeMentionMenu() {
    if (!mentionMenu) return;
    mentionMenu.hidden = true;
    mentionMenu.innerHTML = "";
    mentionItems = [];
  }

  function activeMention(value, pos) {
    value = String(value || "");
    pos = Math.max(0, Math.min(pos || 0, value.length));
    var upto = value.slice(0, pos);
    var at = -1;
    for (var i = upto.length - 1; i >= 0; i--) {
      var ch = upto.charAt(i);
      if (ch === "\n") return null;
      if (ch === "@") {
        if (i === 0 || /\s/.test(upto.charAt(i - 1))) at = i;
        break;
      }
    }
    if (at < 0) return null;
    var query = upto.slice(at + 1);
    var agents = agentsCache || [];
    var closed = false;
    for (var n = 0; n < agents.length; n++) {
      var name = String(agents[n].name || "");
      if (!name || query.length <= name.length) continue;
      if (query.slice(0, name.length).toLowerCase() !== name.toLowerCase()) continue;
      if (!/\s/.test(query.charAt(name.length))) continue;
      var longer = false;
      for (var m = 0; m < agents.length; m++) {
        var other = String(agents[m].name || "");
        if (other.length <= name.length) continue;
        if (query.toLowerCase().indexOf(other.toLowerCase()) === 0) longer = true;
      }
      if (!longer) closed = true;
    }
    if (closed) return null;
    return { start: at, query: query };
  }

  function updateMentionMenu() {
    if (!mentionMenu) {
      closeMentionMenu();
      return;
    }
    var active = activeMention(msg.value, msg.selectionStart);
    if (!active || mentionDismissed === active.query) {
      closeMentionMenu();
      return;
    }
    var q = active.query.toLowerCase();
    if (currentAgentInstanceID && !q.trim()) {
      closeMentionMenu();
      return;
    }
    var agents = currentAgentInstanceID ? [] : (agentsCache || []).filter(function (a) {
      var name = String(a.name || "").toLowerCase();
      var id = String(a.id || "").toLowerCase();
      if (!q.trim()) return true;
      if (name.indexOf(q) >= 0 || id.indexOf(q) >= 0) return true;
      return !!(name && q.indexOf(name) === 0);
    }).map(function (a) {
      return { kind: "agent", id: a.id, name: a.name || a.id };
    });
    var files = [];
    if (q.trim()) {
      if (fileMentionCache.q !== q) {
        fileMentionCache = { q: q, files: [] };
        requestFileMentions(active.query);
      }
      files = (fileMentionCache.files || []).map(function (f) {
        return { kind: "file", id: "", name: f.path || f.name, path: f.path || f.name };
      });
    }
    var list = agents.concat(files);
    mentionItems = list;
    mentionMenu.innerHTML = "";
    mentionMenu.hidden = false;
    if (!list.length) {
      var empty = document.createElement("p");
      empty.className = "mention-empty";
      empty.textContent = q.trim() ? "No match" : "No agent in this workspace";
      mentionMenu.appendChild(empty);
      placeMentionMenu();
      return;
    }
    if (mentionIndex >= list.length) mentionIndex = 0;
    if (mentionIndex < 0) mentionIndex = list.length - 1;
    list.forEach(function (a, i) {
      var b = document.createElement("button");
      b.type = "button";
      b.className = "mention-option" + (i === mentionIndex ? " active" : "");
      b.setAttribute("role", "option");
      var label = document.createElement("span");
      label.className = "mention-name";
      label.textContent = a.name || a.id;
      var id = document.createElement("span");
      id.className = "mention-id";
      id.textContent = a.kind === "file" ? "File" : (a.id || "");
      b.appendChild(label);
      b.appendChild(id);
      b.addEventListener("mousedown", function (e) {
        e.preventDefault();
        applyMention(a);
      });
      mentionMenu.appendChild(b);
    });
    placeMentionMenu();
  }

  function requestFileMentions(query) {
    var token = ++fileMentionToken;
    var key = String(query || "").toLowerCase();
    fetch(apiURL("/api/files?q=" + encodeURIComponent(query)), { headers: headers() })
      .then(function (r) { return r.ok ? r.json() : { files: [] }; })
      .then(function (j) {
        if (token !== fileMentionToken) return;
        fileMentionCache = { q: key, files: (j && j.files) || [] };
        updateMentionMenu();
      })
      .catch(function () {});
  }

  function placeMentionMenu() {
    placePopMenu(mentionMenu);
  }

  var slashMenu = document.getElementById("slash-menu");
  var slashItems = [];
  var slashIndex = 0;
  var slashCommands = null;
  var slashLoading = false;

  function closeSlashMenu() {
    if (!slashMenu) return;
    slashMenu.hidden = true;
    slashMenu.innerHTML = "";
    slashItems = [];
  }

  function activeSlash(value) {
    value = String(value || "");
    if (!value.startsWith("/") || value.indexOf("\n") >= 0) return null;
    var token = value.slice(1);
    if (/\s/.test(token)) return null;
    return token;
  }

  function ensureSlashCommands(done) {
    if (slashCommands) {
      done();
      return;
    }
    if (slashLoading) return;
    slashLoading = true;
    fetch(apiURL("/api/plugin-commands"), { headers: headers() })
      .then(function (r) { return r.ok ? r.json() : { commands: [] }; })
      .then(function (j) {
        slashCommands = (j && j.commands) || [];
        slashLoading = false;
        done();
      })
      .catch(function () { slashLoading = false; });
  }

  function updateSlashMenu() {
    if (!slashMenu) return;
    var query = activeSlash(msg.value);
    if (query === null) {
      closeSlashMenu();
      return;
    }
    if (!slashCommands) {
      ensureSlashCommands(updateSlashMenu);
      return;
    }
    var q = query.toLowerCase();
    var list = slashCommands.filter(function (c) {
      return !q || String(c.name || "").indexOf(q) === 0 || String(c.description || "").toLowerCase().indexOf(q) >= 0;
    });
    slashItems = list;
    slashMenu.innerHTML = "";
    if (!slashCommands.length) {
      closeSlashMenu();
      return;
    }
    slashMenu.hidden = false;
    if (!list.length) {
      var empty = document.createElement("p");
      empty.className = "mention-empty";
      empty.textContent = "No command";
      slashMenu.appendChild(empty);
      placePopMenu(slashMenu);
      return;
    }
    if (slashIndex >= list.length) slashIndex = 0;
    if (slashIndex < 0) slashIndex = list.length - 1;
    list.forEach(function (c, i) {
      var b = document.createElement("button");
      b.type = "button";
      b.className = "mention-option" + (i === slashIndex ? " active" : "");
      b.setAttribute("role", "option");
      var label = document.createElement("span");
      label.className = "mention-name";
      label.textContent = "/" + c.name;
      var id = document.createElement("span");
      id.className = "mention-id";
      id.textContent = c.description || c.plugin || "";
      b.appendChild(label);
      b.appendChild(id);
      b.addEventListener("mousedown", function (e) {
        e.preventDefault();
        applySlash(c);
      });
      slashMenu.appendChild(b);
    });
    placePopMenu(slashMenu);
  }

  function applySlash(cmd) {
    if (!cmd) return;
    msg.value = cmd.body || ("/" + cmd.name);
    fitComposer();
    closeSlashMenu();
    slashCommands = null;
    msg.focus();
    msg.setSelectionRange(msg.value.length, msg.value.length);
    msg.dispatchEvent(new Event("input"));
  }

  function placePopMenu(menu) {
    if (!menu || menu.hidden || !msg) return;
    if (menu.parentElement !== document.body) document.body.appendChild(menu);
    var box = msg.getBoundingClientRect();
    var width = Math.max(220, box.width);
    var left = Math.max(8, box.left);
    if (left + width > window.innerWidth - 8) left = Math.max(8, window.innerWidth - 8 - width);
    menu.style.width = width + "px";
    menu.style.left = left + "px";
    var gap = 6;
    var height = menu.offsetHeight || 0;
    var top = box.top - height - gap;
    if (top < 8) top = Math.min(box.bottom + gap, Math.max(8, window.innerHeight - height - 8));
    menu.style.top = Math.max(8, top) + "px";
  }

  function ensureMentionAgents() {
    if (mentionLoading) return;
    mentionLoading = true;
    Promise.resolve(loadAgents()).finally(function () {
      mentionLoading = false;
      updateMentionMenu();
    });
  }

  function quoteFileMention(path) {
    path = String(path || "");
    var clash = (agentsCache || []).some(function (a) {
      return String(a.name || "").toLowerCase() === path.toLowerCase();
    });
    if (clash) path = "./" + path;
    if (/[\s"]/.test(path)) return '"' + path.replace(/"/g, "") + '"';
    return path;
  }

  function applyMention(agent) {
    var active = activeMention(msg.value, msg.selectionStart);
    if (!active || !agent) return;
    var name = agent.kind === "file" ? quoteFileMention(agent.path || agent.name) : (agent.name || agent.id);
    var insert = "@" + name + " ";
    var caret = active.start + insert.length;
    msg.value = msg.value.slice(0, active.start) + insert + msg.value.slice(msg.selectionStart);
    fitComposer();
    msg.setSelectionRange(caret, caret);
    if (agent.kind !== "file") mentionPicks.push({ id: agent.id, name: agent.name || agent.id });
    mentionDismissed = null;
    closeMentionMenu();
    msg.focus();
  }

  if (msg) {
    msg.addEventListener("input", function () {
      mentionDismissed = null;
      fitComposer();
      if (activeMention(msg.value, msg.selectionStart)) ensureMentionAgents();
      updateMentionMenu();
      updateSlashMenu();
    });
    fitComposer();
    msg.addEventListener("keyup", function (e) {
      if (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Enter" || e.key === "Escape") return;
      updateMentionMenu();
    });
    msg.addEventListener("click", function () { updateMentionMenu(); });
  }
  window.addEventListener("resize", placeMentionMenu);
  var threadEl = document.getElementById("thread");
  if (threadEl) threadEl.addEventListener("scroll", placeMentionMenu);
  var mainEl = document.querySelector("main");
  if (mainEl) mainEl.addEventListener("scroll", placeMentionMenu);

  function renderAttachRow() {
    if (!attachRow) return;
    attachRow.innerHTML = "";
    attachRow.hidden = pendingFiles.length === 0;
    pendingFiles.forEach(function (f, i) {
      var chip = document.createElement("span");
      chip.className = "attach-chip";
      var label = document.createElement("span");
      label.textContent = f.error ? (f.name + " — " + f.error) : (f.uploading ? (f.name + "…") : f.name);
      var remove = document.createElement("button");
      remove.type = "button";
      remove.className = "ghost";
      remove.textContent = "Remove";
      remove.title = "Take this file off the message";
      remove.addEventListener("click", function () {
        pendingFiles.splice(i, 1);
        renderAttachRow();
      });
      chip.appendChild(label);
      chip.appendChild(remove);
      attachRow.appendChild(chip);
    });
  }

  function finishAttachment(slot, response) {
    response
      .then(function (r) {
        return r.text().then(function (t) {
          var j = {};
          try { j = t ? JSON.parse(t) : {}; } catch (e) {}
          if (!r.ok) throw new Error((j && j.message) || t || r.status);
          return j;
        });
      })
      .then(function (j) {
        slot.id = j.id;
        slot.name = j.name || slot.name;
        slot.uploading = false;
        renderAttachRow();
      })
      .catch(function (err) {
        slot.uploading = false;
        slot.error = String((err && err.message) || err || "upload failed");
        renderAttachRow();
      });
  }

  function beginAttachment(name) {
    if (pendingFiles.length >= 8) {
      appendLive("\n[files] Eight files is the limit for one message.\n");
      return null;
    }
    var slot = { id: "", name: name || "file", uploading: true };
    pendingFiles.push(slot);
    renderAttachRow();
    return slot;
  }

  function addAttachmentFile(file) {
    if (!file) return;
    var slot = beginAttachment(file.name || "file");
    if (!slot) return;
    var body = new FormData();
    body.append("file", file, file.name || "file");
    var h = {};
    var s = secret();
    if (s) h.Authorization = "Bearer " + s;
    finishAttachment(slot, fetch(apiURL("/api/attachments"), { method: "POST", headers: h, body: body }));
  }

  function droppedFileName(path) {
    var raw = String(path || "");
    try {
      if (raw.toLowerCase().indexOf("file:") === 0) raw = new URL(raw).pathname;
    } catch (e) { /* keep the raw text */ }
    try { raw = decodeURIComponent(raw); } catch (e) { /* keep the encoded text */ }
    var parts = raw.split("/").filter(Boolean);
    return parts.length ? parts[parts.length - 1] : "file";
  }

  var recentDropKeys = {};
  function dropKey(path) {
    var raw = String(path || "");
    try {
      if (raw.toLowerCase().indexOf("file:") === 0) raw = decodeURIComponent(new URL(raw).pathname);
    } catch (e) { /* keep the raw text */ }
    return raw;
  }
  function addAttachmentPath(path) {
    if (!path) return;
    var key = dropKey(path);
    var now = Date.now();
    if (recentDropKeys[key] && now - recentDropKeys[key] < 1500) return;
    recentDropKeys[key] = now;
    var slot = beginAttachment(droppedFileName(path));
    if (!slot) return;
    finishAttachment(slot, fetch(apiURL("/api/attachments"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ path: path }),
    }));
  }

  window.keimlingAcceptDrop = function (uri) {
    if (composerForm) composerForm.classList.remove("is-drop");
    addAttachmentPath(uri);
  };

  function fileDropURIs(dt) {
    var raw = "";
    try {
      raw = dt.getData("text/uri-list") || dt.getData("text/plain") || "";
    } catch (e) { raw = ""; }
    var out = [];
    String(raw).split(/\r?\n/).forEach(function (line) {
      line = line.trim();
      if (!line || line.charAt(0) === "#") return;
      if (line.toLowerCase().indexOf("file:") === 0) out.push(line);
    });
    return out;
  }

  function dropOffersFiles(dt) {
    if (!dt || !dt.types) return false;
    for (var i = 0; i < dt.types.length; i++) {
      var t = String(dt.types[i] || "").toLowerCase();
      if (t === "files" || t.indexOf("uri-list") >= 0 || t === "text/plain") return true;
    }
    return false;
  }

  function sendMessage(text) {
    text = String(text || "").trim();
    var files = pendingFiles.slice();
    if (!text && !files.length) return;
    if (!sendMessage.releasing) {
      send.disabled = true;
      Promise.resolve(modelSave).then(function () {
        sendMessage.releasing = true;
        try { sendMessage(text); } finally { sendMessage.releasing = false; }
      }, function () {
        send.disabled = false;
      });
      return;
    }
    send.disabled = true;
    if (files.some(function (f) { return f.uploading; })) {
      appendLive("\n[files] Wait for the upload to finish.\n");
      setState(agentState);
      return;
    }
    if (files.some(function (f) { return !f.id || f.error; })) {
      appendLive("\n[files] Remove the file that failed, then send again.\n");
      setState(agentState);
      return;
    }
    clearLive();
    closeMentionMenu();
    var wasBusy = agentState === "running" || agentState === "compacting" || agentState === "awaiting_approval";
    var otherThread = backgroundBusy;
    var picks = currentAgentInstanceID ? [] : mentionPicks.slice();
    fetch(apiURL("/api/message"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ text: text, to: picks, attachments: files.map(function (f) { return f.id; }) }),
    })
      .then(function (r) {
        return r.text().then(function (t) {
          var j = {};
          try { j = t ? JSON.parse(t) : {}; } catch (e) {}
          if (!r.ok) throw new Error(j.error || j.message || t || r.status);
          return j;
        });
      })
      .then(function (j) {
        if (j && j.delivered && j.delivered.length) {
          msg.value = "";
          fitComposer();
          mentionPicks = [];
          pendingFiles = [];
          renderAttachRow();
          var bodyText = j.text != null ? String(j.text) : text;
          var sentFiles = (j && j.files) || files.map(function (f) { return f.name; });
          var shown = currentThreadID;
          var fromID = (j && j.thread_id) || "";
          if (!fromID || fromID === "main") fromID = workspaceThreadID || currentThreadID;
          if (fromID && fromID !== "main") {
            currentThreadID = fromID;
            workspaceThreadID = fromID;
          }
          var threadTitle = "";
          (threadsCache || []).forEach(function (th) {
            if (th && th.id === fromID) threadTitle = th.title || th.name || "";
          });
          j.delivered.forEach(function (d) {
            messages.push({ role: "peer", name: "To " + (d.name || d.id), text: bodyText, files: sentFiles });
          });
          renderThread();
          setState(agentState);
          return Promise.all(j.delivered.map(function (d) {
            return fetch(apiURL("/api/agents/instances/message"), {
              method: "POST",
              headers: headers(),
              body: JSON.stringify({
                id: d.id,
                text: (j && j.wake) || bodyText,
                kind: "thread",
                from: fromID,
                from_name: threadTitle,
                workspace: currentWorkspaceID,
              }),
            }).then(function (r) {
              return r.text().then(function (t) {
                if (!r.ok) throw new Error(t || r.status);
              });
            });
          })).catch(function (err) {
            appendLive("\n[agents] The thread has the message. Waking the agent failed: " + err + "\n");
          }).then(function () {
            if (!fromID || fromID === shown) return;
            return loadSession().then(function () {
              renderThread();
              return loadThreads();
            });
          });
        }
        msg.value = "";
        fitComposer();
        mentionPicks = [];
        pendingFiles = [];
        renderAttachRow();
        messages.push({ role: "user", text: text, files: files.map(function (f) { return f.name; }) });
        if (j && j.queued) {
          messages.push({
            role: "system",
            text: otherThread
              ? "Queued in this thread — runs when this thread is free."
              : (wasBusy
                ? "Queued — steers the current run or runs when idle."
                : "Queued — will run when the agent is idle."),
          });
        } else if (j && j.realtime) {
          speakText(j.text || text);
        } else {
          if (agentState === "idle") resetSideSamples();
          setState("running");
        }
        renderThread();
        setState(agentState); // refresh Send/Queue label + enabled state
      })
      .catch(function (err) {
        var note = String((err && err.message) || err || "");
        if (note.indexOf("choose the agent instance") >= 0) {
          mentionDismissed = null;
          updateMentionMenu();
        }
        appendLive("\n[send] " + note + "\n");
        setState(agentState);
      });
  }

  // /api/stop reaches the worker's appServer both when the GUI is served directly
  // and when it is proxied through the host (/api/ prefix is proxied).
  function stopCurrentRun() {
    if (currentAgentInstanceID) {
      if (stoppingAgents[currentAgentInstanceID]) return;
      stopAgent(currentAgentInstanceID);
      appendLive("\n[stop] Stopping…\n");
      return;
    }
    var runID = runningThreadID || currentThreadID;
    if (!runID || stoppingThreadID === runID) return;
    stoppingThreadID = runID;
    setState(agentState);
    renderThreadListFromCache();
    appendLive("\n[stop] Stopping…\n");
    fetch(apiURL("/api/stop"), { method: "POST", headers: headers() })
      .then(function (r) {
        return r.text().then(function (t) {
          var j = {};
          try { j = t ? JSON.parse(t) : {}; } catch (e) {}
          if (!r.ok) throw new Error(j.error || j.message || t || r.status);
          return j;
        });
      })
      .catch(function (err) {
        if (stoppingThreadID === runID) stoppingThreadID = "";
        setState(agentState);
        renderThreadListFromCache();
        appendLive("\n[stop] " + err + "\n");
      });
  }

  var callPC = null;
  var callDC = null;
  var callAudio = null;
  var callMic = null;
  var callQueue = [];
  var callConfigured = false;
  var callSeeded = false;
  var callDCOpen = false;
  var callSessionReady = false;
  var callSetup = null;
  var callSpeak = 0;
  var callGen = 0;
  var callSeen = {};
  var callUserSaved = {};
  var assistantLive = "";
  var assistantSaved = false;
  var userLive = "";
  var callLimitTimer = 0;
  var callTransport = "";
  var callCapCtx = null;
  var callSpeakEl = null;
  var callAudioItem = "";
  var callAudioIndex = 0;
  var callRespAt = 0;
  var callUserItem = "";

  function startCall() {
    if (callLive || !realtimeReady) return;
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      setVoiceStatus("This window cannot open a call. The microphone is unavailable.");
      return;
    }
    var socketMode = typeof RTCPeerConnection !== "function";
    if (socketMode && typeof WebSocket !== "function") {
      setVoiceStatus("This window cannot open a call. WebSocket is unavailable.");
      return;
    }
    if (socketMode) primeCallSpeaker();
    callLive = true;
    callMuted = false;
    callSentAudio = false;
    callUserSpeaking = false;
    callRecovering = false;
    callRecoverCreate = false;
    callPicturePending = false;
    callConfigured = false;
    callSeeded = false;
    callDCOpen = false;
    callSessionReady = false;
    callGen += 1;
    var gen = callGen;
    setState(agentState);
    setVoiceStatus("Connecting…");
    navigator.mediaDevices.getUserMedia({
      audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true },
    }).then(function (stream) {
      if (gen !== callGen) {
        stream.getTracks().forEach(function (t) { t.stop(); });
        return;
      }
      callMic = stream;
      return fetch(apiURL("/api/realtime/session"), { method: "POST", headers: headers() })
        .then(readJSONResponse)
        .then(function (setup) {
          if (gen !== callGen) return;
          callSetup = setup;
          if (socketMode) return openCallSocket(setup, gen);
          return openCallPeer(setup, gen);
        });
    }).catch(function (err) {
      if (gen !== callGen) return;
      hangUpCall(String((err && err.message) || err || "Call failed"));
    });
  }

  function callSocketURL() {
    var u = new URL(apiURL("/api/realtime/ws"), window.location.href);
    u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
    var sec = pageAccessSecret();
    if (sec) u.searchParams.set("access_secret", sec);
    return u.toString();
  }

  function openCallSocket(setup, gen) {
    callTransport = "ws";
    var ws = new WebSocket(callSocketURL());
    var ch = {
      readyState: "connecting",
      send: function (s) { if (ws.readyState === 1) ws.send(s); },
      close: function () { try { ws.close(); } catch (e) {} },
    };
    callDC = ch;
    ws.onopen = function () {
      if (gen !== callGen) { try { ws.close(); } catch (e) {} return; }
      ch.readyState = "open";
      callDCOpen = true;
      maybeConfigureCall(gen);
      setTimeout(function () {
        if (gen !== callGen || callSessionReady) return;
        callSessionReady = true;
        maybeConfigureCall(gen);
      }, 1200);
      startPCMCapture(gen);
      setVoiceStatus("Live");
      setState(agentState);
      if (callLimitTimer) clearTimeout(callLimitTimer);
      callLimitTimer = setTimeout(function () {
        if (callLive) hangUpCall("The call ended after 60 minutes. The transcript stays in the chat.");
      }, 59 * 60 * 1000);
    };
    ws.onmessage = function (e) { onCallEvent(typeof e.data === "string" ? e.data : "", gen); };
    ws.onerror = function () {
      if (gen !== callGen) return;
      setVoiceStatus("The call could not connect.");
    };
    ws.onclose = function () {
      if (gen !== callGen) return;
      hangUpCall("The call disconnected.");
    };
    return Promise.resolve();
  }

  function resampleTo24k(input, rate) {
    if (!rate || rate === 24000) return input;
    var ratio = rate / 24000;
    var outLen = Math.max(0, Math.floor(input.length / ratio));
    var out = new Float32Array(outLen);
    for (var i = 0; i < outLen; i++) {
      var start = Math.floor(i * ratio);
      var end = Math.min(input.length, Math.floor((i + 1) * ratio));
      if (end <= start) end = Math.min(input.length, start + 1);
      var sum = 0;
      var n = 0;
      for (var j = start; j < end; j++) { sum += input[j]; n++; }
      out[i] = n ? sum / n : 0;
    }
    return out;
  }

  function pcm16Base64(samples) {
    var bytes = new Uint8Array(samples.length * 2);
    var view = new DataView(bytes.buffer);
    for (var i = 0; i < samples.length; i++) {
      var s = samples[i];
      if (s > 1) s = 1;
      if (s < -1) s = -1;
      view.setInt16(i * 2, s < 0 ? s * 0x8000 : s * 0x7fff, true);
    }
    var bin = "";
    for (var o = 0; o < bytes.length; o += 1024) {
      var slice = bytes.subarray(o, Math.min(bytes.length, o + 1024));
      bin += String.fromCharCode.apply(null, slice);
    }
    return btoa(bin);
  }

  function ensureCallAudio() {
    if (!window.AudioContext) return null;
    if (!callCapCtx) callCapCtx = new AudioContext();
    if (callCapCtx.resume) callCapCtx.resume();
    return callCapCtx;
  }

  function startPCMCapture(gen) {
    var ctx = ensureCallAudio();
    if (!callMic || !ctx || !ctx.createScriptProcessor) return;
    var source = ctx.createMediaStreamSource(callMic);
    var micProc = ctx.createScriptProcessor(4096, 1, 1);
    var mute = ctx.createGain();
    mute.gain.value = 0;
    source.connect(micProc);
    micProc.connect(mute);
    mute.connect(ctx.destination);
    micProc.onaudioprocess = function (e) {
      if (gen !== callGen || !callDC || callDC.readyState !== "open") return;
      var input = e.inputBuffer.getChannelData(0);
      var pcm = resampleTo24k(input, ctx.sampleRate);
      if (!pcm.length) return;
      if (callMuted) pcm = new Float32Array(pcm.length);
      else callSentAudio = true;
      callDC.send(JSON.stringify({ type: "input_audio_buffer.append", audio: pcm16Base64(pcm) }));
    };
  }

  function callSpeakerURL() {
    var u = new URL(apiURL("/api/realtime/speaker"), window.location.href);
    var sec = pageAccessSecret();
    if (sec) u.searchParams.set("access_secret", sec);
    u.searchParams.set("t", String(Date.now()));
    return u.toString();
  }

  function primeCallSpeaker() {
    if (!callSpeakEl) callSpeakEl = new Audio();
    callSpeakEl.preload = "auto";
    callSpeakEl.src = callSpeakerURL();
    var started = callSpeakEl.play();
    if (started && started.catch) started.catch(function (e) {
      setVoiceStatus("Playback: " + ((e && e.message) || e));
    });
  }

  function stopCallPlayback() {
    if (!callSpeakEl) return;
    callSpeakEl.pause();
    callSpeakEl.removeAttribute("src");
    try { callSpeakEl.load(); } catch (e) {}
  }

  function stopCallAudio() {
    stopCallPlayback();
    callSpeakEl = null;
    if (callCapCtx) { try { callCapCtx.close(); } catch (e) {} }
    callCapCtx = null;
  }

  function openCallPeer(setup, gen) {
    callTransport = "webrtc";
    var pc = new RTCPeerConnection();
    callPC = pc;
    callAudio = document.createElement("audio");
    callAudio.autoplay = true;
    pc.ontrack = function (e) { callAudio.srcObject = e.streams[0]; };
    callMic.getTracks().forEach(function (t) { pc.addTrack(t, callMic); });
    var dc = pc.createDataChannel("oai-events");
    callDC = dc;
    dc.onopen = function () {
      callDCOpen = true;
      maybeConfigureCall(gen);
      setTimeout(function () {
        if (gen !== callGen || callSessionReady) return;
        callSessionReady = true;
        maybeConfigureCall(gen);
      }, 1200);
    };
    dc.onmessage = function (e) { onCallEvent(e.data, gen); };
    return pc.createOffer().then(function (offer) {
      return pc.setLocalDescription(offer);
    }).then(function () {
      return waitForIce(pc);
    }).then(function () {
      if (gen !== callGen) return;
      return fetch(apiURL("/api/realtime/sdp"), {
        method: "POST",
        headers: headers(),
        body: JSON.stringify({ sdp: pc.localDescription.sdp }),
      }).then(readJSONResponse);
    }).then(function (j) {
      if (!j || gen !== callGen) return;
      return pc.setRemoteDescription({ type: "answer", sdp: j.sdp });
    }).then(function () {
      if (gen !== callGen) return;
      setVoiceStatus("Live");
      setState(agentState);
      if (callLimitTimer) clearTimeout(callLimitTimer);
      callLimitTimer = setTimeout(function () {
        if (callLive) hangUpCall("The call ended after 60 minutes. The transcript stays in the chat.");
      }, 59 * 60 * 1000);
    });
  }

  function waitForIce(pc) {
    if (pc.iceGatheringState === "complete") return Promise.resolve();
    return new Promise(function (resolve) {
      function done() {
        if (pc.iceGatheringState === "complete") {
          pc.removeEventListener("icegatheringstatechange", done);
          resolve();
        }
      }
      pc.addEventListener("icegatheringstatechange", done);
      setTimeout(function () {
        pc.removeEventListener("icegatheringstatechange", done);
        resolve();
      }, 2500);
    });
  }

  function maybeConfigureCall(gen) {
    if (gen !== callGen || !callSessionReady || !callDC || callDC.readyState !== "open" || !callSetup) return;
    var setup = callSetup;
    var audioIn = {
      transcription: { model: setup.transcription_model || "gpt-4o-mini-transcribe" },
      turn_detection: { type: "semantic_vad", eagerness: "medium", interrupt_response: true, create_response: true },
    };
    var audioOut = { voice: setup.voice || "marin" };
    if (callTransport === "ws") {
      audioIn.format = { type: "audio/pcm", rate: 24000 };
      audioOut.format = { type: "audio/pcm", rate: 24000 };
    }
    callDC.send(JSON.stringify({
      type: "session.update",
      session: {
        type: "realtime",
        instructions: setup.instructions || "",
        output_modalities: ["audio"],
        audio: { input: audioIn, output: audioOut },
        tools: setup.tools || [],
        tool_choice: "auto",
      },
    }));
    if (callSeeded) return;
    callSeeded = true;
    callConfigured = true;
    (setup.history || []).forEach(function (row) {
      if (!row || !row.text) return;
      var contentType = row.role === "assistant" ? "output_text" : "input_text";
      var role = row.role === "assistant" ? "assistant" : (row.role === "system" ? "system" : "user");
      callDC.send(JSON.stringify({
        type: "conversation.item.create",
        item: { type: "message", role: role, content: [{ type: contentType, text: row.text }] },
      }));
    });
    var queued = callQueue;
    callQueue = [];
    queued.forEach(function (text) { sendCallText(text); });
  }

  function speakText(text) {
    text = String(text || "").trim();
    if (!text || !callLive) return;
    if (!callConfigured || !callDC || callDC.readyState !== "open") {
      callQueue.push(text);
      return;
    }
    sendCallText(text);
  }

  function sendCallText(text) {
    if (!callDC || callDC.readyState !== "open") return;
    sendCallPicture();
    callDC.send(JSON.stringify({
      type: "conversation.item.create",
      item: { type: "message", role: "user", content: [{ type: "input_text", text: text }] },
    }));
    callDC.send(JSON.stringify({ type: "response.create" }));
  }

  function onCallEvent(raw, gen) {
    if (gen !== callGen) return;
    var ev = {};
    try { ev = JSON.parse(raw); } catch (e) { return; }
    var type = ev.type || "";
    if (type === "session.created" || type === "session.started") {
      callSessionReady = true;
      maybeConfigureCall(gen);
    }
    if (type === "error") {
      var msg = String((ev.error && (ev.error.message || ev.error.code)) || "Call error");
      if (isEmptyAudioError(msg) || isNoResponseError(msg)) return;
      if (isActiveResponseError(msg)) {
        if (callPictureAt && Date.now() - callPictureAt < 1500) {
          callPictureAt = 0;
          callPicturePending = true;
          return;
        }
        if (callRecovering || !callDC || callDC.readyState !== "open") return;
        callRecovering = true;
        callRecoverCreate = !callUserSpeaking;
        callDC.send(JSON.stringify({ type: "response.cancel" }));
        return;
      }
      setVoiceStatus(msg);
      return;
    }
    if (type === "response.created") {
      callRecovering = false;
      callRecoverCreate = false;
      callResponseSpeak = callSpeak;
      finishLiveLine("assistant");
      assistantLive = "";
      assistantSaved = false;
      callAudioItem = "";
      callRespAt = callSpeakEl && isFinite(callSpeakEl.currentTime) ? callSpeakEl.currentTime : 0;
    }
    if (ev.item_id && (type === "response.output_audio.delta" || type === "response.audio.delta" || type === "response.output_audio_transcript.delta")) {
      callAudioItem = ev.item_id;
      if (typeof ev.content_index === "number") callAudioIndex = ev.content_index;
    }
    if (type === "response.output_item.added" && ev.item && ev.item.id) callAudioItem = ev.item.id;
    if (type === "input_audio_buffer.speech_started") {
      callUserSpeaking = true;
      bargeInCall();
      if (!sendCallPicture()) callPicturePending = true;
    }
    if (type === "input_audio_buffer.speech_stopped") {
      callUserSpeaking = false;
      flushCallBgComment();
    }
    if (type === "response.cancelled") {
      if (callPicturePending && callUserSpeaking) sendCallPicture();
      var create = callRecoverCreate && !callUserSpeaking;
      callRecoverCreate = false;
      if (!create) callRecovering = false;
      if (create && callDC && callDC.readyState === "open") {
        callDC.send(JSON.stringify({ type: "response.create" }));
      }
    }
    if (type === "conversation.item.input_audio_transcription.delta") {
      if (ev.item_id && ev.item_id !== callUserItem) {
        finishLiveLine("user");
        callUserItem = ev.item_id;
        userLive = "";
      }
      var piece = ev.delta || "";
      if (!piece && ev.transcript) userLive = ev.transcript;
      else if (piece && userLive && piece.indexOf(userLive) === 0) userLive = piece;
      else userLive += piece;
      showCallLine("user", userLive, false);
    }
    if (type === "conversation.item.input_audio_transcription.completed") {
      if (ev.item_id) callUserItem = ev.item_id;
      saveUserTranscript(ev.item_id, ev.transcript || userLive);
    }
    if (type === "response.output_audio_transcript.delta") {
      assistantLive += ev.delta || "";
      showCallLine("assistant", assistantLive, false);
    }
    if (type === "response.output_audio_transcript.done") {
      assistantLive = ev.transcript || assistantLive;
      showCallLine("assistant", assistantLive, true);
      persistCallLine("assistant", assistantLive);
      assistantSaved = true;
      assistantLive = "";
    }
    if (type === "response.done") {
      var resp = ev.response || {};
      if (!assistantSaved) {
        var said = "";
        (resp.output || []).forEach(function (item) {
          (item.content || []).forEach(function (part) {
            if (part.transcript) said = part.transcript;
            if (part.type === "output_text" && part.text) said = part.text;
          });
        });
        if (said) {
          showCallLine("assistant", said, true);
          persistCallLine("assistant", said);
        }
      }
      var calls = [];
      (resp.output || []).forEach(function (item) {
        if (item && item.type === "function_call" && item.call_id && !callSeen[item.call_id]) calls.push(item);
      });
      if (calls.length && callSpeak === callResponseSpeak) runCallTools(calls, gen);
      else flushCallBgComment();
    }
  }

  function saveUserTranscript(itemID, text) {
    text = String(text || "").trim();
    if (!text) return;
    if (itemID && callUserSaved[itemID]) return;
    if (itemID) callUserSaved[itemID] = true;
    var last = messages[messages.length - 1];
    if (last && last.role === "user" && last.live && String(last.text || "").trim() === text && last.liveDone) return;
    userLive = "";
    showCallLine("user", text, true);
    persistCallLine("user", text);
  }

  function finishLiveLine(role) {
    for (var i = messages.length - 1; i >= 0; i--) {
      var m = messages[i];
      if (m.live && m.role === role && !m.liveDone) {
        m.liveDone = true;
        m.streaming = false;
        return;
      }
    }
  }

  function showCallLine(role, text, done) {
    text = String(text || "");
    if (!text.trim()) return;
    for (var i = messages.length - 1; i >= 0; i--) {
      var m = messages[i];
      if (m.live && m.role === role && !m.liveDone) {
        m.text = text;
        m.streaming = !done;
        m.liveDone = !!done;
        renderThread();
        return;
      }
    }
    messages.push({ role: role, text: text, live: true, streaming: !done, liveDone: !!done });
    renderThread();
  }

  function persistCallLine(role, text) {
    text = String(text || "").trim();
    if (!text) return;
    fetch(apiURL("/api/realtime/line"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ role: role, text: text }),
    }).catch(function (err) {
      setVoiceStatus(String((err && err.message) || err || "Could not save the transcript"));
    });
  }

  function isEmptyAudioError(msg) {
    return /empty|buffer too small|audio is too short|commit.*buffer/i.test(msg);
  }

  function isNoResponseError(msg) {
    return /no active response|no response is active|response not active/i.test(msg);
  }

  function isActiveResponseError(msg) {
    return /active response|already has an active response|response in progress/i.test(msg);
  }

  function commitCallInput() {
    if (!callDC || callDC.readyState !== "open") return;
    if (callTransport === "ws" && !callSentAudio) return;
    callSentAudio = false;
    callDC.send(JSON.stringify({ type: "input_audio_buffer.commit" }));
  }

  function flushCallBgComment() {
    if (!callBgComment || !callLive || !callDC || callDC.readyState !== "open") return;
    if (callUserSpeaking || assistantLive) return;
    var text = callBgComment;
    callBgComment = "";
    callDC.send(JSON.stringify({
      type: "conversation.item.create",
      item: { type: "message", role: "user", content: [{ type: "input_text", text: text }] },
    }));
    callDC.send(JSON.stringify({ type: "response.create" }));
  }

  function runCallTools(calls, gen) {
    var jobs = [];
    (calls || []).forEach(function (item) {
      if (!item || !item.call_id || callSeen[item.call_id]) return;
      callSeen[item.call_id] = true;
      jobs.push({ item: item, done: false, output: "", released: false });
    });
    if (!jobs.length || gen !== callGen || !callDC || callDC.readyState !== "open") return;
    var graceOver = false;
    var replied = false;
    function sendOutput(job, output) {
      if (gen !== callGen || !callDC || callDC.readyState !== "open") return;
      job.released = true;
      callDC.send(JSON.stringify({
        type: "conversation.item.create",
        item: { type: "function_call_output", call_id: job.item.call_id, output: output },
      }));
    }
    function release() {
      if (replied || gen !== callGen || !callDC || callDC.readyState !== "open") return;
      var pending = jobs.filter(function (job) { return !job.done; });
      if (pending.length && !graceOver) return;
      replied = true;
      var later = [];
      jobs.forEach(function (job) {
        if (job.done) sendOutput(job, job.output);
        else {
          sendOutput(job, "Still running. Keep the conversation going. Do not invent the result. It arrives in a later message.");
          later.push(job);
        }
      });
      callDC.send(JSON.stringify({ type: "response.create" }));
      laterLeft = later.length;
      later.forEach(function (job) { if (job.done) noteLater(job); });
    }
    var laterLeft = 0;
    var laterLines = [];
    function noteLater(job) {
      if (!job.released || !job.done || job.noted || gen !== callGen) return;
      job.noted = true;
      laterLines.push((job.item.name || "tool") + ":\n" + (job.output || ""));
      laterLeft--;
      if (laterLeft > 0) return;
      callBgComment = "Background tools finished. Comment on these results before you treat the topic as done.\n\n" + laterLines.join("\n\n");
      flushCallBgComment();
    }
    jobs.forEach(function (job) {
      fetch(apiURL("/api/realtime/tool"), {
        method: "POST",
        headers: headers(),
        body: JSON.stringify({ name: job.item.name, call_id: job.item.call_id, arguments: job.item.arguments || "{}" }),
      }).then(readJSONResponse).then(function (j) {
        job.done = true;
        job.output = (j && j.output) || "";
        if (job.released) noteLater(job);
        else release();
      }).catch(function (err) {
        job.done = true;
        job.output = "INTERRUPTED: " + String((err && err.message) || err);
        if (job.released) noteLater(job);
        else release();
      });
    });
    setTimeout(function () {
      graceOver = true;
      release();
    }, 5000);
  }

  function truncateHeardAudio() {
    if (callTransport !== "ws" || !callAudioItem || !callDC || callDC.readyState !== "open") return;
    var played = 0;
    if (callSpeakEl && isFinite(callSpeakEl.currentTime)) played = Math.max(0, callSpeakEl.currentTime - (callRespAt || 0));
    callDC.send(JSON.stringify({
      type: "conversation.item.truncate",
      item_id: callAudioItem,
      content_index: callAudioIndex || 0,
      audio_end_ms: Math.max(0, Math.round(played * 1000)),
    }));
    callAudioItem = "";
  }

  // The server stops generating when you speak. This window has to stop the
  // file it already buffered, or the rest of the sentence keeps playing.
  function bargeInCall() {
    truncateHeardAudio();
    if (callTransport !== "ws") return;
    stopCallPlayback();
    callRespAt = 0;
    if (callLive) primeCallSpeaker();
  }

  function cancelCallSpeech() {
    callSpeak += 1;
    truncateHeardAudio();
    if (callDC && callDC.readyState === "open") callDC.send(JSON.stringify({ type: "response.cancel" }));
    stopCallPlayback();
    if (callTransport === "ws" && callLive) primeCallSpeaker();
    fetch(apiURL("/api/stop"), { method: "POST", headers: headers() }).catch(function () {});
  }

  function toggleCallMute() {
    if (!callMic) return;
    callMuted = !callMuted;
    callMic.getAudioTracks().forEach(function (t) { t.enabled = !callMuted; });
    if (callMuted) commitCallInput();
    setVoiceStatus(callMuted ? "Muted" : "Live");
    setState(agentState);
  }

  function hangUpCall(note) {
    callGen += 1;
    callLive = false;
    callMuted = false;
    callSentAudio = false;
    callUserSpeaking = false;
    callRecovering = false;
    callRecoverCreate = false;
    callPicturePending = false;
    callConfigured = false;
    callSeeded = false;
    callDCOpen = false;
    callSessionReady = false;
    callSetup = null;
    callQueue = [];
    callSeen = {};
    callUserSaved = {};
    assistantLive = "";
    userLive = "";
    if (callLimitTimer) clearTimeout(callLimitTimer);
    callLimitTimer = 0;
    callTransport = "";
    screenCallItem = "";
    stopCallAudio();
    if (callDC) { try { callDC.close(); } catch (e) {} }
    if (callPC) { try { callPC.close(); } catch (e) {} }
    if (callMic) callMic.getTracks().forEach(function (t) { t.stop(); });
    if (callAudio) callAudio.srcObject = null;
    callDC = null;
    callPC = null;
    callMic = null;
    callAudio = null;
    fetch(apiURL("/api/realtime/end"), { method: "POST", headers: headers() }).catch(function () {});
    setVoiceStatus(note || "");
    setState(agentState);
  }

  var composerForm = document.getElementById("composer");
  composerForm.addEventListener("submit", function (e) {
    e.preventDefault();
    if (recording) {
      stopRecording(true);
      return;
    }
    sendMessage(msg.value);
  });
  if (attachFileBtn && attachInput) {
    attachFileBtn.addEventListener("click", function () { attachInput.click(); });
    attachInput.addEventListener("change", function () {
      Array.prototype.forEach.call(attachInput.files || [], addAttachmentFile);
      attachInput.value = "";
    });
  }
  function takeComposerDrop(e) {
    var dt = e.dataTransfer;
    if (!dt) return;
    var uris = fileDropURIs(dt);
    var files = [];
    if (dt.files) {
      Array.prototype.forEach.call(dt.files, function (f) {
        if (f && f.size > 0) files.push(f);
      });
    }
    if (!uris.length && !files.length) return;
    e.preventDefault();
    e.stopPropagation();
    if (composerForm) composerForm.classList.remove("is-drop");
    if (uris.length) {
      uris.forEach(addAttachmentPath);
      return;
    }
    files.forEach(addAttachmentFile);
  }
  [composerForm, document.getElementById("thread")].forEach(function (el) {
    if (!el) return;
    el.addEventListener("dragover", function (e) {
      if (!dropOffersFiles(e.dataTransfer)) return;
      e.preventDefault();
      if (e.dataTransfer) e.dataTransfer.dropEffect = "copy";
      if (composerForm) composerForm.classList.add("is-drop");
    });
    el.addEventListener("dragleave", function () {
      if (composerForm) composerForm.classList.remove("is-drop");
    });
    el.addEventListener("drop", takeComposerDrop);
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
  if (stopBtn) stopBtn.addEventListener("click", function () {
    if (callLive) cancelCallSpeech();
    var busyish = agentState === "running" || agentState === "compacting" || agentState === "awaiting_approval";
    if (busyish || backgroundBusy) stopCurrentRun();
  });
  var screenBtn = document.getElementById("screen-share");
  var screenPreview = document.getElementById("screen-preview");
  var screenFrame = document.getElementById("screen-frame");
  var screenDismiss = document.getElementById("screen-dismiss");
  var screenBusy = false;
  var screenAbort = null;
  var screenObjectURL = "";
  var screenSaw = false;
  var screenGen = 0;
  var screenLatest = null;
  var screenCallItem = "";

  function concatBytes(a, b) {
    var out = new Uint8Array(a.length + b.length);
    if (a.length) out.set(a, 0);
    if (b.length) out.set(b, a.length);
    return out;
  }

  function scaledScreenDataURL(edge, quality) {
    var img = screenFrame;
    if (!img || !img.complete || !img.naturalWidth || !img.naturalHeight) return "";
    var scale = Math.min(1, edge / Math.max(img.naturalWidth, img.naturalHeight));
    var canvas = document.createElement("canvas");
    canvas.width = Math.max(1, Math.round(img.naturalWidth * scale));
    canvas.height = Math.max(1, Math.round(img.naturalHeight * scale));
    var ctx = canvas.getContext("2d");
    if (!ctx) return "";
    ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    try {
      return canvas.toDataURL("image/jpeg", quality);
    } catch (e) {
      return "";
    }
  }

  var screenTarget = "";
  var screenInput = false;

  function screenShareNote() {
    var note = "Live screen share. This is the current frame and it replaces the previous one.";
    if (screenInput) note += " Keyboard and mouse are shared. screen_click, screen_scroll, screen_key, and screen_type act on this picture. x and y are fractions of the picture, from 0 at the left or top to 1 at the right or bottom.";
    return note;
  }

  function paintScreenInput() {
    var btn = document.getElementById("screen-input");
    if (!btn) return;
    btn.hidden = !screenSaw;
    btn.setAttribute("aria-pressed", screenInput ? "true" : "false");
    btn.textContent = screenInput ? "Stop keyboard and mouse" : "Keyboard and mouse";
    btn.title = screenInput ? "Stop sharing the keyboard and mouse. The picture stays." : "Also share the keyboard and mouse";
  }

  function setScreenStatus(text) {
    var el = document.getElementById("screen-status");
    if (el) el.textContent = text || "";
  }

  function revealScreenShare(text) {
    var block = document.getElementById("side-share");
    if (block) block.hidden = false;
    if (text) setScreenStatus(text);
    selectSidebar("now");
  }

  function screenImageDataURL() {
    if (!screenLatest || !screenLatest.length) return "";
    var edge = screenInput ? 1280 : 768;
    var quality = 0.6;
    var limit = screenInput ? 180000 : 100000;
    var url = "";
    for (var attempt = 0; attempt < 3; attempt++) {
      url = scaledScreenDataURL(edge, quality);
      if (url && url.length <= limit) return url;
      edge = Math.round(edge * 0.7);
      quality = 0.5;
    }
    if (url && url.length <= 220000) return url;
    return "";
  }

  function forgetCallPicture() {
    if (!screenCallItem || !callDC || callDC.readyState !== "open") {
      screenCallItem = "";
      return;
    }
    var id = screenCallItem;
    screenCallItem = "";
    callDC.send(JSON.stringify({ type: "conversation.item.delete", item_id: id }));
  }

  function sendCallPicture() {
    if (!callLive || !callDC || callDC.readyState !== "open") return false;
    var picture = screenImageDataURL();
    if (!picture) return false;
    forgetCallPicture();
    callPicturePending = false;
    callPictureAt = Date.now();
    screenCallItem = "screen_" + Date.now();
    callDC.send(JSON.stringify({
      type: "conversation.item.create",
      item: {
        id: screenCallItem,
        type: "message",
        role: "user",
        content: [
          { type: "input_text", text: screenShareNote() },
          { type: "input_image", image_url: picture },
        ],
      },
    }));
    return true;
  }

  function showScreenJpeg(jpeg) {
    screenLatest = jpeg;
    var url = URL.createObjectURL(new Blob([jpeg], { type: "image/jpeg" }));
    if (screenObjectURL) URL.revokeObjectURL(screenObjectURL);
    screenObjectURL = url;
    if (screenFrame) screenFrame.src = url;
    if (screenPreview) screenPreview.hidden = false;
    if (!screenSaw) {
      screenSaw = true;
      setScreenStatus(screenInput ? "Sharing keyboard and mouse." : "Sharing.");
    }
    paintScreenInput();
  }

  function readScreenFrames(reader) {
    var buf = new Uint8Array(0);
    function take() {
      var last = null;
      while (buf.length >= 4) {
        var n = buf[0] | (buf[1] << 8) | (buf[2] << 16) | (buf[3] << 24);
        if (n <= 0 || n > 8000000) throw new Error("Screen share sent a bad picture.");
        if (buf.length < 4 + n) break;
        last = new Uint8Array(n);
        last.set(buf.subarray(4, 4 + n));
        buf = buf.subarray(4 + n);
      }
      if (last) showScreenJpeg(last);
    }
    function pump() {
      return reader.read().then(function (res) {
        if (res.done) {
          if (!screenSaw) throw new Error("Screen share produced no picture.");
          return;
        }
        buf = concatBytes(buf, res.value);
        take();
        return pump();
      });
    }
    return pump();
  }

  function stopScreenShare() {
    forgetCallPicture();
    screenGen++;
    if (screenAbort) screenAbort.abort();
    screenAbort = null;
    screenBusy = false;
    screenInput = false;
    paintScreenInput();
    if (screenObjectURL) URL.revokeObjectURL(screenObjectURL);
    screenObjectURL = "";
    screenSaw = false;
    screenLatest = null;
    if (screenFrame) screenFrame.removeAttribute("src");
    if (screenPreview) screenPreview.hidden = true;
    var shareBlock = document.getElementById("side-share");
    if (shareBlock) shareBlock.hidden = true;
    setScreenStatus("");
    setVoiceStatus("");
    setState(agentState);
  }

  function askScreenTarget(targets) {
    return new Promise(function (resolve) {
      var dlg = document.getElementById("screen-dialog");
      var list = document.getElementById("screen-targets");
      if (!dlg || !list || typeof dlg.showModal !== "function") {
        resolve(targets[0] ? targets[0].id : "");
        return;
      }
      while (list.firstChild) list.removeChild(list.firstChild);
      targets.forEach(function (t) {
        var b = document.createElement("button");
        b.type = "button";
        b.className = "ghost";
        b.textContent = t.name || t.id;
        b.addEventListener("click", function () {
          dlg.close("pick");
          resolve(t.id);
        });
        list.appendChild(b);
      });
      function done(ev) {
        dlg.removeEventListener("close", done);
        if (!ev || dlg.returnValue !== "pick") resolve("");
      }
      dlg.addEventListener("close", done);
      dlg.returnValue = "";
      dlg.showModal();
    });
  }

  function shareScreen(opts) {
    if (!opts || typeof opts.picked === "undefined") opts = {};
    var wantInput = !!opts.input;
    var target = opts.target || "";
    if (!opts.picked) {
      fetch(apiURL("/api/screen/targets"), { headers: headers() }).then(function (r) {
        if (!r.ok) throw new Error("HTTP " + r.status);
        return r.json();
      }).then(function (info) {
        if (info && info.chooser === "list") {
          var targets = info.targets || [];
          if (!targets.length) {
            revealScreenShare("No screen or window to share.");
            return;
          }
          return askScreenTarget(targets).then(function (id) {
            if (!id) {
              revealScreenShare("Screen share was cancelled.");
              return;
            }
            shareScreen({ input: wantInput, target: id, picked: true });
          });
        }
        if (info && info.chooser === "unavailable") {
          revealScreenShare("Screen share is not available on this system.");
          return;
        }
        shareScreen({ input: wantInput, target: "", picked: true });
      }).catch(function (err) {
        revealScreenShare("Screen: " + ((err && err.message) || err || "failed"));
      });
      return;
    }
    stopScreenShare();
    if (!window.fetch || !window.ReadableStream) {
      revealScreenShare("This window cannot show a live picture.");
      return;
    }
    screenTarget = target;
    screenInput = wantInput;
    paintScreenInput();
    var gen = ++screenGen;
    screenAbort = new AbortController();
    var signal = screenAbort.signal;
    screenBusy = true;
    screenSaw = false;
    setState(agentState);
    revealScreenShare(opts.note || (wantInput ? "Allow keyboard and mouse in the system dialog." : "Choose a screen or window."));
    var q = [];
    if (wantInput) q.push("input=1");
    if (target) q.push("target=" + encodeURIComponent(target));
    fetch(apiURL("/api/screen/share" + (q.length ? "?" + q.join("&") : "")), { method: "POST", headers: headers(), signal: signal }).then(function (r) {
      if (gen !== screenGen || signal.aborted) return;
      if (r.status === 409) {
        var cancel = new Error("cancelled");
        cancel.name = "AbortError";
        throw cancel;
      }
      if (!r.ok) return r.text().then(function (t) { throw new Error(t || ("HTTP " + r.status)); });
      if (!r.body || !r.body.getReader) throw new Error("This window cannot show a live picture.");
      setScreenStatus(wantInput ? "Sharing keyboard and mouse." : "Sharing.");
      return readScreenFrames(r.body.getReader());
    }).catch(function (err) {
      if (gen !== screenGen || signal.aborted) return;
      var name = err && err.name;
      var msg = (err && err.message) || err || "failed";
      if (name === "AbortError" || name === "NotAllowedError") revealScreenShare("Screen share was cancelled.");
      else if (wantInput) {
        revealScreenShare("Keyboard and mouse: " + msg);
        shareScreen({ input: false, target: target, picked: true, note: "Keyboard and mouse: " + msg });
      } else revealScreenShare("Screen: " + msg);
    }).then(function () {
      if (gen !== screenGen) return;
      if (!signal.aborted && screenSaw) setScreenStatus("Screen share stopped.");
      screenBusy = false;
      screenAbort = null;
      setState(agentState);
    });
  }

  if (screenBtn) screenBtn.addEventListener("click", function () {
    var block = document.getElementById("side-share");
    if (block && !block.hidden && (screenBusy || screenSaw)) {
      selectSidebar("now");
      return;
    }
    shareScreen();
  });
  var screenInputBtn = document.getElementById("screen-input");
  if (screenInputBtn) screenInputBtn.addEventListener("click", function () {
    shareScreen({ input: !screenInput, target: screenTarget, picked: true });
  });
  if (screenDismiss) screenDismiss.addEventListener("click", stopScreenShare);
  if (callBtn) callBtn.addEventListener("click", function () {
    if (callLive) hangUpCall();
    else startCall();
  });
  if (callMuteBtn) callMuteBtn.addEventListener("click", function () { toggleCallMute(); });
  if (setRealtimeProvider) setRealtimeProvider.addEventListener("change", function () {
    var on = !!setRealtimeProvider.value;
    if (setRealtimeModel) setRealtimeModel.disabled = !on;
    if (setRealtimeVoice) setRealtimeVoice.disabled = !on;
  });

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
        fitComposer();
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
      setState(agentState);
      stopTracks();
      voiceSendPending = false;
      return;
    }
    mediaRecorder.stop();
  }

  function canUseVoice() {
    // Allowed while the agent is busy — transcribed text can be queued/steered like typed input.
    return connected && voiceOK;
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

    if (slashMenu && !slashMenu.hidden && (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Escape" || (e.key === "Enter" && slashItems.length))) {
      e.preventDefault();
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        slashIndex += e.key === "ArrowDown" ? 1 : -1;
        updateSlashMenu();
        return;
      }
      if (e.key === "Escape") {
        closeSlashMenu();
        return;
      }
      applySlash(slashItems[slashIndex]);
      return;
    }

    if (mentionMenu && !mentionMenu.hidden && (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Escape" || (e.key === "Enter" && mentionItems.length))) {
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault();
        mentionIndex += e.key === "ArrowDown" ? 1 : -1;
        updateMentionMenu();
        return;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        var dismissed = activeMention(msg.value, msg.selectionStart);
        mentionDismissed = dismissed ? dismissed.query : "";
        closeMentionMenu();
        return;
      }
      var chosen = mentionItems[mentionIndex];
      var typing = activeMention(msg.value, msg.selectionStart);
      var exact = chosen && typing && String(chosen.name || "").toLowerCase() === typing.query.trim().toLowerCase();
      if (chosen && !exact) {
        e.preventDefault();
        applyMention(chosen);
        return;
      }
    }

      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        if (msg.value.trim() === "" && !pendingFiles.length && canUseVoice()) {
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
      openSession({ auto: true, stay: !sameOriginLocal(), unlock: true });
      return;
    }

    // Returning visitor with stored secret for this origin → try resume.
    if (prefs.seen && prefs.secret && !prefs.url) {
      secretEl.value = prefs.secret;
      if (loginSecretEl) loginSecretEl.value = prefs.secret;
      setChoice("local");
      openSession({ auto: true, stay: true, unlock: true });
      return;
    }
    if (prefs.seen && prefs.secret && prefs.url) {
      secretEl.value = prefs.secret;
      connectURLEl.value = prefs.url;
      setChoice("remote");
      openSession({ auto: true });
      return;
    }

    location.replace("/");
  })();
  if (loginContinue) loginContinue.addEventListener("click", unlockServer);
  if (loginSecretEl) {
    loginSecretEl.addEventListener("keydown", function (e) {
      if (e.key !== "Enter") return;
      e.preventDefault();
      unlockServer();
    });
  }
})();

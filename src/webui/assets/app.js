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
  var mcpToolDialog = document.getElementById("mcp-tool-dialog");
  var mcpToolName = "";
  var mcpList = document.getElementById("mcp-list");
  var mcpName = document.getElementById("mcp-name");
  var mcpKind = document.getElementById("mcp-kind");
  var mcpCommand = document.getElementById("mcp-command");
  var mcpCwd = document.getElementById("mcp-cwd");
  var mcpCwdRow = document.getElementById("mcp-cwd-row");
  var mcpLogTimer = 0;
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
  var STORAGE_COMPOSER_PAGE = "keimling.composer_page";
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
  var CHAT_MIN_PX = 600;
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
  var paneSpend = document.getElementById("pane-spend");
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
  var credEditSecretKey = document.getElementById("cred-edit-secret-key");
  var credEditInjectLabel = document.getElementById("cred-edit-inject-label");
  var credEditInjectEnvLabel = document.getElementById("cred-edit-inject-env-label");
  var credEditUse = document.getElementById("cred-edit-use");
  var credEditServiceHint = document.getElementById("cred-edit-service-hint");
  var credEditToolsHint = document.getElementById("cred-edit-tools-hint");
  var credEditSecretTitle = document.getElementById("cred-edit-secret-title");
  var credEditSecretHint = document.getElementById("cred-edit-secret-hint");
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
  // Painted conversation. A snapshot replaces the transcript when this changes,
  // including when both conversations are empty or their text happens to match.
  var shownViewKey = "";
  var sessionEpoch = 0;
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
  var setApprovalPolicy = document.getElementById("set-approval-policy");
  var setToolAsk = document.getElementById("set-tool-ask");
  var setArtifactProxy = document.getElementById("set-artifact-proxy");
  var editAutopilotText = document.getElementById("edit-autopilot-text");
  var editReviewText = document.getElementById("edit-review-text");
  var instructionEditor = document.getElementById("instruction-editor");
  var instructionEditorTitle = document.getElementById("instruction-editor-title");
  var instructionEditorText = document.getElementById("instruction-editor-text");
  var instructionEditorErr = document.getElementById("instruction-editor-err");
  var instructionEditorSave = document.getElementById("instruction-editor-save");
  var instructionEditorFull = document.getElementById("instruction-editor-full");
  var instructionKind = "";
  var approvalPolicyBaseline = "";
  // Drafts belong to the chat they were typed in: an agent chat is an instance,
  // a workspace chat is a thread. draftChatKey names the chat in the composer.
  var chatDrafts = {};
  var draftChatKey = "";
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
  var setMCPParallel = document.getElementById("set-mcp-parallel");
  var setContextLimit = document.getElementById("set-context-limit");
  var setVoiceModel = document.getElementById("set-voice-model");
  var setAuxiliaryModel = document.getElementById("set-auxiliary-model");
  var setMainModel = document.getElementById("set-main-model");
  var setVisionModel = document.getElementById("set-vision-model");
  var setSpeechModel = document.getElementById("set-speech-model");
  var setIdleSound = document.getElementById("set-idle-sound");
  var setIdleNotify = document.getElementById("set-idle-notify");
  var idleSoundOn = false;
  var idleNotifyOn = false;
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
  var modelFormOriginalProvider = "";
  var catalogCache = [];
  var catalogsByProvider = {};
  var providerListEl = document.getElementById("provider-list");
  var providerEditor = document.getElementById("provider-editor");
  var providerNewBtn = document.getElementById("provider-new");
  var provEditName = document.getElementById("prov-edit-name");
  var provEditBaseURL = document.getElementById("prov-edit-base-url");
  var provEditKind = document.getElementById("prov-edit-kind");
  var provEditAPIKey = document.getElementById("prov-edit-api-key");
  var provEditManagement = document.getElementById("prov-edit-management-key");
  var provEditManagementLabel = document.getElementById("prov-edit-management-label");
  var provEditManagementHint = document.getElementById("prov-edit-management-hint");
  var provEditCancel = document.getElementById("prov-edit-cancel");
  var provEditSave = document.getElementById("prov-edit-save");
  var provModelList = document.getElementById("prov-model-list");
  var provModelNew = document.getElementById("prov-model-new");
  var provModelEditor = document.getElementById("prov-model-editor");
  var provModelEditorTitle = document.getElementById("prov-model-editor-title");
  var provModelProvider = document.getElementById("prov-model-provider");
  var provModelID = document.getElementById("prov-model-id");
  var provModelName = document.getElementById("prov-model-name");
  var provModelContext = document.getElementById("prov-model-context");
  var provModelReasoning = document.getElementById("prov-model-reasoning");
  var provModelPromptPrice = document.getElementById("prov-model-prompt-price");
  var provModelCompletionPrice = document.getElementById("prov-model-completion-price");
  var provModelInstructions = document.getElementById("prov-model-instructions");
  var provModelTranscribes = document.getElementById("prov-model-transcribes");
  var provModelSpeaks = document.getElementById("prov-model-speaks");
  var provModelRealtime = document.getElementById("prov-model-realtime");
  var provModelImages = document.getElementById("prov-model-images");
  var provModelAudio = document.getElementById("prov-model-audio");
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
  var audioDownloadBtn = document.getElementById("audio-download");
  var audioUploadBtn = document.getElementById("audio-upload");
  var audioInput = document.getElementById("audio-input");
  var callBtn = document.getElementById("call");
  var callMuteBtn = document.getElementById("call-mute");
  var callShareBtn = document.getElementById("call-share");
  var dictateDraft = document.getElementById("dictate-draft");
  var dictateMuteBtn = document.getElementById("dictate-mute");
  var dictateCancelBtn = document.getElementById("dictate-cancel");
  var dictateSendBtn = document.getElementById("dictate-send");
  var composerNarrow = window.matchMedia("(max-width: 840px)");
  var composerMobile = composerNarrow.matches;
  var composerPageId = "text";
  var composerPageThread = "";
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
  var lowContextBtn = document.getElementById("low-context");
  var lowContextOn = false;
  var followThinkingBtn = document.getElementById("follow-thinking");
  var followThinking = false;
  var followDismissedKey = "";
  var applyingFollow = false;
  try { followThinking = localStorage.getItem("keimling.follow_thinking") === "1"; } catch (e) {}
  var voiceStatus = document.getElementById("voice-status");

  var es = null;
  var backendStartedAt = "";
  var backendPoll = 0;
  var messages = [];
  var chapters = [];
  var activeChapter = null;
  var runningThreadID = "";
  var backgroundBusy = false;
  var transcriptSeq = 0;
  var heldOpen = null;
  // Detail rows that already existed when the running turn began. The idle
  // transition closes that run's rows and leaves these alone.
  var runFoldKeys = null;
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
  var lastRecording = null;
  var recording = false;
  var voiceSendPending = false;
  var voiceTarget = "msg";
  var voiceDiscard = false;
  var voiceWorking = false;
  var dictateMuted = false;

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

  var growthLogoOn = false;
  var growthLogoReady = false;
  var growthPose = { rest: 1, p: 0 };
  var growthTarget = { rest: 1, p: 0 };
  var growthPassSeed = false;
  var growthFrame = 0;
  var growthReducedMotion = function () { return false; };

  function growthEase(u) {
    if (u <= 0) return 0;
    if (u >= 1) return 1;
    return u < 0.5 ? 2 * u * u : 1 - Math.pow(-2 * u + 2, 2) / 2;
  }
  function growthProgress(percent) {
    var p = Number(percent);
    if (!isFinite(p)) p = 0;
    if (p < 0) p = 0;
    if (p > 100) p = 100;
    return Math.log1p(9 * p / 100) / Math.log1p(9);
  }
  function growthThreadEmpty() {
    for (var i = 0; i < messages.length; i++) {
      if (messages[i].role === "user" && String(messages[i].text || "").trim()) return false;
    }
    return true;
  }
  function growthSeed(p) {
    if (p <= 0) return { op: 1, scale: 1 };
    if (p >= 0.25) return { op: 0, scale: 0.5 };
    var u = growthEase(p / 0.25);
    return { op: 1 - u, scale: 1 - 0.5 * u };
  }
  function growthBranch(p) {
    if (p <= 0.45) return 1;
    if (p >= 0.7) return 0;
    return 1 - growthEase((p - 0.45) / 0.25);
  }
  function growthFruit(p) {
    if (p <= 0.7) return 0;
    return growthEase((p - 0.7) / 0.3);
  }
  function paintGrowth(pose) {
    var logo = document.getElementById("growth-logo");
    if (!logo) return;
    var rest = pose.rest;
    var p = pose.p;
    if (rest < 0) rest = 0;
    if (rest > 1) rest = 1;
    if (p < 0) p = 0;
    if (p > 1) p = 1;
    var seed = growthSeed(p);
    var plant = 0.025 + p * 0.975;
    function mix(live, still) { return live + (still - live) * rest; }
    var seedEl = logo.querySelector(".kg-seed");
    var plantEl = logo.querySelector(".kg-plant");
    if (seedEl) {
      seedEl.style.opacity = String(mix(seed.op, 0));
      seedEl.style.transform = "scale(" + mix(seed.scale, 0.5) + ")";
    }
    if (plantEl) plantEl.style.transform = "scale(" + mix(plant, 1) + ")";
    var dash = String(mix(growthBranch(p), 1));
    logo.querySelectorAll(".kg-branch").forEach(function (el) { el.style.strokeDashoffset = dash; });
    var fruit = "scale(" + mix(growthFruit(p), 0) + ")";
    logo.querySelectorAll(".kg-fruit").forEach(function (el) { el.style.transform = fruit; });
  }
  function stopGrowth() {
    if (growthFrame) cancelAnimationFrame(growthFrame);
    growthFrame = 0;
  }
  function moveGrowth(target, viaSeed) {
    if (viaSeed) growthPassSeed = true;
    if (growthReducedMotion()) {
      growthPassSeed = false;
      growthPose = { rest: target.rest, p: target.p };
      paintGrowth(growthPose);
      return;
    }
    var points = [{ rest: growthPose.rest, p: growthPose.p }];
    var atSeed = growthPose.rest < 0.08 && growthPose.p < 0.04;
    if (growthPassSeed && !atSeed) points.push({ rest: 0, p: 0 });
    else growthPassSeed = false;
    var last = points[points.length - 1];
    if (Math.abs(last.rest - target.rest) > 0.001 || Math.abs(last.p - target.p) > 0.001) points.push(target);
    if (points.length < 2) {
      growthPose = { rest: target.rest, p: target.p };
      paintGrowth(growthPose);
      return;
    }
    stopGrowth();
    var seg = 900;
    var started = performance.now();
    var nSeg = points.length - 1;
    function frame(now) {
      var t = (now - started) / (seg * nSeg);
      if (t >= 1) {
        growthPose = { rest: points[nSeg].rest, p: points[nSeg].p };
        if (growthPose.rest < 0.08 && growthPose.p < 0.04) growthPassSeed = false;
        paintGrowth(growthPose);
        growthFrame = 0;
        return;
      }
      var x = t * nSeg;
      // Firefox reports a first-frame timestamp that can precede performance.now()
      // taken when scheduling, which would index points[-1].
      if (!(x >= 0)) x = 0;
      var i = Math.min(nSeg - 1, Math.floor(x));
      var u = growthEase(x - i);
      growthPose = {
        rest: points[i].rest + (points[i + 1].rest - points[i].rest) * u,
        p: points[i].p + (points[i + 1].p - points[i].p) * u,
      };
      if (growthPose.rest < 0.08 && growthPose.p < 0.04) growthPassSeed = false;
      paintGrowth(growthPose);
      growthFrame = requestAnimationFrame(frame);
    }
    growthFrame = requestAnimationFrame(frame);
  }
  function growthSame(a, b) {
    return Math.abs(a.rest - b.rest) < 0.001 && Math.abs(a.p - b.p) < 0.001;
  }
  function syncGrowthLogo(usedPct, viaSeed) {
    var target = growthThreadEmpty()
      ? { rest: 1, p: 0 }
      : { rest: 0, p: growthProgress(usedPct) };
    if (viaSeed) growthPassSeed = true;
    if (!viaSeed && growthLogoReady) {
      if (growthFrame && growthSame(target, growthTarget)) return;
      if (!growthFrame && growthSame(target, growthPose)) return;
    }
    growthTarget = target;
    if (!growthLogoOn || !growthLogoReady) return;
    moveGrowth(growthTarget, !!viaSeed);
  }
  function noteGrowthMessage(payload) {
    if (!payload || payload.delta) return;
    var viaSeed = false;
    if ((payload.role || "") === "user") {
      var n = 0;
      messages.forEach(function (m) {
        if (m.role === "user" && String(m.text || "").trim()) n++;
      });
      viaSeed = n === 1;
    }
    if (viaSeed && typeof payload.used_pct !== "number") {
      syncGrowthLogo(0, true);
      return;
    }
    if (typeof payload.used_pct === "number") syncGrowthLogo(payload.used_pct, viaSeed);
  }
  function applyGrowthEnabled(on) {
    var was = growthLogoOn && growthLogoReady;
    growthLogoOn = !!on;
    var box = document.getElementById("growth-logo-toggle");
    if (box) box.checked = growthLogoOn;
    var mark = document.getElementById("brand-mark");
    var logo = document.getElementById("growth-logo");
    // SVG's hidden property does not drop the attribute, so the plant stays display:none.
    function showLogo(el, on) {
      if (!el) return;
      if (on) el.removeAttribute("hidden");
      else el.setAttribute("hidden", "");
    }
    if (!growthLogoOn) {
      stopGrowth();
      growthLogoReady = false;
      growthPassSeed = false;
      growthPose = { rest: 1, p: 0 };
      showLogo(mark, true);
      showLogo(logo, false);
      return;
    }
    showLogo(mark, false);
    showLogo(logo, true);
    growthLogoReady = true;
    if (!was) {
      growthPose = { rest: 1, p: 0 };
      paintGrowth(growthPose);
      moveGrowth(growthTarget, false);
    }
  }
  function bindGrowthLogo() {
    var box = document.getElementById("growth-logo-toggle");
    if (box && !box.dataset.bound) {
      box.dataset.bound = "1";
      box.addEventListener("change", function () {
        applyGrowthEnabled(box.checked);
        saveGuiPrefs({ growth_logo: box.checked }).catch(function () {});
      });
    }
    if (window.matchMedia && !bindGrowthLogo.listening) {
      bindGrowthLogo.listening = true;
      var mq = window.matchMedia("(prefers-reduced-motion: reduce)");
      growthReducedMotion = function () { return mq.matches; };
      if (mq.addEventListener) {
        mq.addEventListener("change", function () {
          if (growthLogoOn && growthLogoReady) moveGrowth(growthTarget, false);
        });
      }
    }
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
        if (!p) {
          bindGrowthLogo();
          applyGrowthEnabled(true);
          return;
        }
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
        bindGrowthLogo();
        applyGrowthEnabled(p.growth_logo !== false);
      })
      .catch(function () {
        bindGrowthLogo();
        applyGrowthEnabled(true);
      });
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

  function showPluginGitDialog(text) {
    var dlg = document.getElementById("plugin-git-dialog");
    var out = document.getElementById("plugin-git-out");
    if (!dlg || !out) {
      showPluginErr(text);
      return;
    }
    showPluginErr("");
    out.textContent = text || "";
    if (!dlg.open) dlg.showModal();
  }

  function showPluginFailure(err) {
    var text = String((err && err.message) || err || "");
    if (text.indexOf("\n") >= 0) showPluginGitDialog(text);
    else showPluginErr(text);
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
      if (p.ssh_port) bits.push("SSH port " + p.ssh_port);
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
      if (p.layout === "marketplace" && p.plugins && p.plugins.length) {
        var units = document.createElement("div");
        units.className = "plugin-units";
        if (!p.enabled) {
          var held = document.createElement("p");
          held.className = "hint";
          held.textContent = "Marketplace is off. Checked plugins come back when you turn it on.";
          units.appendChild(held);
        }
        var active = {};
        (p.active || []).forEach(function (unitName) { active[unitName] = true; });
        p.plugins.forEach(function (unitName) {
          var row = document.createElement("label");
          row.className = "check";
          var pick = document.createElement("input");
          pick.type = "checkbox";
          pick.checked = !!active[unitName];
          pick.disabled = !p.enabled;
          pick.addEventListener("change", function () {
            pick.disabled = true;
            fetch(apiURL("/api/plugins/unit"), {
              method: "POST",
              headers: headers(),
              body: JSON.stringify({ id: p.id, name: unitName, enabled: pick.checked }),
            })
              .then(readAPIResult)
              .then(loadPlugins)
              .catch(function (err) {
                pick.checked = !pick.checked;
                showPluginErr(String(err.message || err));
              })
              .finally(function () { pick.disabled = !p.enabled; });
          });
          row.appendChild(pick);
          row.appendChild(document.createTextNode(unitName));
          units.appendChild(row);
        });
        li.appendChild(units);
      }
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
      on.className = "check";
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
      on.appendChild(document.createTextNode(p.layout === "marketplace" ? "Marketplace on" : "On"));
      actions.appendChild(on);
      if (p.kind === "git") {
        var refresh = document.createElement("button");
        refresh.type = "button";
        refresh.className = "ghost";
        refresh.textContent = "Refresh";
        refresh.addEventListener("click", function () {
          fetch(apiURL("/api/plugins/refresh"), { method: "POST", headers: headers(), body: JSON.stringify({ id: p.id }) })
            .then(readAPIResult)
            .then(function (j) {
              if (j && j.error) showPluginFailure({ message: j.error });
              return loadPlugins();
            })
            .catch(function (err) { showPluginFailure(err); });
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
    if (paneSpend) paneSpend.hidden = tab !== "spend";
    saveSettingsBtn.hidden = true;
    connectBtn.hidden = tab !== "connection";
    if (tab === "secrets" && connected) loadCredentialsUI();
    if (tab === "plugins" && connected) loadPlugins();
    if (tab === "skills" && connected) {
      loadSkillsList().catch(function (err) { showSkillsErr(String(err.message || err)); });
    }
    if (tab === "mcp" && connected) {
      syncMcpKindUI();
      loadMcpList().catch(function (err) { showMcpErr(String(err.message || err)); });
    }
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
      credEditSecret.placeholder = c && c.secret_set ? "Set — leave blank to keep" : "Secret value";
    }
    if (credEditSecretKey) {
      credEditSecretKey.value = "";
      credEditSecretKey.placeholder = c && c.secret_set ? "Set — leave blank to keep" : "Paste the private key file, from BEGIN through END";
    }
    if (credEditDelete) credEditDelete.hidden = !c;
    syncCredentialForm();
  }

  function syncCredentialForm() {
    var ssh = credEditType && credEditType.value === "ssh_key";
    if (credEditSecret) credEditSecret.hidden = !!ssh;
    if (credEditSecretKey) credEditSecretKey.hidden = !ssh;
    if (credEditInjectLabel) credEditInjectLabel.hidden = !!ssh;
    if (credEditInjectEnvLabel) credEditInjectEnvLabel.hidden = !!ssh;
    if (credEditUse) {
      credEditUse.textContent = ssh
        ? "A private key for Git or SSH. Paste the key file below. For one shell command, Keimling loads the key into its own ssh-agent and then drops it. The model never reads the key."
        : "A value the model never reads. A shell command can receive it as an environment variable. An MCP tool can receive it as one argument.";
    }
    if (credEditServiceHint) {
      credEditServiceHint.textContent = ssh
        ? "The hosts this key is for, separated by commas, such as git.example.org, github.com. The agent uses this to choose the key. The key is not limited to those hosts."
        : "Which service this belongs to. Several hosts can be written here, separated by commas. The agent uses this to choose. It does not limit where the value is sent.";
    }
    if (credEditToolsHint) {
      credEditToolsHint.textContent = ssh
        ? "Write exec. Git and ssh run as a shell command."
        : "Write exec for a shell command, or an MCP tool name such as mcp__server__tool. Separate several names with commas.";
    }
    if (credEditSecretTitle) credEditSecretTitle.textContent = ssh ? "Private key" : "Secret";
    if (credEditSecretHint) {
      credEditSecretHint.textContent = ssh
        ? "The key file, from BEGIN through END, each line on its own line. A key that asks for its own passphrase cannot be loaded. The vault password protects it. When you edit, leave this blank to keep the saved key."
        : "The value itself. It is stored encrypted and is not shown again. When you edit, leave this blank to keep the saved value.";
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
    var ssh = credEditType && credEditType.value === "ssh_key";
    var secret = ssh ? (credEditSecretKey ? credEditSecretKey.value : "") : (credEditSecret ? credEditSecret.value : "");
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
      var count = (p.models || []).length;
      var li = document.createElement("li");
      var label = document.createElement("div");
      label.className = "profile-pick";
      var meta = [];
      if (p.base_url) meta.push(p.base_url);
      meta.push(count === 1 ? "1 model" : count + " models");
      label.innerHTML = esc(p.name || p.id) +
        '<span class="profile-meta">' + esc(meta.join(" · ")) + "</span>";
      var actions = document.createElement("div");
      actions.className = "profile-row-actions";
      var edit = document.createElement("button");
      edit.type = "button";
      edit.className = "ghost profile-edit-btn";
      edit.textContent = "Edit";
      edit.addEventListener("click", function () { openProviderEditor(p); });
      actions.appendChild(edit);
      var del = document.createElement("button");
      del.type = "button";
      del.className = "ghost profile-edit-btn danger";
      del.textContent = "Remove";
      del.addEventListener("click", function () { removeProvider(p.id); });
      actions.appendChild(del);
      li.appendChild(label);
      li.appendChild(actions);
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
    if (provModelInstructions) provModelInstructions.value = "";
    setModelCapabilities(null);
  }

  function fillModelForm(m) {
    if (!m) { clearModelForm(); return; }
    if (provModelID) provModelID.value = m.id || "";
    if (provModelName) provModelName.value = m.name || "";
    if (provModelContext) provModelContext.value = m.context_limit ? String(m.context_limit) : "";
    if (provModelReasoning) provModelReasoning.value = m.reasoning || "";
    if (provModelPromptPrice) provModelPromptPrice.value = usdPerMillionFromToken(m.prompt_price);
    if (provModelCompletionPrice) provModelCompletionPrice.value = usdPerMillionFromToken(m.completion_price);
    if (provModelInstructions) provModelInstructions.value = m.instructions || "";
    setModelCapabilities(m);
  }

  function providerByID(id) {
    for (var i = 0; i < providersCache.length; i++) {
      if (providersCache[i].id === id) return providersCache[i];
    }
    return null;
  }

  function fillModelProviderSelect(selected) {
    if (!provModelProvider) return;
    provModelProvider.innerHTML = "";
    providersCache.forEach(function (p) {
      var opt = document.createElement("option");
      opt.value = p.id;
      opt.textContent = p.name || p.id;
      provModelProvider.appendChild(opt);
    });
    if (selected) provModelProvider.value = selected;
    if (!provModelProvider.value && providersCache[0]) provModelProvider.value = providersCache[0].id;
    showProviderCatalog(provModelProvider.value);
  }

  function closeModelForm() {
    modelFormMode = null;
    modelFormOriginalID = "";
    modelFormOriginalProvider = "";
    clearModelForm();
    if (provCatalogStatus) provCatalogStatus.hidden = true;
    if (provModelEditor) provModelEditor.hidden = true;
    if (provModelNew) provModelNew.hidden = false;
  }

  function openModelFormAdd() {
    if (!providersCache.length) {
      gateErr.hidden = false;
      gateErr.textContent = "Save a provider first";
      return;
    }
    modelFormMode = "add";
    modelFormOriginalID = "";
    modelFormOriginalProvider = "";
    clearModelForm();
    fillModelProviderSelect(activeProviderID);
    if (provModelEditorTitle) provModelEditorTitle.textContent = "Add model";
    if (provModelEditor) provModelEditor.hidden = false;
    if (provModelNew) provModelNew.hidden = true;
    if (provModelID) provModelID.focus();
  }

  function openModelFormEdit(m, providerID) {
    if (!m) return;
    modelFormMode = "edit";
    modelFormOriginalID = m.id;
    modelFormOriginalProvider = providerID || "";
    fillModelForm(m);
    fillModelProviderSelect(providerID);
    if (provModelEditorTitle) provModelEditorTitle.textContent = "Edit model";
    if (provModelEditor) provModelEditor.hidden = false;
    if (provModelNew) provModelNew.hidden = true;
    if (provModelID) provModelID.focus();
  }

  function persistProviderCache(statusEl, doneText) {
    if (statusEl) statusEl.textContent = "Saving…";
    return putSettings({
      model_providers: providersPayload(),
      active_model_provider_id: activeProviderID,
    }).then(function () {
      if (statusEl) statusEl.textContent = doneText || "Saved";
      return true;
    }).catch(function (e) {
      if (statusEl) statusEl.textContent = "";
      gateErr.hidden = false;
      gateErr.textContent = String(e);
      return false;
    });
  }

  function renderProvModelList() {
    if (!provModelList) return;
    provModelList.innerHTML = "";
    providersCache.forEach(function (p) {
      (p.models || []).forEach(function (m) {
        var li = document.createElement("li");
        var label = document.createElement("div");
        label.className = "profile-pick";
        var meta = [p.name || p.id];
        if (m.context_limit) meta.push(m.context_limit + " ctx");
        if (m.transcribes) meta.push("transcribes");
        if (m.speaks) meta.push("speaks");
        if (m.realtime) meta.push("realtime");
        label.innerHTML = esc(m.name || m.id) +
          '<span class="profile-meta">' + esc(meta.join(" · ")) + "</span>";
        var actions = document.createElement("div");
        actions.className = "profile-row-actions";
        var edit = document.createElement("button");
        edit.type = "button";
        edit.className = "ghost profile-edit-btn";
        edit.textContent = "Edit";
        edit.addEventListener("click", function () { openModelFormEdit(m, p.id); });
        actions.appendChild(edit);
        var del = document.createElement("button");
        del.type = "button";
        del.className = "ghost profile-edit-btn danger";
        del.textContent = "Remove";
        del.addEventListener("click", function () {
          var labelName = m.name || m.id;
          askConfirm("Remove “" + labelName + "” from " + (p.name || p.id) + "?", "Remove " + labelName).then(function (yes) {
            if (!yes) return;
            p.models = (p.models || []).filter(function (x) { return x.id !== m.id; });
            if ((p.active_model_id || p.model) === m.id) {
              p.active_model_id = p.models[0] ? p.models[0].id : "";
              p.model = p.active_model_id;
            }
            if (modelFormOriginalID === m.id && modelFormOriginalProvider === p.id) closeModelForm();
            persistProviderCache(document.getElementById("prov-model-status"), "Model removed");
          });
        });
        actions.appendChild(del);
        li.appendChild(label);
        li.appendChild(actions);
        provModelList.appendChild(li);
      });
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
    var promptPrice = usdPerTokenFromMillion(provModelPromptPrice ? provModelPromptPrice.value : "");
    var completionPrice = usdPerTokenFromMillion(provModelCompletionPrice ? provModelCompletionPrice.value : "");
    if (promptPrice == null || completionPrice == null) {
      gateErr.hidden = false;
      gateErr.textContent = "Enter the price in USD per million tokens";
      return;
    }
    var row = {
      id: id,
      name: provModelName ? provModelName.value.trim() : "",
      context_limit: ctx || 0,
      reasoning: provModelReasoning ? provModelReasoning.value : "",
      prompt_price: promptPrice,
      completion_price: completionPrice,
      instructions: provModelInstructions ? provModelInstructions.value.trim() : "",
      transcribes: !!(provModelTranscribes && provModelTranscribes.checked),
      speaks: !!(provModelSpeaks && provModelSpeaks.checked),
      realtime: !!(provModelRealtime && provModelRealtime.checked),
      sees_images: !!(provModelImages && provModelImages.checked),
      hears_audio: !!(provModelAudio && provModelAudio.checked),
    };
    if (row.sees_images) row.use_images = true;
    if (row.hears_audio) row.use_audio = true;
    var providerID = provModelProvider ? provModelProvider.value : "";
    var target = providerByID(providerID);
    if (!target) {
      gateErr.hidden = false;
      gateErr.textContent = "Choose a saved provider";
      return;
    }
    var models = (target.models || []).slice();
    var duplicate = models.some(function (x) {
      return x.id === id && !(modelFormMode === "edit" && modelFormOriginalProvider === providerID && x.id === modelFormOriginalID);
    });
    if (duplicate) {
      gateErr.hidden = false;
      gateErr.textContent = "That provider already has this model id";
      return;
    }
    if (modelFormMode === "edit" && modelFormOriginalProvider && modelFormOriginalProvider !== providerID) {
      var previous = providerByID(modelFormOriginalProvider);
      if (previous) {
        previous.models = (previous.models || []).filter(function (x) { return x.id !== modelFormOriginalID; });
        if ((previous.active_model_id || previous.model) === modelFormOriginalID) {
          previous.active_model_id = previous.models[0] ? previous.models[0].id : "";
          previous.model = previous.active_model_id;
        }
      }
    }
    var placed = false;
    if (modelFormMode === "edit" && modelFormOriginalProvider === providerID) {
      for (var i = 0; i < models.length; i++) {
        if (models[i].id === modelFormOriginalID) {
          models[i] = row;
          if ((target.active_model_id || target.model) === modelFormOriginalID) {
            target.active_model_id = id;
            target.model = id;
          }
          placed = true;
          break;
        }
      }
    }
    if (!placed) models.push(row);
    if (!target.active_model_id && !target.model) {
      target.active_model_id = id;
      target.model = id;
    }
    target.models = models;
    gateErr.hidden = true;
    var modelStatus = document.getElementById("prov-model-status");
    persistProviderCache(modelStatus, "Model saved").then(function (ok) {
      if (ok) closeModelForm();
    });
  }

  function applyCatalogPrefill(entry) {
    if (!entry) return;
    if (provModelID) provModelID.value = entry.id || "";
    if (provModelName && entry.name) provModelName.value = entry.name;
    if (provModelContext && entry.context_limit) provModelContext.value = String(entry.context_limit);
    if (provModelPromptPrice && entry.prompt_price) provModelPromptPrice.value = usdPerMillionFromToken(entry.prompt_price);
    if (provModelCompletionPrice && entry.completion_price) provModelCompletionPrice.value = usdPerMillionFromToken(entry.completion_price);
    if (entry.capabilities_known) setModelCapabilities(entry);
  }

  function setModelCapabilities(m) {
    if (provModelImages) provModelImages.checked = !!(m && m.sees_images && m.use_images !== false);
    if (provModelAudio) provModelAudio.checked = !!(m && m.hears_audio && m.use_audio !== false);
    if (provModelTranscribes) provModelTranscribes.checked = !!(m && m.transcribes);
    if (provModelSpeaks) provModelSpeaks.checked = !!(m && m.speaks);
    if (provModelRealtime) provModelRealtime.checked = !!(m && m.realtime);
  }

  function showProviderCatalog(providerID) {
    catalogCache = catalogsByProvider[providerID] || [];
    if (!provCatalogDatalist) return;
    provCatalogDatalist.innerHTML = "";
    catalogCache.forEach(function (m) {
      var opt = document.createElement("option");
      opt.value = m.id;
      opt.label = m.name ? (m.name + " (" + m.id + ")") : m.id;
      if (m.outputs) opt.label += " · " + m.outputs;
      provCatalogDatalist.appendChild(opt);
    });
    if (provCatalogStatus && !catalogCache.length) {
      provCatalogStatus.hidden = true;
      provCatalogStatus.textContent = "";
    }
  }

  function fillCatalogDatalist(models, providerID) {
    catalogsByProvider[providerID] = models || [];
    if (provModelProvider && provModelProvider.value === providerID) showProviderCatalog(providerID);
  }

  function fetchModelCatalog() {
    if (!connected) return;
    if (provCatalogStatus) {
      provCatalogStatus.hidden = false;
      provCatalogStatus.textContent = "Fetching catalog…";
    }
    var catalogProvider = providerByID(provModelProvider ? provModelProvider.value : "");
    if (!catalogProvider || !catalogProvider.base_url) {
      if (provCatalogStatus) provCatalogStatus.textContent = "Choose a saved provider";
      return;
    }
    var fetchedFor = catalogProvider.id;
    var body = { base_url: catalogProvider.base_url, kind: catalogProvider.kind || "auto", provider_id: catalogProvider.id };
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
          fillCatalogDatalist(j.models || [], fetchedFor);
          var still = provModelProvider && provModelProvider.value === fetchedFor;
          if (still && provCatalogStatus) {
            provCatalogStatus.hidden = false;
            provCatalogStatus.textContent = (j.count || 0) + " models for this provider — pick an id to prefill.";
          }
          if (!still) return;
          var typed = provModelID ? provModelID.value.trim() : "";
          var list = catalogsByProvider[fetchedFor] || [];
          for (var i = 0; i < list.length; i++) {
            if (list[i].id === typed) {
              applyCatalogPrefill(list[i]);
              break;
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

  function providerWantsManagementKey() {
    var kind = provEditKind ? provEditKind.value : "auto";
    var base = ((provEditBaseURL && provEditBaseURL.value) || "").toLowerCase();
    return kind === "openrouter" || (kind === "auto" && base.indexOf("openrouter") >= 0);
  }
  function syncManagementKeyField() {
    var show = providerWantsManagementKey();
    if (provEditManagementLabel) provEditManagementLabel.hidden = !show;
    if (provEditManagementHint) provEditManagementHint.hidden = !show;
  }

  function openProviderEditor(p) {
    editingProviderID = p ? p.id : "";
    if (providerEditor) providerEditor.hidden = false;
    if (provEditName) provEditName.value = p ? (p.name || "") : "";
    if (provEditBaseURL) provEditBaseURL.value = p ? (p.base_url || "") : "";
    if (provEditKind) provEditKind.value = (p && p.kind) || "auto";
    if (provEditAPIKey) {
      provEditAPIKey.value = "";
      provEditAPIKey.placeholder = p && p.api_key_set ? "Set — leave blank to keep" : "API key";
    }
    if (provEditManagement) {
      provEditManagement.value = "";
      provEditManagement.placeholder = p && p.management_key_set ? "Set — leave blank to keep" : "Management key";
    }
    if (provEditManagementHint) {
      provEditManagementHint.textContent = p && p.management_key_set
        ? "Management key is set. Leave blank to keep it. Chat still uses the API key."
        : "Reads the OpenRouter account total. Chat still uses the API key.";
    }
    syncManagementKeyField();
    if (apiKeyHint) {
      apiKeyHint.textContent = p && p.api_key_set
        ? "Key is set. Leave blank to keep it, or paste a new key."
        : "Required for OpenRouter and most remote APIs.";
    }
  }

  function closeProviderEditor() {
    editingProviderID = null;
    if (providerEditor) providerEditor.hidden = true;
  }

  function saveProviderEditor() {
    var name = provEditName ? provEditName.value.trim() : "";
    var base = provEditBaseURL ? provEditBaseURL.value.trim() : "";
    var kind = provEditKind ? provEditKind.value : "auto";
    var key = provEditAPIKey ? provEditAPIKey.value.trim() : "";
    var management = provEditManagement ? provEditManagement.value.trim() : "";
    if (!base) {
      gateErr.hidden = false;
      gateErr.textContent = "Base URL required";
      return;
    }
    var kept = providerByID(editingProviderID);
    var row = {
      name: name || base,
      base_url: base,
      kind: kind || "auto",
      models: kept ? (kept.models || []).slice() : [],
      active_model_id: kept ? (kept.active_model_id || kept.model || "") : "",
      model: kept ? (kept.model || kept.active_model_id || "") : "",
      reasoning: kept ? (kept.reasoning || "") : "",
    };
    if (!editingProviderID) {
      row.id = "provider-" + Date.now();
      if (key) { row.api_key = key; row.api_key_set = true; }
      else { row.api_key_set = false; }
      if (management) { row.management_key = management; row.management_key_set = true; }
      else { row.management_key_set = false; }
      providersCache.push(row);
      activeProviderID = row.id;
    } else {
      for (var j = 0; j < providersCache.length; j++) {
        if (providersCache[j].id === editingProviderID) {
          row.id = editingProviderID;
          row.api_key_set = providersCache[j].api_key_set;
          row.management_key_set = providersCache[j].management_key_set;
          if (key) { row.api_key = key; row.api_key_set = true; }
          else if (providersCache[j].api_key) row.api_key = providersCache[j].api_key;
          if (management) { row.management_key = management; row.management_key_set = true; }
          else if (providersCache[j].management_key) row.management_key = providersCache[j].management_key;
          providersCache[j] = row;
          break;
        }
      }
    }
    syncHiddenProviderFields();
    renderProviderList();
    var status = document.getElementById("prov-edit-status");
    if (status) status.textContent = "Saving provider…";
    putSettings({
      model_providers: providersPayload(),
      active_model_provider_id: activeProviderID,
    }).then(function () {
      if (status) status.textContent = "Provider saved";
      closeProviderEditor();
    }).catch(function (e) {
      gateErr.hidden = false;
      gateErr.textContent = String(e);
      if (status) status.textContent = "";
    });
  }

  function removeProvider(id) {
    var current = providerByID(id);
    if (!current) return;
    var name = current.name || current.id || "this provider";
    askConfirm("Remove “" + name + "”? The saved API key is removed with it.", "Remove " + name).then(function (yes) {
      if (!yes) return;
      providersCache = providersCache.filter(function (p) { return p.id !== id; });
      if (activeProviderID === id) activeProviderID = providersCache[0] ? providersCache[0].id : "";
      if (editingProviderID === id) closeProviderEditor();
      if (modelFormOriginalProvider === id) closeModelForm();
      syncHiddenProviderFields();
      renderProviderList();
      renderProvModelList();
      putSettings({
        model_providers: providersPayload(),
        active_model_provider_id: activeProviderID,
      }).catch(function (e) {
        gateErr.hidden = false;
        gateErr.textContent = String(e);
      });
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
    if (setRealtimeVoice) setRealtimeVoice.value = j.realtime_voice || "marin";
    realtimeReady = !!(j.realtime_provider_id && j.realtime_model);
    setState(agentState);
  }

  // fillAuxiliaryModel lists the saved provider models plus the empty choice
  // that means "use the chat model".
  function fillAuxiliaryModel(j) {
    if (!setAuxiliaryModel) return;
    var wanted = (j && j.auxiliary_model) || "";
    setAuxiliaryModel.innerHTML = '<option value="">Use the chat model</option>';
    providersCache.forEach(function (p) {
      (p.models || []).forEach(function (m) {
        var opt = document.createElement("option");
        opt.value = m.id;
        opt.textContent = (p.name || p.id) + " · " + (m.name || m.id);
        setAuxiliaryModel.appendChild(opt);
      });
    });
    setAuxiliaryModel.value = wanted;
    if (setAuxiliaryModel.value !== wanted) setAuxiliaryModel.value = "";
  }

  // auxiliaryModelPayload is the saved value. Empty means the chat model.
  function auxiliaryModelPayload() {
    return setAuxiliaryModel ? String(setAuxiliaryModel.value || "").trim() : "";
  }

  function fillVoiceModels(selected) {
    if (!setVoiceModel) return;
    if (setVoiceModel.tagName !== "SELECT") {
      setVoiceModel.value = selected || "";
      syncVoiceModelEnabled();
      return;
    }
    var pid = setVoiceProvider ? setVoiceProvider.value : "";
    setVoiceModel.innerHTML = '<option value="">Off</option>';
    providersCache.forEach(function (p) {
      if (pid && p.id !== pid) return;
      (p.models || []).forEach(function (m) {
        if (!m.transcribes) return;
        var opt = document.createElement("option");
        opt.value = m.id;
        opt.textContent = m.name || m.id;
        setVoiceModel.appendChild(opt);
      });
    });
    setVoiceModel.value = selected || "";
    if (setVoiceModel.value !== (selected || "")) setVoiceModel.value = "";
    syncVoiceModelEnabled();
  }

  function fillCapabilitySelect(select, wanted, allow) {
    if (!select) return;
    var label = "Off";
    var first = select.querySelector("option");
    if (first && first.textContent) label = first.textContent;
    select.innerHTML = "";
    var off = document.createElement("option");
    off.value = "";
    off.textContent = label;
    select.appendChild(off);
    providersCache.forEach(function (p) {
      (p.models || []).forEach(function (m) {
        if (allow && !allow(m)) return;
        var opt = document.createElement("option");
        opt.value = p.id + "\n" + m.id;
        opt.textContent = (p.name || p.id) + " · " + (m.name || m.id);
        select.appendChild(opt);
      });
    });
    select.value = wanted || "";
    if (select.value !== (wanted || "")) select.value = "";
  }

  function fillPurposeChoices(j) {
    j = j || {};
    var main = "";
    if (activeProviderID) {
      var active = null;
      for (var i = 0; i < providersCache.length; i++) {
        if (providersCache[i].id === activeProviderID) { active = providersCache[i]; break; }
      }
      var m = activeModelOf(active);
      if (active && m) main = active.id + "\n" + m.id;
    }
    fillCapabilitySelect(setMainModel, main, null);
    if (setMainModel && setMainModel.options.length > 1) {
      var blankMain = setMainModel.querySelector('option[value=""]');
      if (blankMain) blankMain.remove();
    }
    var vision = (j.vision_provider_id && j.vision_model) ? j.vision_provider_id + "\n" + j.vision_model : "";
    fillCapabilitySelect(setVisionModel, vision, function (m) { return !!m.sees_images && m.use_images !== false; });
    var speech = (j.speech_provider_id && j.speech_model) ? j.speech_provider_id + "\n" + j.speech_model : "";
    fillCapabilitySelect(setSpeechModel, speech, function (m) { return !!m.speaks; });
    var voice = (j.voice_provider_id && j.voice_model) ? j.voice_provider_id + "\n" + j.voice_model : "";
    fillCapabilitySelect(setVoiceModel, voice, function (m) { return !!m.transcribes; });
    var live = (j.realtime_provider_id && j.realtime_model) ? j.realtime_provider_id + "\n" + j.realtime_model : "";
    fillCapabilitySelect(setRealtimeModel, live, function (m) { return !!m.realtime; });
  }

  function purposePair(value) {
    var parts = String(value || "").split("\n");
    return { provider: parts[0] || "", model: parts[1] || "" };
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
        management_key_set: !!p.management_key_set,
      };
    });
    activeProviderID = j.active_model_provider_id || (providersCache[0] && providersCache[0].id) || "";
    setOpenAIAPIKey.value = "";
    if (openaiKeyHint) openaiKeyHint.hidden = true;
    setMode.value = j.mode || "auto";
    fillVoiceAndSearch(j);
    fillRealtime(j);
    fillAuxiliaryModel(j);
    fillPurposeChoices(j);
    setWeb.checked = !!j.web;
    setAutopilot.checked = !!j.autopilot;
    applyAutopilotSettingsFields(j);
    approvalPolicyBaseline = j.approval_policy || "";
    if (setApprovalPolicy) setApprovalPolicy.value = approvalPolicyBaseline;
    renderToolAsk(j.tool_ask);
    if (setArtifactProxy) setArtifactProxy.checked = !!j.artifact_proxy;
    if (setAutopilotInstructionsUndo) setAutopilotInstructionsUndo.hidden = true;
    setMaxSteps.value = j.max_steps != null ? j.max_steps : 200;
    if (setMCPParallel) setMCPParallel.value = j.mcp_parallel != null ? j.mcp_parallel : 10;
    if (setContextLimit) setContextLimit.value = j.context_limit != null ? j.context_limit : 128000;
    idleSoundOn = !!j.idle_sound;
    idleNotifyOn = !!j.idle_notify;
    if (setIdleSound) setIdleSound.checked = idleSoundOn;
    if (setIdleNotify) setIdleNotify.checked = idleNotifyOn;
    syncHiddenProviderFields();
    renderProviderList();
    renderProvModelList();
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

  function modelPayload(m) {
    var model = {
      id: m.id,
      name: m.name || "",
      context_limit: m.context_limit || 0,
      reasoning: m.reasoning || "",
      prompt_price: m.prompt_price || "",
      completion_price: m.completion_price || "",
      instructions: m.instructions || "",
      transcribes: !!m.transcribes,
      speaks: !!m.speaks,
      realtime: !!m.realtime,
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
  }

  function providersPayload() {
    return providersCache.map(function (p) {
      var row = {
        id: p.id,
        name: p.name,
        base_url: p.base_url,
        kind: p.kind || "auto",
        model: p.model,
        reasoning: p.reasoning || "",
        active_model_id: p.active_model_id || p.model || "",
        models: (p.models || []).map(modelPayload),
      };
      if (p.api_key) row.api_key = p.api_key;
      if (p.management_key) row.management_key = p.management_key;
      return row;
    });
  }

  function putSettings(body) {
    return fetch(apiURL("/api/settings"), {
      method: "PUT",
      headers: headers(),
      body: JSON.stringify(body),
    }).then(function (r) {
      return r.text().then(function (t) {
        var j = {};
        try { j = JSON.parse(t); } catch (e) {}
        if (!r.ok) throw new Error((j && j.message) || t || ("HTTP " + r.status));
        providersCache.forEach(function (p) {
          delete p.api_key;
          delete p.management_key;
        });
        fillAgentSettings(j);
        applyProviderGate(!!j.needs_provider);
        if (typeof loadSession === "function") loadSession();
        if (typeof loadThreadModel === "function") loadThreadModel();
        if (modeEl) modeEl.textContent = j.mode || "—";
        return j;
      });
    });
  }

  function savePurposes() {
    var status = document.getElementById("set-purposes-status");
    var main = purposePair(setMainModel && setMainModel.value);
    if (main.provider && main.model) {
      activeProviderID = main.provider;
      providersCache.forEach(function (p) {
        if (p.id === main.provider) {
          p.active_model_id = main.model;
          p.model = main.model;
        }
      });
    }
    var vision = purposePair(setVisionModel && setVisionModel.value);
    var speech = purposePair(setSpeechModel && setSpeechModel.value);
    var voice = purposePair(setVoiceModel && setVoiceModel.value);
    var live = purposePair(setRealtimeModel && setRealtimeModel.value);
    if (status) status.textContent = "Saving purposes…";
    putSettings({
      auxiliary_model: auxiliaryModelPayload(),
      voice_model: voice.model,
      voice_provider_id: voice.provider,
      vision_model: vision.model,
      vision_provider_id: vision.provider,
      speech_model: speech.model,
      speech_provider_id: speech.provider,
      realtime_model: live.model,
      realtime_provider_id: live.provider,
      realtime_voice: setRealtimeVoice ? setRealtimeVoice.value : "marin",
      active_model_provider_id: activeProviderID,
      model_providers: providersPayload(),
    }).then(function () {
      if (status) status.textContent = "Purposes saved";
    }).catch(function (e) {
      if (status) status.textContent = "";
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
      mcp_parallel: setMCPParallel ? parseInt(setMCPParallel.value, 10) : 10,
      idle_sound: !!(setIdleSound && setIdleSound.checked),
      idle_notify: !!(setIdleNotify && setIdleNotify.checked),
    };
    if (agentSettingsLoaded) {
      body.web = setWeb ? setWeb.checked : false;
      body.search_kind = setSearchKind ? setSearchKind.value : "";
      body.search_url = setSearchURL ? setSearchURL.value.trim() : "";
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
    var behaviourStatus = document.getElementById("set-behaviour-status");
    if (behaviourStatus) behaviourStatus.textContent = "Saving…";
    saveSettingsBtn.disabled = true;
    putSettings(body)
      .then(function () {
        if (behaviourStatus) behaviourStatus.textContent = "Behaviour saved";
        refreshBudget(true);
        closeGate();
      })
      .catch(function (e) {
        if (behaviourStatus) behaviourStatus.textContent = "";
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
      camera: ["M3.5 7.5h4l1.8-2h5.4l1.8 2h4v11h-17z", "M12 15.4a3.2 3.2 0 1 0 0-6.4a3.2 3.2 0 0 0 0 6.4"],
      mic: ["M9 4.5a3 3 0 0 1 6 0v6a3 3 0 0 1-6 0z", "M6.5 11a5.5 5.5 0 0 0 11 0", "M12 16.5V20"],
      "mic-off": ["M9 4.5a3 3 0 0 1 6 0v6a3 3 0 0 1-6 0z", "M6.5 11a5.5 5.5 0 0 0 11 0", "M12 16.5V20", "M5 5l14 14"],
      cassette: ["M4 8h16v9H4z", "M8 12.5a1.6 1.6 0 1 0 3.2 0a1.6 1.6 0 1 0-3.2 0", "M12.8 12.5a1.6 1.6 0 1 0 3.2 0a1.6 1.6 0 1 0-3.2 0"],
      phone: ["M8 4h2.8l1.2 3.6-2 1.1a11 11 0 0 0 5.3 5.3l1.1-2 3.6 1.2V16a2 2 0 0 1-2 2A13 13 0 0 1 6 6a2 2 0 0 1 2-2z"],
      menu: ["M4 7h16", "M4 12h16", "M4 17h16"],
      sidebar: ["M4 5h16v14H4z", "M15 5v14"],
      dock: ["M4 5h16v14H4z", "M15 5v14"],
      focus: ["M9 5H5v4", "M15 5h4v4", "M19 15v4h-4", "M5 15v4h4"],
      fullscreen: ["M14 4h6v6", "M20 4l-5 5", "M4 10V4h6", "M4 4l5 5", "M4 14v6h6", "M4 20l5-5", "M14 20h6v-6", "M20 20l-5-5"],
      close: ["M6 6l12 12", "M18 6L6 18"]
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

  function moreMenuHome() {
    return document.querySelector("#composer .composer-more");
  }
  function clearMoreMenuStyle() {
    if (!composerMoreMenu) return;
    composerMoreMenu.classList.remove("is-portaled");
    composerMoreMenu.style.position = "";
    composerMoreMenu.style.left = "";
    composerMoreMenu.style.right = "";
    composerMoreMenu.style.top = "";
    composerMoreMenu.style.bottom = "";
    composerMoreMenu.style.zIndex = "";
    composerMoreMenu.style.maxHeight = "";
    composerMoreMenu.style.overflowY = "";
  }
  function setComposerMore(open) {
    if (!composerMoreMenu || !composerMoreBtn) return;
    var home = moreMenuHome();
    composerMoreBtn.setAttribute("aria-expanded", open ? "true" : "false");
    if (!(open && composerMobile)) {
      if (home && composerMoreMenu.parentElement !== home) home.appendChild(composerMoreMenu);
      clearMoreMenuStyle();
      composerMoreMenu.hidden = !open;
      return;
    }
    document.body.appendChild(composerMoreMenu);
    composerMoreMenu.classList.add("is-portaled");
    composerMoreMenu.hidden = false;
    composerMoreMenu.style.position = "fixed";
    composerMoreMenu.style.right = "auto";
    composerMoreMenu.style.zIndex = "70";
    composerMoreMenu.style.maxHeight = "";
    composerMoreMenu.style.overflowY = "auto";
    var r = composerMoreBtn.getBoundingClientRect();
    var menuW = composerMoreMenu.offsetWidth || 220;
    var space = Math.max(96, r.top - 12);
    composerMoreMenu.style.maxHeight = space + "px";
    composerMoreMenu.style.left = Math.max(8, Math.min(r.left, window.innerWidth - menuW - 8)) + "px";
    composerMoreMenu.style.top = "auto";
    composerMoreMenu.style.bottom = (window.innerHeight - r.top + 6) + "px";
  }

  function composerPageList() {
    var list = ["text"];
    if (voiceOK) list.push("dictate");
    if (realtimeReady) list.push("call");
    return list;
  }

  function paintComposerTrack(shift) {
    var list = composerPageList();
    var track = document.getElementById("composer-track");
    var dots = document.querySelector(".composer-dots");
    var n = list.length;
    if (list.indexOf(composerPageId) < 0) composerPageId = "text";
    if (dots) dots.hidden = !composerMobile || n < 2;
    document.querySelectorAll(".composer-dot").forEach(function (dot) {
      var id = dot.getAttribute("data-page");
      dot.hidden = list.indexOf(id) < 0;
      if (id === composerPageId) dot.setAttribute("aria-current", "true");
      else dot.removeAttribute("aria-current");
    });
    document.querySelectorAll(".composer-page").forEach(function (page) {
      page.hidden = list.indexOf(page.getAttribute("data-page")) < 0;
    });
    if (!track) return;
    document.querySelectorAll(".composer-page").forEach(function (page) {
      if (!composerMobile) {
        page.style.width = "";
        page.style.flex = "";
        page.style.maxWidth = "";
      }
    });
    if (!composerMobile) {
      track.style.transform = "";
      track.style.width = "";
      track.style.height = "";
      return;
    }
    var index = Math.max(0, list.indexOf(composerPageId));
    var stage = track.parentElement;
    var pageW = stage ? stage.clientWidth : 0;
    if (pageW < 1) pageW = track.getBoundingClientRect().width || 0;
    track.style.width = (n * pageW) + "px";
    list.forEach(function (id) {
      var page = document.getElementById("composer-page-" + id);
      if (!page) return;
      page.style.flex = "0 0 " + pageW + "px";
      page.style.width = pageW + "px";
      page.style.maxWidth = pageW + "px";
    });
    track.style.transform = "translateX(" + ((-index * pageW) + (shift || 0)) + "px)";
    if (!shift) fitComposer();
  }

  function saveComposerPage() {
    if (!currentThreadID) return;
    try {
      localStorage.setItem(STORAGE_COMPOSER_PAGE + "." + currentThreadID, composerPageId);
    } catch (e) { /* private mode */ }
  }

  // Per-thread input mode: only text/dictate/call, never recording or call state.
  function restoreComposerPage() {
    var saved = "";
    if (currentThreadID) {
      try { saved = localStorage.getItem(STORAGE_COMPOSER_PAGE + "." + currentThreadID) || ""; } catch (e) { saved = ""; }
    }
    if (composerPageList().indexOf(saved) < 0) saved = "text";
    if (saved === composerPageId) return;
    showComposerPage(saved, true);
  }

  function showComposerPage(id, restoring) {
    var list = composerPageList();
    composerPageId = list.indexOf(id) < 0 ? "text" : id;
    if (!restoring) saveComposerPage();
    setComposerMore(false);
    placeComposerControls();
  }

  function composerRubber(dx) {
    var list = composerPageList();
    var index = list.indexOf(composerPageId);
    if ((index <= 0 && dx > 0) || (index >= list.length - 1 && dx < 0)) return dx * 0.28;
    return dx;
  }

  function syncDictateChrome() {
    if (dictateDraft) dictateDraft.hidden = !(voiceWorking || (dictateDraft.value && dictateDraft.value.trim()));
    if (dictateMuteBtn) {
      dictateMuteBtn.hidden = !recording;
      setControlLabel(dictateMuteBtn, dictateMuted ? "Unmute" : "Mute", dictateMuted ? "Turn the microphone back on" : "Pause the microphone");
      setGlyph(dictateMuteBtn, dictateMuted ? "mic-off" : "mic");
    }
    if (dictateSendBtn && send) {
      var cap = send.querySelector(".btn-caption");
      dictateSendBtn.textContent = cap ? cap.textContent : "Send";
      dictateSendBtn.disabled = !!send.disabled || voiceWorking || (!recording && !(dictateDraft && dictateDraft.value.trim()));
    }
    if (dictateCancelBtn) dictateCancelBtn.disabled = voiceWorking || (!recording && !(dictateDraft && dictateDraft.value.trim()));
    var fab = document.getElementById("dictate-fab");
    if (fab) fab.classList.toggle("is-working", !!voiceWorking);
    if (dictateDraft && dictateDraft.hidden) dictateDraft.style.height = "";
    fitComposer();
  }

  function placeMobileControls() {
    var textPage = document.getElementById("composer-page-text");
    var textBar = document.getElementById("composer-text-bar");
    var field = document.querySelector(".composer-field");
    var actions = document.querySelector(".composer-actions");
    var dictateFab = document.getElementById("dictate-fab");
    var callFab = document.getElementById("call-fab");
    var muteSlot = document.getElementById("call-mute-slot");
    var stopSlot = document.getElementById("call-stop-slot");
    if (textPage && field && textBar && field.parentElement !== textPage) textPage.insertBefore(field, textBar);
    if (textBar && actions && textBar.firstElementChild !== actions) textBar.insertBefore(actions, textBar.firstChild);
    if (textBar && send && send.parentElement !== textBar) textBar.appendChild(send);
    if (dictateFab && voiceBtn && voiceBtn.parentElement !== dictateFab) dictateFab.appendChild(voiceBtn);
    if (callFab && callBtn && callBtn.parentElement !== callFab) callFab.appendChild(callBtn);
    if (muteSlot && callMuteBtn && callMuteBtn.parentElement !== muteSlot) muteSlot.appendChild(callMuteBtn);
    if (stopBtn && textBar && stopSlot) {
      var stopHome = composerPageId === "text" && !stopBtn.hidden ? textBar : stopSlot;
      if (stopBtn.parentElement !== stopHome) {
        if (stopHome === textBar) textBar.insertBefore(stopBtn, send);
        else stopSlot.appendChild(stopBtn);
      }
      if (composerPageId === "call") stopBtn.hidden = false;
    }
    if (callShareBtn) callShareBtn.hidden = !callLive;
    syncDictateChrome();
    paintComposerTrack(0);
  }

  function placeDesktopControls() {
    var col = document.querySelector(".composer-send-col");
    var field = document.querySelector(".composer-field");
    var main = document.querySelector(".composer-main");
    var stage = document.querySelector(".composer-stage");
    var actions = document.querySelector(".composer-actions");
    if (field && main && field.parentElement !== main) {
      if (stage && stage.parentElement === main) main.insertBefore(field, stage);
      else main.appendChild(field);
    }
    if (field && actions && actions.parentElement !== field) field.appendChild(actions);
    if (col && send && send.parentElement !== col) col.appendChild(send);
    paintComposerTrack(0);
    if (!composerMoreMenu || !col || !send) return;
    function parked(btn, where) {
      if (!btn) return true;
      var inMenu = where === "menu";
      return btn.parentElement === (inMenu ? composerMoreMenu : col) && btn.classList.contains("icon-btn") !== inMenu;
    }
    var screen = typeof screenBtn !== "undefined" ? screenBtn : null;
    if (parked(attachFileBtn, "menu") && parked(screen, "menu") && parked(voiceBtn, "col") && parked(callBtn, "col") && parked(callMuteBtn, "col") && parked(stopBtn, "col")) return;
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

  function placeComposerControls() {
    var form = document.getElementById("composer");
    if (form) form.classList.toggle("is-mobile", composerMobile);
    if (composerMobile) placeMobileControls();
    else placeDesktopControls();
    pinComposer();
  }

  function fieldMin(el) {
    var style = window.getComputedStyle(el);
    var line = parseFloat(style.lineHeight);
    if (!isFinite(line) || line <= 0) line = (parseFloat(style.fontSize) || 16) * 1.35;
    var pad = (parseFloat(style.paddingTop) || 0) + (parseFloat(style.paddingBottom) || 0);
    return Math.ceil(line + pad);
  }
  // Resting height of a dictation or call page, without a grown transcript.
  function composerRestHeight(page) {
    if (!page || page.hidden) return 0;
    var extra = 0;
    var draft = page.querySelector("#dictate-draft");
    if (draft && !draft.hidden) extra = draft.offsetHeight;
    return Math.max(0, page.offsetHeight - extra);
  }
  function fitComposer() {
    var form = document.getElementById("composer");
    if (!form) return;
    var fields = [];
    if (msg) fields.push(msg);
    if (dictateDraft && !dictateDraft.hidden) fields.push(dictateDraft);
    if (!fields.length) { pinComposer(); return; }
    fields = fields.filter(function (el) { return !!el.offsetParent; });
    if (!fields.length) { pinComposer(); return; }
    var track = document.getElementById("composer-track");
    var cap = Math.floor(window.innerHeight * (composerMobile ? 0.7 : 0.5));
    var allPages = composerMobile ? Array.prototype.slice.call(document.querySelectorAll("#composer .composer-page")) : [];
    allPages.forEach(function (page) { page.style.alignSelf = "flex-start"; });
    fields.forEach(function (el) { el.style.height = fieldMin(el) + "px"; });
    var outside = composerMobile && track ? Math.max(0, form.offsetHeight - track.offsetHeight) : 0;
    fields.forEach(function (el) {
      var page = el.closest(".composer-page");
      var min = fieldMin(el);
      var chrome = page ? (page.offsetHeight - el.offsetHeight) + outside : form.offsetHeight - el.offsetHeight;
      var max = Math.max(min, cap - chrome);
      var need = el.scrollHeight;
      var next = Math.min(Math.max(need, min), max);
      el.style.height = next + "px";
      el.style.overflowY = need > next + 1 ? "auto" : "hidden";
    });
    var floor = 0;
    if (composerMobile) {
      floor = Math.max(
        composerRestHeight(document.getElementById("composer-page-call")),
        composerRestHeight(document.getElementById("composer-page-dictate"))
      );
      var textPage = document.getElementById("composer-page-text");
      var textField = textPage ? textPage.querySelector("textarea") : null;
      if (textPage && textField && floor > textPage.offsetHeight) {
        textField.style.height = (textField.offsetHeight + floor - textPage.offsetHeight) + "px";
      }
    }
    var visible = document.getElementById("composer-page-" + composerPageId);
    var trackH = visible ? visible.offsetHeight : 0;
    if (composerMobile && floor > trackH) trackH = floor;
    allPages.forEach(function (page) { page.style.alignSelf = ""; });
    if (!track) return;
    track.style.height = composerMobile && trackH ? trackH + "px" : "";
    pinComposer();
  }
  function pinComposer() {
    var form = document.getElementById("composer");
    if (!form || !composerMobile) return;
    document.documentElement.style.setProperty("--composer-h", Math.ceil(form.offsetHeight) + "px");
  }

  // A run ends with its Thinking rows and tool cards closed. The chat lines
  // the agent wrote stay as they are, the user message stays visible, and every
  // row can be opened again by hand. Rows that were already there when the run
  // began keep their state, so reading an older output is not interrupted.
  function beginRunFolds() {
    runFoldKeys = {};
    if (!thread) return;
    var nodes = thread.querySelectorAll("[data-dom-key]");
    for (var i = 0; i < nodes.length; i++) {
      var key = nodes[i].dataset && nodes[i].dataset.domKey;
      if (key) runFoldKeys[key] = true;
    }
  }

  function idleWindowQuiet() {
    return !!(document.hasFocus && document.hasFocus() && document.visibilityState !== "hidden");
  }

  function idlePingURL() {
    var rate = 22050;
    var n = Math.floor(rate * 0.12);
    var data = n * 2;
    var buf = new ArrayBuffer(44 + data);
    var view = new DataView(buf);
    function write(at, text) {
      for (var i = 0; i < text.length; i++) view.setUint8(at + i, text.charCodeAt(i));
    }
    write(0, "RIFF");
    view.setUint32(4, 36 + data, true);
    write(8, "WAVE");
    write(12, "fmt ");
    view.setUint32(16, 16, true);
    view.setUint16(20, 1, true);
    view.setUint16(22, 1, true);
    view.setUint32(24, rate, true);
    view.setUint32(28, rate * 2, true);
    view.setUint16(32, 2, true);
    view.setUint16(34, 16, true);
    write(36, "data");
    view.setUint32(40, data, true);
    for (var i = 0; i < n; i++) {
      var env = Math.min(1, i / 220) * Math.min(1, (n - i) / 900);
      var sample = Math.sin(2 * Math.PI * 880 * i / rate) * env * 0.35;
      view.setInt16(44 + i * 2, sample * 32767, true);
    }
    return URL.createObjectURL(new Blob([buf], { type: "audio/wav" }));
  }

  function noticeIdle() {
    if (idleWindowQuiet()) return;
    if (idleSoundOn) {
      try {
        var audio = new Audio(idlePingURL());
        var played = audio.play();
        if (played && played.catch) played.catch(function () {});
      } catch (e) {}
    }
    if (idleNotifyOn && typeof Notification === "function" && Notification.permission === "granted") {
      try { new Notification("Keimling", { body: "The run is idle." }); } catch (e) {}
    }
  }

  function collapseRunFolds() {
    var before = runFoldKeys || {};
    runFoldKeys = null;
    if (!thread) return;
    var dropped = [];
    var folds = thread.querySelectorAll("details");
    for (var i = 0; i < folds.length; i++) {
      var fold = folds[i];
      if (!fold.open || !fold.classList) continue;
      if (!fold.classList.contains("thinking-fold") &&
          !fold.classList.contains("tool-line") &&
          !fold.classList.contains("tool-output-fold")) continue;
      var owner = fold.closest ? fold.closest("[data-dom-key]") : null;
      var ownerKey = owner && owner.dataset && owner.dataset.domKey;
      if (ownerKey && before[ownerKey]) continue;
      // Clearing the follow marker first keeps the toggle handler from reading
      // this close as the user dismissing the live row.
      delete fold.dataset.autoFollow;
      fold.open = false;
      if (fold.dataset.domKey) dropped.push(fold.dataset.domKey);
    }
    if (heldOpen) {
      for (var n = 0; n < dropped.length; n++) delete heldOpen[dropped[n]];
    }
  }

  function setState(s) {
    var prev = agentState;
    agentState = s || "idle";
    var wasRun = prev === "running" || prev === "compacting" || prev === "awaiting_approval";
    var isRun = agentState === "running" || agentState === "compacting" || agentState === "awaiting_approval";
    if (isRun && !wasRun) {
      beginRunFolds();
      lastRunActivityAt = Date.now();
    }
    if (wasRun && agentState === "idle") {
      closeSideRun();
      collapseRunFolds();
      if (typeof noticeIdle === "function") noticeIdle();
    }
    stateEl.textContent = agentState;
    stateEl.className = "state " + agentState;
    if (agentState === "running" && lastRunActivityAt) {
      var quietMin = Math.floor((Date.now() - lastRunActivityAt) / 60000);
      if (quietMin >= 3) {
        stateEl.textContent = "running · " + quietMin + " min quiet";
        stateEl.title = "No output for " + quietMin + " minutes. Stop the run if it is stuck.";
      }
    }
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
    var queueHere = busyish;
    send.disabled = !connected || needsProvider;
    msg.disabled = !connected || needsProvider;
    if (attachFileBtn) attachFileBtn.disabled = !connected || needsProvider;
    if (msg) msg.placeholder = needsProvider ? "Add a model provider before sending" : "Message Keimling…";
    if (modelEl) modelEl.disabled = !connected || needsProvider;
    voiceBtn.disabled = !connected || !voiceOK || callLive;
    voiceBtn.hidden = !voiceOK;
    updateAudioActions();
    if (callBtn) {
      callBtn.hidden = !realtimeReady;
      callBtn.disabled = !connected || !realtimeReady || recording;
      callBtn.classList.toggle("live", callLive);
      setControlLabel(callBtn, callLive ? "Hang up" : "Call", callLive ? "Hang up. The transcript stays in the chat." : "Live voice call");
      setGlyph(callBtn, callLive ? "hangup" : "phone");
    }
    if (screenBtn) screenBtn.disabled = !connected || screenBusy;
    if (cameraBtn) cameraBtn.disabled = !connected || cameraBusy;
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
      var stopVisible = connected && (busyish || callLive);
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
      if (composerNarrow.matches) closeArtifactPane();
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
        if (composerNarrow.matches) closeArtifactPane();
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

  function pruneWorkspaces() {
    var status = document.getElementById("workspace-prune-status");
    if (status) status.textContent = "Checking…";
    return fetch(apiURL("/api/workspaces/prune"), { method: "POST", headers: headers(), body: "{}" })
      .then(readAPIResult)
      .then(function (j) {
        var n = (j && j.count) || 0;
        if (status) status.textContent = n ? n + " removed" : "Nothing to remove";
        return loadWorkspaces();
      })
      .then(function () { renderWorkspaceManage(); })
      .catch(function (err) { if (status) status.textContent = String(err); });
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
          renderSideChatRef();
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

  function refreshSidebar() {
    if (!connected) return Promise.resolve();
    resetSideSamples();
    renderSidePerf();
    var jobs = [
      refreshSideContext(),
      loadThreadAutopilot(),
      loadArtifactsList().then(function (items) {
        var key = artifactKey(currentArtifact);
        if (!key) return;
        var still = (items || []).some(function (art) { return artifactKey(art) === key; });
        if (still) return;
        if (items && items.length) showArtifact(sortArtifacts(items)[0]);
        else {
          currentArtifact = null;
          setArtifactEmpty(true);
        }
      }).catch(function () {}),
      loadMcpList().catch(function () {})
    ];
    if (sideTab === "agents" || sideTab === "now") jobs.push(loadAgentBoard());
    return Promise.all(jobs);
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
    clearLive();
    hideApproval();
    return loadSession()
      .then(function () {
        connectEvents();
        return Promise.all([loadWorkspaces(), loadThreads(), loadAgents(), refreshSidebar()]);
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
    clearLive();
    hideApproval();
    return loadSession()
      .then(function () {
        connectEvents();
        return Promise.all([loadWorkspaces(), loadThreads(), loadAgents(), refreshSidebar()]);
      })
      .then(function () {
        if (msg) msg.focus();
      });
  }

  // A spawn does not attach: the host leaves the open view alone, so the chat
  // stays where it is and the new agent is opened from the sidebar.
  function afterAgentSpawn(name) {
    var label = name || "The agent";
    return Promise.all([loadAgents(), refreshSidebar()]).then(function () {
      appendLive("[agents] " + label + " runs in the background. Open it in the sidebar.");
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

  function chatRefAddress() {
    if (currentAgentInstanceID || !currentWorkspaceID || !currentThreadID || currentThreadID === "main") return "";
    return "chat:" + currentWorkspaceID + "/" + currentThreadID;
  }

  function renderSideChatRef() {
    var foot = document.getElementById("side-chat-ref");
    var label = document.getElementById("side-chat-ref-label");
    if (!foot || !label) return;
    var addr = chatRefAddress();
    foot.hidden = !addr;
    if (!addr) return;
    var wsName = currentWorkspaceID;
    for (var i = 0; i < workspacesCache.length; i++) {
      if (workspacesCache[i].id === currentWorkspaceID) {
        wsName = workspacesCache[i].name || wsName;
        break;
      }
    }
    var threadName = "This chat";
    for (var t = 0; t < threadsCache.length; t++) {
      if (threadsCache[t].id === currentThreadID) {
        threadName = threadsCache[t].title || threadsCache[t].name || threadName;
        break;
      }
    }
    label.textContent = wsName + " / " + threadName;
    label.title = addr;
  }

  function copyChatRef() {
    var addr = chatRefAddress();
    var btn = document.getElementById("side-chat-ref-copy");
    if (!addr || !btn) return;
    var mark = function () {
      btn.textContent = "Copied";
      window.setTimeout(function () { btn.textContent = "Copy"; }, 1500);
    };
    var fallback = function () {
      var ta = document.createElement("textarea");
      ta.value = addr;
      ta.setAttribute("readonly", "");
      ta.style.position = "fixed";
      ta.style.left = "-9999px";
      document.body.appendChild(ta);
      ta.select();
      try { document.execCommand("copy"); } catch (e) {}
      ta.remove();
    };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(addr).then(mark).catch(function () { fallback(); mark(); });
      return;
    }
    fallback();
    mark();
  }

  var threadListDrawn = "";

  function renderThreadListFromCache() {
    renderSideChatRef();
    if (!threadListEl) return;
    var query = threadSearchEl ? String(threadSearchEl.value || "").trim() : "";
    var threads = visibleThreads(threadsCache, currentThreadID);
    var sig = [
      query, currentThreadID, currentAgentInstanceID, runningThreadID, stoppingThreadID,
      callLive ? 1 : 0, threadsShowAll ? 1 : 0, threadsCache.length,
      threads.map(function (th) { return th.id + "\t" + (th.title || ""); }).join("\n")
    ].join("\u0001");
    if (sig === threadListDrawn && threadListEl.childElementCount) return;
    threadListDrawn = sig;
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

  // A draft belongs to the chat it was typed in. An agent chat is an instance,
  // a workspace chat is a thread, so the instance id wins when both are set.
  function chatDraftKey() {
    if (currentAgentInstanceID) return "agent:" + currentAgentInstanceID;
    if (currentThreadID) return "thread:" + currentThreadID;
    return "";
  }

  // Keep what is in the composer under the chat it belongs to.
  function saveChatDraft() {
    if (!msg) return;
    if (!draftChatKey) draftChatKey = chatDraftKey();
    if (!draftChatKey) return;
    chatDrafts[draftChatKey] = msg.value;
  }

  // The open chat changed: park the old draft and show the new chat one.
  function syncChatDraft() {
    if (!msg) return;
    var next = chatDraftKey();
    if (!draftChatKey) {
      draftChatKey = next;
      return;
    }
    if (next === draftChatKey) return;
    chatDrafts[draftChatKey] = msg.value;
    draftChatKey = next;
    msg.value = next ? (chatDrafts[next] || "") : "";
    fitComposer();
  }

  // A sent message is no longer a draft.
  function forgetChatDraft() {
    if (draftChatKey) delete chatDrafts[draftChatKey];
  }

  function switchThread(id) {
    if (!id) return;
    if (!currentAgentInstanceID && id === currentThreadID) return;
    saveChatDraft();
    if (stateEl) stateEl.textContent = "opening";
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
        clearLive();
        hideApproval();
        syncChatDraft();
        return loadSession();
      })
      .then(function () {
        connectEvents();
        return Promise.all([loadThreads(), loadAgents(), refreshSidebar()]);
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
        return Promise.all([loadThreads(), loadAgents(), refreshSidebar()]);
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

  var agentsPollDrawn = "";

  function agentsPollSig() {
    return (currentAgentInstanceID || "") + "\n" + agentState + "\n" + agentsCache.map(function (ag) {
      return [ag.id, ag.name || "", ag.busy ? 1 : 0, ag.running ? 1 : 0, ag.attached ? 1 : 0, ag.phase || "", ag.call ? 1 : 0, stoppingAgents[ag.id] ? 1 : 0].join("\t");
    }).join("\n");
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
          syncChatDraft();
          var sig = agentsPollSig();
          if (opts.poll && sig === agentsPollDrawn) return;
          agentsPollDrawn = sig;
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
      if (sideTab === "agents" || sideTab === "now") loadAgentBoard();
      if (artifactPane && !artifactPane.hidden) {
        loadApprovals();
        if (sideTab === "gateways") loadGateways();
        if (sideTab === "escalations") loadEscalations();
      }
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
      box.disabled = row.allowed === false || row.fixed === true;
      box.dataset.accessId = row.id || row.name;
      if (row.allowed === false) box.dataset.allowed = "0";
      if (row.fixed === true) {
        box.dataset.fixed = "1";
        label.title = "This contact comes from the definition. Change it there.";
      }
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

  function renderPeerChoices(el, rows, selected, skipID) {
    if (!el) return;
    var want = {};
    (selected || []).forEach(function (id) { if (id) want[id] = true; });
    el.innerHTML = "";
    (rows || []).forEach(function (row) {
      if (!row || !row.id || row.id === skipID) return;
      var label = document.createElement("label");
      label.className = "agent-access-row";
      var box = document.createElement("input");
      box.type = "checkbox";
      box.checked = !!want[row.id];
      box.dataset.peerId = row.id;
      var text = document.createElement("span");
      text.textContent = row.name || row.id;
      label.appendChild(box);
      label.appendChild(text);
      el.appendChild(label);
    });
  }

  function checkedPeerIDs(el) {
    var ids = [];
    if (!el) return ids;
    el.querySelectorAll("input[type=checkbox]").forEach(function (box) {
      if (box.checked) ids.push(box.dataset.peerId);
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
            return { id: p.id, label: p.name || p.id, granted: !!p.granted, fixed: !!p.fixed };
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
        var spendUSD = document.getElementById("agent-access-spend-usd");
        var spendTokens = document.getElementById("agent-access-spend-tokens");
        if (spendUSD) {
          spendUSD.value = j.spend_limit_usd ? String(j.spend_limit_usd) : "";
          spendUSD.oninput = markAgentAccessUnsaved;
        }
        if (spendTokens) {
          spendTokens.value = j.spend_limit_tokens ? String(j.spend_limit_tokens) : "";
          spendTokens.oninput = markAgentAccessUnsaved;
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
    var spendUSD = document.getElementById("agent-access-spend-usd");
    var spendTokens = document.getElementById("agent-access-spend-tokens");
    if (spendUSD) body.spend_limit_usd = spendUSD.value.trim();
    if (spendTokens) body.spend_limit_tokens = spendTokens.value.trim();
    if (agentAccessPeers && agentAccessPeers.dataset.ready === "1") {
      body.peers = checkedAccess(agentAccessPeers, true);
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
    if (name === "create" || name === "edit") placeDefinitionTests(name);
    var note = document.getElementById("agent-def-test-draft-note");
    if (note) note.hidden = name !== "create";
    showAgentDefErr("");
  }

  function placeDefinitionTests(name) {
    var section = document.getElementById("agent-def-tests");
    var slot = document.getElementById(name === "create" ? "agent-def-tests-slot-create" : "agent-def-tests-slot-edit");
    if (section && slot && section.parentNode !== slot) slot.appendChild(section);
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
    renderPeerChoices(document.getElementById("agent-instance-peers"), allInstancesCache, []);
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
      renderPeerChoices(document.getElementById("agent-def-edit-peers"), definitionsCache, def.peers || []);
      renderToolRows(readToolsFromForm());
    });
    clearDefinitionTestForm();
    renderDefinitionTests(def);
  }

  var editingDefinitionTestID = "";
  var definitionTestUndo = null;
  var creatingTests = [];

  function currentEditingDefinition() {
    for (var i = 0; i < definitionsCache.length; i++) {
      if (definitionsCache[i].id === editingDefinitionID) return definitionsCache[i];
    }
    return null;
  }

  function testsOnScreen() {
    if (editingDefinitionID) {
      var def = currentEditingDefinition();
      return (def && def.tests) || [];
    }
    return creatingTests;
  }

  function toolResultsList(value) {
    if (Array.isArray(value)) return value.slice();
    if (value == null || value === "") return value == null ? [] : [""];
    return [String(value)];
  }

  function grantedToolsForTestForm() {
    var root = document.getElementById(editingDefinitionID ? "agent-def-edit-tools" : "agent-def-create-tools");
    return checkedToolNames(root).filter(function (name) {
      return name && name !== "message_agent";
    });
  }

  function fieldInput(value, placeholder) {
    var input = document.createElement("input");
    input.type = "text";
    input.value = value || "";
    if (placeholder) input.placeholder = placeholder;
    input.autocomplete = "off";
    return input;
  }

  function partyKindSelect(kind) {
    var select = document.createElement("select");
    [["thread", "Thread"], ["agent", "Agent"]].forEach(function (pair) {
      var opt = document.createElement("option");
      opt.value = pair[0];
      opt.textContent = pair[1];
      select.appendChild(opt);
    });
    select.value = kind === "thread" ? "thread" : "agent";
    return select;
  }

  function removeButton(kind) {
    var button = document.createElement("button");
    button.type = "button";
    button.className = "ghost";
    button.textContent = "Remove";
    button.dataset.testRemove = kind;
    return button;
  }

  function readPartiesFromForm() {
    var rows = [];
    document.querySelectorAll("#agent-def-test-parties .agent-test-party").forEach(function (row) {
      var inputs = row.querySelectorAll("input");
      var kind = row.querySelector("select");
      rows.push({
        id: inputs[0] ? inputs[0].value : "",
        name: inputs[1] ? inputs[1].value : "",
        kind: kind ? kind.value : "agent",
      });
    });
    return rows;
  }

  function partyOptions(selected) {
    var select = document.createElement("select");
    var none = document.createElement("option");
    none.value = "";
    none.textContent = "—";
    select.appendChild(none);
    readPartiesFromForm().forEach(function (p) {
      var id = String(p.id || "").trim();
      if (!id) return;
      var opt = document.createElement("option");
      opt.value = id;
      var label = String(p.name || "").trim();
      opt.textContent = label && label !== id ? label + " (" + id + ")" : id;
      select.appendChild(opt);
    });
    select.value = selected || "";
    if (select.value !== (selected || "")) select.value = "";
    return select;
  }

  function renderPartyRows(parties) {
    var host = document.getElementById("agent-def-test-parties");
    if (!host) return;
    host.textContent = "";
    var list = parties && parties.length ? parties : [{ id: "", name: "", kind: "thread" }];
    list.forEach(function (p) {
      var row = document.createElement("div");
      row.className = "agent-test-party";
      row.appendChild(fieldInput(p.id, "id"));
      row.appendChild(fieldInput(p.name, "Name"));
      row.appendChild(partyKindSelect(p.kind));
      row.appendChild(removeButton("party"));
      host.appendChild(row);
    });
  }

  function renderStepRows(steps) {
    var host = document.getElementById("agent-def-test-steps");
    if (!host) return;
    host.textContent = "";
    var list = steps && steps.length ? steps : [{ from: "", text: "", to: "", check: "" }];
    list.forEach(function (step) {
      var row = document.createElement("div");
      row.className = "agent-test-step";
      [["Writes to this agent", step.from, step.text, "What they send"], ["This agent writes to", step.to, step.check, "What that message must mean"]].forEach(function (spec, index) {
        var line = document.createElement("div");
        line.className = "agent-test-step-line";
        var label = document.createElement("span");
        label.textContent = spec[0];
        line.appendChild(label);
        line.appendChild(partyOptions(spec[1]));
        line.appendChild(fieldInput(spec[2], spec[3]));
        line.appendChild(index === 1 ? removeButton("step") : document.createElement("span"));
        row.appendChild(line);
      });
      host.appendChild(row);
    });
  }

  function readStepsFromForm() {
    var steps = [];
    document.querySelectorAll("#agent-def-test-steps .agent-test-step").forEach(function (row) {
      var lines = row.querySelectorAll(".agent-test-step-line");
      function part(line) {
        if (!line) return { who: "", text: "" };
        var select = line.querySelector("select");
        var input = line.querySelector("input");
        return { who: select ? select.value : "", text: input ? input.value : "" };
      }
      var write = part(lines[0]);
      var expect = part(lines[1]);
      steps.push({ from: write.who, text: write.text, to: expect.who, check: expect.text });
    });
    return steps;
  }

  function syncStepPartySelects() {
    document.querySelectorAll("#agent-def-test-steps select").forEach(function (select) {
      select.replaceWith(partyOptions(select.value));
    });
  }

  function readToolsFromForm() {
    var tools = {};
    document.querySelectorAll("#agent-def-test-tools .agent-test-tool").forEach(function (block) {
      var name = block.dataset.tool || "";
      if (!name) return;
      var results = [];
      block.querySelectorAll("input").forEach(function (input) {
        results.push(input.value);
      });
      tools[name] = results;
    });
    return tools;
  }

  function renderToolRows(tools, onlyGranted) {
    var host = document.getElementById("agent-def-test-tools");
    if (!host) return;
    var saved = tools == null ? readToolsFromForm() : tools;
    host.textContent = "";
    var names = grantedToolsForTestForm();
    if (!onlyGranted) {
      Object.keys(saved).forEach(function (name) {
        if (name !== "message_agent" && names.indexOf(name) < 0) names.push(name);
      });
    }
    if (!names.length) {
      var empty = document.createElement("p");
      empty.className = "hint";
      empty.textContent = "No simulated tools. message_agent is the script. Check a tool above to set what it returns.";
      host.appendChild(empty);
      return;
    }
    names.forEach(function (name) {
      var results = toolResultsList(saved[name]);
      if (!results.length) results = [""];
      var block = document.createElement("div");
      block.className = "agent-test-tool";
      block.dataset.tool = name;
      var title = document.createElement("span");
      title.className = "agent-test-label";
      title.textContent = name;
      block.appendChild(title);
      results.forEach(function (result) {
        var row = document.createElement("div");
        row.className = "agent-test-tool-call";
        var label = document.createElement("span");
        label.textContent = "Call " + (block.querySelectorAll(".agent-test-tool-call").length + 1);
        row.appendChild(label);
        row.appendChild(fieldInput(result, "Result, empty is allowed"));
        row.appendChild(removeButton("tool"));
        block.appendChild(row);
      });
      var add = document.createElement("button");
      add.type = "button";
      add.className = "ghost";
      add.textContent = "Add the next call";
      add.dataset.testAdd = "tool";
      add.dataset.tool = name;
      block.appendChild(add);
      host.appendChild(block);
    });
  }

  function closeDefinitionTestChat() {
    var dlg = document.getElementById("agent-test-chat-dialog");
    if (dlg && dlg.open) dlg.close();
  }

  function closeDefinitionTestEditor() {
    var dlg = document.getElementById("agent-test-dialog");
    if (dlg && dlg.open) dlg.close();
  }

  function showDefinitionTestErr(msg) {
    var el = document.getElementById("agent-test-dialog-err");
    if (!el) return;
    el.hidden = !msg;
    el.textContent = msg || "";
  }

  function openDefinitionTestEditor(test) {
    if (test) fillDefinitionTestForm(test);
    else clearDefinitionTestForm();
    var title = document.getElementById("agent-test-dialog-title");
    if (title) title.textContent = test ? "Edit test" : "New test";
    showDefinitionTestErr("");
    var dlg = document.getElementById("agent-test-dialog");
    if (dlg && !dlg.open) dlg.showModal();
  }

  function openDefinitionTestChat(test, index) {
    var runs = test.last_runs || [];
    var run = runs[index];
    var dlg = document.getElementById("agent-test-chat-dialog");
    var log = document.getElementById("agent-def-test-chat-log");
    var title = document.getElementById("agent-def-test-chat-title");
    var detail = document.getElementById("agent-def-test-chat-detail");
    if (!dlg || !log || !run) return;
    if (title) title.textContent = (test.name || "Test") + " · Run " + (index + 1);
    if (detail) detail.textContent = run.pass ? "Passed" : (run.detail || "Failed");
    log.textContent = "";
    if (run.transcript && run.transcript.length) appendMessageList(log, run.transcript);
    else log.appendChild(document.createTextNode("This run has no chat. Run the test again to keep one."));
    if (!dlg.open) dlg.showModal();
  }

  function clearDefinitionTestForm() {
    editingDefinitionTestID = "";
    var name = document.getElementById("agent-def-test-name");
    if (name) name.value = "";
    renderPartyRows([]);
    renderStepRows([]);
    renderToolRows({});
  }

  function localTestProblem(test) {
    if (!test.name) return "Test name required.";
    if (!test.parties.length) return "Add at least one party. Parties are the other side, not this agent.";
    if (!test.steps.length) return "Add a step: someone writes to this agent, or this agent writes to someone.";
    return "";
  }

  function definitionTestFormTouched() {
    if ((document.getElementById("agent-def-test-name").value || "").trim()) return true;
    if (readPartiesFromForm().some(function (p) { return String(p.id || "").trim() || String(p.name || "").trim(); })) return true;
    if (readStepsFromForm().some(function (s) { return String(s.from || s.to || s.text || s.check || "").trim(); })) return true;
    var tools = readToolsFromForm();
    return Object.keys(tools).some(function (name) {
      var results = tools[name] || [];
      return results.length > 1 || results.some(function (r) { return String(r) !== ""; });
    });
  }

  function parseDefinitionTestForm() {
    var name = (document.getElementById("agent-def-test-name").value || "").trim();
    var parties = readPartiesFromForm().filter(function (p) {
      return String(p.id || "").trim();
    }).map(function (p) {
      var id = String(p.id).trim();
      return { id: id, name: String(p.name || "").trim() || id, kind: p.kind || "agent" };
    });
    var steps = [];
    readStepsFromForm().forEach(function (s) {
      var step = {};
      if (String(s.from || "").trim()) {
        step.from = String(s.from).trim();
        step.text = String(s.text || "").trim();
      }
      if (String(s.to || "").trim()) {
        step.to = String(s.to).trim();
        step.check = String(s.check || "").trim();
      }
      if (step.from || step.to) steps.push(step);
    });
    var tools = {};
    var raw = readToolsFromForm();
    Object.keys(raw).forEach(function (tool) {
      var results = raw[tool] || [];
      var keep = results.length > 1 || results.some(function (r) { return String(r) !== ""; });
      if (keep) tools[tool] = results;
    });
    var test = { name: name, parties: parties, steps: steps, tools: tools };
    if (editingDefinitionTestID) test.id = editingDefinitionTestID;
    return test;
  }

  function fillDefinitionTestForm(test) {
    editingDefinitionTestID = test.id || "";
    document.getElementById("agent-def-test-name").value = test.name || "";
    renderPartyRows(test.parties || []);
    renderStepRows((test.steps || []).map(function (step) {
      return { from: step.from || "", text: step.text || "", to: step.to || "", check: step.check || "" };
    }));
    var tools = {};
    Object.keys(test.tools || {}).forEach(function (name) {
      tools[name] = toolResultsList(test.tools[name]);
    });
    renderToolRows(tools);
    closeDefinitionTestChat();
  }

  function renderDefinitionTests(def) {
    var list = document.getElementById("agent-def-test-list");
    if (!list) return;
    list.textContent = "";
    var tests = editingDefinitionID ? ((def && def.tests) || []) : creatingTests;
    if (!tests.length) {
      var empty = document.createElement("li");
      empty.className = "hint";
      empty.textContent = "No tests yet.";
      list.appendChild(empty);
      return;
    }
    tests.forEach(function (test) {
      var row = document.createElement("li");
      row.className = "agent-test-row";
      var name = document.createElement("span");
      name.className = "agent-test-name";
      name.textContent = test.name || test.id;
      row.appendChild(name);
      var runs = test.last_runs || [];
      if (runs.length) {
        var block = document.createElement("div");
        block.className = "agent-test-runs";
        runs.forEach(function (run, i) {
          var line = document.createElement("div");
          line.className = "agent-test-run";
          var label = document.createElement("span");
          label.textContent = "Run " + (i + 1);
          var state = document.createElement("span");
          state.className = "agent-test-run-state" + (run.pass ? "" : " fail");
          state.textContent = run.pass ? "Passed" : (run.detail || "Failed");
          var chat = document.createElement("button");
          chat.type = "button";
          chat.className = "ghost";
          chat.textContent = "Chat";
          chat.dataset.testAct = "chat";
          chat.dataset.testId = test.id;
          chat.dataset.runIndex = String(i);
          line.appendChild(label);
          line.appendChild(state);
          line.appendChild(chat);
          block.appendChild(line);
        });
        var all = runs.length === 3 && runs.every(function (run) { return run.pass; });
        var overall = document.createElement("p");
        overall.className = "hint";
        overall.textContent = all ? "Test passed." : "Test failed.";
        block.appendChild(overall);
        row.appendChild(block);
      }
      [["Run", "run"], ["Edit", "edit"], ["Delete", "delete"]].forEach(function (pair) {
        var button = document.createElement("button");
        button.type = "button";
        button.className = pair[1] === "run" ? "" : "ghost";
        button.textContent = pair[0];
        button.dataset.testAct = pair[1];
        button.dataset.testId = test.id;
        if (pair[1] === "run" && !editingDefinitionID) {
          button.disabled = true;
          button.title = "Save the definition before running this test.";
        }
        row.appendChild(button);
      });
      list.appendChild(row);
    });
  }

  function showDefinitionTestUndo(test) {
    definitionTestUndo = test;
    var bar = document.getElementById("agent-def-test-undo");
    if (!bar) return;
    bar.hidden = false;
    bar.textContent = "";
    bar.appendChild(document.createTextNode("Removed “" + (test.name || test.id) + "”. "));
    var undo = document.createElement("button");
    undo.type = "button";
    undo.className = "ghost";
    undo.textContent = "Undo";
    undo.addEventListener("click", function () {
      if (!definitionTestUndo) return;
      saveDefinitionTest(definitionTestUndo).catch(function (err) { showAgentDefErr(String(err)); });
    });
    bar.appendChild(undo);
  }

  function rememberDraftTest(test) {
    if (!test.id) test.id = "draft-" + Date.now();
    var next = [];
    var replaced = false;
    creatingTests.forEach(function (one) {
      if (one.id === test.id) {
        next.push(test);
        replaced = true;
      } else next.push(one);
    });
    if (!replaced) next.push(test);
    creatingTests = next;
    definitionTestUndo = null;
    var bar = document.getElementById("agent-def-test-undo");
    if (bar) bar.hidden = true;
    clearDefinitionTestForm();
    renderDefinitionTests(null);
    closeDefinitionTestEditor();
  }

  function draftTestsForSave(extra) {
    var tests = creatingTests.map(function (test) {
      var copy = JSON.parse(JSON.stringify(test));
      if (String(copy.id || "").indexOf("draft-") === 0) delete copy.id;
      delete copy.last_runs;
      return copy;
    });
    if (extra) {
      var copy = JSON.parse(JSON.stringify(extra));
      if (String(copy.id || "").indexOf("draft-") === 0) delete copy.id;
      delete copy.last_runs;
      tests.push(copy);
    }
    return tests;
  }

  function saveDefinitionTest(test) {
    if (!editingDefinitionID) {
      rememberDraftTest(test);
      return Promise.resolve();
    }
    return fetch(apiURL("/api/agents/recipes/tests"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: editingDefinitionID, test: test }),
    }).then(readAPIResult).then(function () {
      definitionTestUndo = null;
      var bar = document.getElementById("agent-def-test-undo");
      if (bar) bar.hidden = true;
      clearDefinitionTestForm();
      closeDefinitionTestEditor();
      return loadDefinitions();
    }).then(function () {
      renderDefinitionTests(currentEditingDefinition());
    });
  }

  function runDefinitionTest(testID, button) {
    if (button) {
      button.disabled = true;
      button.textContent = "Running…";
    }
    return fetch(apiURL("/api/agents/recipes/tests/run"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: editingDefinitionID, test_id: testID }),
    }).then(readAPIResult).then(function () {
      return loadDefinitions();
    }).then(function () {
      renderDefinitionTests(currentEditingDefinition());
    }).catch(function (err) {
      showAgentDefErr(String(err));
      return loadDefinitions().then(function () {
        renderDefinitionTests(currentEditingDefinition());
      });
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
    // An empty list is a real choice and grants nothing. Say so before saving,
    // so nobody wonders later why the agent cannot act.
    if (!tools.length && typeof confirm === "function" && !confirm("No tool is checked. This definition will grant no tools at all. Save anyway?")) {
      return;
    }
    var createModel = document.getElementById("agent-def-model");
    var model = createModel ? String(createModel.value || "").trim() : "";
    var createMode = document.getElementById("agent-def-mode");
    var mode = createMode ? createMode.value : "auto";
    var peers = checkedPeerIDs(document.getElementById("agent-def-create-peers"));
    var pendingTest = null;
    if (definitionTestFormTouched()) pendingTest = parseDefinitionTestForm();
    var body = { name: name, instructions: instructions, model: model, tools: tools, peers: peers, mode: mode, tests: draftTestsForSave(pendingTest) };
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
        creatingTests = [];
        clearDefinitionTestForm();
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
    var peers = checkedPeerIDs(document.getElementById("agent-def-edit-peers"));
    var body = { id: editingDefinitionID, name: name, instructions: instructions, model: model, tools: tools, peers: peers, mode: mode };
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
        peers: checkedPeerIDs(document.getElementById("agent-instance-peers")),
      }),
    })
      .then(readAPIResult)
      .then(function (j) {
        if (agentDefDialog && agentDefDialog.open) agentDefDialog.close();
        return afterAgentSpawn((j.instance && (j.instance.name || j.instance.id)) || "");
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
        var names = (j.instances || []).map(function (inst) { return inst.name || inst.id; }).join(", ");
        return afterAgentSpawn(names);
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

  function toolInputText(raw) {
    var s = String(raw || "");
    if (!s) return "";
    try {
      return JSON.stringify(JSON.parse(s), null, 2);
    } catch (e) {
      return s;
    }
  }

  function toolFirstLine(cmd) {
    var line = String(cmd || "").trim().split("\n")[0] || "";
    if (line.length > 96) return line.slice(0, 93) + "…";
    return line;
  }

  function toolSummaryRest(name, cmd) {
    var line = toolFirstLine(cmd);
    var n = String(name || "").trim();
    if (!n || n === "exec") return line;
    if (line === n) return "";
    if (line.indexOf(n + " ") === 0) return line.slice(n.length).trim();
    return line;
  }

  function toolBox(className, labelHTML, bodyHTML) {
    var box = document.createElement("div");
    box.className = "tool-box " + className;
    box.innerHTML =
      '<div class="tool-box-label">' + labelHTML + "</div>" +
      '<pre class="tool-body">' + bodyHTML + "</pre>";
    return box;
  }

  function formatReviewTime(ms) {
    var s = Number(ms) / 1000;
    if (!isFinite(s) || s <= 0) return "";
    if (s < 10) return s.toFixed(1) + "s";
    return Math.round(s) + "s";
  }

  function toolCard(m) {
    var name = m.name || "tool";
    var cmd = m.command || m.text || name;
    var inputBody = toolInputText(m.input);
    if (!inputBody && cmd && cmd !== name) inputBody = cmd;
    if (!inputBody) inputBody = "—";
    var wrap = document.createElement("details");
    wrap.className = "tool-line";
    if (m.pending) wrap.className += " pending";
    var filtered = !!m.filtered && !!(m.raw && m.raw !== m.result);
    var fullBody = filtered ? (m.raw || "") : (m.raw || m.result || "");
    var filteredBody = filtered ? (m.result || "") : "";

    var status = m.status || (m.pending ? "pending" : (m.ok === false ? "failed" : "ran"));
    if (status === "failed") wrap.className += " failed";
    if (status === "stopped") wrap.className += " stopped";
    if (m.reviewing) wrap.className += " reviewing";
    var ranLabel = status === "failed" ? "Failed" : status === "stopped" ? "Stopped" : "Ran";
    if (m.pending) ranLabel = m.reviewing ? "Reviewing" : m.waiting ? "Awaiting approval" : "Running";
    var reviewNote = formatReviewTime(m.reviewMs);
    var summary = document.createElement("summary");
    var rest = toolSummaryRest(name, cmd);
    summary.innerHTML =
      '<span class="tool-line-ran">' + ranLabel + "</span>" +
      '<span class="tool-line-name">' + esc(name) + "</span>" +
      (rest ? '<code class="tool-line-cmd">' + esc(rest) + "</code>" : "") +
      (reviewNote ? '<span class="tool-line-review" title="Time in the approval review">checked ' + reviewNote + "</span>" : "") +
      (filtered ? '<span class="tool-line-funnel" title="Output filtered for model">▽</span>' : "") +
      (m.pending ? '<span class="tool-line-pending">…</span>' : "");
    wrap.appendChild(summary);

    var stack = document.createElement("div");
    stack.className = "tool-stack";
    stack.appendChild(toolBox("tool-box-cmd", "Input", esc(inputBody)));

    var pendingText = m.reviewing ? "Waiting for the approval review…" : m.waiting ? "Waiting for your approval…" : "Running…";
    var outputHTML = fullBody ? window.ansiToHtml(fullBody) : esc(m.pending ? pendingText : "—");
    if (filtered && filteredBody) {
      var fold = document.createElement("details");
      fold.className = "tool-output-fold";
      fold.dataset.domKey = ensureDomKey(m) + ":output";
      if (heldOpen && heldOpen[fold.dataset.domKey]) fold.open = true;
      var foldSum = document.createElement("summary");
      foldSum.textContent = "Output";
      var foldBody = document.createElement("pre");
      foldBody.className = "tool-body";
      foldBody.innerHTML = outputHTML;
      fold.appendChild(foldSum);
      fold.appendChild(foldBody);
      stack.appendChild(fold);
      stack.appendChild(toolBox("tool-box-filtered", 'Filtered <span class="tool-line-funnel">▽</span>', window.ansiToHtml(filteredBody)));
    } else {
      stack.appendChild(toolBox("tool-box-raw", "Output", outputHTML));
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
        return wrapChatTables(highlightFences(html));
      }
    } catch (e) {}
    return window.ansiToHtml(raw);
  }

  function wrapChatTables(html) {
    if (!html || html.indexOf("<table") < 0) return html;
    var wrap = document.createElement("div");
    wrap.innerHTML = html;
    wrap.querySelectorAll("table").forEach(function (table) {
      if (table.parentElement && table.parentElement.classList.contains("md-table")) return;
      var scroller = document.createElement("div");
      scroller.className = "md-table";
      table.parentNode.insertBefore(scroller, table);
      scroller.appendChild(table);
    });
    return wrap.innerHTML;
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
      return ["tool", m.name || "", m.command || "", m.input || "", m.pending ? "1" : "0", m.reviewing ? "1" : "0", m.waiting ? "1" : "0", String(m.reviewMs || 0), m.status || "", m.result || "", m.raw || "", m.filtered ? "1" : "0"].join("\u0001");
    }
    return [m.role || "", m.text || "", m.name || "", (m.files || []).join(","), m.streaming ? "1" : "0", m.collapsed ? "1" : "0", m.line || 0, m.branch_line || 0, m.branch_index || 0, m.branch_count || 0].join("\u0001");
  }

  // True when two signatures differ only by the streaming flag, so the node can stay.
  function sameMessageShape(prevSig, nextSig) {
    var a = String(prevSig || "").split("\u0001");
    var b = String(nextSig || "").split("\u0001");
    if (a.length !== b.length || a.length < 5 || a[0] === "tool") return false;
    a[4] = "0";
    b[4] = "0";
    return a.join("\u0001") === b.join("\u0001");
  }

  function transcriptMatches(live, incoming) {
    if (!live || live.length !== (incoming || []).length) return false;
    for (var i = 0; i < live.length; i++) {
      var a = live[i];
      var b = incoming[i];
      if ((a.role || "") !== (b.role || "")) return false;
      if (a.role === "tool_group") {
        if ((a.name || "") !== (b.name || "")) return false;
        if ((a.command || "") !== (b.command || "")) return false;
        if ((a.result || "") !== (b.result || "")) return false;
        if (!!a.pending) return false;
        continue;
      }
      if (String(a.text || "").trim() !== String(b.text || "").trim()) return false;
    }
    return true;
  }

  function adoptSessionFields(live, incoming) {
    for (var i = 0; i < live.length; i++) {
      var a = live[i];
      var b = incoming[i];
      // event would pull the live row into a chapter on the next paint.
      if (b.line) a.line = b.line;
      if (b.branch_line) a.branch_line = b.branch_line;
      if (b.branch_index != null) a.branch_index = b.branch_index;
      if (b.branch_count) a.branch_count = b.branch_count;
      if (b.files) a.files = b.files;
      delete a.streaming;
    }
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
      if (d.dataset && d.dataset.autoFollow === "1") return;
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
    if ((m.role || "") === "thinking") {
      var thought = document.createElement("article");
      thought.className = "msg thinking";
      var fold = document.createElement("details");
      fold.className = "thinking-fold";
      var sum = document.createElement("summary");
      sum.textContent = "Thinking";
      var body = document.createElement("pre");
      body.className = "thinking-body";
      body.textContent = m.text || "";
      fold.appendChild(sum);
      fold.appendChild(body);
      thought.appendChild(fold);
      stampMessageEl(thought, m, "thinking");
      fold.dataset.domKey = ensureDomKey(m) + ":think";
      var userOpen = heldOpen && heldOpen[fold.dataset.domKey];
      var follow = followThinking && m.streaming && followDismissedKey !== ensureDomKey(m);
      if (userOpen || follow) {
        applyingFollow = true;
        fold.open = true;
        applyingFollow = false;
        if (follow && !userOpen) fold.dataset.autoFollow = "1";
      }
      parent.appendChild(thought);
      return "thinking";
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
    // An agent chat may stop its run for the fork, so its button stays usable.
    var force = !!currentAgentInstanceID;
    edit.disabled = !force && agentBusy();
    edit.addEventListener("click", function () { openBranchEditor(el, m, source, force); });
    nav.appendChild(edit);
    body.appendChild(nav);
  }

  function openBranchEditor(el, m, source, force) {
    if (agentBusy() && !force) {
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
      function submit(forceNow) {
        sendEditedMessage(source, text, forceNow).catch(function (err) {
          send.disabled = false;
          cancel.disabled = false;
          appendLive("\n[branch] " + ((err && err.message) || err) + "\n");
        });
      }
      if (force && agentBusy()) {
        askConfirm("The agent is working. Sending stops the current run and continues from this message.", "Stop and send").then(function (ok) {
          if (!ok) {
            // Cancelling keeps the editor open and stops nothing.
            send.disabled = false;
            cancel.disabled = false;
            return;
          }
          submit(true);
        });
        return;
      }
      submit(force);
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

  function sendEditedMessage(line, text, force) {
    // A forced fork stops the running agent run first; workspace threads wait.
    if (agentBusy() && !force) return Promise.reject(new Error("The agent is still working."));
    return postJSON("/api/branch", { line: line, text: text, force: !!force }).then(function () {
      if (agentState === "idle") resetSideSamples();
      clearLive();
      setState("running");
      return loadSession();
    }).then(function () {
      scrollTranscriptToEnd();
      return refreshSideContext();
    });
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
    if (m.role === "user") return true;
    if (m.role === "assistant" && String(m.text || "").trim()) return true;
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

  // Within this many pixels of the bottom, new transcript growth stays at the bottom.
  var transcriptFollowPx = 200;
  var transcriptStick = true;
  var transcriptHoldUntil = 0;
  var transcriptTouchY = null;

  function transcriptFromBottom() {
    var el = document.scrollingElement || document.documentElement;
    return el.scrollHeight - el.scrollTop - el.clientHeight;
  }

  function holdTranscriptFollow() {
    transcriptHoldUntil = Date.now() + 1500;
    transcriptStick = false;
  }

  function followWanted() {
    if (Date.now() < transcriptHoldUntil) return false;
    return transcriptFromBottom() <= transcriptFollowPx;
  }

  function noteTranscriptStick() {
    if (Date.now() < transcriptHoldUntil) {
      transcriptStick = false;
      return;
    }
    transcriptStick = transcriptFromBottom() <= transcriptFollowPx;
  }

  window.addEventListener("scroll", noteTranscriptStick, { passive: true });

  // Layout changes can also dispatch scroll events. Track actual input separately
  // so an idle refresh can distinguish a page reset from someone scrolling away.
  var transcriptInputRevision = 0;
  function noteTranscriptInput() { transcriptInputRevision++; }
  window.addEventListener("wheel", function (e) {
    noteTranscriptInput();
    if (e.deltaY < 0) holdTranscriptFollow();
  }, { passive: true, capture: true });
  window.addEventListener("touchstart", function (e) {
    noteTranscriptInput();
    if (e.touches && e.touches[0]) transcriptTouchY = e.touches[0].clientY;
  }, { passive: true, capture: true });
  window.addEventListener("touchmove", function (e) {
    if (!e.touches || !e.touches[0] || transcriptTouchY == null) return;
    var y = e.touches[0].clientY;
    if (y > transcriptTouchY + 8) holdTranscriptFollow();
    transcriptTouchY = y;
  }, { passive: true, capture: true });
  window.addEventListener("pointerdown", noteTranscriptInput, { passive: true, capture: true });
  window.addEventListener("keydown", function (e) {
    if (/^(ArrowUp|ArrowDown|PageUp|PageDown|Home|End| |Spacebar)$/.test(e.key)) noteTranscriptInput();
    if (e.key === "ArrowUp" || e.key === "PageUp" || e.key === "Home") holdTranscriptFollow();
  }, true);

  // force is the reader's own send or a thread change: go to the bottom
  // even when they were reading higher up. Follow skips the jump if they
  // have since scrolled away.
  function scrollTranscript(force, guard) {
    requestAnimationFrame(function () {
      if (guard && !guard()) return;
      if (!force && !transcriptStick) return;
      var el = document.scrollingElement || document.documentElement;
      // Request a valid scroll offset instead of overshooting the document height.
      window.scrollTo(0, Math.max(0, el.scrollHeight - el.clientHeight));
      transcriptStick = true;
    });
  }

  function scrollTranscriptToEnd(guard) {
    scrollTranscript(true, guard);
  }

  function followTranscript() {
    scrollTranscript(false);
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
      if (kids[k].dataset.sig !== sig && sameMessageShape(kids[k].dataset.sig, sig)) {
        kids[k].classList.toggle("streaming", !!list[k].streaming);
        kids[k].dataset.sig = sig;
        prev = kids[k].dataset.msgRole || prev;
        continue;
      }
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
    if (thread.dataset.rebuild === "1") {
      heldOpen = openMapFrom(thread);
      // Build off-screen so the page never collapses to an empty transcript
      // while a session snapshot is replacing the live messages.
      var replacement = document.createElement("div");
      renderThreadContents(replacement);
      var fragment = document.createDocumentFragment();
      while (replacement.firstChild) fragment.appendChild(replacement.firstChild);
      thread.replaceChildren(fragment);
      delete thread.dataset.rebuild;
      return;
    }
    renderThreadContents(thread);
  }

  function renderThreadContents(target) {
    if (!messages.length) {
      target.innerHTML = "";
      var hint = document.createElement("div");
      hint.className = "msg system";
      hint.innerHTML = '<p class="who">Keimling</p><p class="body">Send a message to continue this session.</p>';
      target.appendChild(hint);
      return;
    }

    // Single chapter: full transcript for that range.
    if (activeChapter) {
      var prevHeld = heldOpen;
      var seenOpen = openMapFrom(target);
      if (Object.keys(seenOpen).length) heldOpen = seenOpen;
      target.innerHTML = "";
      var head = document.createElement("div");
      head.className = "msg system";
      var sum = activeChapter.summary ? "\n\n" + activeChapter.summary : "";
      head.innerHTML = '<p class="who">Chapter</p><div class="body">' + esc(activeChapter.title + sum) + "</div>";
      target.appendChild(head);
      appendMessageList(target, messages.filter(inChapter));
      heldOpen = prevHeld;
      return;
    }

    // Chapters stay put; only the live tail grows, so an opened fold is not rebuilt.
    if (chapters.length) {
      var sig = chapterBlockSig();
      var block = target.querySelector(":scope > .chapter-block");
      var tail = target.querySelector(":scope > .thread-tail");
      if (!block || block.dataset.sig !== sig || !tail) {
        var chapterOpen = openMapFrom(target);
        var prevOpen = heldOpen;
        if (Object.keys(chapterOpen).length) heldOpen = chapterOpen;
        target.innerHTML = "";
        block = document.createElement("div");
        block.className = "chapter-block";
        block.dataset.sig = sig;
        chapters.forEach(function (ch) {
          block.appendChild(renderChapterOverview(ch));
        });
        target.appendChild(block);
        tail = document.createElement("div");
        tail.className = "thread-tail";
        target.appendChild(tail);
        appendMessageList(tail, messagesOutsideChapters());
        heldOpen = prevOpen;
      } else if (!syncMessageList(tail, messagesOutsideChapters())) {
        fillMessageList(tail, messagesOutsideChapters());
      }
      return;
    }

    if (target.querySelector(".chapter-block, .thread-tail")) target.innerHTML = "";
    if (!syncMessageList(target, messages)) fillMessageList(target, messages);
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
    var follow = followWanted();
    renderThread();
    if (follow) scrollTranscript(true);
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
    // Same markdown as the settled message, so lists and checkboxes do not jump when the reply ends.
    var follow = followWanted();
    body.innerHTML = renderAssistantHTML(last.text || "");
    if (follow) scrollTranscript(true);
  }

  function patchThinking() {
    var last = messages[messages.length - 1];
    if (!last || last.role !== "thinking") {
      paintTranscript();
      return;
    }
    var nodes = thread.querySelectorAll(".msg.thinking");
    var el = nodes.length ? nodes[nodes.length - 1] : null;
    var body = el && el.querySelector(".thinking-body");
    if (!body) {
      paintTranscript();
      return;
    }
    var follow = followWanted();
    body.textContent = last.text || "";
    if (follow) scrollTranscript(true);
    if (followThinking && last.streaming && followDismissedKey !== ensureDomKey(last)) {
      var fold = el.querySelector(".thinking-fold");
      if (fold && !fold.open) {
        applyingFollow = true;
        fold.open = true;
        fold.dataset.autoFollow = "1";
        applyingFollow = false;
      }
    }
  }

  function finishLiveThinking() {
    var live = false;
    messages.forEach(function (m) {
      if (m.role === "thinking" && m.streaming) {
        delete m.streaming;
        live = true;
      }
    });
    followDismissedKey = "";
    if (!live) return;
    applyingFollow = true;
    document.querySelectorAll(".thinking-fold[data-auto-follow='1']").forEach(function (fold) {
      fold.open = false;
      delete fold.dataset.autoFollow;
    });
    applyingFollow = false;
  }

  function handleThinking(payload) {
    if (payload && payload.thread_id && payload.thread_id !== currentThreadID) return;
    if (!eventsForThisView()) return;
    var text = payload.text || "";
    if (!text) return;
    var last = messages[messages.length - 1];
    if (last && last.role === "thinking" && last.streaming) {
      last.text = (last.text || "") + text;
      patchThinking();
      return;
    }
    messages.push({ role: "thinking", text: text, streaming: true });
    paintTranscript();
  }

  function handleStructuredMessage(payload) {
    if (payload && payload.thread_id) {
      if (payload.thread_id !== currentThreadID) return;
    } else if (!eventsForThisView()) return;
    var text = payload.text || "";
    var role = payload.role || "assistant";
    finishLiveThinking();
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
      noteGrowthMessage(payload);
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
    if (role === "user") scrollTranscriptToEnd();
    noteGrowthMessage(payload);
  }

  function pendingToolCard(name) {
    for (var i = messages.length - 1; i >= 0; i--) {
      var m = messages[i];
      if (m.role === "tool_group" && m.pending && m.name === (name || "tool")) return m;
    }
    return null;
  }

  function handleStructuredTool(payload) {
    if (payload && payload.thread_id && payload.thread_id !== currentThreadID) return;
    if (!eventsForThisView()) return;
    finishLiveThinking();
    if (payload.phase === "start") {
      // Finalize any open stream so tool sits in the monologue correctly.
      var open = messages[messages.length - 1];
      if (open && open.streaming) delete open.streaming;
      messages.push({
        role: "tool_group",
        name: payload.name || "tool",
        command: payload.command || payload.name || "tool",
        input: payload.input || "",
        result: "",
        raw: "",
        pending: true,
        reviewing: false,
        reviewMs: 0,
      });
      // Collapsed by default (no approval) — avoids expand/collapse scroll jump.
      paintTranscript();
      return;
    }
    if (payload.phase === "review") {
      // The call is with the approval review and has not started yet.
      var rev = pendingToolCard(payload.name);
      if (!rev) return;
      rev.reviewing = true;
      rev.reviewAt = Date.now();
      rev.reviewMs = 0;
      paintTranscript();
      return;
    }
    if (payload.phase === "ask") {
      // The gate wants a human; the call has not started.
      var asking = pendingToolCard(payload.name);
      if (!asking) return;
      if (asking.reviewAt) {
        asking.reviewMs = Date.now() - asking.reviewAt;
        asking.reviewAt = 0;
      }
      asking.reviewing = false;
      asking.waiting = true;
      paintTranscript();
      return;
    }
    if (payload.phase === "go") {
      // The gate passed; the call starts now.
      var started = pendingToolCard(payload.name);
      if (!started) return;
      if (started.reviewAt) {
        started.reviewMs = Date.now() - started.reviewAt;
        started.reviewAt = 0;
      }
      started.reviewing = false;
      started.waiting = false;
      paintTranscript();
      return;
    }
    if (payload.phase === "output") {
      var live = pendingToolCard(payload.name);
      if (!live) return;
      live.result = payload.result || "";
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
      input: payload.input || "",
      result: resultText,
      raw: payload.filtered ? (payload.raw || "") : "",
      filtered: !!payload.filtered,
      pending: false,
      reviewing: false,
      waiting: false,
      ok: payload.ok,
      status: status,
    };
    for (var i = messages.length - 1; i >= 0; i--) {
      if (messages[i].role === "tool_group" && messages[i].pending && messages[i].name === resultMsg.name) {
        // Keep the time spent in review, so the card can show it apart from the run.
        if (messages[i].reviewAt) {
          resultMsg.reviewMs = Date.now() - messages[i].reviewAt;
        } else {
          resultMsg.reviewMs = messages[i].reviewMs || 0;
        }
        messages[i] = resultMsg;
        paintTranscript();
        return;
      }
    }
    messages.push(resultMsg);
    paintTranscript();
  }

  function usdPerMillionFromToken(s) {
    s = String(s || "").trim();
    if (!s) return "";
    var n = Number(s);
    if (!isFinite(n) || n < 0) return "";
    var m = Math.round(n * 1e6 * 1e8) / 1e8;
    return String(m);
  }
  function usdPerTokenFromMillion(s) {
    s = String(s || "").trim();
    if (!s) return "";
    var n = Number(s);
    if (!isFinite(n) || n < 0) return null;
    var out = (n / 1e6).toFixed(12).replace(/0+$/, "").replace(/\.$/, "");
    return out === "" ? "0" : out;
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
      if (prov.account_remaining_usd != null) parts.push("account " + formatUSD(prov.account_remaining_usd));
      else if (prov.account_error) parts.push("account ?");
      if (prov.error) parts.push("key budget ?");
      else if (prov.remaining_usd != null) {
        var kind = prov.kind || "";
        parts.push((kind === "openrouter" ? "key left " : "left ") + formatUSD(prov.remaining_usd));
      }
    }
    budgetLineEl.textContent = parts.length ? parts.join(" · ") : "—";
    var tip = [];
    if (sess.priced && sess.estimate_usd != null) tip.push("Session estimate from token usage × model prices");
    if (prov.supported && prov.error) tip.push(prov.error);
    if (prov.account_remaining_usd != null) tip.push("OpenRouter account credits minus usage");
    if (prov.account_error) tip.push(prov.account_error);
    if (prov.supported && !prov.error && prov.remaining_usd != null) {
      tip.push(prov.kind === "openrouter"
        ? "OpenRouter key limit remaining"
        : "Key budget minus spend");
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

  // Settings → Spend: aggregated model cost, per-model table and the balance
  // history. The server sends hourly/daily buckets; single calls stay in the
  // JSONL ledger only.
  var spendState = { granularity: "day", days: 30 };

  function spendRangeLabel(data) {
    var since = new Date((data && data.since) || "");
    var until = new Date((data && data.until) || "");
    if (isNaN(since.getTime()) || isNaN(until.getTime())) return "";
    return since.toLocaleDateString() + " – " + until.toLocaleDateString();
  }

  function spendBucketLabel(start, granularity) {
    var d = new Date(start);
    if (isNaN(d.getTime())) return String(start || "");
    if (granularity === "hour") {
      return d.toLocaleString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
    }
    return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  }

  function loadSpend() {
    if (!connected) return Promise.resolve();
    var summary = document.getElementById("spend-summary");
    if (summary) summary.textContent = "Loading spend…";
    var q = "?granularity=" + encodeURIComponent(spendState.granularity) + "&days=" + spendState.days;
    return fetch(apiURL("/api/spend" + q), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error("HTTP " + r.status);
          renderSpend(j);
        });
      })
      .catch(function (e) {
        if (summary) summary.textContent = "Could not load spend: " + String((e && e.message) || e);
      });
  }

  function renderSpend(data) {
    data = data || {};
    renderSpendSummary(data);
    renderSpendChart(data);
    renderSpendModels(data);
    renderSpendBudget(data);
  }

  function renderSpendSummary(data) {
    var el = document.getElementById("spend-summary");
    if (!el) return;
    var t = data.totals || {};
    var parts = [];
    parts.push(formatUSD(t.usd || 0) + " priced");
    parts.push(((t.input_tokens || 0) + (t.output_tokens || 0)) + " tokens");
    parts.push((t.calls || 0) + ((t.calls || 0) === 1 ? " call" : " calls"));
    if (t.unpriced_calls) parts.push(t.unpriced_calls + " without a known price");
    var range = spendRangeLabel(data);
    el.textContent = parts.join(" · ") + (range ? " · " + range : "");
  }

  function renderSpendChart(data) {
    var host = document.getElementById("spend-chart");
    if (!host) return;
    var buckets = data.buckets || [];
    var max = 0;
    buckets.forEach(function (b) {
      var v = b.usd || 0;
      if (v > max) max = v;
    });
    if (!buckets.length || max <= 0) {
      host.textContent = "No priced spend in this window.";
      return;
    }
    var W = 720;
    var H = 170;
    var base = H - 22;
    var top = 8;
    var n = buckets.length;
    var gap = n > 60 ? 0 : 1;
    var slot = (W - gap * (n - 1)) / n;
    var svg = ['<svg viewBox="0 0 ' + W + ' ' + H + '" class="spend-svg" role="img" aria-label="Cost per ' + esc(data.granularity || "day") + '">'];
    svg.push('<line x1="0" y1="' + base + '" x2="' + W + '" y2="' + base + '" class="spend-axis"></line>');
    svg.push('<text x="0" y="10" class="spend-max">' + esc(formatUSD(max)) + '</text>');
    buckets.forEach(function (b, i) {
      var v = b.usd || 0;
      if (v <= 0) return;
      var h = Math.round((base - top) * (v / max));
      if (h < 1) h = 1;
      var x = i * (slot + gap);
      svg.push('<rect x="' + x.toFixed(1) + '" y="' + (base - h) + '" width="' + Math.max(1, slot).toFixed(1) + '" height="' + h + '" class="spend-bar"><title>' + esc(spendBucketLabel(b.start, data.granularity)) + ": " + esc(formatUSD(v)) + '</title></rect>');
    });
    var marks = [];
    [0, Math.floor((n - 1) / 2), n - 1].forEach(function (i) {
      if (marks.indexOf(i) < 0) marks.push(i);
    });
    marks.forEach(function (i, k) {
      var x = i * (slot + gap) + slot / 2;
      var anchor = k === 0 ? "start" : k === marks.length - 1 ? "end" : "middle";
      svg.push('<text x="' + x.toFixed(1) + '" y="' + (H - 6) + '" text-anchor="' + anchor + '" class="spend-tick">' + esc(spendBucketLabel(buckets[i].start, data.granularity)) + '</text>');
    });
    svg.push("</svg>");
    host.innerHTML = svg.join("");
  }

  function renderSpendModels(data) {
    var host = document.getElementById("spend-models");
    if (!host) return;
    var rows = data.by_model || [];
    if (!rows.length) {
      host.textContent = "No model calls recorded yet.";
      return;
    }
    var max = 0;
    rows.forEach(function (r) {
      if ((r.usd || 0) > max) max = r.usd || 0;
    });
    var html = ['<table class="spend-table"><thead><tr><th>Model</th><th>Provider</th><th>Calls</th><th>Tokens</th><th>USD</th></tr></thead><tbody>'];
    rows.forEach(function (r) {
      var usd = r.usd || 0;
      var unpriced = r.unpriced_calls || 0;
      var pct = max > 0 ? Math.round((usd / max) * 100) : 0;
      html.push("<tr" + (usd === 0 && unpriced > 0 ? ' class="spend-row-unpriced"' : "") + ">");
      html.push("<td>" + esc(r.model || "unknown") + (unpriced ? ' <span class="hint">(' + unpriced + " unpriced)</span>" : "") + "</td>");
      html.push("<td>" + esc(r.provider || "unknown") + "</td>");
      html.push('<td class="num">' + (r.calls || 0) + "</td>");
      html.push('<td class="num">' + ((r.input_tokens || 0) + (r.output_tokens || 0)) + "</td>");
      html.push('<td class="spend-bar-cell"><div class="spend-bar-track"><div class="spend-bar-fill" style="width:' + pct + '%"></div></div><span class="spend-amount">' + (usd > 0 ? esc(formatUSD(usd)) : "—") + "</span></td>");
      html.push("</tr>");
    });
    html.push("</tbody></table>");
    host.innerHTML = html.join("");
  }

  function renderSpendBudget(data) {
    var host = document.getElementById("spend-budget");
    if (!host) return;
    var rows = data.budget || [];
    if (!rows.length) {
      host.textContent = "No balance snapshots yet. OpenRouter and LiteLLM record one on each real budget fetch.";
      return;
    }
    var html = ['<table class="spend-table"><thead><tr><th>When</th><th>Provider</th><th>Key left</th><th>Account left</th></tr></thead><tbody>'];
    rows.slice(-20).reverse().forEach(function (r) {
      var when = new Date(r.time);
      html.push("<tr>");
      html.push("<td>" + esc(isNaN(when.getTime()) ? (r.time || "") : when.toLocaleString()) + "</td>");
      html.push("<td>" + esc(r.provider || "unknown") + (r.kind ? ' <span class="hint">(' + esc(r.kind) + ")</span>" : "") + "</td>");
      html.push('<td class="num">' + (r.remaining_usd != null ? esc(formatUSD(r.remaining_usd)) : "—") + "</td>");
      html.push('<td class="num">' + (r.account_remaining_usd != null ? esc(formatUSD(r.account_remaining_usd)) : "—") + "</td>");
      html.push("</tr>");
    });
    html.push("</tbody></table>");
    host.innerHTML = html.join("");
  }

  function setSpendRange(granularity) {
    spendState.granularity = granularity === "hour" ? "hour" : "day";
    spendState.days = spendState.granularity === "hour" ? 3 : 30;
    var dayBtn = document.getElementById("spend-range-days");
    var hourBtn = document.getElementById("spend-range-hours");
    if (dayBtn) dayBtn.setAttribute("aria-pressed", spendState.granularity === "day" ? "true" : "false");
    if (hourBtn) hourBtn.setAttribute("aria-pressed", spendState.granularity === "hour" ? "true" : "false");
    return loadSpend();
  }

  (function bindSpendControls() {
    var dayBtn = document.getElementById("spend-range-days");
    var hourBtn = document.getElementById("spend-range-hours");
    var refreshBtn = document.getElementById("spend-refresh");
    if (dayBtn) dayBtn.addEventListener("click", function () { setSpendRange("day"); });
    if (hourBtn) hourBtn.addEventListener("click", function () { setSpendRange("hour"); });
    if (refreshBtn) refreshBtn.addEventListener("click", function () { loadSpend(); });
  })();

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

  function sessionViewKey(j) {
    j = j || {};
    return [j.agent_instance_id || "", j.workspace_id || "", j.thread_id || ""].join("\u0001");
  }

  function applySession(j) {
    var openedThread = currentThreadID;
    applyProviderGate(!!j.needs_provider);
    modelEl.textContent = needsProvider ? "—" : (j.model || "—");
    applyModelChrome(j);
    modeEl.textContent = j.mode || "—";
    var incoming = j.messages || [];
    var total = Number(j.messages_total || 0);
    if (total > incoming.length && incoming.length) {
      var missing = total - incoming.length;
      if (!incoming[0] || incoming[0].role !== "system" || String(incoming[0].text || "").indexOf("earlier messages") < 0) {
        incoming = [{ role: "system", text: missing + " earlier messages are not shown." }].concat(incoming);
      }
    }
    var nextKey = sessionViewKey(j);
    var sameView = shownViewKey !== "" && nextKey === shownViewKey;
    if (!sameView) activeChapter = null;
    var keepThread = sameView && transcriptMatches(messages, incoming);
    if (keepThread) {
      adoptSessionFields(messages, incoming);
      chapters = j.chapters || [];
      if (thread) {
        thread.querySelectorAll(".msg.streaming").forEach(function (el) { el.classList.remove("streaming"); });
        applyingFollow = true;
        thread.querySelectorAll(".thinking-fold[data-auto-follow='1']").forEach(function (fold) {
          fold.open = false;
          delete fold.dataset.autoFollow;
        });
        applyingFollow = false;
      }
    } else {
      messages = incoming;
      chapters = j.chapters || [];
      if (thread) thread.dataset.rebuild = "1";
    }
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
    syncChatDraft();
    renderSideChatRef();
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
    applyLowContext(!!j.low_context);
    shownViewKey = nextKey;
    renderChapters();
    if (!keepThread) {
      renderThread();
      if (!(j.thread_id && j.thread_id !== openedThread)) followTranscript();
    }
    if (j.growth) syncGrowthLogo(j.growth.used_pct, false);
    if (j.thread_id && j.thread_id !== openedThread) scrollTranscriptToEnd();
    var next = j.busy ? "running" : (j.state || "idle");
    // The run can already be idle while this snapshot still says busy.
    if (next === "running" && sideClosed) next = "idle";
    setState(next);
    if (j.approval) showApproval(j.approval);
    if (!j.busy) clearLive();
  }

  function handleGrowth(payload) {
    if (!eventsForThisView()) return;
    if (!payload || typeof payload.used_pct !== "number") return;
    syncGrowthLogo(payload.used_pct, false);
  }

  function loadSession() {
    sessionEpoch += 1;
    var epoch = sessionEpoch;
    return fetch(apiURL("/api/session?limit=600"), { headers: headers() }).then(function (r) {
      return r.json().then(function (j) {
        if (epoch !== sessionEpoch) return;
        if (!r.ok) throw new Error((j && j.message) || ("HTTP " + r.status));
        applySession(j);
        if (currentThreadID !== composerPageThread) {
          composerPageThread = currentThreadID;
          restoreComposerPage();
        }
      });
    });
  }

  function connectEvents() {
    if (es) es.close();
    var ctrl = new AbortController();
    fetch(apiURL("/events"), { headers: headers(), signal: ctrl.signal })
      .then(function (r) {
        if (!r.ok) throw new Error("events " + r.status);
        // A fresh stream is the moment to trust the server again: a run that
        // ended while this tab was disconnected never sent its idle event.
        loadSession().catch(function () {});
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
              if (ev === "message" || ev === "thinking" || ev === "tool" || ev === "progress" || ev === "status") {
                lastRunActivityAt = Date.now();
              }
              if (ev === "ready") noteBackendStarted(payload);
              if (ev === "message") handleStructuredMessage(payload);
              if (ev === "thinking") handleThinking(payload);
              if (ev === "realtime_inject") speakText(payload && payload.text);
              if (ev === "tool") handleStructuredTool(payload);
              if (ev === "growth") handleGrowth(payload);
              if (ev === "approval" && eventsForThisView()) showApproval(payload);
              if (ev === "progress") updateProgress(payload);
              if (ev === "artifact") {
                showArtifact(payload);
                loadArtifactsList();
              }
              if (ev === "compaction") updateCompactionProgress(payload);
              if (ev === "thread") updateThreadTitle(payload);
              if (ev === "status" && payload.state) {
                var foreignStatus = payload.thread_id && currentThreadID && payload.thread_id !== currentThreadID && !currentAgentInstanceID;
                if (foreignStatus) {
                  if (payload.state === "idle") {
                    if (payload.thread_id === stoppingThreadID) stoppingThreadID = "";
                    if (runningThreadID === payload.thread_id) runningThreadID = "";
                    backgroundBusy = !!runningThreadID && runningThreadID !== currentThreadID;
                    loadThreads();
                  } else {
                    runningThreadID = payload.thread_id;
                    backgroundBusy = true;
                  }
                } else {
                if (payload.thread_id) runningThreadID = payload.thread_id;
                var otherView = !currentAgentInstanceID && runningThreadID && currentThreadID && runningThreadID !== currentThreadID;
                if (payload.state === "idle") {
                  if (currentAgentInstanceID) delete stoppingAgents[currentAgentInstanceID];
                  if (!currentAgentInstanceID && (!payload.thread_id || payload.thread_id === stoppingThreadID)) stoppingThreadID = "";
                  runningThreadID = "";
                  backgroundBusy = false;
                  // Capture follow intent before idle hides UI or the session
                  // replaces nodes: either can make WebView report a scroll to 0.
                  var idleFollow = transcriptStick;
                  var idleScrollEl = document.scrollingElement || document.documentElement;
                  var idleTop = idleScrollEl.scrollTop;
                  var idleInput = transcriptInputRevision;
                  var idleView = shownViewKey;
                  var idleThread = currentThreadID;
                  var idleWorkspace = currentWorkspaceID;
                  var idleAgent = currentAgentInstanceID;
                  setState("idle");
                  var idleLoad = loadSession();
                  var idleEpoch = sessionEpoch;
                  var idleGuard = function () {
                    if (transcriptInputRevision !== idleInput || agentState !== "idle" ||
                        sessionEpoch !== idleEpoch || shownViewKey !== idleView || currentThreadID !== idleThread ||
                        currentWorkspaceID !== idleWorkspace || currentAgentInstanceID !== idleAgent) return false;
                    var el = document.scrollingElement || document.documentElement;
                    // An away-from-bottom scroll at a nonzero position is also
                    // respected when assistive scrolling has no input event.
                    return !idleFollow || transcriptStick || el.scrollTop <= 0;
                  };
                  var restoreIdleScroll = function () {
                    if (idleFollow) {
                      scrollTranscriptToEnd(idleGuard);
                    } else if (idleTop > 0) {
                      requestAnimationFrame(function () {
                        if (!idleGuard()) return;
                        var el = document.scrollingElement || document.documentElement;
                        if (el.scrollTop <= 0) window.scrollTo(0, Math.min(idleTop, Math.max(0, el.scrollHeight - el.clientHeight)));
                      });
                    }
                  };
                  restoreIdleScroll();
                  idleLoad
                    .then(function () {
                      // This is needed even for matching snapshots: closing a
                      // thinking fold or other idle chrome can reset the page.
                      restoreIdleScroll();
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
        watchBackendRestart();
        watchQuietRun();
      });
    es = { close: function () { ctrl.abort(); } };
  }

  // noteBackendStarted records which process serves this page. The ready event
  // carries the backend start time, so a replaced backend is visible.
  function noteBackendStarted(payload) {
    if (payload && payload.started_at) backendStartedAt = String(payload.started_at);
    stopBackendPoll();
  }

  function stopBackendPoll() {
    if (backendPoll) {
      clearInterval(backendPoll);
      backendPoll = 0;
    }
  }

  // watchBackendRestart keeps asking the health endpoint while the event stream
  // is gone. A new start time means the backend was replaced: the page reloads
  // and picks up the new process. The same process means only the stream broke,
  // so it reconnects. A backend that stays down keeps the browser on the page.
  // Keeps the "running · N min quiet" label honest without new server events.
  var quietRunTimer = null;
  function watchQuietRun() {
    if (quietRunTimer || typeof setInterval !== "function") return;
    quietRunTimer = setInterval(function () {
      if (agentState === "running") setState(agentState);
    }, 20000);
  }

  function watchBackendRestart() {
    if (backendPoll) return;
    backendPoll = setInterval(function () {
      fetch(apiURL("/api/health"), { headers: headers() })
        .then(function (r) {
          if (!r.ok) return null;
          return r.json();
        })
        .then(function (j) {
          if (!j || !j.started_at) return;
          var started = String(j.started_at);
          if (backendStartedAt && started === backendStartedAt) {
            stopBackendPoll();
            connectEvents();
            loadSession().catch(function () {});
            return;
          }
          if (!backendStartedAt) {
            backendStartedAt = started;
            stopBackendPoll();
            connectEvents();
            loadSession().catch(function () {});
            return;
          }
          stopBackendPoll();
          window.location.reload();
        })
        .catch(function () {});
    }, 2000);
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
        watchMcpScripts();
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
  if (credEditType) credEditType.addEventListener("change", syncCredentialForm);
  if (providerNewBtn) providerNewBtn.addEventListener("click", function () {
    openProviderEditor(null);
  });
  var purposeSave = document.getElementById("set-purposes-save");
  if (purposeSave) purposeSave.addEventListener("click", savePurposes);
  var behaviourSave = document.getElementById("set-behaviour-save");
  if (behaviourSave) behaviourSave.addEventListener("click", saveAgentSettings);
  if (setIdleNotify) setIdleNotify.addEventListener("change", function () {
    if (setIdleNotify.checked && typeof Notification === "function" && Notification.permission === "default") {
      Notification.requestPermission();
    }
  });
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
  if (provEditKind) provEditKind.addEventListener("change", syncManagementKeyField);
  if (provEditBaseURL) provEditBaseURL.addEventListener("input", syncManagementKeyField);
  if (provModelNew) provModelNew.addEventListener("click", openModelFormAdd);
  if (provModelSave) provModelSave.addEventListener("click", saveEditorModel);
  if (provModelID) provModelID.addEventListener("change", function () {
    if (!catalogCache.length) return;
    var id = provModelID.value.trim();
    for (var i = 0; i < catalogCache.length; i++) {
      if (catalogCache[i].id === id) {
        applyCatalogPrefill(catalogCache[i]);
        break;
      }
    }
  });
  if (provModelCancel) provModelCancel.addEventListener("click", closeModelForm);
  if (provCatalogFetch) provCatalogFetch.addEventListener("click", fetchModelCatalog);
  if (provModelProvider) provModelProvider.addEventListener("change", function () {
    showProviderCatalog(provModelProvider.value);
  });
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
  function budgetLabelText(b) {
    b = b || {};
    var pct = typeof b.used_pct === "number" ? b.used_pct : 0;
    return "used " + (b.used || 0) + " / " + (b.limit || 0) +
      " (" + pct.toFixed(1) + "%) · remaining " + (b.remaining || 0) +
      " · " + (b.source || "estimate");
  }
  // One meter for both views; each view keeps its own element ids.
  function applyBudgetMeter(labelID, fillID, b) {
    b = b || {};
    var pct = typeof b.used_pct === "number" ? b.used_pct : 0;
    var label = document.getElementById(labelID);
    if (label) label.textContent = budgetLabelText(b);
    var fill = document.getElementById(fillID);
    if (!fill) return;
    fill.style.width = Math.min(100, Math.max(0, pct)) + "%";
    fill.classList.toggle("warn", !!b.warn);
  }
  // The Now view repeats the Context tab's meter from the same /api/context payload.
  function renderNowBudget(b) {
    applyBudgetMeter("now-budget-label", "now-budget-fill", b);
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
    applyBudgetMeter("budget-label", "budget-fill", b);
    renderNowBudget(b);
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
    [mcpErr, document.getElementById("mcp-tool-err")].forEach(function (el) {
      if (!el) return;
      if (!msg) {
        el.hidden = true;
        el.textContent = "";
        return;
      }
      el.hidden = false;
      el.textContent = msg;
    });
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
  function mcpQuoteArg(s) {
    s = String(s || "");
    if (!s || /[\s"']/.test(s)) return '"' + s.replace(/\\/g, "\\\\").replace(/"/g, '\\"') + '"';
    return s;
  }
  function mcpSplitCommand(line) {
    var out = [];
    var cur = "";
    var quote = "";
    for (var i = 0; i < line.length; i++) {
      var c = line.charAt(i);
      if (quote) {
        if (c === "\\" && quote === '"' && i + 1 < line.length) {
          cur += line.charAt(++i);
          continue;
        }
        if (c === quote) quote = "";
        else cur += c;
        continue;
      }
      if (c === '"' || c === "'") {
        quote = c;
        continue;
      }
      if (/\s/.test(c)) {
        if (cur) {
          out.push(cur);
          cur = "";
        }
        continue;
      }
      cur += c;
    }
    if (cur) out.push(cur);
    return out;
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
    if (mcpCwdRow) mcpCwdRow.hidden = remote;
    if (mcpEnvRow) mcpEnvRow.hidden = remote;
    if (mcpUrlRow) mcpUrlRow.hidden = !remote;
    if (mcpAuthRow) mcpAuthRow.hidden = !remote;
    if (!remote && mcpAuth) mcpAuth.value = "none";
    syncMcpAuthUI();
  }
  function mcpSignIn(name, url, before) {
    var payload = { name: name, url: url, origin: location.origin };
    if (before) payload.before = before;
    return fetch(apiURL("/api/mcp/sign-in"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify(payload),
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
  function mcpServerByName(name) {
    var found = null;
    (mcpServersCache || []).forEach(function (s) {
      if (s && s.name === name) found = s;
    });
    return found;
  }
  function mcpServerNote(s) {
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
    return {
      line: (s.transport || "") + " · " + (s.status || "") + from + authNote +
        (errText ? " — " + errText : "") +
        (s.tools && s.tools.length ? " · " + s.tools.length + " tools" : ""),
      errText: errText,
      from: from,
    };
  }
  function mcpToolNames(s) {
    var names = (s.tools || []).slice();
    (s.allow || []).forEach(function (name) {
      if (names.indexOf(name) < 0) names.push(name);
    });
    return names;
  }
  function fillMcpToolDialog(s) {
    var title = document.getElementById("mcp-tool-title");
    var meta = document.getElementById("mcp-tool-meta");
    var list = document.getElementById("mcp-tool-list");
    var actions = document.getElementById("mcp-tool-actions");
    if (!list || !s) return;
    if (title) title.textContent = s.name || "Details";
    var note = mcpServerNote(s);
    if (meta) meta.textContent = (s.transport || "") + " · " + (s.status || "") + note.from + (note.errText ? " — " + note.errText : "");
    var before = document.getElementById("mcp-before");
    if (before && document.activeElement !== before) before.value = s.before || "";
    list.innerHTML = "";
    var names = mcpToolNames(s);
    if (!names.length) {
      var empty = document.createElement("p");
      empty.className = "hint";
      empty.textContent = "This server has not listed tools yet. Reload after it connects, then open it again.";
      list.appendChild(empty);
    } else {
      var hint = document.createElement("p");
      hint.className = "hint";
      hint.textContent = "Always allow runs that tool in ask and auto without a question. Uncheck to ask again.";
      list.appendChild(hint);
      var allowed = {};
      (s.allow || []).forEach(function (name) { allowed[name] = true; });
      names.forEach(function (toolName) {
        var row = document.createElement("label");
        row.className = "check";
        var box = document.createElement("input");
        box.type = "checkbox";
        box.checked = !!allowed[toolName];
        box.addEventListener("change", function () {
          box.disabled = true;
          showMcpErr("");
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
        list.appendChild(row);
      });
    }
    if (!actions) return;
    actions.innerHTML = "";
    var logBtn = document.createElement("button");
    logBtn.type = "button";
    logBtn.className = "ghost";
    logBtn.textContent = "Log";
    logBtn.addEventListener("click", function () { openMcpLog(s.name); });
    actions.appendChild(logBtn);
    var reloadBtn = document.createElement("button");
    reloadBtn.type = "button";
    reloadBtn.className = "ghost";
    reloadBtn.textContent = "Reload";
    reloadBtn.title = "Reconnect this server";
    reloadBtn.addEventListener("click", function () {
      showMcpErr("");
      reloadBtn.disabled = true;
      mcpReloadOne(s.name).catch(function (e) { showMcpErr(e.message || e); })
        .finally(function () { reloadBtn.disabled = false; });
    });
    actions.appendChild(reloadBtn);
    if (s.source === "user") {
      var editBtn = document.createElement("button");
      editBtn.type = "button";
      editBtn.className = "ghost";
      editBtn.textContent = "Edit";
      editBtn.addEventListener("click", function () { showMcpEdit(s); });
      actions.appendChild(editBtn);
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
      actions.appendChild(sign);
    }
    if (s.source !== "plugin") {
      var rm = document.createElement("button");
      rm.type = "button";
      rm.className = "ghost";
      rm.textContent = "Remove";
      rm.addEventListener("click", function (ev) {
        ev.preventDefault();
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
              if (mcpToolDialog && typeof mcpToolDialog.close === "function") mcpToolDialog.close();
              mcpToolName = "";
              return loadMcpList().then(function () {
                if (j && j.warning) showMcpErr(j.warning);
              });
            })
            .catch(function (e) { showMcpErr(e.message || e); })
            .finally(function () { rm.disabled = false; });
        });
      });
      actions.appendChild(rm);
    }
  }
  function mcpReloadOne(name) {
    return fetch(apiURL("/api/mcp/reload-one"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ name: name }),
    }).then(function (r) {
      return r.text().then(function (t) {
        var j = {};
        try { j = t ? JSON.parse(t) : {}; } catch (e) {}
        if (!r.ok) throw new Error((j && (j.error || j.message)) || t || r.status);
        renderMcpServers(j.servers || []);
        return j;
      });
    });
  }
  function showMcpEdit(s) {
    var box = document.getElementById("mcp-edit");
    var stdioBox = document.getElementById("mcp-edit-stdio");
    var urlBox = document.getElementById("mcp-edit-url-row");
    var cwdBox = document.getElementById("mcp-edit-cwd-row");
    var command = document.getElementById("mcp-edit-command");
    var cwd = document.getElementById("mcp-edit-cwd");
    var url = document.getElementById("mcp-edit-url");
    if (!box || !s) return;
    var stdio = s.transport === "stdio" || (!s.url && !!s.command);
    if (stdioBox) stdioBox.hidden = !stdio;
    if (cwdBox) cwdBox.hidden = !stdio;
    if (urlBox) urlBox.hidden = stdio;
    if (stdio) {
      var bits = [];
      if (s.command) bits.push(s.command);
      (s.args || []).forEach(function (arg) { bits.push(arg); });
      if (command) command.value = bits.map(mcpQuoteArg).join(" ");
      if (cwd) cwd.value = s.cwd || "";
    } else if (url) {
      url.value = s.url || "";
    }
    box.hidden = false;
  }
  function saveMcpStart() {
    var input = document.getElementById("mcp-start");
    var save = document.getElementById("mcp-start-save");
    var command = input ? input.value.trim() : "";
    if (save) save.disabled = true;
    showMcpErr("");
    watchMcpScripts();
    fetch(apiURL("/api/mcp/start"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ command: command }),
    }).then(function (r) {
      return r.text().then(function (t) {
        var j = {};
        try { j = t ? JSON.parse(t) : {}; } catch (e) {}
        if (!r.ok) throw new Error((j && (j.error || j.message)) || t || r.status);
        return j;
      });
    }).then(function (j) {
      renderMcpServers(j.servers || []);
      if (j.error) showMcpErr(j.error);
    }).catch(function (e) {
      showMcpErr(e.message || e);
    }).finally(function () {
      if (save) save.disabled = false;
    });
  }
  function saveMcpBefore() {
    var name = mcpToolName;
    if (!name) return;
    var input = document.getElementById("mcp-before");
    var save = document.getElementById("mcp-before-save");
    var command = input ? input.value.trim() : "";
    if (save) save.disabled = true;
    showMcpErr("");
    watchMcpScripts();
    fetch(apiURL("/api/mcp/before"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ name: name, command: command }),
    }).then(function (r) {
      return r.text().then(function (t) {
        var j = {};
        try { j = t ? JSON.parse(t) : {}; } catch (e) {}
        if (!r.ok) throw new Error((j && (j.error || j.message)) || t || r.status);
        return j;
      });
    }).then(function (j) {
      renderMcpServers(j.servers || []);
      var fresh = mcpServerByName(name);
      if (fresh && input && document.activeElement !== input) input.value = fresh.before || "";
    }).catch(function (e) {
      showMcpErr(e.message || e);
    }).finally(function () {
      if (save) save.disabled = false;
    });
  }
  function saveMcpEdit() {
    var name = mcpToolName;
    if (!name) return;
    var stdioBox = document.getElementById("mcp-edit-stdio");
    var stdio = !stdioBox || !stdioBox.hidden;
    var body = { name: name, update: true };
    if (stdio) {
      var line = ((document.getElementById("mcp-edit-command") || {}).value || "").trim();
      if (!line) {
        showMcpErr("Command required.");
        return;
      }
      var parts = mcpSplitCommand(line);
      body.command = parts[0];
      body.args = parts.slice(1);
      body.cwd = ((document.getElementById("mcp-edit-cwd") || {}).value || "").trim();
    } else {
      body.url = ((document.getElementById("mcp-edit-url") || {}).value || "").trim();
      if (!body.url) {
        showMcpErr("URL required.");
        return;
      }
    }
    var noteEl = document.getElementById("mcp-note");
    body.note = noteEl ? noteEl.value.trim() : "";
    var save = document.getElementById("mcp-edit-save");
    if (save) save.disabled = true;
    showMcpErr("");
    watchMcpScripts();
    fetch(apiURL("/api/mcp/add"), {
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
    }).then(function (j) {
      var edit = document.getElementById("mcp-edit");
      if (edit) edit.hidden = true;
      renderMcpServers(j.servers || []);
    }).catch(function (e) {
      showMcpErr(e.message || e);
    }).finally(function () {
      if (save) save.disabled = false;
    });
  }
  function openMcpLog(name) {
    var dlg = document.getElementById("mcp-log-dialog");
    var title = document.getElementById("mcp-log-title");
    var text = document.getElementById("mcp-log-text");
    if (!dlg || !text) return;
    if (title) title.textContent = "Log · " + (name || "");
    function pull() {
      return fetch(apiURL("/api/mcp/log?name=" + encodeURIComponent(name)), { headers: headers() })
        .then(function (r) {
          return r.json().then(function (j) {
            if (!r.ok) throw new Error((j && j.error) || r.status);
            return j;
          });
        })
        .then(function (j) {
          var next = (j && j.log) || "";
          if (!next) next = "No output yet. Connection errors land here. A local command also writes its error stream here.";
          if (text.value !== next) {
            var atEnd = text.scrollTop + text.clientHeight >= text.scrollHeight - 24;
            text.value = next;
            if (atEnd) text.scrollTop = text.scrollHeight;
          }
        })
        .catch(function (e) { text.value = String(e.message || e); });
    }
    pull();
    if (mcpLogTimer) clearInterval(mcpLogTimer);
    mcpLogTimer = setInterval(function () {
      if (!dlg.open && !dlg.hasAttribute("open")) {
        clearInterval(mcpLogTimer);
        mcpLogTimer = 0;
        return;
      }
      pull();
    }, 1000);
    if (typeof dlg.showModal === "function") dlg.showModal();
    else dlg.setAttribute("open", "");
  }
  function openMcpToolDialog(name) {
    var s = mcpServerByName(name);
    if (!s || !mcpToolDialog) return;
    var edit = document.getElementById("mcp-edit");
    if (edit) edit.hidden = true;
    mcpToolName = name;
    var note = document.getElementById("mcp-note");
    if (note) note.value = s.note || "";
    showMcpErr("");
    fillMcpToolDialog(s);
    if (typeof mcpToolDialog.showModal === "function") mcpToolDialog.showModal();
    else mcpToolDialog.setAttribute("open", "");
  }
  function renderMcpServers(servers) {
    mcpServersCache = servers || [];
    renderSideMcp(mcpServersCache);
    if (!mcpList) return;
    mcpList.innerHTML = "";
    if (!servers || !servers.length) {
      var empty = document.createElement("li");
      empty.className = "skill-meta";
      empty.textContent = "No MCP servers configured.";
      mcpList.appendChild(empty);
    } else {
      servers.forEach(function (s) {
        var li = document.createElement("li");
        var head = document.createElement("div");
        head.className = "skill-head";
        var title = document.createElement("div");
        title.className = "skill-name";
        title.textContent = s.name || "";
        var open = document.createElement("button");
        open.type = "button";
        open.className = "ghost";
        open.textContent = "Details";
        open.addEventListener("click", function () { openMcpToolDialog(s.name); });
        head.appendChild(title);
        head.appendChild(open);
        var meta = document.createElement("div");
        meta.className = "skill-meta";
        meta.textContent = mcpServerNote(s).line;
        li.appendChild(head);
        li.appendChild(meta);
        mcpList.appendChild(li);
      });
    }
    if (mcpToolName && mcpToolDialog && (mcpToolDialog.open || mcpToolDialog.hasAttribute("open"))) {
      var fresh = mcpServerByName(mcpToolName);
      if (fresh) fillMcpToolDialog(fresh);
    }
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
        var start = document.getElementById("mcp-start");
        if (start && document.activeElement !== start && j.start != null) start.value = j.start;
        renderMcpServers(mcpServersCache);
        applyMcpScriptBanner(j);
      });
  }
  function applyMcpScriptBanner(j) {
    var box = document.getElementById("mcp-script-banner");
    var text = document.getElementById("mcp-script-banner-text");
    var retry = document.getElementById("mcp-script-reload");
    if (!box || !text) return;
    var script = j && j.script;
    var message = script && script.message ? String(script.message) : "";
    var failed = !!(script && script.state === "failed" && message);
    if (!message) {
      box.hidden = true;
      text.textContent = "";
      if (retry) retry.hidden = true;
      return;
    }
    if (text.textContent !== message) text.textContent = message;
    box.hidden = false;
    box.classList.toggle("failed", failed);
    box.classList.toggle("running", !failed);
    if (retry) retry.hidden = !failed;
  }
  function pullMcpScriptBanner() {
    return fetch(apiURL("/api/mcp"), { headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          return j;
        });
      })
      .then(function (j) { applyMcpScriptBanner(j); return j; });
  }
  var mcpScriptTimer = 0;
  function watchMcpScripts() {
    if (mcpScriptTimer) clearTimeout(mcpScriptTimer);
    var ticks = 0;
    function step() {
      if (!connected) { mcpScriptTimer = 0; return; }
      pullMcpScriptBanner().then(function () {
        ticks++;
        var box = document.getElementById("mcp-script-banner");
        var running = box && !box.hidden && box.classList.contains("running");
        var visible = box && !box.hidden;
        if (running || visible || ticks < 6) mcpScriptTimer = setTimeout(step, running ? 2000 : 5000);
        else mcpScriptTimer = 0;
      }).catch(function () {
        ticks++;
        if (ticks < 6) mcpScriptTimer = setTimeout(step, 2000);
        else mcpScriptTimer = 0;
      });
    }
    step();
  }
  function reloadAllMcp() {
    watchMcpScripts();
    showMcpErr("");
    if (mcpReloadBtn) mcpReloadBtn.disabled = true;
    var retry = document.getElementById("mcp-script-reload");
    if (retry) retry.disabled = true;
    return fetch(apiURL("/api/mcp/reload"), { method: "POST", headers: headers() })
      .then(function (r) {
        return r.json().then(function (j) {
          if (!r.ok) throw new Error((j && j.error) || r.status);
          return j;
        });
      })
      .then(function (j) { renderMcpServers(j.servers || []); return pullMcpScriptBanner(); })
      .catch(function (e) { showMcpErr(e.message || e); })
      .finally(function () {
        if (mcpReloadBtn) mcpReloadBtn.disabled = false;
        if (retry) retry.disabled = false;
      });
  }
  function openMcpSettings() {
    if (!connected) return;
    showMcpErr("");
    showGate({ manage: true, keepApp: true, tab: "mcp" });
  }
  var mcpEditSave = document.getElementById("mcp-edit-save");
  if (mcpEditSave) mcpEditSave.addEventListener("click", saveMcpEdit);
  var mcpStartSave = document.getElementById("mcp-start-save");
  if (mcpStartSave) mcpStartSave.addEventListener("click", saveMcpStart);
  var mcpBeforeSave = document.getElementById("mcp-before-save");
  if (mcpBeforeSave) mcpBeforeSave.addEventListener("click", saveMcpBefore);
  function saveMcpNote() {
    var name = mcpToolName;
    var s = mcpServerByName(name);
    var noteEl = document.getElementById("mcp-note");
    if (!name || !s) return;
    var body = { name: name, update: true, note: noteEl ? noteEl.value.trim() : "" };
    if (s.url) body.url = s.url;
    else {
      body.command = s.command || "";
      body.args = s.args || [];
      body.cwd = s.cwd || "";
    }
    showMcpErr("");
    fetch(apiURL("/api/mcp/add"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify(body),
    }).then(function (r) {
      return r.text().then(function (t) {
        var j = {};
        try { j = t ? JSON.parse(t) : {}; } catch (e) {}
        if (!r.ok) throw new Error((j && (j.error || j.message)) || t || r.status);
        renderMcpServers(j.servers || []);
      });
    }).catch(function (e) { showMcpErr(e.message || e); });
  }
  var mcpNoteSave = document.getElementById("mcp-note-save");
  if (mcpNoteSave) mcpNoteSave.addEventListener("click", saveMcpNote);
  var mcpToolForm = document.getElementById("mcp-tool-form");
  if (mcpToolForm) mcpToolForm.addEventListener("keydown", function (e) {
    if (e.key !== "Enter" || !e.target || e.target.tagName !== "INPUT") return;
    if (e.target.id === "mcp-before") {
      e.preventDefault();
      saveMcpBefore();
      return;
    }
    if (String(e.target.id || "").indexOf("mcp-edit-") !== 0) return;
    e.preventDefault();
    saveMcpEdit();
  });
  var mcpLogDialog = document.getElementById("mcp-log-dialog");
  if (mcpLogDialog) mcpLogDialog.addEventListener("close", function () {
    if (mcpLogTimer) clearInterval(mcpLogTimer);
    mcpLogTimer = 0;
  });
  var mcpLogFull = document.getElementById("mcp-log-full");
  if (mcpLogFull) mcpLogFull.addEventListener("click", function () {
    if (!mcpLogDialog) return;
    if (document.fullscreenElement === mcpLogDialog) document.exitFullscreen();
    else if (mcpLogDialog.requestFullscreen) mcpLogDialog.requestFullscreen();
  });
  var sideMcpManage = document.getElementById("side-mcp-manage");
  if (sideMcpManage) sideMcpManage.addEventListener("click", openMcpSettings);
  var sideChatRefCopy = document.getElementById("side-chat-ref-copy");
  if (sideChatRefCopy) sideChatRefCopy.addEventListener("click", copyChatRef);

  function artifactSrc(art) {
    if (!art) return "";
    var u = art.url || "";
    if (/^https?:\/\//i.test(u)) return u;
    if (art.token) return apiURL("/files/" + art.token);
    if (u.charAt(0) === "/") return apiURL(u);
    return u;
  }
  function artifactMaxPx() {
    var side = 0;
    var el = document.querySelector(".sidebar");
    if (el) side = el.getBoundingClientRect().width;
    var room = Math.floor(window.innerWidth - side - CHAT_MIN_PX);
    var ratio = Math.floor(window.innerWidth * ARTIFACT_WIDTH_MAX_RATIO);
    var max = Math.min(ratio, room);
    return max < 0 ? 0 : max;
  }
  function clampArtifactWidth(px) {
    var max = artifactMaxPx();
    if (max < ARTIFACT_WIDTH_MIN) {
      if (px < 0) return 0;
      if (px > max) return max;
      return Math.round(px);
    }
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
    var known = { now: 1, agents: 1, escalations: 1, gateways: 1, artifacts: 1, context: 1, connections: 1, instructions: 1, chapters: 1 };
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
    var narrow = composerNarrow.matches;
    artifactPane.classList.toggle("mode-overlay", overlay);
    artifactPane.classList.toggle("mode-dock", !overlay);
    if (shellEl) {
      shellEl.classList.toggle("artifact-overlay", overlay && !artifactPane.hidden);
      shellEl.classList.toggle("artifact-open", !artifactPane.hidden && !overlay && !narrow);
    }
    if (artifactBackdrop) artifactBackdrop.hidden = artifactPane.hidden || (!overlay && !narrow);
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
    syncMobileBar();
    if (sidebarToggle) sidebarToggle.setAttribute("aria-pressed", "true");
  }
  function syncSidebarOpen() {
    if (!artifactOpenExt) return;
    artifactOpenExt.hidden = !(sideTab === "artifacts" && artifactSrc(currentArtifact));
  }
  function selectSidebar(tab, opts) {
    opts = opts || {};
    var known = { now: 1, agents: 1, escalations: 1, gateways: 1, artifacts: 1, context: 1, connections: 1, instructions: 1, chapters: 1 };
    if (tab === "chapters" && !composerNarrow.matches) tab = "now";
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
    if (tab === "escalations") loadEscalations();
    if (tab === "gateways") loadGateways();
    if (tab === "now") {
      renderSidePerf();
      loadAgentBoard();
      loadApprovals();
    }
    if (tab === "now" || tab === "context") refreshSideContext({ report: tab === "context" });
    if (tab === "artifacts") {
      loadArtifactsList().then(function (items) {
        if (!currentArtifact && items && items.length) showArtifact(sortArtifacts(items)[0]);
      }).catch(function () {});
    }
    if (tab === "connections") loadMcpList().catch(function () {});
    if (tab === "instructions") loadThreadAutopilot();
    if (tab === "chapters") renderChapters();
    syncSidePoll();
  }
  function syncMobileBar() {
    var bar = document.querySelector(".paper-toolbar");
    if (!bar) return;
    var bottom = Math.ceil(bar.getBoundingClientRect().bottom);
    if (bottom <= 0) return;
    document.documentElement.style.setProperty("--mobile-bar", bottom + "px");
  }
  function setNavOpen(open) {
    var side = document.getElementById("app-sidebar");
    var bd = document.getElementById("nav-backdrop");
    var btn = document.getElementById("nav-toggle");
    open = !!open && composerNarrow.matches;
    if (shellEl) shellEl.classList.toggle("nav-open", open);
    if (bd) bd.hidden = !open;
    if (btn) btn.setAttribute("aria-expanded", open ? "true" : "false");
    if (side) side.setAttribute("aria-hidden", composerNarrow.matches && !open ? "true" : "false");
    if (open) {
      syncMobileBar();
      setComposerMore(false);
    }
  }
  function placeChapters() {
    var mount = document.getElementById("chapters-mount");
    var flyout = document.getElementById("chapters-flyout");
    if (!chaptersEl || !mount || !flyout) return;
    var narrow = composerNarrow.matches;
    var dest = narrow ? mount : flyout;
    if (chaptersEl.parentElement !== dest) dest.appendChild(chaptersEl);
    if (narrow && chaptersFlyout) {
      chaptersFlyout.hidden = true;
      if (chaptersToggle) chaptersToggle.setAttribute("aria-expanded", "false");
    }
    syncMobileBar();
    if (!narrow) setNavOpen(false);
    if (!narrow && sideTab === "chapters") selectSidebar("now", { keepClosed: true });
    var side = document.getElementById("app-sidebar");
    if (side) side.setAttribute("aria-hidden", narrow && !(shellEl && shellEl.classList.contains("nav-open")) ? "true" : "false");
    requestAnimationFrame(syncMobileBar);
  }
  var sidePollTimer = null;
  function syncSidePoll() {
    if (sidePollTimer) {
      clearInterval(sidePollTimer);
      sidePollTimer = null;
    }
    if (!connected) return;
    if (sideTab !== "connections" && sideTab !== "context" && sideTab !== "now") return;
    sidePollTimer = setInterval(function () {
      if (!connected) return;
      if (sideTab === "connections") loadMcpList().catch(function () {});
      if (sideTab === "context" || sideTab === "now") refreshSideContext();
    }, 4000);
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
    if (shellEl && shellEl.classList.contains("nav-open")) {
      e.preventDefault();
      setNavOpen(false);
      return;
    }
    if (!artifactPane || artifactPane.hidden) return;
    if (artifactViewMode === "overlay" || composerNarrow.matches) {
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
  if (mcpReloadBtn) mcpReloadBtn.addEventListener("click", reloadAllMcp);
  var mcpScriptReload = document.getElementById("mcp-script-reload");
  if (mcpScriptReload) mcpScriptReload.addEventListener("click", reloadAllMcp);
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
      var beforeAdd = document.getElementById("mcp-before-add");
      if (beforeAdd && beforeAdd.value.trim()) body.before = beforeAdd.value.trim();
      if (kind === "stdio") {
        var line = (mcpCommand && mcpCommand.value || "").trim();
        if (!line) {
          showMcpErr("Command required.");
          return;
        }
        var parts = mcpSplitCommand(line);
        body.command = parts[0];
        body.args = parts.slice(1);
        body.env = mcpReadPairs(mcpEnvList);
        body.cwd = (mcpCwd && mcpCwd.value || "").trim();
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
          mcpSignIn(name, url, body.before).then(function () {
            if (mcpName) mcpName.value = "";
            if (mcpUrl) mcpUrl.value = "";
            if (mcpToken) mcpToken.value = "";
            if (beforeAdd) beforeAdd.value = "";
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
      watchMcpScripts();
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
          if (mcpCwd) mcpCwd.value = "";
          if (mcpUrl) mcpUrl.value = "";
          if (mcpToken) mcpToken.value = "";
          if (beforeAdd) beforeAdd.value = "";
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

  function renderEscalations(list) {
    var mount = document.getElementById("escalation-list");
    if (!mount) return;
    mount.textContent = "";
    if (!list.length) {
      var empty = document.createElement("li");
      empty.className = "escalation-empty";
      empty.textContent = "Nothing raised.";
      mount.appendChild(empty);
      return;
    }
    list.forEach(function (e) {
      var li = document.createElement("li");
      li.className = "escalation-item" + (e.done ? " escalation-done" : "");
      var open = document.createElement("button");
      open.type = "button";
      open.className = "escalation-open";
      open.title = e.agent_id ? "Open this agent" : "";
      var head = document.createElement("div");
      head.className = "escalation-head";
      var who = document.createElement("strong");
      who.textContent = e.agent_name || e.agent_id || "agent";
      var when = document.createElement("span");
      when.className = "hint";
      when.textContent = e.created_at ? new Date(e.created_at).toLocaleString() : "";
      head.appendChild(who);
      head.appendChild(when);
      var body = document.createElement("p");
      body.textContent = e.text || "";
      open.appendChild(head);
      open.appendChild(body);
      open.addEventListener("click", function () {
        if (e.agent_id) attachAgent(e.agent_id);
      });
      li.appendChild(open);
      var actions = document.createElement("div");
      actions.className = "escalation-actions";
      var toggle = document.createElement("button");
      toggle.type = "button";
      toggle.className = "ghost";
      toggle.textContent = e.done ? "Reopen" : "Done";
      toggle.addEventListener("click", function () { setEscalationDone(e.id, !e.done); });
      actions.appendChild(toggle);
      li.appendChild(actions);
      mount.appendChild(li);
    });
  }

  function setEscalationDone(id, done) {
    return fetch(apiURL("/api/escalations/done"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: id, done: !!done }),
    })
      .then(readAPIResult)
      .then(function () { return loadEscalations(); })
      .catch(function (err) { appendLive("\n[escalations] " + err + "\n"); });
  }

  function loadEscalations() {
    if (!connected) return Promise.resolve();
    return fetch(apiURL("/api/escalations"), { headers: headers() })
      .then(function (r) { return r.json().then(function (j) { if (!r.ok) throw new Error((j && j.error) || r.status); return j; }); })
      .then(function (j) {
        var list = (j && j.escalations) || [];
        renderEscalations(list);
        var open = list.filter(function (e) { return !e.done; }).length;
        var btn = document.querySelector('#sidebar-tabs button[data-side="escalations"]');
        if (btn) btn.textContent = open ? "Escalations (" + open + ")" : "Escalations";
      })
      .catch(function () { renderEscalations([]); });
  }

  function openApproval(item) {
    if (!item) return;
    if (item.kind === "agent" && item.ref) {
      attachAgent(item.ref);
      return;
    }
    if (item.kind === "workspace" && item.ref) {
      if (item.ref === currentWorkspaceID && !currentAgentInstanceID && item.thread_id) {
        switchThread(item.thread_id);
        return;
      }
      switchWorkspace(item.ref);
    }
  }

  function decideApproval(item, allow) {
    if (!item) return;
    var url = item.worker ? "/api/approvals/decide" : "/api/approve";
    var body = item.worker
      ? { worker: item.worker, thread_id: item.thread_id || "", allow: !!allow }
      : { thread_id: item.thread_id || "", allow: !!allow };
    return fetch(apiURL(url), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify(body),
    })
      .then(readAPIResult)
      .then(function () { return loadApprovals(); })
      .catch(function (err) { appendLive("\n[approvals] " + err + "\n"); });
  }

  function renderApprovals(list) {
    var mount = document.getElementById("now-approvals");
    var empty = document.getElementById("now-approvals-empty");
    if (!mount) return;
    mount.textContent = "";
    list = list || [];
    if (empty) empty.hidden = list.length > 0;
    var btn = document.querySelector('#sidebar-tabs button[data-side="now"]');
    if (btn) btn.textContent = list.length ? "Now (" + list.length + ")" : "Now";
    list.forEach(function (item) {
      var li = document.createElement("li");
      li.className = "escalation-item";
      var open = document.createElement("button");
      open.type = "button";
      open.className = "escalation-open";
      var head = document.createElement("div");
      head.className = "escalation-head";
      var who = document.createElement("strong");
      who.textContent = item.name || item.ref || "waiting";
      var tool = document.createElement("span");
      tool.className = "hint";
      tool.textContent = item.tool || "approval";
      head.appendChild(who);
      head.appendChild(tool);
      var body = document.createElement("p");
      body.textContent = item.reason || item.command || "Needs a decision.";
      open.appendChild(head);
      open.appendChild(body);
      open.addEventListener("click", function () { openApproval(item); });
      var actions = document.createElement("div");
      actions.className = "escalation-actions";
      var allow = document.createElement("button");
      allow.type = "button";
      allow.textContent = "Allow";
      allow.addEventListener("click", function (ev) { ev.stopPropagation(); decideApproval(item, true); });
      var deny = document.createElement("button");
      deny.type = "button";
      deny.className = "ghost";
      deny.textContent = "Deny";
      deny.addEventListener("click", function (ev) { ev.stopPropagation(); decideApproval(item, false); });
      actions.appendChild(deny);
      actions.appendChild(allow);
      li.appendChild(open);
      li.appendChild(actions);
      mount.appendChild(li);
    });
  }

  function loadApprovals() {
    if (!connected) return Promise.resolve();
    return fetch(apiURL("/api/approvals"), { headers: headers() })
      .then(function (r) { return r.json().then(function (j) { if (!r.ok) throw new Error((j && j.error) || r.status); return j; }); })
      .then(function (j) { renderApprovals((j && j.approvals) || []); })
      .catch(function () { renderApprovals([]); });
  }

  function renderGateways(j) {
    var edges = (j && j.edges) || [];
    var pending = (j && j.pending) || [];
    var edgeMount = document.getElementById("gateway-edges");
    var waitMount = document.getElementById("gateway-pending");
    var empty = document.getElementById("gateway-pending-empty");
    var btn = document.querySelector('#sidebar-tabs button[data-side="gateways"]');
    if (btn) btn.textContent = pending.length ? "Gateways (" + pending.length + ")" : "Gateways";
    if (edgeMount) {
      edgeMount.textContent = "";
      if (!edges.length) {
        var none = document.createElement("li");
        none.className = "escalation-empty";
        none.textContent = "No contacts yet.";
        edgeMount.appendChild(none);
      }
      edges.forEach(function (edge) {
        var li = document.createElement("li");
        li.className = "gateway-edge";
        var who = document.createElement("span");
        who.textContent = (edge.from_name || edge.from) + " → " + (edge.to_name || edge.to);
        var label = document.createElement("label");
        var box = document.createElement("input");
        box.type = "checkbox";
        box.checked = !!edge.inspect;
        box.addEventListener("change", function () {
          fetch(apiURL("/api/gateways/inspect"), {
            method: "POST",
            headers: headers(),
            body: JSON.stringify({ from: edge.from, to: edge.to, inspect: box.checked }),
          })
            .then(readAPIResult)
            .then(renderGateways)
            .catch(function (err) { appendLive("\n[gateways] " + err + "\n"); });
        });
        label.appendChild(box);
        label.appendChild(document.createTextNode("Inspect"));
        li.appendChild(who);
        li.appendChild(label);
        edgeMount.appendChild(li);
      });
    }
    if (!waitMount) return;
    waitMount.textContent = "";
    if (empty) empty.hidden = pending.length > 0;
    pending.forEach(function (msg) {
      var li = document.createElement("li");
      li.className = "escalation-item";
      var head = document.createElement("div");
      head.className = "escalation-head";
      var who = document.createElement("strong");
      who.textContent = (msg.from_name || msg.from) + " → " + (msg.to_name || msg.to);
      var when = document.createElement("span");
      when.className = "hint";
      when.textContent = msg.created_at ? new Date(msg.created_at).toLocaleString() : "";
      head.appendChild(who);
      head.appendChild(when);
      var body = document.createElement("p");
      body.textContent = msg.text || "";
      var actions = document.createElement("div");
      actions.className = "escalation-actions";
      var block = document.createElement("button");
      block.type = "button";
      block.className = "ghost";
      block.textContent = "Block";
      block.addEventListener("click", function () { decideGateway(msg.id, false); });
      var allow = document.createElement("button");
      allow.type = "button";
      allow.textContent = "Allow";
      allow.addEventListener("click", function () { decideGateway(msg.id, true); });
      actions.appendChild(block);
      actions.appendChild(allow);
      li.appendChild(head);
      li.appendChild(body);
      li.appendChild(actions);
      waitMount.appendChild(li);
    });
  }

  function decideGateway(id, allow) {
    return fetch(apiURL("/api/gateways/decide"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ id: id, allow: !!allow }),
    })
      .then(readAPIResult)
      .then(renderGateways)
      .catch(function (err) { appendLive("\n[gateways] " + err + "\n"); });
  }

  function loadGateways() {
    if (!connected) return Promise.resolve();
    return fetch(apiURL("/api/gateways"), { headers: headers() })
      .then(function (r) { return r.json().then(function (j) { if (!r.ok) throw new Error((j && j.error) || r.status); return j; }); })
      .then(renderGateways)
      .catch(function () { renderGateways({ edges: [], pending: [] }); });
  }

  function loadAgentBoard() {
    if (!connected) return Promise.resolve();
    return fetch(apiURL("/api/agents/instances?all=1"), { headers: headers() })
      .then(function (r) { return r.json().then(function (j) { if (!r.ok) throw new Error((j && j.error) || r.status); return j; }); })
      .then(function (j) {
        var list = (j && j.instances) || [];
        renderAgentBoard(list);
        renderNowAgents(list);
      })
      .catch(function () { renderAgentBoard([]); renderNowAgents([]); });
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

  // The same fields the board reads decide whether an agent is doing something.
  function agentIsActive(ag) {
    if (!ag) return false;
    if (ag.call) return true;
    return ag.phase === "running" || ag.phase === "waiting";
  }

  var nowAgentsDrawn = null;

  // Slim list for the Now tab: only running, on-a-call, or waiting agents.
  function renderNowAgents(list) {
    var holder = document.getElementById("now-agents");
    if (!holder) return;
    if (sideTab !== "now") return;
    var active = (list || []).filter(agentIsActive);
    var sig = active.map(function (ag) {
      return [ag.id, ag.name || "", ag.recipe_name || ag.recipe_id || "", ag.phase || "", ag.call ? 1 : 0, ag.waiting || ""].join("\t");
    }).join("\n");
    if (sig === nowAgentsDrawn) return;
    nowAgentsDrawn = sig;
    holder.innerHTML = "";
    var empty = document.getElementById("now-agents-empty");
    if (empty) empty.hidden = active.length > 0;
    active.forEach(function (ag) {
      var row = document.createElement("li");
      row.className = "now-agent";
      var open = document.createElement("button");
      open.type = "button";
      open.className = "now-agent-open";
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
      holder.appendChild(row);
    });
  }

  var agentBoardDrawn = "";

  function renderAgentBoard(list) {
    var board = document.getElementById("agent-board");
    if (!board) return;
    var sig = (list || []).map(function (ag) {
      return [ag.id, ag.name || "", ag.recipe_name || ag.recipe_id || "", ag.phase || "", ag.call ? 1 : 0, ag.waiting || ""].join("\t");
    }).join("\n");
    if (sig === agentBoardDrawn && board.childElementCount) return;
    agentBoardDrawn = sig;
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
  var workspacePruneBtn = document.getElementById("workspace-prune-btn");
  if (workspacePruneBtn) {
    workspacePruneBtn.addEventListener("click", function () { pruneWorkspaces(); });
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
  function paintComposerMore() {
    if (!composerMoreBtn) return;
    var on = lowContextOn || followThinking;
    composerMoreBtn.classList.toggle("low-on", on);
    var bits = [];
    if (lowContextOn) bits.push("Reduce context is on");
    if (followThinking) bits.push("Follow thinking is on");
    composerMoreBtn.title = bits.length ? bits.join(". ") : "More";
  }
  function applyLowContext(on) {
    lowContextOn = !!on;
    if (lowContextBtn) {
      lowContextBtn.setAttribute("aria-checked", lowContextOn ? "true" : "false");
      lowContextBtn.title = lowContextOn
        ? "On. Internal tools are names until a search loads one."
        : "Off. Internal tools stay in the request.";
    }
    var state = document.getElementById("low-context-state");
    if (state) state.textContent = lowContextOn ? "On" : "Off";
    paintComposerMore();
  }
  function applyFollowThinking(on) {
    followThinking = !!on;
    try {
      if (followThinking) localStorage.setItem("keimling.follow_thinking", "1");
      else localStorage.removeItem("keimling.follow_thinking");
    } catch (e) {}
    if (followThinkingBtn) {
      followThinkingBtn.setAttribute("aria-checked", followThinking ? "true" : "false");
      followThinkingBtn.title = followThinking
        ? "On. The thinking row that is receiving text stays open."
        : "Off. Thinking stays closed.";
    }
    var state = document.getElementById("follow-thinking-state");
    if (state) state.textContent = followThinking ? "On" : "Off";
    paintComposerMore();
  }
  applyFollowThinking(followThinking);
  if (followThinkingBtn) {
    followThinkingBtn.addEventListener("click", function (e) {
      e.preventDefault();
      e.stopPropagation();
      applyFollowThinking(!followThinking);
    });
  }
  if (thread) {
    thread.addEventListener("toggle", function (e) {
      if (applyingFollow) return;
      var d = e.target;
      if (!d || !d.classList || !d.classList.contains("thinking-fold")) return;
      if (d.open) {
        delete d.dataset.autoFollow;
        followDismissedKey = "";
        return;
      }
      if (d.dataset.autoFollow === "1") {
        followDismissedKey = String(d.dataset.domKey || "").replace(/:think$/, "");
        delete d.dataset.autoFollow;
      }
    }, true);
  }
  if (lowContextBtn) {
    lowContextBtn.addEventListener("click", function (e) {
      e.preventDefault();
      e.stopPropagation();
      if (!connected) return;
      var next = !lowContextOn;
      applyLowContext(next);
      fetch(apiURL("/api/threads/low-context"), {
        method: "PUT",
        headers: headers(),
        body: JSON.stringify({ enabled: next }),
      }).then(function (r) {
        if (!r.ok) throw new Error("save failed");
      }).catch(function () {
        applyLowContext(!next);
      });
    });
  }
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
    if (e.target.closest && (e.target.closest(".composer-more") || e.target.closest("#composer-more-menu"))) return;
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
  var navToggle = document.getElementById("nav-toggle");
  if (navToggle) {
    navToggle.addEventListener("click", function () {
      setNavOpen(!(shellEl && shellEl.classList.contains("nav-open")));
    });
  }
  var navBackdrop = document.getElementById("nav-backdrop");
  if (navBackdrop) navBackdrop.addEventListener("click", function () { setNavOpen(false); });
  var appSidebar = document.getElementById("app-sidebar");
  if (appSidebar) {
    appSidebar.addEventListener("click", function (e) {
      if (!composerNarrow.matches) return;
      if (e.target.closest("#thread-list button, #agent-list button, #new-chat, #open-connection")) setNavOpen(false);
    });
  }
  placeChapters();
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
      var portInput = document.getElementById("plugin-port");
      var location = input ? String(input.value || "").trim() : "";
      var port = portInput ? String(portInput.value || "").trim() : "";
      if (!location) {
        showPluginErr("Git URL or directory required");
        return;
      }
      showPluginErr("");
      pluginAdd.disabled = true;
      var addLabel = pluginAdd.textContent;
      pluginAdd.textContent = "Adding…";
      fetch(apiURL("/api/plugins/add"), { method: "POST", headers: headers(), body: JSON.stringify({ location: location, port: port }) })
        .then(readAPIResult)
        .then(function () {
          if (input) input.value = "";
          if (portInput) portInput.value = "";
          loadPlugins();
        })
        .catch(function (err) { showPluginFailure(err); })
        .finally(function () {
          pluginAdd.disabled = false;
          pluginAdd.textContent = addLabel;
        });
    });
  }

  settingsTabs.querySelectorAll(".settings-tab").forEach(function (btn) {
    btn.addEventListener("click", function () {
      setSettingsTab(btn.getAttribute("data-tab"));
      if ((btn.getAttribute("data-tab") === "models" || btn.getAttribute("data-tab") === "agent") && connected) loadAgentSettings();
      if (btn.getAttribute("data-tab") === "secrets" && connected) loadCredentialsUI();
      if (btn.getAttribute("data-tab") === "plugins" && connected) loadPlugins();
      if (btn.getAttribute("data-tab") === "spend" && connected) loadSpend();
    });
  });
  function renderToolAsk(rows) {
    if (!setToolAsk) return;
    setToolAsk.innerHTML = "";
    (rows || []).forEach(function (row) {
      var label = document.createElement("label");
      label.className = "check";
      var input = document.createElement("input");
      input.type = "checkbox";
      input.checked = !!row.ask;
      input.setAttribute("data-tool", row.name);
      input.addEventListener("change", saveToolAsk);
      label.appendChild(input);
      label.appendChild(document.createTextNode(" " + row.name));
      setToolAsk.appendChild(label);
    });
  }
  function saveToolAsk() {
    if (!setToolAsk || !connected) return;
    var map = {};
    setToolAsk.querySelectorAll("input[data-tool]").forEach(function (el) {
      map[el.getAttribute("data-tool")] = el.checked;
    });
    fetch(apiURL("/api/settings"), {
      method: "PUT",
      headers: headers(),
      body: JSON.stringify({ tool_ask: map }),
    }).then(function (r) { return r.json(); }).then(function (j) {
      if (j && j.tool_ask) renderToolAsk(j.tool_ask);
    }).catch(function () {});
  }
  function openInstructionEditor(kind) {
    if (!instructionEditor || !instructionEditorText) return;
    instructionKind = kind;
    if (instructionEditorErr) instructionEditorErr.hidden = true;
    if (instructionEditorTitle) {
      instructionEditorTitle.textContent = kind === "review" ? "Auto-review instructions" : "Autopilot instructions";
    }
    instructionEditorText.value = kind === "review"
      ? (setApprovalPolicy ? setApprovalPolicy.value : approvalPolicyBaseline)
      : (setAutopilotInstructions ? setAutopilotInstructions.value : "");
    if (typeof instructionEditor.showModal === "function") instructionEditor.showModal();
  }
  function saveInstructionEditor() {
    if (!instructionEditorText) return;
    var text = instructionEditorText.value;
    var body = {};
    if (instructionKind === "review") {
      body.approval_policy = text;
    } else {
      if (!text.trim()) {
        if (instructionEditorErr) {
          instructionEditorErr.hidden = false;
          instructionEditorErr.textContent = "Autopilot instructions cannot be empty. Use Reset to go back to the built-in prompt.";
        }
        return;
      }
      body.autopilot_instructions = text;
    }
    fetch(apiURL("/api/settings"), {
      method: "PUT",
      headers: headers(),
      body: JSON.stringify(body),
    }).then(function (r) {
      return r.json().then(function (j) {
        if (!r.ok) throw new Error((j && j.message) || ("HTTP " + r.status));
        fillAgentSettings(j);
        if (instructionEditor && instructionEditor.open) instructionEditor.close();
      });
    }).catch(function (e) {
      if (instructionEditorErr) {
        instructionEditorErr.hidden = false;
        instructionEditorErr.textContent = String(e.message || e);
      }
    });
  }
  if (editAutopilotText) editAutopilotText.addEventListener("click", function () { openInstructionEditor("autopilot"); });
  if (editReviewText) editReviewText.addEventListener("click", function () { openInstructionEditor("review"); });
  if (instructionEditorSave) instructionEditorSave.addEventListener("click", saveInstructionEditor);
  if (instructionEditorFull) instructionEditorFull.addEventListener("click", function () {
    if (!instructionEditor) return;
    if (document.fullscreenElement === instructionEditor) document.exitFullscreen();
    else if (instructionEditor.requestFullscreen) instructionEditor.requestFullscreen();
  });
  if (setArtifactProxy) setArtifactProxy.addEventListener("change", function () {
    fetch(apiURL("/api/settings"), {
      method: "PUT",
      headers: headers(),
      body: JSON.stringify({ artifact_proxy: setArtifactProxy.checked }),
    }).catch(function () {});
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
      editingDefinitionID = "";
      creatingTests = [];
      clearDefinitionTestForm();
      renderDefinitionTests(null);
      setAgentDefPane("create");
      loadModelSuggestions();
      loadMcpServers().then(function () {
        renderToolMatrix(document.getElementById("agent-def-create-tools"), { granted: null });
        renderPeerChoices(document.getElementById("agent-def-create-peers"), definitionsCache, []);
        renderToolRows({});
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
  var agentDefTestList = document.getElementById("agent-def-test-list");
  if (agentDefTestList) {
    agentDefTestList.addEventListener("click", function (e) {
      var button = e.target.closest("button");
      if (!button || !button.dataset.testAct) return;
      e.preventDefault();
      var test = null;
      testsOnScreen().forEach(function (one) {
        if (one.id === button.dataset.testId) test = one;
      });
      if (!test) return;
      if (button.dataset.testAct === "chat") {
        openDefinitionTestChat(test, Number(button.dataset.runIndex) || 0);
        return;
      }
      if (button.dataset.testAct === "edit") {
        openDefinitionTestEditor(test);
        return;
      }
      if (button.dataset.testAct === "run") {
        if (!editingDefinitionID) {
          showAgentDefErr("Save the definition before running this test.");
          return;
        }
        runDefinitionTest(test.id, button);
        return;
      }
      if (button.dataset.testAct === "delete") {
        askConfirm("Delete test “" + (test.name || test.id) + "”?", "Delete").then(function (yes) {
          if (!yes) return;
          if (!editingDefinitionID) {
            creatingTests = creatingTests.filter(function (one) { return one.id !== test.id; });
            showDefinitionTestUndo(test);
            renderDefinitionTests(null);
            return;
          }
          fetch(apiURL("/api/agents/recipes/tests/delete"), {
            method: "POST",
            headers: headers(),
            body: JSON.stringify({ id: editingDefinitionID, test_id: test.id }),
          }).then(readAPIResult).then(function () {
            showDefinitionTestUndo(test);
            return loadDefinitions();
          }).then(function () {
            renderDefinitionTests(currentEditingDefinition());
          }).catch(function (err) {
            showAgentDefErr(String(err));
          });
        });
      }
    });
  }
  var agentDefTestSave = document.getElementById("agent-def-test-save");
  if (agentDefTestSave) {
    agentDefTestSave.addEventListener("click", function (e) {
      e.preventDefault();
      var test;
      try { test = parseDefinitionTestForm(); }
      catch (err) { showDefinitionTestErr(String(err.message || err)); return; }
      var problem = localTestProblem(test);
      if (problem) { showDefinitionTestErr(problem); return; }
      showDefinitionTestErr("");
      saveDefinitionTest(test).catch(function (err) { showDefinitionTestErr(String(err.message || err)); });
    });
  }
  var agentDefTestNew = document.getElementById("agent-def-test-new");
  if (agentDefTestNew) {
    agentDefTestNew.addEventListener("click", function (e) {
      e.preventDefault();
      openDefinitionTestEditor(null);
    });
  }
  ["agent-test-dialog-close", "agent-test-dialog-cancel"].forEach(function (id) {
    var el = document.getElementById(id);
    if (!el) return;
    el.addEventListener("click", function (e) {
      e.preventDefault();
      closeDefinitionTestEditor();
    });
  });
  var agentTestDialog = document.getElementById("agent-test-dialog");
  if (agentTestDialog) agentTestDialog.addEventListener("close", clearDefinitionTestForm);
  var agentDefDialog = document.getElementById("agent-def-dialog");
  if (agentDefDialog) {
    agentDefDialog.addEventListener("close", function () {
      closeDefinitionTestEditor();
      closeDefinitionTestChat();
    });
  }
  var agentDefTests = document.getElementById("agent-test-dialog");
  if (agentDefTests) {
    agentDefTests.addEventListener("click", function (e) {
      var add = e.target.closest("[data-test-add]");
      if (add && add.dataset.testAdd !== "tool") {
        e.preventDefault();
        if (add.dataset.testAdd === "party") {
          var parties = readPartiesFromForm();
          parties.push({ id: "", name: "", kind: "agent" });
          renderPartyRows(parties);
          syncStepPartySelects();
        } else if (add.dataset.testAdd === "step") {
          var steps = readStepsFromForm();
          steps.push({ from: "", text: "", to: "", check: "" });
          renderStepRows(steps);
        }
        return;
      }
      if (add && add.dataset.testAdd === "tool") {
        e.preventDefault();
        var tools = readToolsFromForm();
        var toolName = add.dataset.tool;
        tools[toolName] = toolResultsList(tools[toolName]);
        if (!tools[toolName].length) tools[toolName] = [""];
        tools[toolName].push("");
        renderToolRows(tools);
        return;
      }
      var remove = e.target.closest("[data-test-remove]");
      if (!remove) return;
      e.preventDefault();
      if (remove.dataset.testRemove === "party") {
        var left = [];
        document.querySelectorAll("#agent-def-test-parties .agent-test-party").forEach(function (row) {
          if (row.contains(remove)) return;
          var inputs = row.querySelectorAll("input");
          var kind = row.querySelector("select");
          left.push({ id: inputs[0] ? inputs[0].value : "", name: inputs[1] ? inputs[1].value : "", kind: kind ? kind.value : "agent" });
        });
        renderPartyRows(left);
        syncStepPartySelects();
        return;
      }
      if (remove.dataset.testRemove === "step") {
        var kept = [];
        document.querySelectorAll("#agent-def-test-steps .agent-test-step").forEach(function (row) {
          if (!row.contains(remove)) {
            var lines = row.querySelectorAll(".agent-test-step-line");
            function part(line) {
              if (!line) return { who: "", text: "" };
              var select = line.querySelector("select");
              var input = line.querySelector("input");
              return { who: select ? select.value : "", text: input ? input.value : "" };
            }
            var write = part(lines[0]);
            var expect = part(lines[1]);
            kept.push({ from: write.who, text: write.text, to: expect.who, check: expect.text });
          }
        });
        renderStepRows(kept);
        return;
      }
      if (remove.dataset.testRemove === "tool") {
        var block = remove.closest(".agent-test-tool");
        var current = readToolsFromForm();
        if (block) {
          var rows = block.querySelectorAll(".agent-test-tool-call");
          var index = Array.prototype.indexOf.call(rows, remove.closest(".agent-test-tool-call"));
          var seq = toolResultsList(current[block.dataset.tool]);
          if (index >= 0) seq.splice(index, 1);
          if (!seq.length) seq = [""];
          current[block.dataset.tool] = seq;
        }
        renderToolRows(current);
      }
    });
    var partyHost = document.getElementById("agent-def-test-parties");
    if (partyHost) partyHost.addEventListener("change", syncStepPartySelects);
  }
  var agentDefTestInfo = document.getElementById("agent-def-test-info");
  if (agentDefTestInfo) {
    agentDefTestInfo.addEventListener("click", function (e) {
      e.preventDefault();
      var example = document.getElementById("agent-def-test-example");
      if (!example) return;
      example.hidden = !example.hidden;
      agentDefTestInfo.setAttribute("aria-expanded", example.hidden ? "false" : "true");
    });
  }
  var agentDefTestChatBack = document.getElementById("agent-def-test-chat-back");
  if (agentDefTestChatBack) {
    agentDefTestChatBack.addEventListener("click", function (e) {
      e.preventDefault();
      closeDefinitionTestChat();
    });
  }
  ["agent-def-create-tools", "agent-def-edit-tools"].forEach(function (id) {
    var el = document.getElementById(id);
    if (!el) return;
    el.addEventListener("change", function () {
      if (!document.getElementById("agent-def-tests") || document.getElementById("agent-def-tests").hidden) return;
      renderToolRows(readToolsFromForm(), true);
    });
  });
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
        clearLive();
        hideApproval();
        return loadSession();
      })
      .then(function () {
        connectEvents();
        return Promise.all([loadThreads(), loadAgents(), refreshSidebar()]);
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
      saveChatDraft();
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

  function clearSentDraft(opts) {
    if (opts && opts.dictate) {
      if (dictateDraft) dictateDraft.value = "";
      syncDictateChrome();
      return;
    }
    msg.value = "";
    forgetChatDraft();
    fitComposer();
  }

  function sendMessage(text, opts) {
    opts = opts || {};
    text = String(text || "").trim();
    var files = opts.dictate ? [] : pendingFiles.slice();
    if (!text && !files.length) return;
    if (!sendMessage.releasing) {
      send.disabled = true;
      Promise.resolve(modelSave).then(function () {
        sendMessage.releasing = true;
        try { sendMessage(text, opts); } finally { sendMessage.releasing = false; }
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
    var sentFrom = currentThreadID;
    var otherThread = backgroundBusy;
    var picks = currentAgentInstanceID ? [] : mentionPicks.slice();
    fetch(apiURL("/api/message"), {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ text: text, to: picks, thread_id: currentThreadID || "", attachments: files.map(function (f) { return f.id; }) }),
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
          clearSentDraft(opts);
          if (!opts.dictate) {
            mentionPicks = [];
            pendingFiles = [];
          }
          renderAttachRow();
          var bodyText = j.text != null ? String(j.text) : text;
          var sentFiles = (j && j.files) || files.map(function (f) { return f.name; });
          var shown = currentThreadID;
          var fromID = (j && j.thread_id) || "";
          if (!fromID || fromID === "main") fromID = workspaceThreadID || currentThreadID;
          // A message the user typed in this chat is never a reason to move the
          // view to another chat. That silent switch is what looked like the
          // reply landing in the wrong transcript.
          var movedChat = !!(fromID && fromID !== "main" && sentFrom && sentFrom !== "main" && fromID !== sentFrom);
          if (fromID && fromID !== "main" && !movedChat) {
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
          if (movedChat) {
            messages.push({ role: "system", text: "This message was recorded in another chat (" + (threadTitle || fromID) + "), not here." });
          }
          renderThread();
          scrollTranscriptToEnd();
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
        clearSentDraft(opts);
        if (!opts.dictate) {
          mentionPicks = [];
          pendingFiles = [];
        }
        renderAttachRow();
        messages.push({ role: "user", text: text, files: files.map(function (f) { return f.name; }) });
        lastUserEchoText = String(text || "").trim();
        lastUserEchoAt = Date.now();
        noteGrowthMessage({ role: "user" });
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
        scrollTranscriptToEnd();
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
  // Last time the server said something while a run was open. A run that sits
  // silent is shown as such instead of a plain "running".
  var lastRunActivityAt = 0;
  // A sent message is shown at once; the server echo of the same line must not
  // add a second copy. Text plus time is the guard.
  var lastUserEchoText = "";
  var lastUserEchoAt = 0;
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
    if (lastUserEchoText && lastUserEchoText === text && Date.now() - lastUserEchoAt < 5000) {
      lastUserEchoText = "";
      return;
    }
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
        if (role === "user") scrollTranscriptToEnd();
        else followTranscript();
        return;
      }
    }
    messages.push({ role: role, text: text, live: true, streaming: !done, liveDone: !!done });
    renderThread();
    if (role === "user") scrollTranscriptToEnd();
    else followTranscript();
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
  if (callShareBtn) callShareBtn.addEventListener("click", function () {
    if (screenBtn) screenBtn.click();
  });
  var screenInputBtn = document.getElementById("screen-input");
  if (screenInputBtn) screenInputBtn.addEventListener("click", function () {
    shareScreen({ input: !screenInput, target: screenTarget, picked: true });
  });
  if (screenDismiss) screenDismiss.addEventListener("click", stopScreenShare);

  // The camera share is its own browser stream. It keeps running beside a
  // screen share, and the newest frame goes with the next chat turn.
  var cameraBtn = document.getElementById("camera-share");
  var cameraPreview = document.getElementById("camera-preview");
  var cameraVideo = document.getElementById("camera-video");
  var cameraDismiss = document.getElementById("camera-dismiss");
  var cameraStream = null;
  var cameraTimer = null;
  var cameraBusy = false;
  var cameraSending = false;
  var cameraRun = 0;

  // A camera frame is scaled down to a small edge and sent as a JPEG.
  function cameraFrameFit(width, height, maxEdge) {
    if (!width || !height || !maxEdge) return { width: 0, height: 0 };
    var scale = Math.min(1, maxEdge / Math.max(width, height));
    return {
      width: Math.max(1, Math.round(width * scale)),
      height: Math.max(1, Math.round(height * scale)),
    };
  }

  // Desktop browsers take any camera. Phones prefer the back camera and fall
  // back to whatever camera the browser offers.
  function cameraVideoConstraints(facing) {
    var video = { width: { ideal: 1280 }, height: { ideal: 720 }, frameRate: { ideal: 15, max: 30 } };
    if (facing === "environment" || facing === "user") video.facingMode = { ideal: facing };
    return { audio: false, video: video };
  }

  function setCameraStatus(text) {
    var el = document.getElementById("camera-status");
    if (el) el.textContent = text || "";
  }

  function revealCameraShare(text) {
    var block = document.getElementById("side-camera");
    if (block) block.hidden = false;
    if (text) setCameraStatus(text);
    selectSidebar("now");
  }

  function postCameraFrame(blob) {
    if (!blob || cameraSending) return;
    cameraSending = true;
    var h = headers();
    h["Content-Type"] = "image/jpeg";
    fetch(apiURL("/api/camera/share"), { method: "POST", headers: h, body: blob })
      .catch(function () {})
      .then(function () { cameraSending = false; });
  }

  function canvasJpeg(canvas) {
    if (canvas.toBlob) {
      return new Promise(function (resolve) {
        canvas.toBlob(function (blob) { resolve(blob || null); }, "image/jpeg", 0.6);
      });
    }
    var url = canvas.toDataURL("image/jpeg", 0.6);
    return fetch(url).then(function (r) { return r.blob(); }).catch(function () { return null; });
  }

  // Five frames a second are enough for the chat model and stay cheap on a phone.
  function pumpCameraFrames() {
    if (!cameraStream || !cameraVideo) return;
    var width = cameraVideo.videoWidth;
    var height = cameraVideo.videoHeight;
    if (!width || !height) return;
    var fit = cameraFrameFit(width, height, 1280);
    var canvas = document.createElement("canvas");
    canvas.width = fit.width;
    canvas.height = fit.height;
    var ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.drawImage(cameraVideo, 0, 0, fit.width, fit.height);
    canvasJpeg(canvas).then(postCameraFrame).catch(function () {});
  }

  function stopCameraTracks() {
    if (cameraStream) {
      cameraStream.getTracks().forEach(function (track) {
        try { track.stop(); } catch (e) { /* the camera already closed */ }
      });
    }
    cameraStream = null;
  }

  function stopCameraShare() {
    cameraRun++;
    if (cameraTimer) clearInterval(cameraTimer);
    cameraTimer = null;
    stopCameraTracks();
    if (cameraVideo) {
      if (cameraVideo.pause) cameraVideo.pause();
      cameraVideo.srcObject = null;
    }
    cameraSending = false;
    cameraBusy = false;
    if (cameraPreview) cameraPreview.hidden = true;
    var block = document.getElementById("side-camera");
    if (block) block.hidden = true;
    setCameraStatus("");
    // The server forgets the frame too, so the next turn carries no picture.
    fetch(apiURL("/api/camera/share?stop=1"), { method: "POST", headers: headers() }).catch(function () {});
    setState(agentState);
  }

  function startCameraShare() {
    var block = document.getElementById("side-camera");
    if (cameraStream) {
      if (block) block.hidden = false;
      selectSidebar("now");
      return;
    }
    var media = navigator && navigator.mediaDevices;
    if (!media || !media.getUserMedia) {
      revealCameraShare("This window cannot share a camera.");
      return;
    }
    var gen = ++cameraRun;
    cameraBusy = true;
    setState(agentState);
    revealCameraShare("Allow the camera in the browser dialog.");
    var phone = /Android|iPhone|iPad|iPod|Mobile/i.test((navigator.userAgent || ""));
    media.getUserMedia(cameraVideoConstraints(phone ? "environment" : ""))
      .catch(function () { return media.getUserMedia({ audio: false, video: true }); })
      .then(function (stream) {
        if (gen !== cameraRun) {
          stream.getTracks().forEach(function (track) { track.stop(); });
          return;
        }
        cameraStream = stream;
        if (cameraVideo) {
          cameraVideo.srcObject = stream;
          var played = cameraVideo.play && cameraVideo.play();
          if (played && played.catch) played.catch(function () {});
        }
        if (cameraPreview) cameraPreview.hidden = false;
        setCameraStatus("Sharing the camera.");
        cameraTimer = setInterval(pumpCameraFrames, 200);
        cameraBusy = false;
        setState(agentState);
      })
      .catch(function (err) {
        if (gen !== cameraRun) return;
        cameraBusy = false;
        setState(agentState);
        var msg = (err && err.message) || err || "failed";
        if (err && err.name === "NotAllowedError") revealCameraShare("Camera share was cancelled.");
        else revealCameraShare("Camera: " + msg);
      });
  }

  if (cameraBtn) cameraBtn.addEventListener("click", function () {
    var block = document.getElementById("side-camera");
    if (block && !block.hidden && cameraStream) {
      selectSidebar("now");
      return;
    }
    startCameraShare();
  });
  if (cameraDismiss) cameraDismiss.addEventListener("click", stopCameraShare);
  if (callBtn) callBtn.addEventListener("click", function () {
    if (callLive) hangUpCall();
    else startCall();
  });
  if (callMuteBtn) callMuteBtn.addEventListener("click", function () { toggleCallMute(); });

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

  function recordingName(blob) {
    var type = blob.type || "";
    var ext = type.indexOf("mp4") >= 0 ? "mp4" : type.indexOf("ogg") >= 0 ? "ogg" : "webm";
    return "voice." + ext;
  }

  function updateAudioActions() {
    if (audioDownloadBtn) audioDownloadBtn.disabled = !lastRecording;
    if (audioUploadBtn) audioUploadBtn.disabled = !connected || !voiceOK || callLive || recording;
  }

  if (audioDownloadBtn) audioDownloadBtn.addEventListener("click", function () {
    if (!lastRecording) return;
    var url = window.URL.createObjectURL(lastRecording);
    var link = document.createElement("a");
    link.href = url;
    link.download = recordingName(lastRecording);
    document.body.appendChild(link);
    link.click();
    link.remove();
    setTimeout(function () { window.URL.revokeObjectURL(url); }, 60000);
  });

  if (audioUploadBtn && audioInput) audioUploadBtn.addEventListener("click", function () {
    if (!audioUploadBtn.disabled) audioInput.click();
  });
  if (audioInput) audioInput.addEventListener("change", function () {
    var file = audioInput.files && audioInput.files[0];
    audioInput.value = "";
    if (!file || !canUseVoice() || callLive || recording) return;
    if (file.size < 44 || file.size > 25 * 1024 * 1024) {
      setVoiceStatus("Voice: audio file size must be between 44 bytes and 25 MiB.");
      return;
    }
    uploadRecording(file, file.name || recordingName(file), false);
  });

  function voiceDraftEl() {
    return voiceTarget === "dictate" && dictateDraft ? dictateDraft : msg;
  }

  function uploadRecording(blob, filename, sendAfter) {
    voiceWorking = true;
    syncDictateChrome();
    setVoiceStatus("Transcribing…");
    var fd = new FormData();
    fd.append("audio", blob, filename || recordingName(blob));
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
        voiceWorking = false;
        text = String(text || "").trim();
        var el = voiceDraftEl();
        if (!text) {
          setVoiceStatus("No speech detected.");
          syncDictateChrome();
          return;
        }
        if (el.value && !/\s$/.test(el.value)) el.value += " ";
        el.value += text;
        if (el === msg) fitComposer();
        setVoiceStatus("");
        syncDictateChrome();
        if (sendAfter) {
          sendMessage(el.value, el === dictateDraft ? { dictate: true } : undefined);
          return;
        }
        // Keep the focus where it was: focusing the draft would open the
        // on-screen keyboard and shift the whole mobile layout.
      })
      .catch(function (err) {
        voiceWorking = false;
        syncDictateChrome();
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
    if (!canUseVoice() || recording || voiceWorking) return;
    if (composerMobile) {
      voiceTarget = "dictate";
      showComposerPage("dictate");
    } else voiceTarget = "msg";
    dictateMuted = false;
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
          var sendAfter = voiceSendPending;
          var discard = voiceDiscard;
          voiceSendPending = false;
          voiceDiscard = false;
          recording = false;
          dictateMuted = false;
          voiceBtn.classList.remove("recording");
          var type = mime || (mediaChunks[0] && mediaChunks[0].type) || "audio/webm";
          var blob = new Blob(mediaChunks, { type: type });
          mediaChunks = [];
          if (blob.size && !discard) lastRecording = blob;
          setState(agentState);
          stopTracks();
          if (discard) {
            setVoiceStatus("");
            syncDictateChrome();
            return;
          }
          if (blob.size < 64) {
            setVoiceStatus("Recording too short.");
            return;
          }
          uploadRecording(blob, recordingName(blob), sendAfter);
        };
        mediaRecorder.start();
        recording = true;
        voiceSendPending = false;
        voiceBtn.classList.add("recording");
        setVoiceStatus("Recording… Space = draft · Enter = send");
        setState(agentState);
        if (voiceTarget !== "dictate") msg.focus();
      })
      .catch(function (err) {
        setVoiceStatus("Mic permission: " + err);
      });
  }

  voiceBtn.addEventListener("click", function () {
    if (voiceWorking) return;
    if (recording) stopRecording(false);
    else startRecording();
  });

  function sendDictate() {
    if (voiceWorking) return;
    if (recording) {
      voiceTarget = "dictate";
      stopRecording(true);
      return;
    }
    if (!dictateDraft || !dictateDraft.value.trim()) return;
    sendMessage(dictateDraft.value, { dictate: true });
  }

  function cancelDictate() {
    dictateMuted = false;
    if (recording) {
      voiceDiscard = true;
      stopRecording(false);
    }
    if (dictateDraft) dictateDraft.value = "";
    setVoiceStatus("");
    syncDictateChrome();
  }

  if (dictateCancelBtn) dictateCancelBtn.addEventListener("click", cancelDictate);
  if (dictateSendBtn) dictateSendBtn.addEventListener("click", sendDictate);
  if (dictateDraft) {
    dictateDraft.addEventListener("input", syncDictateChrome);
    dictateDraft.addEventListener("keydown", function (e) {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        sendDictate();
      }
    });
  }
  if (dictateMuteBtn) dictateMuteBtn.addEventListener("click", function () {
    if (!recording || !mediaStream) return;
    dictateMuted = !dictateMuted;
    mediaStream.getAudioTracks().forEach(function (t) { t.enabled = !dictateMuted; });
    syncDictateChrome();
  });
  document.querySelectorAll(".composer-dot").forEach(function (dot) {
    dot.addEventListener("click", function () { showComposerPage(dot.getAttribute("data-page")); });
  });
  (function bindComposerSwipe() {
    var row = document.querySelector(".composer-row");
    if (!row) return;
    var drag = null;
    row.addEventListener("pointerdown", function (e) {
      if (!composerMobile || composerPageList().length < 2) return;
      if (e.target.closest && e.target.closest("button, a, input, label")) return;
      drag = { x: e.clientX, y: e.clientY, dx: 0, locked: false };
    });
    row.addEventListener("pointermove", function (e) {
      if (!drag) return;
      var dx = e.clientX - drag.x;
      var dy = e.clientY - drag.y;
      if (!drag.locked) {
        if (Math.abs(dx) < 12 && Math.abs(dy) < 12) return;
        if (Math.abs(dy) > Math.abs(dx)) {
          drag = null;
          return;
        }
        drag.locked = true;
        var track = document.getElementById("composer-track");
        if (track) track.classList.add("is-dragging");
        try { row.setPointerCapture(e.pointerId); } catch (err) {}
      }
      drag.dx = dx;
      paintComposerTrack(composerRubber(dx));
    });
    function endDrag() {
      var g = drag;
      drag = null;
      var track = document.getElementById("composer-track");
      if (track) track.classList.remove("is-dragging");
      if (!g || !g.locked) {
        paintComposerTrack(0);
        return;
      }
      var list = composerPageList();
      var index = list.indexOf(composerPageId);
      if (g.dx > 48 && index > 0) showComposerPage(list[index - 1]);
      else if (g.dx < -48 && index < list.length - 1) showComposerPage(list[index + 1]);
      else paintComposerTrack(0);
    }
    row.addEventListener("pointerup", endDrag);
    row.addEventListener("pointercancel", endDrag);
  })();
  if (composerNarrow.addEventListener) {
    composerNarrow.addEventListener("change", function () {
      composerMobile = composerNarrow.matches;
      setComposerMore(false);
      placeComposerControls();
      placeChapters();
      fitComposer();
      if (artifactPane && !artifactPane.hidden) syncArtifactModeUI();
    });
  }

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
  window.addEventListener("scroll", syncMobileBar, true);
  window.addEventListener("resize", function () {
    if (geometryTimer) clearTimeout(geometryTimer);
    geometryTimer = setTimeout(persistWindowGeometry, 400);
    syncMobileBar();
    if (composerMobile) paintComposerTrack(0);
    else fitComposer();
    if (!shellEl || !artifactPane || artifactPane.hidden || artifactViewMode === "overlay") return;
    var saved = 0;
    try { saved = parseInt(localStorage.getItem(STORAGE_ARTIFACT_WIDTH) || "", 10); } catch (e) {}
    if (!(saved > 0)) saved = artifactPane.getBoundingClientRect().width;
    applyArtifactWidth(saved);
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
    if (a.hasAttribute("download") && href.indexOf("blob:") === 0) return;
    e.preventDefault();
    e.stopPropagation();
    try { keimlingOpenExternal(a.href); } catch (err) {}
  }
  document.addEventListener("click", openLinkExternally, true);
  document.addEventListener("auxclick", openLinkExternally, true);

  placeComposerControls();

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

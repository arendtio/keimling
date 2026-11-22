"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

const app = fs.readFileSync(path.join(__dirname, "../assets/app.js"), "utf8");
function section(start, end) {
  const a = app.indexOf(start);
  const b = app.indexOf(end, a);
  assert.ok(a >= 0 && b > a, `missing app section: ${start}`);
  return app.slice(a, b);
}
const sessionCode = [
  section("  function transcriptMatches(", "  function stampMessageEl("),
  section("  // Within this many pixels of the bottom", "  function appendLive("),
  section("  function sessionViewKey(", "  function loadSession("),
  section("  // A draft belongs to the chat it was typed in.", "  function switchThread("),
].join("\n");

// Model a browser that clamps the scroll position when an attached transcript
// is emptied, then dispatches its scroll event after the new rows are added.
function makeSession({ position = 9400, matching = false, rowHeight = 4700, snapshotMessages, activeChapter = null, snapshotChapters = [] } = {}) {
  const page = { scrollHeight: 10000, clientHeight: 600, scrollTop: position };
  const frames = [];
  const calls = [];
  let scrollListener;
  let pendingScroll = false;

  class Container {
    constructor(attached = false) {
      this.attached = attached;
      this.children = [];
      this.dataset = {};
      this.classList = { remove() {}, toggle() {} };
    }
    set innerHTML(value) {
      this.html = value;
      this.children = [];
      if (this.attached) {
        page.scrollHeight = 600;
        page.scrollTop = 0;
        pendingScroll = true;
      }
    }
    get innerHTML() { return this.html || ""; }
    get firstChild() { return this.children[0] || null; }
    get childNodes() { return this.children; }
    appendChild(child) {
      if (child.parent) child.parent.children.splice(child.parent.children.indexOf(child), 1);
      child.parent = this;
      this.children.push(child);
      if (this.attached) page.scrollHeight = 600 + this.children.length * rowHeight;
      return child;
    }
    cloneNode() {
      const clone = new Container();
      clone.dataset = { ...this.dataset };
      return clone;
    }
    replaceChildren(fragment) {
      this.children = [];
      while (fragment.firstChild) this.appendChild(fragment.firstChild);
      const max = Math.max(0, page.scrollHeight - page.clientHeight);
      if (page.scrollTop > max) {
        page.scrollTop = max;
        pendingScroll = true;
      }
    }
    querySelector() { return null; }
    querySelectorAll() { return []; }
  }

  const thread = new Container(true);
  // Existing nodes are deliberately not reconstructed: only their persisted
  // document height and scroll position matter before the session refresh.
  thread.children = [{ dataset: { domKey: "old" } }];
  const live = [{ role: "user", text: "Question" }];
  const incoming = snapshotMessages !== undefined ? snapshotMessages : matching
    ? [{ role: "user", text: "Question" }]
    : [{ role: "user", text: "Question" }, { role: "assistant", text: "Answer" }];
  const context = {
    document: {
      scrollingElement: page,
      documentElement: page,
      createElement() { return new Container(); },
      createDocumentFragment() { return new Container(); },
    },
    window: {
      addEventListener(type, listener) { if (type === "scroll") scrollListener = listener; },
      scrollTo(x, y) {
        calls.push({ x, y });
        page.scrollTop = Math.max(0, Math.min(y, page.scrollHeight - page.clientHeight));
      },
    },
    requestAnimationFrame(callback) { frames.push(callback); },
    thread,
    messages: live,
    chapters: [],
    activeChapter,
    heldOpen: {},
    shownViewKey: ["", "ws", "main"].join("\u0001"),
    currentThreadID: "main",
    currentWorkspaceID: "ws",
    currentAgentInstanceID: "",
    msg: { value: "" },
    fitComposer() {},
    chatDrafts: {},
    draftChatKey: "",
    runningThreadID: "main",
    workspaceThreadID: "",
    stoppingThreadID: "",
    stoppingAgents: {},
    backgroundBusy: false,
    sideClosed: false,
    callLive: false,
    voiceOK: false,
    realtimeReady: false,
    needsProvider: false,
    agentState: "running",
    applyingFollow: false,
    modelEl: {},
    modeEl: {},
    voiceBtn: {},
    workspaceLabel: null,
    workspaceRootEl: null,
    openMapFrom() { return {}; },
    appendMessageNode(parent, message) {
      parent.appendChild({ dataset: { domKey: message.role }, message });
      return message.role;
    },
    appendMessageList(parent, list) {
      list.forEach((message) => context.appendMessageNode(parent, message));
    },
    inChapter(message) {
      return !message.event || (message.event >= activeChapter.start && message.event <= activeChapter.end);
    },
    messagesOutsideChapters() {
      return context.messages.filter((message) => !message.event || !context.chapters.some((chapter) =>
        message.event >= chapter.start && message.event <= chapter.end));
    },
    renderChapterOverview(chapter) {
      const node = new Container();
      node.chapter = chapter;
      return node;
    },
    esc(value) { return String(value); },
    applyProviderGate() {},
    applyModelChrome() {},
    renderSideChatRef() {},
    renderBudgetLine() {},
    applyLowContext() {},
    renderChapters() {},
    setState() {},
    clearLive() {},
  };
  vm.runInNewContext(sessionCode, context);
  // Initialize follow state from the position before the idle refresh.
  scrollListener();
  return {
    page,
    calls,
    thread,
    apply(threadID = "main") {
      context.applySession({ workspace_id: "ws", thread_id: threadID, messages: incoming, chapters: snapshotChapters });
    },
    userScroll(top) {
      page.scrollTop = top;
      scrollListener();
    },
    frame() {
      if (pendingScroll) {
        pendingScroll = false;
        scrollListener();
      }
      while (frames.length) frames.shift()();
    },
  };
}

test("an idle session rebuild keeps a follower at the bottom", () => {
  const session = makeSession();
  session.apply();
  session.frame();
  assert.equal(session.page.scrollTop, 9400);
  assert.deepEqual(session.thread.children.map((row) => row.message.text), ["Question", "Answer"]);
});

test("an idle session rebuild preserves the position of a reader higher up", () => {
  const session = makeSession({ position: 9000 });
  session.apply();
  session.frame();
  assert.equal(session.page.scrollTop, 9000);
  assert.deepEqual(session.calls, []);
});

test("a matching idle session keeps the existing transcript and reader position", () => {
  const session = makeSession({ position: 9000, matching: true });
  const oldRow = session.thread.children[0];
  session.apply();
  session.frame();
  assert.equal(session.thread.children[0], oldRow);
  assert.equal(session.page.scrollTop, 9000);
  assert.deepEqual(session.calls, []);
});

test("opening another thread still forces the page to its bottom", () => {
  const session = makeSession({ position: 9000 });
  session.apply("other");
  session.frame();
  assert.equal(session.page.scrollTop, 9400);
});

test("a follower reaches the new bottom when the snapshot grows", () => {
  const session = makeSession({ rowHeight: 5200 });
  session.apply();
  session.frame();
  assert.equal(session.page.scrollTop, 10400);
});

test("a reader can scroll away after the idle rebuild but before follow runs", () => {
  const session = makeSession();
  session.apply();
  session.userScroll(9000);
  session.frame();
  assert.equal(session.page.scrollTop, 9000);
  assert.deepEqual(session.calls, []);
});

test("an empty snapshot renders the empty-session hint", () => {
  const session = makeSession({ snapshotMessages: [] });
  session.apply();
  session.frame();
  assert.equal(session.thread.children.length, 1);
  assert.match(session.thread.children[0].innerHTML, /Send a message to continue this session/);
});

test("a rebuilt chapter view retains its heading and filters messages", () => {
  const chapter = { id: "one", title: "Chapter one", start: 1, end: 1 };
  const session = makeSession({
    activeChapter: chapter,
    snapshotChapters: [chapter],
    snapshotMessages: [
      { role: "user", text: "In chapter", event: 1 },
      { role: "assistant", text: "Outside chapter", event: 2 },
    ],
  });
  session.apply();
  session.frame();
  assert.match(session.thread.children[0].innerHTML, /Chapter one/);
  assert.deepEqual(session.thread.children.slice(1).map((row) => row.message.text), ["In chapter"]);
});

test("a rebuilt chapter overview contains its chapter and live tail", () => {
  const chapter = { id: "one", title: "Chapter one", start: 1, end: 1 };
  const session = makeSession({
    snapshotChapters: [chapter],
    snapshotMessages: [
      { role: "user", text: "In chapter", event: 1 },
      { role: "assistant", text: "Live tail", event: 2 },
    ],
  });
  session.apply();
  session.frame();
  const [block, tail] = session.thread.children;
  assert.equal(block.className, "chapter-block");
  assert.equal(block.children[0].chapter, chapter);
  assert.equal(tail.className, "thread-tail");
  assert.deepEqual(tail.children.map((row) => row.message.text), ["Live tail"]);
});

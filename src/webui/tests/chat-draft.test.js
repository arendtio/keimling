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
const draftCode = section("  // A draft belongs to the chat it was typed in.", "  function switchThread(");
const switchCode = section("  function switchThread(", "  function deleteThread(");

// The composer holds one draft at a time. It belongs to the open chat: the
// agent instance when an agent chat is open, otherwise the workspace thread.
function makeComposer({ agent = "", thread = "main", value = "" } = {}) {
  const calls = { fitted: 0 };
  const context = {
    msg: { value },
    fitComposer: () => { calls.fitted += 1; },
    currentAgentInstanceID: agent,
    currentThreadID: thread,
    chatDrafts: {},
    draftChatKey: "",
  };
  vm.runInNewContext(draftCode, context);
  return {
    context,
    calls,
    get value() { return context.msg.value; },
    type(text) { context.msg.value = text; context.saveChatDraft(); },
    open(agentID, threadID) {
      context.currentAgentInstanceID = agentID;
      context.currentThreadID = threadID;
      context.syncChatDraft();
    },
    send() { context.msg.value = ""; context.forgetChatDraft(); },
  };
}

function makeSwitch({ agent = "", thread = "main", value = "" } = {}) {
  const calls = { posts: [], sessions: 0 };
  const context = {
    msg: { value },
    fitComposer: () => {},
    currentAgentInstanceID: agent,
    currentThreadID: thread,
    workspaceThreadID: "",
    chatDrafts: {},
    draftChatKey: "",
    stateEl: { textContent: "" },
    es: null,
    apiURL: p => p,
    headers: () => ({}),
    fetch: (url, opts) => {
      calls.posts.push({ url, body: JSON.parse(opts.body) });
      return Promise.resolve({ ok: true, text: () => Promise.resolve("") });
    },
    clearLive: () => {},
    hideApproval: () => {},
    loadSession: () => { calls.sessions += 1; return Promise.resolve(); },
    connectEvents: () => {},
    loadThreads: () => Promise.resolve(),
    loadAgents: () => Promise.resolve(),
    refreshSidebar: () => Promise.resolve(),
    appendLive: () => {},
  };
  vm.runInNewContext(draftCode + "\n" + switchCode, context);
  return { context, calls };
}

const flush = () => new Promise(resolve => setImmediate(resolve));

test("a draft typed in an agent chat stays out of the workspace thread", () => {
  const chat = makeComposer({ agent: "agent-1", thread: "t1" });
  chat.open("agent-1", "t1");
  chat.type("half a thought");
  chat.open("", "t2");
  assert.equal(chat.value, "");
  chat.open("agent-1", "t1");
  assert.equal(chat.value, "half a thought");
});

test("a workspace draft does not follow into an agent chat", () => {
  const chat = makeComposer({ agent: "", thread: "t1" });
  chat.open("", "t1");
  chat.type("for the thread");
  chat.open("agent-1", "t1");
  assert.equal(chat.value, "");
  chat.open("", "t1");
  assert.equal(chat.value, "for the thread");
});

test("an agent instance and a thread with the same id keep separate drafts", () => {
  const chat = makeComposer({ agent: "same", thread: "same" });
  chat.open("same", "same");
  chat.type("agent text");
  chat.open("", "same");
  assert.equal(chat.value, "");
  chat.type("thread text");
  chat.open("same", "same");
  assert.equal(chat.value, "agent text");
  chat.open("", "same");
  assert.equal(chat.value, "thread text");
});

test("the instance id names the chat while an agent chat is open", () => {
  const chat = makeComposer({ agent: "agent-1", thread: "t1" });
  assert.equal(chat.context.chatDraftKey(), "agent:agent-1");
  chat.context.currentAgentInstanceID = "";
  assert.equal(chat.context.chatDraftKey(), "thread:t1");
  chat.context.currentThreadID = "";
  assert.equal(chat.context.chatDraftKey(), "");
});

test("a session refresh of the same chat keeps what is being typed", () => {
  const chat = makeComposer({ agent: "agent-1", thread: "t1" });
  chat.open("agent-1", "t1");
  chat.context.msg.value = "still typing";
  chat.open("agent-1", "t1");
  assert.equal(chat.value, "still typing");
  assert.equal(chat.calls.fitted, 0);
});

test("a sent message is not restored as a draft", () => {
  const chat = makeComposer({ agent: "agent-1", thread: "t1" });
  chat.open("agent-1", "t1");
  chat.type("send me");
  chat.send();
  chat.open("", "t2");
  chat.open("agent-1", "t1");
  assert.equal(chat.value, "");
});

test("switching chats restores the draft and refits the composer", () => {
  const chat = makeComposer({ agent: "", thread: "t1" });
  chat.open("", "t1");
  chat.type("one");
  chat.open("", "t2");
  assert.equal(chat.calls.fitted, 1);
  chat.type("two");
  chat.open("", "t1");
  assert.equal(chat.value, "one");
  assert.equal(chat.calls.fitted, 2);
});

test("the first known chat does not clear the composer", () => {
  const chat = makeComposer({ agent: "", thread: "", value: "early" });
  chat.open("agent-1", "t1");
  assert.equal(chat.value, "early");
});

test("switching a thread parks the draft and restores the other thread one", async () => {
  const chat = makeSwitch({ agent: "", thread: "t1", value: "draft one" });
  chat.context.syncChatDraft();
  chat.context.switchThread("t2");
  await flush();
  assert.equal(chat.context.currentThreadID, "t2");
  assert.equal(chat.context.msg.value, "");
  chat.context.msg.value = "draft two";
  chat.context.switchThread("t1");
  await flush();
  assert.equal(chat.context.msg.value, "draft one");
  assert.equal(chat.calls.sessions, 2);
});

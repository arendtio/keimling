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
const branchCode = [
  section("  function attachBranchControls(", "  function openBranchEditor("),
  section("  function openBranchEditor(", "  function sendEditedMessage("),
  section("  function sendEditedMessage(", "  function selectContinuation("),
].join("\n");

function makeEl(tag) {
  return {
    tagName: tag.toUpperCase(),
    children: [],
    attrs: {},
    listeners: {},
    bySel: {},
    className: "",
    hidden: false,
    disabled: false,
    textContent: "",
    value: "",
    appendChild(child) {
      child.parent = this;
      this.children.push(child);
      return child;
    },
    setAttribute(name, value) { this.attrs[name] = value; },
    addEventListener(name, fn) { (this.listeners[name] = this.listeners[name] || []).push(fn); },
    click() { (this.listeners.click || []).forEach(fn => fn({ preventDefault() {} })); },
    querySelector(sel) { return this.bySel[sel] || null; },
    remove() {
      if (this.parent) this.parent.children = this.parent.children.filter(c => c !== this);
    },
    focus() {},
    setSelectionRange() {},
  };
}

function makeChat({ instance = "", state = "idle" } = {}) {
  const textEl = makeEl("span");
  const bodyEl = makeEl("div");
  bodyEl.bySel[".msg-text"] = textEl;
  const el = makeEl("div");
  el.bySel[".body"] = bodyEl;
  const calls = { post: [], cleared: 0, states: [], appended: [] };
  let confirmAnswer = null;
  const context = {
    document: { createElement: tag => makeEl(tag) },
    currentAgentInstanceID: instance,
    agentState: state,
    agentBusy: () => state === "running" || state === "compacting" || state === "awaiting_approval",
    appendLive: text => { calls.appended.push(text); },
    postJSON: (url, payload) => {
      calls.post.push({ url, payload });
      return Promise.resolve({ accepted: true });
    },
    clearLive: () => { calls.cleared += 1; },
    setState: s => { calls.states.push(s); },
    loadSession: () => Promise.resolve(),
    scrollTranscriptToEnd: () => {},
    refreshSideContext: () => {},
    resetSideSamples: () => {},
    askConfirm: () => new Promise(resolve => { confirmAnswer = resolve; }),
  };
  vm.runInNewContext(branchCode, context);
  return {
    el, bodyEl, calls, context,
    answer(value) { assert.ok(confirmAnswer, "no confirmation was asked"); confirmAnswer(value); },
    asked() { return confirmAnswer !== null; },
    flush: () => new Promise(resolve => setImmediate(resolve)),
    editButton() {
      const nav = this.bodyEl.children.find(c => c.className === "branch-nav");
      return nav.children.find(c => c.textContent === "Edit");
    },
    sendButton() {
      const box = this.bodyEl.children.find(c => c.className === "branch-edit");
      const row = box.children.find(c => c.className === "row");
      return row.children.find(c => c.textContent === "Send");
    },
    editor() {
      return this.bodyEl.children.find(c => c.className === "branch-edit");
    },
  };
}

test("an agent chat keeps the edit button usable while the agent runs", () => {
  const chat = makeChat({ instance: "agent-1", state: "running" });
  chat.context.attachBranchControls(chat.el, { text: "old text", line: 7 });
  assert.equal(chat.editButton().disabled, false);
  chat.editButton().click();
  assert.ok(chat.editor(), "the editor opens while the agent runs");
});

test("a workspace thread keeps the edit button disabled while the agent runs", () => {
  const chat = makeChat({ instance: "", state: "running" });
  chat.context.attachBranchControls(chat.el, { text: "old text", line: 7 });
  assert.equal(chat.editButton().disabled, true);
  chat.editButton().click();
  assert.equal(chat.editor(), undefined);
  assert.match(chat.calls.appended.join(""), /still working/);
});

test("sending in a running agent chat asks before stopping the run", async () => {
  const chat = makeChat({ instance: "agent-1", state: "running" });
  chat.context.attachBranchControls(chat.el, { text: "old text", line: 7 });
  chat.editButton().click();
  chat.sendButton().click();
  assert.ok(chat.asked(), "a confirmation is required before stopping the run");
  assert.equal(chat.calls.post.length, 0);
  chat.answer(false);
  await chat.flush();
  assert.equal(chat.calls.post.length, 0, "cancelling sends nothing");
  assert.ok(chat.editor(), "the editor stays open after cancelling");
  chat.sendButton().click();
  chat.answer(true);
  await chat.flush();
  assert.equal(chat.calls.post.length, 1);
  assert.equal(chat.calls.post[0].url, "/api/branch");
  assert.equal(chat.calls.post[0].payload.force, true);
});

test("an idle agent chat sends without a confirmation", async () => {
  const chat = makeChat({ instance: "agent-1", state: "idle" });
  chat.context.attachBranchControls(chat.el, { text: "old text", line: 7 });
  chat.editButton().click();
  chat.sendButton().click();
  await chat.flush();
  assert.equal(chat.asked(), false);
  assert.equal(chat.calls.post.length, 1);
  assert.equal(chat.calls.post[0].payload.force, true);
});

test("sending an edited message reports the fork and reloads the chat", async () => {
  const chat = makeChat({ instance: "agent-1", state: "running" });
  await chat.context.sendEditedMessage(7, "new text", true);
  // Copy into this realm so strict deep equality also compares prototypes.
  const sent = JSON.parse(JSON.stringify(chat.calls.post[0]));
  assert.deepEqual(sent, { url: "/api/branch", payload: { line: 7, text: "new text", force: true } });
  assert.equal(chat.calls.cleared, 1);
  assert.deepEqual(chat.calls.states, ["running"]);
});

test("a workspace thread sends without force and still waits while busy", async () => {
  const chat = makeChat({ instance: "", state: "running" });
  await assert.rejects(() => chat.context.sendEditedMessage(7, "new text"));
  assert.equal(chat.calls.post.length, 0);

  const idle = makeChat({ instance: "", state: "idle" });
  await idle.context.sendEditedMessage(7, "new text");
  assert.equal(idle.calls.post[0].payload.force, false);
});

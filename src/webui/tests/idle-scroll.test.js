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
const code = [
  section("  // Within this many pixels of the bottom", "  function chapterBlockSig()"),
  section("  function connectEvents()", "  // Remote attach navigates the WebView"),
].join("\n");

// The actual SSE idle handler is exercised with a controlled session response.
// A WebView-like layout reset dispatches a scroll event without user input.
function makeIdle({ position = 9400, resetOnIdle = false } = {}) {
  const page = { scrollHeight: 10000, clientHeight: 600, scrollTop: position };
  const listeners = {};
  const frames = [];
  const calls = [];
  let resolveSession;
  const session = new Promise(resolve => { resolveSession = resolve; });
  let firstRead = true;
  const encoder = new TextEncoder();
  const context = {
    Promise, TextDecoder, AbortController,
    document: { scrollingElement: page, documentElement: page },
    window: {
      addEventListener(type, listener) { (listeners[type] ||= []).push(listener); },
      scrollTo(x, y) { calls.push({ x, y }); page.scrollTop = Math.max(0, Math.min(y, page.scrollHeight - page.clientHeight)); },
    },
    requestAnimationFrame(callback) { frames.push(callback); },
    es: null, uiFlash: null, currentThreadID: "main", currentWorkspaceID: "ws", currentAgentInstanceID: "",
    shownViewKey: "ws/main", agentState: "running", runningThreadID: "main", stoppingThreadID: "",
    stoppingAgents: {}, backgroundBusy: false, sessionEpoch: 0,
    apiURL: p => p, headers: () => ({}),
    fetch(url) {
      assert.equal(url, "/events");
      return Promise.resolve({ ok: true, body: { getReader: () => ({ read() {
        if (!firstRead) return new Promise(() => {});
        firstRead = false;
        return Promise.resolve({ done: false, value: encoder.encode('event: status\ndata: {"state":"idle"}\n\n') });
      } }) } });
    },
    setState(state) {
      context.agentState = state;
      if (resetOnIdle && state === "idle") {
        page.scrollTop = 0;
        fire("scroll");
      }
    },
    loadSession() { context.sessionEpoch++; return session; },
    clearLive() {}, refreshBudget() { return Promise.resolve(); }, refreshSideContext() {},
    loadThreads() {}, resetSideSamples() {}, appendLive() {},
  };
  function fire(type, event = {}) { (listeners[type] || []).forEach(fn => fn(event)); }
  vm.runInNewContext(code, context);
  fire("scroll");
  context.connectEvents();
  return {
    page, calls, context,
    flush: () => new Promise(resolve => setImmediate(resolve)),
    frame() { while (frames.length) frames.shift()(); },
    layoutReset() { page.scrollTop = 0; fire("scroll"); },
    input(type, event) { fire(type, event); },
    userScroll(top) { page.scrollTop = top; fire("scroll"); },
    finishSession() { resolveSession(); },
  };
}

test("a follower recovers from a top reset during idle and a matching session refresh", async () => {
  const chat = makeIdle({ resetOnIdle: true });
  await chat.flush();
  chat.frame();
  assert.equal(chat.page.scrollTop, 9400, "idle UI changes must not leave a follower at the top");
  chat.layoutReset();
  chat.finishSession();
  await chat.flush();
  chat.frame();
  assert.equal(chat.page.scrollTop, 9400, "even a matching snapshot must restore the bottom");
});

test("a reader who was away from the bottom is not forced down when idle", async () => {
  const chat = makeIdle({ position: 8000 });
  await chat.flush();
  chat.finishSession();
  await chat.flush();
  chat.frame();
  assert.equal(chat.page.scrollTop, 8000);
  assert.deepEqual(chat.calls, []);
});

test("a reader higher up is restored after an idle layout reset to the top", async () => {
  const chat = makeIdle({ position: 8000, resetOnIdle: true });
  await chat.flush();
  chat.frame();
  assert.equal(chat.page.scrollTop, 8000);
  chat.layoutReset();
  chat.finishSession();
  await chat.flush();
  chat.frame();
  assert.equal(chat.page.scrollTop, 8000);
});

test("intentional scrolling during the asynchronous refresh cancels the idle follow", async () => {
  const chat = makeIdle();
  await chat.flush();
  chat.frame();
  chat.input("wheel", { deltaY: -200 });
  chat.userScroll(8000);
  chat.finishSession();
  await chat.flush();
  chat.frame();
  assert.equal(chat.page.scrollTop, 8000);
});

test("scrolling to the top before the scheduled idle frame is not undone", async () => {
  const chat = makeIdle();
  await chat.flush();
  chat.input("keydown", { key: "Home" });
  chat.userScroll(0);
  chat.frame();
  chat.finishSession();
  await chat.flush();
  chat.frame();
  assert.equal(chat.page.scrollTop, 0);
});

test("an assistive scroll away from the bottom is respected even without an input event", async () => {
  const chat = makeIdle();
  await chat.flush();
  chat.frame();
  chat.userScroll(8000);
  chat.finishSession();
  await chat.flush();
  chat.frame();
  assert.equal(chat.page.scrollTop, 8000);
});

test("typing a non-scroll key does not cancel recovery from a layout reset", async () => {
  const chat = makeIdle();
  await chat.flush();
  chat.frame();
  chat.input("keydown", { key: "a" });
  chat.layoutReset();
  chat.finishSession();
  await chat.flush();
  chat.frame();
  assert.equal(chat.page.scrollTop, 9400);
});

test("an old idle refresh cannot scroll a newly opened thread", async () => {
  const chat = makeIdle();
  await chat.flush();
  chat.context.shownViewKey = "ws/other";
  chat.context.currentThreadID = "other";
  chat.context.sessionEpoch++;
  chat.userScroll(2000);
  chat.finishSession();
  await chat.flush();
  chat.frame();
  assert.equal(chat.page.scrollTop, 2000);
});

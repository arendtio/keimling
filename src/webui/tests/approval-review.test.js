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
const code = section("  function pendingToolCard(", "  function usdPerMillionFromToken(");

function harness() {
  let now = 5000;
  const sandbox = {
    messages: [],
    currentThreadID: "th-1",
    eventsForThisView: () => true,
    finishLiveThinking: () => {},
    paintTranscript: () => {},
    refreshSideContext: () => {},
    Date: { now: () => now },
  };
  vm.runInNewContext(code, sandbox);
  return {
    messages: sandbox.messages,
    at(ms) { now = ms; },
    send(payload) { sandbox.handleStructuredTool(payload); },
  };
}

test("a reviewed call shows review state before it runs", () => {
  const h = harness();
  h.send({ phase: "start", name: "exec", command: "rm -rf build", input: "{}" });
  assert.equal(h.messages.length, 1);
  assert.equal(h.messages[0].pending, true);
  assert.ok(!h.messages[0].reviewing);

  h.send({ phase: "review", name: "exec", command: "rm -rf build" });
  assert.equal(h.messages[0].reviewing, true);
  assert.equal(h.messages[0].reviewMs, 0);

  h.at(8000);
  h.send({ phase: "go", name: "exec", command: "rm -rf build" });
  assert.equal(h.messages[0].reviewing, false);
  assert.equal(h.messages[0].pending, true);
  assert.equal(h.messages[0].reviewMs, 3000);
});

test("the review time survives into the finished card", () => {
  const h = harness();
  h.send({ phase: "start", name: "exec", command: "ls" });
  h.send({ phase: "review", name: "exec", command: "ls" });
  h.at(6500);
  h.send({ phase: "go", name: "exec", command: "ls" });
  h.at(7000);
  h.send({ phase: "result", name: "exec", command: "ls", result: "a\nb", ok: true });
  assert.equal(h.messages.length, 1);
  assert.equal(h.messages[0].pending, false);
  assert.equal(h.messages[0].status, "ran");
  assert.equal(h.messages[0].reviewMs, 1500);
});

test("a refused call keeps the review time and fails", () => {
  const h = harness();
  h.send({ phase: "start", name: "exec", command: "curl x" });
  h.send({ phase: "review", name: "exec", command: "curl x" });
  h.at(9000);
  h.send({ phase: "result", name: "exec", command: "curl x", result: "DENIED: no", ok: false });
  assert.equal(h.messages[0].status, "failed");
  assert.equal(h.messages[0].reviewMs, 4000);
});

test("a call without review carries no review time", () => {
  const h = harness();
  h.send({ phase: "start", name: "read_file", command: "read_file a" });
  h.send({ phase: "go", name: "read_file", command: "read_file a" });
  h.send({ phase: "result", name: "read_file", command: "read_file a", result: "x", ok: true });
  assert.equal(h.messages[0].reviewMs, 0);
});

test("a call that asks the human says it waits instead of running", () => {
  const h = harness();
  h.send({ phase: "start", name: "exec", command: "git push" });
  h.send({ phase: "review", name: "exec", command: "git push" });
  h.at(7000);
  h.send({ phase: "ask", name: "exec", command: "git push" });
  assert.equal(h.messages[0].reviewing, false);
  assert.equal(h.messages[0].waiting, true);
  assert.equal(h.messages[0].pending, true);
  assert.equal(h.messages[0].reviewMs, 2000);

  h.at(9000);
  h.send({ phase: "go", name: "exec", command: "git push" });
  assert.equal(h.messages[0].waiting, false);
  assert.equal(h.messages[0].reviewMs, 2000);
});

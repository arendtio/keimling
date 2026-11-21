"use strict";

// A run ends with its Thinking rows and tool cards closed, while the chat lines
// and the user's own opened rows keep their state.

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
const foldCode = section(
  "  // A run ends with its Thinking rows and tool cards closed.",
  "  function setState(s) {",
);
const stateCode =
  section("  function setState(s) {", "    stateEl.textContent = agentState;") + "\n}";

function transcript() {
  const nodes = [];
  const folds = [];
  const byKey = {};
  function node(key) {
    const n = { dataset: { domKey: key } };
    byKey[key] = n;
    nodes.push(n);
    return n;
  }
  function fold(classes, ownerKey, { open = false, dataset = {} } = {}) {
    const owner = byKey[ownerKey] || node(ownerKey);
    const d = {
      open,
      dataset: Object.assign({}, dataset),
      classList: { contains: name => classes.indexOf(name) >= 0 },
      closest: () => owner,
    };
    folds.push(d);
    return d;
  }
  return {
    node, fold,
    thread: {
      querySelectorAll(selector) {
        return selector === "details" ? folds.slice() : nodes.slice();
      },
    },
  };
}

function load(doc, { heldOpen = null, agentState = "idle" } = {}) {
  const context = { thread: doc.thread, heldOpen, runFoldKeys: null, agentState };
  vm.runInNewContext(foldCode, context);
  return context;
}

test("idle closes the run's Thinking row and tool cards, not the user's own opens", () => {
  const doc = transcript();
  const older = doc.fold(["tool-line"], "m1", { open: true });
  const olderOutput = doc.fold(["tool-output-fold"], "m1", { open: true });
  const heldOpen = { m1: true, m2: true, "m2:output": true };
  const ui = load(doc, { heldOpen });

  ui.beginRunFolds();
  const card = doc.fold(["tool-line"], "m2", { open: true, dataset: { domKey: "m2" } });
  const liveOutput = doc.fold(["tool-output-fold"], "m2", { open: true, dataset: { domKey: "m2:output" } });
  const thought = doc.fold(["thinking-fold"], "m3", { open: true, dataset: { domKey: "m3:think", autoFollow: "1" } });
  const chapter = doc.fold(["chapter-fold"], "m4", { open: true });
  const notice = doc.fold(["notice"], "m5", { open: true });
  doc.node("m6");

  ui.collapseRunFolds();

  assert.equal(card.open, false, "a tool card of the finished run closes");
  assert.equal(liveOutput.open, false, "the full output inside that card closes");
  assert.equal(thought.open, false, "a Thinking row of the finished run closes");
  assert.equal(thought.dataset.autoFollow, undefined, "the follow marker goes with the close");
  assert.equal(older.open, true, "a row opened before the run stays open");
  assert.equal(olderOutput.open, true, "its inner output stays open too");
  assert.equal(chapter.open, true, "a chapter fold is not a run row");
  assert.equal(notice.open, true, "a system notice keeps the state the user gave it");
  assert.equal(heldOpen.m2, undefined, "a closed row is not restored open by the next paint");
  assert.equal(heldOpen["m2:output"], undefined);
  assert.equal(heldOpen.m1, true, "an open row before the run is still restored");
});

test("rows that are already closed stay closed", () => {
  const doc = transcript();
  const ui = load(doc);
  ui.beginRunFolds();
  const card = doc.fold(["tool-line"], "m2");
  const thought = doc.fold(["thinking-fold"], "m3");

  ui.collapseRunFolds();

  assert.equal(card.open, false);
  assert.equal(thought.open, false);
  assert.equal(ui.runFoldKeys, null, "the run baseline is spent after the close");
});

test("without a baseline the open run rows still close", () => {
  const doc = transcript();
  const ui = load(doc);
  const card = doc.fold(["tool-line"], "m2", { open: true, dataset: { domKey: "m2" } });

  ui.collapseRunFolds();

  assert.equal(card.open, false);
});

test("setState closes once when the run ends and captures the rows when it starts", () => {
  const calls = [];
  const context = {
    agentState: "idle",
    beginRunFolds() { calls.push("begin"); },
    collapseRunFolds() { calls.push("collapse"); },
    closeSideRun() { calls.push("side"); },
  };
  vm.runInNewContext(stateCode, context);

  context.setState("running");
  context.setState("running");
  assert.deepEqual(calls, ["begin"], "the run baseline is captured once");

  context.setState("idle");
  assert.deepEqual(calls, ["begin", "side", "collapse"]);

  context.setState("idle");
  assert.deepEqual(calls, ["begin", "side", "collapse"], "a repeated idle event closes nothing again");

  // A later turn starts a new baseline, so its own rows close at its own end.
  context.setState("awaiting_approval");
  context.setState("idle");
  assert.deepEqual(calls, ["begin", "side", "collapse", "begin", "side", "collapse"]);
});

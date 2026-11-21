"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

const app = fs.readFileSync(path.join(__dirname, "../assets/app.js"), "utf8");
const html = fs.readFileSync(path.join(__dirname, "../index.html"), "utf8");
function section(start, end) {
  const a = app.indexOf(start);
  const b = app.indexOf(end, a);
  assert.ok(a >= 0 && b > a, `missing app section: ${start}`);
  return app.slice(a, b);
}
const code = [
  section("  function budgetLabelText(", "  function renderContextDialog("),
  section("  function agentPhaseLabel(", "  var agentBoardDrawn"),
].join("\n");

class El {
  constructor(tag) {
    this.tag = tag;
    this.children = [];
    this.style = {};
    this.className = "";
    this.dataset = {};
    this.hidden = false;
    this.textContent = "";
    this.html = "";
    const classes = new Set();
    this.classes = classes;
    this.classList = {
      add: (c) => classes.add(c),
      remove: (c) => classes.delete(c),
      contains: (c) => classes.has(c),
      toggle: (c, on) => { if (on === undefined ? !classes.has(c) : on) classes.add(c); else classes.delete(c); },
    };
  }
  set innerHTML(value) { this.html = value; this.children = []; }
  get innerHTML() { return this.html; }
  get childElementCount() { return this.children.length; }
  appendChild(child) { this.children.push(child); return child; }
  addEventListener(type, fn) { (this.listeners || (this.listeners = {}))[type] = fn; }
  click() { if (this.listeners && this.listeners.click) this.listeners.click(); }
}

function makeNow({ sideTab = "now" } = {}) {
  const nodes = new Map();
  const lookups = [];
  const attached = [];
  const sandbox = {
    sideTab: sideTab,
    attachAgent(id) { attached.push(id); },
    document: {
      getElementById(id) {
        lookups.push(id);
        return nodes.get(id) || null;
      },
      createElement(tag) { return new El(tag); },
    },
  };
  const byID = (id) => {
    if (!nodes.has(id)) nodes.set(id, new El("div"));
    return nodes.get(id);
  };
  ["budget-label", "budget-fill", "now-budget-label", "now-budget-fill", "now-agents", "now-agents-empty"]
    .forEach(byID);
  vm.createContext(sandbox);
  vm.runInContext(code, sandbox);
  return {
    sandbox,
    nodes,
    lookups,
    attached,
    renderBudget(budget) {
      lookups.length = 0;
      sandbox.__budget = budget;
      vm.runInContext("renderNowBudget(__budget)", sandbox);
      return { label: nodes.get("now-budget-label").textContent, fill: nodes.get("now-budget-fill") };
    },
    renderAgents(list) {
      sandbox.__list = list;
      vm.runInContext("renderNowAgents(__list)", sandbox);
      return nodes.get("now-agents");
    },
    setTab(tab) { sandbox.sideTab = tab; },
  };
}

test("the Now budget repeats the Context tab wording from the same payload", () => {
  const now = makeNow();
  const { label, fill } = now.renderBudget({
    used: 12345, limit: 100000, used_pct: 12.345, remaining: 87655, source: "measured", warn: false,
  });
  assert.equal(label, "used 12345 / 100000 (12.3%) · remaining 87655 · measured");
  assert.equal(fill.style.width, "12.345%");
  assert.equal(fill.classes.has("warn"), false);
  assert.deepEqual(now.lookups, ["now-budget-label", "now-budget-fill"]);
});

test("the Now budget shows the warn fill when the limit is passed", () => {
  const now = makeNow();
  const { label, fill } = now.renderBudget({ used: 120000, limit: 100000, used_pct: 120, remaining: -20000, warn: true });
  assert.equal(label, "used 120000 / 100000 (120.0%) · remaining -20000 · estimate");
  assert.equal(fill.style.width, "100%");
  assert.equal(fill.classes.has("warn"), true);
});

test("the Now budget clamps a negative fill to zero width", () => {
  const now = makeNow();
  const { fill } = now.renderBudget({ used: 10, limit: 100, used_pct: -5 });
  assert.equal(fill.style.width, "0%");
});

test("the Now tab lists running and waiting agents and hides idle ones", () => {
  const now = makeNow();
  const list = now.renderAgents([
    { id: "idle", name: "Ada", recipe_name: "Analyst", phase: "idle" },
    { id: "run", name: "Bo", recipe_name: "Builder", phase: "running" },
    { id: "call", name: "Cy", recipe_id: "ops", phase: "idle", call: true },
    { id: "wait", name: "Di", phase: "waiting", waiting: "watch" },
    { id: "paused", name: "El", phase: "paused" },
  ]);
  assert.deepEqual(list.children.map((row) => row.children[1].textContent),
    ["Running", "On a call", "Waiting for a watch"]);
  assert.deepEqual(list.children.map((row) => row.children[0].children[0].textContent), ["Bo", "Cy", "Di"]);
  assert.deepEqual(list.children.map((row) => row.children[0].children[1].textContent), ["Builder", "ops", "Definition"]);
  assert.equal(now.nodes.get("now-agents-empty").hidden, true);
});

test("an empty Now list keeps the quiet hint", () => {
  const now = makeNow();
  const list = now.renderAgents([{ id: "idle", name: "Ada", phase: "idle" }]);
  assert.equal(list.children.length, 0);
  assert.equal(now.nodes.get("now-agents-empty").hidden, false);
});

test("a Now row opens the same agent as the board", () => {
  const now = makeNow();
  const list = now.renderAgents([{ id: "run", name: "Bo", phase: "running" }]);
  list.children[0].children[0].click();
  assert.deepEqual(now.attached, ["run"]);
});

test("the Now agent list is left alone while another tab is shown", () => {
  const now = makeNow({ sideTab: "agents" });
  const list = now.renderAgents([{ id: "run", name: "Bo", phase: "running" }]);
  assert.equal(list.children.length, 0);
  assert.deepEqual(now.attached, []);
});

test("the Now blocks keep their own ids and follow the Performance block", () => {
  const ids = ["now-budget-label", "now-budget-fill", "now-agents", "now-agents-empty"];
  ids.forEach((id) => {
    const count = (html.match(new RegExp('id="' + id + '"', "g")) || []).length;
    assert.equal(count, 1, `expected one #${id}`);
  });
  assert.equal((html.match(/id="budget-fill"/g) || []).length, 1);
  const perf = html.indexOf('id="side-perf"');
  const budget = html.indexOf('id="now-budget-label"');
  const plan = html.indexOf('id="side-plan"');
  const agents = html.indexOf('id="now-agents"');
  assert.ok(perf > 0 && budget > perf && plan > budget && agents > plan);
});

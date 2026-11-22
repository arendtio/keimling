"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

const app = fs.readFileSync(path.join(__dirname, "../assets/app.js"), "utf8");
const html = fs.readFileSync(path.join(__dirname, "../index.html"), "utf8");

function codeOf(name) {
  const start = app.indexOf("function " + name + "(");
  assert.ok(start >= 0, name + " must exist in app.js");
  const end = app.indexOf("\n  }\n", start);
  assert.ok(end > start, name + " must end at the module indent");
  return app.slice(start, end + 4);
}

// select emulates a real <select>: a value that no option carries falls back to "".
function makeSelect() {
  const sel = {
    options: [],
    _value: "",
    appendChild(opt) {
      sel.options.push(opt);
    },
    get value() {
      return sel._value;
    },
    set value(v) {
      const wanted = String(v == null ? "" : v);
      sel._value = wanted === "" || sel.options.some(o => o.value === wanted) ? wanted : "";
    },
  };
  Object.defineProperty(sel, "innerHTML", {
    get() {
      return "";
    },
    set() {
      sel.options = [{ value: "", textContent: "Use the chat model" }];
      sel._value = "";
    },
  });
  return sel;
}

function runFill(snapshot, providers, select) {
  const context = {
    setAuxiliaryModel: select,
    providersCache: providers,
    document: {
      createElement(tag) {
        assert.equal(tag, "option");
        return { value: "", textContent: "" };
      },
    },
  };
  vm.createContext(context);
  vm.runInContext(codeOf("fillAuxiliaryModel"), context);
  context.fillAuxiliaryModel(snapshot);
  return select;
}

function runPayload(select) {
  const context = { setAuxiliaryModel: select };
  vm.createContext(context);
  vm.runInContext(codeOf("auxiliaryModelPayload"), context);
  return context.auxiliaryModelPayload();
}

const providers = [
  { id: "p1", name: "Provider one", models: [{ id: "aux-1" }, { id: "aux-2", name: "Second" }] },
  { id: "p2", name: "Provider two", models: [{ id: "aux-3" }] },
];

test("settings offer the auxiliary model with an empty chat-model choice", () => {
  assert.match(html, /id="set-auxiliary-model"/);
  assert.match(html, /<option value="">Use the chat model<\/option>/);
  assert.match(app, /body\.auxiliary_model = auxiliaryModelPayload\(\)/);
  assert.match(app, /fillAuxiliaryModel\(j\)/);
});

test("saved provider models fill the select and the saved value is chosen", () => {
  const select = runFill({ auxiliary_model: "aux-2" }, providers, makeSelect());
  assert.deepEqual(select.options.map(o => o.value), ["", "aux-1", "aux-2", "aux-3"]);
  assert.equal(select.value, "aux-2");
  assert.equal(runPayload(select), "aux-2");
});

test("an empty setting stores the chat model", () => {
  const select = runFill({}, providers, makeSelect());
  assert.equal(select.value, "");
  assert.equal(runPayload(select), "");
  assert.equal(runPayload(makeSelect()).trim(), "", "trimmed empty value means the chat model");
  select.value = "   ";
  assert.equal(runPayload(select), "");
});

test("a model that no provider offers falls back to the chat model", () => {
  const select = runFill({ auxiliary_model: "gone" }, providers, makeSelect());
  assert.equal(select.value, "");
  assert.equal(runPayload(select), "");
});

"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

// Exercise the same follow code used by the embedded UI without loading the app.
const app = fs.readFileSync(path.join(__dirname, "../assets/app.js"), "utf8");
const start = app.indexOf("  // Within this many pixels of the bottom");
const end = app.indexOf("  function chapterBlockSig()", start);
assert.ok(start >= 0 && end > start, "transcript follow code must exist");
const followCode = app.slice(start, end);

function makeTranscript(options = {}) {
  const root = { scrollHeight: 2000, clientHeight: 600, scrollTop: 1400 };
  const frames = [];
  const calls = [];
  let onScroll;
  const context = {
    document: {
      scrollingElement: options.noScrollingElement ? null : root,
      documentElement: root,
    },
    window: {
      addEventListener(type, listener) {
        assert.equal(type, "scroll");
        onScroll = listener;
      },
      scrollTo(x, y) {
        calls.push({ x, y });
        root.scrollTop = Math.max(0, Math.min(y, root.scrollHeight - root.clientHeight));
      },
    },
    requestAnimationFrame(callback) { frames.push(callback); },
  };
  vm.runInNewContext(followCode, context);
  return {
    root,
    calls,
    userScroll(top) { root.scrollTop = top; onScroll(); },
    grow(height) { root.scrollHeight = height; },
    follow() { context.followTranscript(); },
    force() { context.scrollTranscriptToEnd(); },
    frame() { while (frames.length) frames.shift()(); },
  };
}

test("following requests the actual bottom, not an out-of-range scroll position", () => {
  const chat = makeTranscript();
  chat.userScroll(1320); // 80 px from the bottom
  chat.grow(2400);
  chat.follow();
  chat.frame();
  assert.deepEqual(chat.calls, [{ x: 0, y: 1800 }]);
  assert.equal(chat.root.scrollTop, 1800);
});

test("following leaves readers more than 200 px from the bottom alone", () => {
  const chat = makeTranscript();
  chat.userScroll(1000);
  chat.grow(2400);
  chat.follow();
  chat.frame();
  assert.deepEqual(chat.calls, []);
  assert.equal(chat.root.scrollTop, 1000);
});

test("a forced scroll reaches the bottom even after reading higher up", () => {
  const chat = makeTranscript();
  chat.userScroll(1000);
  chat.grow(2400);
  chat.force();
  chat.frame();
  assert.deepEqual(chat.calls, [{ x: 0, y: 1800 }]);
});

test("a reader can scroll away after follow was scheduled", () => {
  const chat = makeTranscript();
  chat.follow();
  chat.userScroll(1000);
  chat.frame();
  assert.deepEqual(chat.calls, []);
});

test("a short document and the scrolling-element fallback never request a negative position", () => {
  const chat = makeTranscript({ noScrollingElement: true });
  chat.root.scrollHeight = 400;
  chat.root.clientHeight = 600;
  chat.force();
  chat.frame();
  assert.deepEqual(chat.calls, [{ x: 0, y: 0 }]);
});

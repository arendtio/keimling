"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

// Exercise the growth logo animation from the embedded UI without loading the app.
const app = fs.readFileSync(path.join(__dirname, "../assets/app.js"), "utf8");
const start = app.indexOf("  var growthLogoOn = false;");
const end = app.indexOf("  function growthSame(a, b) {", start);
assert.ok(start >= 0 && end > start, "growth logo code must exist");
const growthCode = app.slice(start, end);

function makeGrowth() {
  let now = 1000;
  let nextID = 1;
  const frames = new Map();
  const context = {
    document: { getElementById() { return null; } },
    performance: { now() { return now; } },
    requestAnimationFrame(callback) {
      const id = nextID++;
      frames.set(id, callback);
      return id;
    },
    cancelAnimationFrame(id) { frames.delete(id); },
  };
  vm.runInNewContext(growthCode, context);
  return {
    context,
    move(target, viaSeed) { context.moveGrowth(target, viaSeed); },
    pose() { return context.growthPose; },
    pending() { return frames.size; },
    runFrame(timestamp) {
      const queued = [...frames.values()];
      frames.clear();
      queued.forEach((callback) => callback(timestamp));
    },
  };
}

test("a first frame earlier than the scheduling time does not break the growth animation", () => {
  const growth = makeGrowth();
  growth.move({ rest: 0, p: 0.6 }, false);
  assert.equal(growth.pending(), 1);
  // Firefox can report a frame timestamp before performance.now() of the scheduling call.
  growth.runFrame(900);
  const pose = growth.pose();
  assert.ok(Number.isFinite(pose.rest) && Number.isFinite(pose.p), `pose ${JSON.stringify(pose)}`);
  assert.equal(pose.rest, 1);
  assert.equal(pose.p, 0);
  assert.equal(growth.pending(), 1);
});

test("the growth animation reaches its target pose", () => {
  const growth = makeGrowth();
  growth.move({ rest: 0, p: 0.6 }, false);
  for (let timestamp = 1000; timestamp <= 2000; timestamp += 20) growth.runFrame(timestamp);
  const pose = growth.pose();
  assert.equal(pose.rest, 0);
  assert.ok(Math.abs(pose.p - 0.6) < 0.01, `p ${pose.p}`);
  assert.equal(growth.pending(), 0);
});

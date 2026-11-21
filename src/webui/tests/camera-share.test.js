"use strict";

// The camera share sends small JPEG frames of its own stream. The browser keeps
// the frame small and prefers the back camera on a phone.

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
const cameraCode = section(
  "  // A camera frame is scaled down to a small edge and sent as a JPEG.",
  "  function setCameraStatus(",
);

function load() {
  const context = {};
  vm.runInNewContext(cameraCode, context);
  return context;
}

test("a camera frame is scaled down to the small edge and never up", () => {
  const ui = load();
  const wide = ui.cameraFrameFit(1920, 1080, 1280);
  assert.deepEqual({ width: wide.width, height: wide.height }, { width: 1280, height: 720 });
  const tall = ui.cameraFrameFit(720, 1440, 1280);
  assert.deepEqual({ width: tall.width, height: tall.height }, { width: 640, height: 1280 });
  const small = ui.cameraFrameFit(640, 480, 1280);
  assert.deepEqual({ width: small.width, height: small.height }, { width: 640, height: 480 });
  const none = ui.cameraFrameFit(0, 480, 1280);
  assert.deepEqual({ width: none.width, height: none.height }, { width: 0, height: 0 });
});

test("the camera stream asks for a facing mode when the device has one", () => {
  const ui = load();
  const back = ui.cameraVideoConstraints("environment");
  assert.equal(back.audio, false);
  assert.equal(back.video.width.ideal, 1280);
  assert.equal(back.video.facingMode.ideal, "environment");

  const front = ui.cameraVideoConstraints("user");
  assert.equal(front.video.facingMode.ideal, "user");

  const desktop = ui.cameraVideoConstraints("");
  assert.equal(desktop.audio, false);
  assert.equal(desktop.video.facingMode, undefined);
});

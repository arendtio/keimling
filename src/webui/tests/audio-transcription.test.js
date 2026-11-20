"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

const app = fs.readFileSync(path.join(__dirname, "../assets/app.js"), "utf8");
const html = fs.readFileSync(path.join(__dirname, "../index.html"), "utf8");
const gui = fs.readFileSync(path.join(__dirname, "../../gui_webview.go"), "utf8");
const start = app.indexOf("  function pickMime()");
const end = app.indexOf("  voiceBtn.addEventListener(\"click\"", start);
assert.ok(start >= 0 && end > start, "voice code must exist");
const voiceCode = app.slice(start, end);

function button() {
  return {
    disabled: false,
    handlers: {},
    classList: { add() {}, remove() {} },
    addEventListener(type, fn) { this.handlers[type] = fn; },
    click() { if (!this.disabled && this.handlers.click) this.handlers.click(); },
  };
}

function makeVoice() {
  const download = button();
  const upload = button();
  const input = button();
  input.value = "";
  input.files = [];
  const requests = [];
  const downloads = [];
  const statuses = [];
  const sent = [];
  let nextResponse = { ok: true, statusText: "OK", text: () => Promise.resolve('{"text":"transcript"}') };
  let recorder;
  let stopped = 0;
  class Recorder {
    static isTypeSupported(type) { return type === "audio/webm;codecs=opus"; }
    constructor() { this.state = "inactive"; recorder = this; }
    start() { this.state = "recording"; }
    stop() {
      this.state = "inactive";
      this.ondataavailable({ data: this.data || new Blob(["x".repeat(100)], { type: "audio/webm" }) });
      this.onstop();
    }
  }
  class FormDataMock {
    constructor() { this.fields = []; }
    append(...args) { this.fields.push(args); }
  }
  const context = {
    Blob, Promise, MediaRecorder: Recorder,
    window: { MediaRecorder: Recorder, URL: {
      createObjectURL(blob) { downloads.push({ blob }); return "blob:test"; },
      revokeObjectURL(url) { downloads[downloads.length - 1].revoked = url; },
    } },
    document: {
      body: { appendChild(link) { link.attached = true; } },
      createElement(tag) {
        assert.equal(tag, "a");
        return {
          attached: false,
          click() {
            assert.equal(this.attached, true, "download link must be attached when clicked");
            Object.assign(downloads[downloads.length - 1], { href: this.href, name: this.download });
          },
          remove() { this.attached = false; },
        };
      },
    },
    navigator: { mediaDevices: { getUserMedia: () => Promise.resolve({ getTracks: () => [{ stop() { stopped++; } }] }) } },
    FormData: FormDataMock,
    fetch(url, options) { requests.push({ url, options }); return Promise.resolve(nextResponse); },
    headers: () => ({ "Content-Type": "application/json", Authorization: "Bearer test" }),
    apiURL: p => p,
    setTimeout(fn) { fn(); },
    setVoiceStatus(s) { statuses.push(s); },
    fitComposer() {},
    setState() { context.updateAudioActions(); },
    sendMessage(s) { sent.push(s); },
    msg: { value: "", focus() {} },
    voiceBtn: button(), audioDownloadBtn: download, audioUploadBtn: upload, audioInput: input,
    connected: true, voiceOK: true, callLive: false, agentState: "idle",
    mediaRecorder: null, mediaChunks: [], mediaStream: null, recording: false,
    voiceSendPending: false, lastRecording: null,
  };
  vm.runInNewContext(voiceCode, context);
  context.updateAudioActions();
  const flush = () => new Promise(resolve => setImmediate(resolve));
  return {
    context, download, upload, input, requests, downloads, statuses, sent, flush,
    get recorder() { return recorder; },
    get stopped() { return stopped; },
    response(response) { nextResponse = response; },
    async record(sendAfter = false, data) {
      context.startRecording();
      await flush();
      if (data) recorder.data = data;
      context.stopRecording(sendAfter);
      await flush();
    },
    async choose(file) {
      input.files = file ? [file] : [];
      input.value = file ? file.name : "";
      input.handlers.change();
      await flush();
    },
  };
}

function file(name, bytes, type = "audio/mpeg") {
  const blob = new Blob(["a".repeat(bytes)], { type });
  blob.name = name;
  return blob;
}

test("More exposes separate audio download and transcription actions", () => {
  assert.match(html, /id="audio-download"[^>]*role="menuitem"[^>]*disabled/);
  assert.match(html, /id="audio-upload"[^>]*role="menuitem"[^>]*disabled/);
  assert.match(html, /id="audio-input"[^>]*type="file"[^>]*accept="audio\/\*/);
  assert.match(app.slice(app.indexOf("  function setState(s)"), app.indexOf("  function setVoiceStatus")), /updateAudioActions\(\)/);
});

test("last recording remains downloadable after transcription fails, and is replaced by the next recording", async () => {
  const voice = makeVoice();
  assert.equal(voice.download.disabled, true);
  voice.response({ ok: false, status: 502, statusText: "Bad Gateway", text: () => Promise.resolve("provider failed") });
  await voice.record();
  assert.equal(voice.stopped, 1);
  assert.match(voice.statuses.at(-1), /provider failed/);
  assert.equal(voice.download.disabled, false);
  voice.download.click();
  assert.equal(voice.downloads[0].blob.size, 100);
  assert.equal(voice.downloads[0].name, "voice.webm");
  assert.equal(voice.downloads[0].href, "blob:test");
  assert.equal(voice.downloads[0].revoked, "blob:test");
  await voice.record(false, new Blob(["y".repeat(120)], { type: "audio/webm" }));
  voice.download.click();
  assert.equal(voice.downloads[1].blob.size, 120);
});

test("uploaded audio preserves its filename, appends to drafts and never sends automatically", async () => {
  const voice = makeVoice();
  voice.context.msg.value = "existing draft";
  const audio = file("meeting.mp3", 100);
  await voice.choose(audio);
  assert.equal(voice.requests.length, 1);
  assert.equal(voice.requests[0].url, "/api/transcribe");
  assert.equal(voice.requests[0].options.headers.Authorization, "Bearer test");
  assert.equal(voice.requests[0].options.headers["Content-Type"], undefined);
  assert.deepEqual(voice.requests[0].options.body.fields[0], ["audio", audio, "meeting.mp3"]);
  assert.equal(voice.context.msg.value, "existing draft transcript");
  assert.deepEqual(voice.sent, []);
  assert.equal(voice.input.value, "");
  await voice.choose(audio);
  assert.equal(voice.requests.length, 2);
  assert.equal(voice.context.msg.value, "existing draft transcript transcript");
});

test("uploading does not replace the last microphone recording, even when it was short", async () => {
  const voice = makeVoice();
  await voice.record(false, new Blob(["a".repeat(50)], { type: "audio/webm" }));
  assert.match(voice.statuses.at(-1), /too short/i);
  assert.equal(voice.requests.length, 0);
  await voice.choose(file("another.mp3", 100));
  voice.download.click();
  assert.equal(voice.downloads[0].blob.size, 50);
  assert.equal(voice.downloads[0].name, "voice.webm");
});

test("record-and-send still sends, while upload after a send writes a new draft", async () => {
  const voice = makeVoice();
  await voice.record(true);
  assert.deepEqual(voice.sent, ["transcript"]);
  voice.context.msg.value = "";
  await voice.choose(file("voice.webm", 100, "audio/webm"));
  assert.equal(voice.context.msg.value, "transcript");
  assert.deepEqual(voice.sent, ["transcript"]);
});

test("invalid file sizes are reported without uploading or losing the last recording", async () => {
  const voice = makeVoice();
  await voice.record();
  const count = voice.requests.length;
  await voice.choose(file("tiny.wav", 43, "audio/wav"));
  assert.equal(voice.requests.length, count);
  assert.match(voice.statuses.at(-1), /size|small|44/i);
  await voice.choose(file("large.wav", (25 << 20) + 1, "audio/wav"));
  assert.equal(voice.requests.length, count);
  assert.match(voice.statuses.at(-1), /size|large|25/i);
  assert.equal(voice.download.disabled, false);
});

test("download stays available offline; uploading is disabled offline, during a call or recording", async () => {
  const voice = makeVoice();
  await voice.record();
  voice.context.connected = false;
  voice.context.updateAudioActions();
  assert.equal(voice.download.disabled, false);
  assert.equal(voice.upload.disabled, true);
  voice.context.connected = true;
  voice.context.callLive = true;
  voice.context.updateAudioActions();
  assert.equal(voice.upload.disabled, true);
  voice.context.callLive = false;
  voice.context.recording = true;
  voice.context.updateAudioActions();
  assert.equal(voice.upload.disabled, true);
});

test("native external-link interception does not consume blob downloads", () => {
  assert.match(gui, /hasAttribute\(["']download["']\)/);
  const linkStart = app.indexOf("  function openLinkExternally(e)");
  const linkEnd = app.indexOf("  document.addEventListener(\"click\", openLinkExternally", linkStart);
  assert.ok(linkStart >= 0 && linkEnd > linkStart);
  const opened = [];
  const context = { keimlingOpenExternal(href) { opened.push(href); } };
  vm.runInNewContext(app.slice(linkStart, linkEnd), context);
  let prevented = false;
  const anchor = {
    href: "blob:test",
    getAttribute(name) { return name === "href" ? this.href : null; },
    hasAttribute(name) { return name === "download"; },
  };
  context.openLinkExternally({
    target: { closest() { return anchor; } },
    preventDefault() { prevented = true; },
    stopPropagation() {},
  });
  assert.equal(prevented, false);
  assert.deepEqual(opened, []);
});

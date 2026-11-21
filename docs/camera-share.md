# Camera share

A camera runs beside the chat the same way a screen does. The browser takes the
picture, and the server keeps only the newest frame so the next chat message can
carry it. A screen share and a camera share are separate: both can run, and
stopping one leaves the other alone.

## Turning the camera on

The composer menu has **Camera** next to **Screen**. Choosing it asks the browser
for `getUserMedia({audio: false, video: …})`. A phone asks for the camera behind
the screen first (`facingMode: "environment"`) and falls back to whatever camera
the browser offers; a desktop asks for its normal camera. No sound is requested.

The Now view then shows the moving picture under Performance and above the Plan,
in the block **Camera**, with a status line and Stop. Starting the camera opens
that view. The block hides itself again when the share ends.

## Frames

Once a second counts as slow, and a full-size still is expensive, so the browser
draws the video into a canvas whose longest edge is at most 1280 pixels and
encodes it as JPEG. About five frames a second are posted to
`POST /api/camera/share`; only one request is in flight at a time, and a newer
frame replaces the older one.

Each request carries one JPEG as its body. The server checks the JPEG marker and
a 4 MiB limit and keeps the bytes in `cameraShareState` (`src/camera_frame.go`),
the newest frame only. That is deliberately simpler than the screen's
length-prefixed stream: the browser already owns the frame loop, and every
mobile browser can post a single picture.

## What the model sees

`withCameraShare` appends the newest frame to a chat message as
`keimling-camera.jpg`, next to the screen's `keimling-screen.jpg`. Only the
newest camera frame in a thread is put into the model context (as an
`input_image`); an older one is marked as replaced. When models cannot see
pictures at all, the frame stays a path. When both shares run, both pictures go
out.

## The camera_snapshot tool

`camera_snapshot` writes the newest frame to a temporary JPEG and returns its
path with the usual picture hint, so the assistant can look at it with
`delegate` and `image` set to that path. Without a running camera share it
returns a short request to turn the camera on rather than an error. The tool is
read-only: no approval is needed, and it is granted in the agent tool catalog
under **Desktop**.

## Stopping and cleanup

Stop stops the camera tracks, clears the interval, hides the block, and posts
`POST /api/camera/share?stop=1`, which drops the frame from memory. No picture is
written while the share runs; the only files are the chat attachment of a sent
message and the snapshot JPEG of an explicit tool call.

## Tests

- `src/camera_frame_test.go` covers the newest frame, clearing it, the
  independence of camera and screen frames, the newest-frame-only rule in the
  model input, the `camera_snapshot` answer with and without a frame, and the
  endpoint.
- `src/webui/tests/camera-share.test.js` covers the frame scaling and the camera
  constraints.

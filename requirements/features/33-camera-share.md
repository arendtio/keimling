# Feature: Camera share

## Intent

A person can share the live picture of a camera in the chat, and the assistant
can take a still from that camera on its own. A camera share and a screen share
run side by side and do not stop each other.

## REQ-CAM-1 One live camera picture

- **Statement:** Keimling shall let the user turn the camera on in the sidebar and shall keep the newest picture of that camera available for the chat.
- **Origin:** User, 2026-11-21 (quote: "Kamera-Sharing, in beiden Varianten — (a) Live-Video wie beim Screen-Sharing").
- **Rationale:** The user wants the same continuous picture they already have for a shared screen, from the camera instead.
- **Fit criterion:** After the user turns the camera on and the browser allows it, the Now view beside the chat shows the moving camera picture with a status line and Stop. The chat model receives the newest frame with the next message. Stop ends the stream, stops the camera tracks, and the sidebar block goes away.
- **Priority:** shall
- **Status:** confirmed

## REQ-CAM-2 Camera and screen stay independent

- **Statement:** A camera share and a screen share shall each keep running when the other one starts or stops. When both run, the model shall receive both pictures.
- **Origin:** User, 2026-11-21 (quote: "Kamera und Screen müssen unabhängig voneinander funktionieren und sich nicht gegenseitig abschalten; wenn beides läuft, werden beide Bilder angehängt.").
- **Fit criterion:** With a screen shared, starting the camera leaves the screen picture running, and the next message carries the camera frame and the screen frame. Stopping the camera keeps the screen frame. Stopping the screen keeps the camera frame.
- **Priority:** shall
- **Status:** confirmed

## REQ-CAM-3 Only the current camera frame is in the model context

- **Statement:** The model context shall hold the current camera frame only. A newer frame replaces the previous one, as with the screen share.
- **Origin:** Same share rule as REQ-SS-3, applied to the camera.
- **Fit criterion:** After the camera picture has changed, the model answers from the new frame, and an older camera frame in the thread is marked as replaced instead of appearing again.
- **Priority:** shall
- **Status:** confirmed

## REQ-CAM-4 The camera_snapshot tool

- **Statement:** The assistant shall be able to take the current camera picture with a `camera_snapshot` tool, and shall get it as a picture it can look at. When no camera share runs, the tool shall say so and ask the user to turn the camera on instead of failing.
- **Origin:** User, 2026-11-21 (quote: "ein Werkzeug `camera_snapshot` für Standbilder" and "auch wenn gerade kein Live-Share aktiv ist: dann eine kurze Aufforderung an den User, die Kamera einzuschalten, statt eines Fehlers").
- **Rationale:** A still is what the assistant needs for one look at the room, without a running share for the rest of the conversation.
- **Fit criterion:** With the camera on, a `camera_snapshot` call returns a JPEG path of the newest frame plus the usual picture hint, and the picture can be looked at. With the camera off, the call returns a request to turn the camera on and does not report an error.
- **Priority:** shall
- **Status:** confirmed

## REQ-CAM-5 No sound and no stored pictures

- **Statement:** The camera share shall carry no sound, shall not recognise faces, and shall keep no pictures beyond the newest frame in memory and the normal chat attachment.
- **Origin:** User, 2026-11-21 (quote: "Kein Ton, keine Gesichtserkennung, keine Speicherung von Bildern außer dem flüchtigen Frame im State und der normalen Chat-Upload-Ablage. Aufräumen: beim Stop Tracks stoppen, State leeren.").
- **Fit criterion:** The camera stream asks for video only. No file is written while the share runs. After Stop, the server holds no camera frame and the next message carries none.
- **Priority:** shall
- **Status:** confirmed

## REQ-CAM-6 Mobile and desktop

- **Statement:** The camera share shall work in a mobile browser and on the desktop, asking for the rear camera where the device has one and falling back to the offered camera.
- **Origin:** User, 2026-11-21 (quote: "Plattform: mobile Browser und Desktop; `getUserMedia` mit `video: {facingMode: …}`-Fallback.").
- **Fit criterion:** A phone asks for the camera behind the screen first and still starts when the browser offers only one camera. A desktop starts with the camera it has. Frames are at most 1280 pixels wide and go out at about five per second.
- **Priority:** shall
- **Status:** confirmed

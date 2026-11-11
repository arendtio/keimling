# Feature: Live screen share

## Intent

A person can share a live picture of a screen or a window. The picture keeps
moving after the choice. A new click does not take another still.

## REQ-SS-1 One continuous share

- **Statement:** Keimling shall let the user share a live stream of the current desktop, a screen, or a window.
- **Origin:** User, 2026-11-07 (quote: "ein Tool haben, mit dem ich einen Livestream von meinem aktuellen Desktop, meinem Fenster oder so übertragen kann. Also nicht, dass ich jedes Mal, wenn ich irgendwo draufgeklickt hab, einen neuen Screenshot ziehen muss.").
- **Fit criterion:** The system chooser asks which screen or window to share. After the choice, the picture appears without waiting for that window to change, and it keeps updating as the window changes, without another capture. Stopping the share ends the stream. A single still after the chooser does not meet this. Waiting until the window changes before the first picture appears does not meet this.
- **Priority:** shall
- **Status:** confirmed

## REQ-SS-2 The model sees the share

- **Statement:** While a screen or window is shared, the model in the chat and in the live call shall see that picture.
- **Origin:** User, 2026-11-07 (quote: "Aber das Model kan anscheinend trotzdem ncihts sehen"). User, 2026-11-07 (quote: "wenn ich dann im Chat oder auch im Call die KI nach diesem Bild frage … sagt die KI mir, es würde nicht, wäre nichts da … es doch irgendwie nicht an die KI wirklich übertragen wird.").
- **Fit criterion:** With a share running, a question in the chat or in the live call can be answered from that screen. A picture that only updates in the window does not meet this.

## REQ-SS-3 Only the current frame is in the model context

- **Statement:** While a screen or window is shared, the model context shall hold the current frame only. A newer frame replaces the previous one.
- **Origin:** User, 2026-11-08 (quote: "Werden die Bilder eigentlich immer an den Kontext appended oder wird quasi immer nur ein Bild reingehängt und das ist das Aktuelle und das wird immer wieder ausgetauscht? Das wäre ja, glaube ich, deutlich sinnvoller, ne?").
- **Fit criterion:** After the shared picture has updated, a question is answered from the latest frame. Earlier frames are not still in the model context.
- **Priority:** shall
- **Status:** confirmed

## REQ-SS-4 Keyboard and mouse on the shared picture

- **Statement:** While a screen or window is shared, the user shall be able to additionally share the keyboard and mouse, and the model shall then be able to click, scroll, type, and press keys in that picture. Adding keyboard and mouse shall keep the same window or screen. Turning that share off shall stop further input.
- **Origin:** User, 2026-11-10 (quote: "dass man über den gleichen Mechanismus den Desktop so shared, dass die KI auch selber darin Klicks machen kann und Tastatureingaben"). User, 2026-11-10 (quote: "wenn ich den Bildschirm geteilt hab, ich zusätzlich noch Tastatur und Maus teilen kann"). User, 2026-11-10 (quote: "bei der ersten Screen-Freigabe hab ich ihm ein Fenster freigegeben, und als er dann nach Tastatur und Maus gefragt hat, hat er den gesamten Bild, also beide Bildschirme als einen gemeinsamen geteilten Bereich, freigegeben. Und ich hätte jetzt erwartet, dass es trotzdem bei der Fensterteilung bleibt").
- **Rationale:** Seeing the desktop is not enough when the user wants the assistant to operate the shared screen instead of only talking about it. A window share that becomes both screens changes what the user allowed.
- **Fit criterion:** With a picture shared and keyboard and mouse not shared, a click, scroll, typed text, or key from the model does not change the desktop. After the user turns keyboard and mouse on and the system allows it, a click, a scroll, typed text, and a key from the model land in the shared picture. If that picture was a window, it stays that window. Turning it off stops the next one. The picture keeps updating either way.
- **Priority:** shall
- **Status:** confirmed

## REQ-SS-5 The same share on Linux, macOS, and Windows

- **Statement:** Sharing a screen or window, and additionally sharing the keyboard and mouse, shall work on Linux, macOS, and Windows.
- **Origin:** User, 2026-11-10 (quote: "dass beides auch für 'n Mac und auch für Windows funktioniert").
- **Rationale:** The desktop client is used on more than one operating system, and the same job has to be possible on each.
- **Fit criterion:** On Linux, macOS, and Windows the user can choose a screen or window, see that picture keep updating, and turn keyboard and mouse on so a click and typed text from the model land in that picture.
- **Priority:** shall
- **Status:** confirmed

## REQ-SS-6 The live picture sits beside the chat

- **Statement:** While a screen is being shared, the live picture, the switch for keyboard and mouse, and the control that stops the share shall be in the Now view beside the chat, under the performance and above the plan. The picture shall be at most about one third of the viewport height. Starting a share shall open that view. The chat composer shall not show the picture.
- **Origin:** User, 2026-11-11 (quote: "Mir gefällt die UX noch nciht so richtig. Wollen wir die Preview vom Sharing sonst in die sidebar packen (vielleicht mit bei Now?) und das teilen der eingabe könnte man dort dann auch aktivieren."). Same day (quote: "ein Drittel der Bildschirmhöhe sollte schon drin sein als maximale Höhe" and "ganz oben die Performance reinnehmen, darunter dann die Preview und dann den Plan").
- **Rationale:** The picture in the composer covers the place where the user writes. The share is something they look at beside the chat, together with what is happening now. A cap near 200 pixels makes a shared screen unreadably small.
- **Fit criterion:** After the user starts a share, the Now view is open and shows the moving picture under the performance and above the plan, with a switch for keyboard and mouse and Stop. The picture uses the width of that view and is no taller than about one third of the viewport. The composer does not show that picture. With the share running, choosing Screen again shows that same view and does not start a second share. Stop ends it and the picture leaves Now.
- **Priority:** shall
- **Status:** confirmed

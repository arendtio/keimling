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
- **Priority:** shall
- **Status:** confirmed

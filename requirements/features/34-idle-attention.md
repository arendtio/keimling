# Feature: Notice when a run goes idle

## REQ-IA-1 A finished run can sound and notify

- **Bin:** explicit requirement
- **Statement:** Settings shall let the user turn on a sound, a desktop notification, or both, for the moment a run returns to idle. With both off, finishing a run shall stay quiet. While the window is focused, that moment shall stay quiet even when a notice is turned on. The sound shall be generated when it plays. The product shall not ship a stored audio file for it.
- **Origin:** User, 2026-11-25 (quote: "in den Einstellungen die Möglichkeit hätte, das zu aktivieren, dass er einen Sound abspielt oder auch eine Desktop-Notification macht, wenn er in den Idle-Modus zurückgeht" and "der Sound … keine Wave-Datei … generiert" and "nur so 'n kleiner Pin-Sound"). Same day, later (quote: "der sollte … nicht aufkommen, wenn das Fenster aktuell fokussiert ist").
- **Rationale:** A long run finishes while the person is in another window. A ping while they are already looking at the chat is noise. A stored sound file would sit in the binary for one ping.
- **Fit criterion:** With both options off, a run that reaches idle plays nothing and posts no notification. With the window focused, a run that reaches idle stays quiet. With the window unfocused and sound on, that transition plays a short generated ping and adds no audio file to the binary. With the window unfocused and the notification on, that transition posts a desktop notification. The choices survive a restart.
- **Priority:** should
- **Status:** confirmed

## REQ-IA-2 The ping plays the way the desktop call already plays

- **Bin:** explicit constraint
- **Statement:** On the desktop window the ping shall play through an HTML audio element, the same kind of playback the live call uses when that window has no WebRTC. It shall not depend on an audio-context output, which that window does not use for playback.
- **Origin:** User, 2026-11-25 (quote: "unter macOS … Ton abzuspielen … bei den Calls" and "bestimmte APIs nicht zur Verfügung standen" and "unter macOS mit der GUI funktioniert das … nicht" and "ob die überhaupt zur Verfügung steht unter macOS mit dem WebKit").
- **Rationale:** The call already found that this WebKit plays a WAV through an audio element and does not play the spoken reply through the audio context. The context in that window is the microphone path.
- **Fit criterion:** With the desktop window unfocused and the sound on, the ping is heard from an audio element fed with generated WAV bytes. The call's microphone path is unchanged. A browser that already plays the call over WebRTC still hears the ping.
- **Priority:** shall
- **Status:** confirmed

Whether the CLI has the same notice is open. See `requirements/open-questions.md`.

# Feature: Growth logo

## REQ-GL-1 The sidebar plant follows the open chat's context

- **Bin:** explicit requirement
- **Statement:** The sidebar logo shall be the animated plant. Its stage shall follow how full the open chat's context is, on the same curve as the growth slider: early fill moves the plant farther, and later fill moves it less. An empty chat may keep the still plant. After the first message of that chat is sent, the plant shall begin as a seedling and then move to the stage for that fill. Each later change in fill shall move it again. The plant shall also move backward when the fill drops. A change of chat shall move it from the stage it is showing to the stage of the chat that is now open. Those moves shall be smooth.
- **Origin:** User, 2026-11-19 (quote: "dieses animierte Logo verwendet wird und der Zustand des Logos davon abgeleitet wird, wie viel % des Kontextes verwendet werden (analog des sliders, inkl. der logarithmik). Ein ganz leerer Chat darf gerne einen ähnlichen Zustand wie aktuell anzeigen, aber sobald die erste Nachricht abgesendet wird, soll es mit dem seedling starten und dann langsam wachsen bei jeder Nachricht. … auch beim Thread-Wechsel … sollen die Übergänge immer smooth animiert sein"). Same day, the seedling start is the moment of the first send, and a drop in fill moves the plant back (answers: start at the seed and glide to the measured fill; shrink smoothly both ways).
- **Rationale:** The plant is the mark of the product, and the context fill is the thing a long chat is quietly spending. A jump on send or on a thread change would make the mark look broken.
- **Fit criterion:** An empty chat shows the still plant. Sending its first message plays from the seedling to the stage for that chat's context fill, using the slider's logarithmic curve. A later message moves the plant to the new fill. After compaction, or after opening a chat with a smaller fill, the plant moves back along that curve. Opening another chat moves the plant from the stage on screen to that chat's stage without a snap. Login and the connection card keep the still drawing.
- **Priority:** shall
- **Status:** confirmed

## REQ-GL-2 The animation can be turned off

- **Bin:** explicit requirement
- **Statement:** Settings shall offer a control that turns the animated logo off. It shall be on by default, and the choice shall still be the same after a restart. Off shall show the still plant. When the system asks for reduced motion, the plant shall show the stage for the current fill as a still frame.
- **Origin:** User, 2026-11-19 (quote: "in den Optionen eine Möglichkeit geben das animierte Logo abzuschalten, default soll aber aktiv sein"). Same day (answer: reduced motion keeps a still frame at the matching stage; off shows today's still plant).
- **Rationale:** The motion is a pleasure, not a requirement for using the chat. Someone who does not want it, or whose system asks for less motion, still needs a readable mark.
- **Fit criterion:** A new install shows the animated plant. Turning the control off shows the still plant immediately, and turning it on returns to the animated plant at the open chat's stage. After a restart the control is still off. With reduced motion requested, the plant does not play; it shows the still frame for the open chat's fill.
- **Priority:** shall
- **Status:** confirmed

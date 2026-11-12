# Feature: Sidebar

The side area is the place for what the user looks at beside the chat. Artifacts are one kind of content there. The records below are the stakeholder ask. Tab names, layout, and how a rate is computed are not decided here.

## REQ-SB-1 One side area for the things beside the chat

- **Bin:** explicit requirement
- **Statement:** The product shall show artifacts, the current context, the autopilot instructions, the agent's current plan and its status, live performance, and the MCP connections in one side area. Each of these shall be viewable beside the chat, as a focus overlay, and fullscreen. Closing the area shall leave the chat in place.
- **Origin:** User, 2026-11-08 (paraphrase of using the artifact sidebar more generally, moving Context and Autopilot instructions out of More, and using Focus, Fullscreen, and Close for all of that content: "alle Sachen, die ich gerade aufgezählt hab, sowohl in der Sidebar anzeigen könnte, als auch … als Fullscreen oder als Focus Overlay").
- **Rationale:** Those views are the same job as the artifact panel: look at something next to the chat, and give it more room when the chat column is too narrow.
- **Fit criterion:** Context, autopilot instructions, a plan, the performance line, and the MCP connections each open in the side area. Each can be switched to a focus overlay and to fullscreen, and back beside the chat. Close hides the area and the chat remains. An artifact can still be opened outside the app.
- **Priority:** shall
- **Status:** confirmed

## REQ-SB-2 The entry is the sidebar

- **Bin:** explicit requirement
- **Statement:** The product shall replace the More entry with an entry named Sidebar that opens this side area. Whether that area is open, which view is selected, and how wide it is beside the chat shall be restored when the app is opened again.
- **Origin:** User, 2026-11-08 (quote: "diesen More-Button … generischer darstellen und den irgendwie Sidebar nennen" and "die ganzen Dinge, die da jetzt drin sind, dann eben in die Sidebar verfrachten"). Same day (quote: "Können wir auch speichern, ob die Sidebar geöffnet oder geschlossen ist und welcher Tab selektiert war und wie breit die Sidebar war?").
- **Rationale:** More is a list of unrelated dialogs. The user thinks of them as one side area.
- **Fit criterion:** The toolbar control is named Sidebar. Choosing it opens the side area on the view last used. Context, artifacts, and autopilot instructions are no longer separate entries under a More menu. After the app is opened again, the side area is open or closed as it was, shows the same view, and has the same width beside the chat.
- **Priority:** shall
- **Status:** confirmed

## REQ-SB-3 The plan and the performance share a view

- **Bin:** explicit constraint
- **Statement:** The side area should not give every topic its own tab. The current plan and the live performance should be seen together. Other topics should be split or combined by how much room they need and whether the user looks at them at the same time.
- **Origin:** User, 2026-11-08 (quote: "nicht sinnvoll jetzt für jedes der Themen … 'n eigenen Tab" and "Manche Sachen lassen sich vielleicht auch kombinieren, wie der Plan … und die Performance-Daten" and "wie gut es inhaltlich zusammenpasst, ob das aus User-Sicht gemeinsam betrachtet werden sollte").
- **Rationale:** A tab per meter makes the user hunt. The plan and the pulse of the run are read together while waiting.
- **Fit criterion:** With a plan present and a run in progress, the plan's steps and the performance figures are visible without changing tabs. Context is not squeezed into that same view. Autopilot instructions are not edited on that same view.
- **Priority:** should
- **Status:** confirmed

## REQ-SB-4 The current plan and its status stay visible

- **Bin:** explicit requirement
- **Statement:** While the agent has a plan, the side area shall show that plan and the status of each part, and shall keep showing it for the rest of the run.
- **Origin:** User, 2026-11-08 (quote: "dauerhaft sehen könnte, welchen Plan der Agent gerade abarbeitet … und wo welchen Status der hat"). User, 2026-11-11 (quote: "für den plan auch die unicode zeichen nutzen die wir in der cli nutzen in ggf. noch das halbvolle für in progress").
- **Rationale:** The plan is the answer to what the agent is doing, and it is lost when it only exists inside the model context.
- **Fit criterion:** After the agent records a plan with one part in progress and one part pending, both parts and those statuses are visible in the side area without asking the agent. Pending is an empty ring, in progress is a half mark, and done is a check, the same family as the command-line plan. When a part later becomes done, the side area shows the check. With no plan, the view says there is no plan.
- **Priority:** shall
- **Status:** confirmed

## REQ-SB-5 Live performance of the model

- **Bin:** explicit requirement
- **Statement:** The side area shall show how many tokens per second are being processed, how long it has been since the model last responded, and how long the model has been thinking. Token speed shall be read against a typical rate, and a model that stays faster than that typical rate shall read as faster. A single opening figure that is far above the speeds which follow shall not set that scale. Recent speeds shall be visible as columns about as wide as two capital letters, as many as fit across the side area. While the model is thinking, a light shall travel from one side to the other and back, under the speed columns, in the same green as the rest of the interface, as a thin solid bar that meets both ends. A long quiet stretch while the user is waiting shall read as the model still working. The time since the model last checked in shall increase only while the user is waiting for the model. When the model has finished and the run is waiting for the user, that time shall stay still, the finished mark beside the heading shall show, and the speed columns shall remain, with one zero-height column at the end. The word Done is not shown. Speed above the typical rate shall show as a taller column in the same colour.
- **Origin:** User, 2026-11-08 (quote: "so und so viel Token pro Sekunde werden gerade verarbeitet, so und so lange ist es her, dass … das Model sich zuletzt zurückgemeldet hat, so und so lange ist es gerade schon am Denken"). Same day, after seeing the first figures (quote: "kleines Dashboard in Miniaturansicht … jetzt gibt er richtig Gas" and "wenn das Model fertig gearbeitet hat und auf User Input wartet, dann muss die Zeit … nicht weiterlaufen"). Same day, after seeing the first dashboard (quote: "häufig in der ersten Sekunde … ein absurd hoher Token per Second Wert" and "nur normalisierte Token" and "ein Maximum irgendwo bei 90 Token pro Sekunde oder 120 oder 200" and "sehen, wenn das Modell quasi extrem schnell ist, weil es schneller ist als sonst" and "eher 'n Balkendiagramm, was so durchläuft … die aktuellsten 20 Balken" and "bei Thinking … Knight Rider … Zyklon von Battlestar Galactica"). Same day, after seeing the bar chart (quote: "Statt quiet vielleicht eher done" and "den thinking teil würde ich eher unter dem token/s diagramm anzeigen" and "ohne farbeverläuft, etwas dünner und in dem gleichen grün"). Same day (quote: "auf der linken Seite geht er mal bis zur Hälfte weg und auf der rechten geht er gar nicht bis zum Rand" and "da braucht man gar keine extra Farbe" and "die Performance Daten sind dann ja einfach weg" and "ein Balken … der quasi die Höhe null hat"). Same day (quote: "ein Balken ungefähr so breit ist wie zwei Buch, Großbuchstaben, und dann aber so viele Balken angezeigt werden, wie reinpassen"). User, 2026-11-11 (quote: "ganz oben die Performance" and "eine Baseline" and "hinter der Überschrift irgendwie so ein kleines Icon, was den Status berichtet"). Same day (quote: "das Done unter dem performance graphen brauchen wir nicht mehr. und die Linie darunter dann auch nicht mehr. Visuell reicht dann glaube ich die baseline des charts."). Same day (quote: "das layout etwas unruhig, wenn die neuen elemente erscheinend und wieder verschwinden" and "der platz den es einnimmt sich nicht verändert zwischen stop, thinking und so weiter"). Same day (quote: "die token per second und thinking text anzeige würde ich noch nach rechts schieben, weil dort immer der aktuelle stand kommt").
- **Rationale:** A quiet run looks stuck. An opening spike makes every later speed look tiny, and a line does not show the last moments as clearly as a run of columns. The sweep is how a long think looks alive without pretending tokens are arriving.
- **Fit criterion:** While tokens arrive steadily below a typical rate, the columns stay under a visible mark for that typical rate. A model that stays faster than the mark draws columns past it. One opening sample far above the speeds that follow does not raise the mark and is not the tall column. Each column is about as wide as two capital letters. A wider side area shows more of the latest speeds and a narrower one shows fewer, newest at one end. While the model is thinking, a thin solid light in the interface green travels under the columns and meets both ends, and a duration increases. Columns above the typical rate stay the same colour and are taller. When the run is idle and waiting for the user, the finished mark shows beside the heading, the word Done is not shown, the columns remain with a zero-height column at the end, that duration does not increase, and the speed is not shown as a live rate. The columns stand on a baseline. There is no extra line under the performance block. The performance block keeps the same height when the run is finished, thinking, or producing tokens. The token rate and the thinking line sit on the right, under the latest column. A mark beside the Performance heading shows finished, still working, tokens arriving, or faster than the usual rate. Performance is the first block in that view.
- **Priority:** shall
- **Status:** confirmed

## REQ-SB-6 MCP connections are visible and can be made ready

- **Bin:** explicit requirement
- **Statement:** The side area shall list the MCP connections and whether each one is usable. When a connection needs an action before it can run, including starting authentication, the user shall be able to start that action from this list. The list shall not be where a connection is edited. A control on this list shall open Manage MCP servers for that.
- **Origin:** User, 2026-11-08 (quote: "welche MCP-Connections es gibt und ob die alle healthy sind, also ob da irgendwie Authentifizierung notwendig ist"). Confirmed the same day (quote: "primär read-only" and "authentifizierung starten sollte darüber auch gehen, also aktionen die man machen muss, damit er läuft" and "Zum anpassen sollte es einen button richtung Manage MCP Servers geben").
- **Rationale:** The user needs to see why a tool is missing and to unblock it, without a second place that edits the server definition.
- **Fit criterion:** A connected server is shown as usable. A server that needs authentication is shown as needing it, and starting authentication from that row begins the existing sign-in. The row does not rename the server, change its command, or change its address. The Manage MCP servers control opens the existing management screen.
- **Priority:** shall
- **Status:** confirmed

## REQ-SB-7 An artifact the agent opens comes forward

- **Bin:** explicit requirement
- **Statement:** When the agent shows or opens an artifact, that artifact shall become the visible content of the side area, including when the user was looking at something else there.
- **Origin:** User, 2026-11-08 (quote: "ein neues Artifact sollte nach vorne kommen, wenn der agent es zeigen/öffnen möchte").
- **Rationale:** The agent opens an artifact so the user sees it. Leaving the side area on another view hides that.
- **Fit criterion:** With the side area showing the plan, an agent request to show an artifact opens the side area on that artifact. A later request to show a second artifact replaces the visible one. The plan remains available without being the visible content.
- **Priority:** shall
- **Status:** confirmed

## REQ-SB-8 The chat keeps its width

- **Bin:** explicit requirement
- **Statement:** The chat shall keep a width of its own. A long message shall not make the chat wider. While the side area is open beside the chat, widening that area shall leave the chat at least about 600 pixels wide.
- **Origin:** User, 2026-11-12 (quote: "die Breite von unserem Gesprächs-Chat irgendwie wachsen kann" and "der soll nicht breiter werden, nur weil da irgendwie lange Nachrichten reinkommen" and "wenn man die Sidebar öffnet, dann kann man sie quasi so weit öffnen, wie man will, der Gesprächs-Chat hat keine Mindestbreite" and "so 'ne Mindestbreite von 600 Pixeln").
- **Rationale:** The chat is the place the user reads. A long line or a wide side area should not push that place out of shape.
- **Fit criterion:** A message with a very long line leaves the chat column the same width, and the line scrolls or wraps inside it. Dragging the side area wider stops while the chat is still about 600 pixels wide. On a window too narrow for that floor plus the side area, the chat keeps the room it can.
- **Priority:** shall
- **Status:** confirmed

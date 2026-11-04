# Feature: Artifact panel

## REQ-AR-1 An empty panel follows the theme

- **Bin:** explicit requirement
- **Statement:** When the artifact panel is open and nothing is loaded, the preview area shall follow the current light or dark background instead of showing a blank white sheet, and it shall say that nothing is open.
- **Origin:** User, 2026-11-04 (quote: "wenn wir kein Artefakt geladen haben, haben wir da immer so 'n weißes Rechteck im Empty State. Ich hätte da gern 'n Empty State, der sich eher auch am Theme orientiert … und 'n kleiner Text, der in der Mitte steht").
- **Rationale:** A white sheet in a dark window looks like a broken preview.
- **Fit criterion:** With no artifact loaded, the preview area uses the same background tone as the app in both light and dark, and a short sentence is centered in it. No white page shows through.
- **Priority:** shall
- **Status:** confirmed

## REQ-AR-2 Artifacts in the panel can be switched without scrolling past the preview

- **Bin:** explicit requirement
- **Statement:** The open artifact panel shall list the artifacts currently in the workspace, newest first, where they can be reached without scrolling past the preview, and choosing one shall show it.
- **Origin:** User, 2026-11-04 (quote: "wenn ich die Artefakte aufgeklappt hab, das im Moment ganz unten drunter irgendwie 'ne Liste ist … kommt man irgendwie durch Scrollen auch nur schwer ran … die Liste der Artefakte, die es in dem Workspace gerade gibt. Wahrscheinlich am besten sortiert nach Aktualität … selber switchen").
- **Rationale:** The list under the preview is off-screen for a tall panel.
- **Fit criterion:** With two artifacts, both names are visible at the top of the open panel without scrolling the page. The newer one is first. Choosing the other replaces the preview. Choosing the first again brings it back.
- **Priority:** shall
- **Status:** confirmed

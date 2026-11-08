# Feature: Themes

## REQ-TH-1 A theme is two palettes

- **Statement:** A theme shall carry a light palette and a dark palette. The person shall choose automatic, light, or dark. Automatic follows the system color scheme. The built-in theme is Keimling: the page background stays the KDE gray, and the dark surfaces stay in that gray-green family rather than blue.
- **Origin:** User, 2026-11-08 (paraphrase: themes are color palettes; the dark sheet is blue while the chrome is gray-green and they clash; keep the gray that matches KDE; each theme needs light and dark so they can be switched). Same day: the Themes page shall switch among automatic, light, and dark.
- **Fit criterion:** In dark mode the main background stays the KDE gray. The chat sheet is a darker green well. User bubbles and the composer sit lighter on that well. Themes offers Automatic, Light, and Dark. Automatic follows the system scheme. Light and Dark stay put when the system scheme changes. The choice survives a restart.
- **Priority:** shall
- **Status:** confirmed

## REQ-TH-2 Manage themes in Settings

- **Statement:** Settings shall have a Themes page where a person can create a theme, edit its palettes, use it, and delete a theme they created. Keimling can be edited and reset, and cannot be deleted.
- **Origin:** User, 2026-11-08 (paraphrase: a settings page to create, adjust, and delete themes, plus a finished default theme).
- **Fit criterion:** A new theme can be named, given the nine roles for light and dark, saved, and selected. The editor shows a light and a dark sample of a frame, a sheet, a raised row, text, a bubble, a button, and a danger label, and the sample follows the pickers. Deleting a theme asks first. Keimling has no delete. Reset Keimling restores its built-in palettes, including the leaf-green accent. The choice survives a restart.
- **Priority:** shall
- **Status:** confirmed

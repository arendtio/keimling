//go:build cgo && !nogui

package main

import (
	"encoding/json"
	"fmt"
	"strings"

	webview "github.com/samcharles93/webview_go"
)

func openWebViewImpl(url, title, configPath string) error {
	return openWebViewWithHomeImpl(url, title, configPath, "")
}

// openWebViewWithHomeImpl opens the WebView. When localHome is non-empty (desktop
// embedded agent), "switch to local" navigates back there; otherwise it clears
// gui-prefs and relaunches a local process (pure --connect session).
func openWebViewWithHomeImpl(url, title, configPath, localHome string) error {
	if title == "" {
		title = "Keimling"
	}
	prefs, _ := readGuiPrefs(configPath)
	width, height := windowSizeFromPrefs(prefs)
	prepareGUIIdentity()
	w, err := webview.NewWithOptions(webview.Options{
		Title:  title,
		Width:  width,
		Height: height,
		Hint:   webview.HintNone,
		URL:    url,
		Debug:  false,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", errGUIUnavailable, err)
	}
	defer w.Destroy()
	applyWindowIcon(w)
	allowMediaPermissions(w)
	interceptExternalLinks(w)
	_ = w.Bind("keimlingSaveGuiGeometry", func(width, height int) {
		_ = saveGuiWindowSize(configPath, width, height)
	})
	_ = w.Bind("keimlingOpenExternal", func(href string) {
		_ = openExternalURL(href)
	})
	_ = w.Bind("keimlingLocalHomeURL", func() string {
		return localHome
	})
	_ = w.Bind("keimlingUseLocalAgent", func() string {
		_ = mergeGuiPrefsConnect(configPath, "", "")
		if localHome != "" {
			return localHome
		}
		scheduleRelaunchLocal(configPath, func() {
			w.Dispatch(func() { w.Terminate() })
		})
		return "relaunch"
	})
	_ = w.Bind("keimlingRememberRemote", func(connect, secret string) {
		_ = upsertClientRemote(configPath, "", connect, secret)
		_ = mergeGuiPrefsConnect(configPath, connect, secret)
	})
	_ = w.Bind("keimlingGetClientConnections", func() string {
		f, err := readClientConnections(configPath)
		if err != nil {
			return `{"connections":[],"active_id":""}`
		}
		b, err := json.Marshal(f.public())
		if err != nil {
			return `{"connections":[],"active_id":""}`
		}
		return string(b)
	})
	_ = w.Bind("keimlingPutClientConnections", func(raw string) string {
		var body struct {
			Connections *[]connectionProfile `json:"connections"`
			ActiveID    *string              `json:"active_id"`
		}
		if err := json.Unmarshal([]byte(raw), &body); err != nil {
			return `{"ok":false,"error":"invalid json"}`
		}
		err := updateClientConnections(configPath, func(f *clientConnectionsFile) {
			if body.Connections != nil {
				prevByID := map[string]connectionProfile{}
				for _, c := range f.Connections {
					prevByID[c.ID] = c
				}
				next := *body.Connections
				for i := range next {
					if strings.TrimSpace(next[i].AccessSecret) != "" {
						continue
					}
					if prev, ok := prevByID[strings.TrimSpace(next[i].ID)]; ok && prev.AccessSecret != "" {
						next[i].AccessSecret = prev.AccessSecret
					}
				}
				f.Connections = next
			}
			if body.ActiveID != nil {
				f.ActiveID = strings.TrimSpace(*body.ActiveID)
			}
		})
		if err != nil {
			return fmt.Sprintf(`{"ok":false,"error":%q}`, err.Error())
		}
		f, err := readClientConnections(configPath)
		if err != nil {
			return `{"ok":false}`
		}
		b, _ := json.Marshal(f.public())
		return string(b)
	})
	_ = w.Bind("keimlingRelaunchWorkspace", func(workspaceID string) string {
		scheduleRelaunchWorkspace(configPath, workspaceID, func() {
			w.Dispatch(func() { w.Terminate() })
		})
		return "relaunch"
	})
	// Ensure clicks never navigate the embedded UI away before app.js binds handlers.
	_ = w.Init(`document.addEventListener("click",function(e){
  if(typeof keimlingOpenExternal!=="function")return;
  var a=e.target&&e.target.closest&&e.target.closest("a[href]");
  if(!a)return;
  var href=a.getAttribute("href");
  if(!href||href.charAt(0)==="#"||href.indexOf("javascript:")===0)return;
  e.preventDefault();e.stopPropagation();
  try{keimlingOpenExternal(a.href);}catch(err){}
},true);`)
	if err := w.Run(); err != nil {
		return fmt.Errorf("%w: %v", errGUIUnavailable, err)
	}
	return nil
}

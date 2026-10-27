//go:build cgo && !nogui

package main

import (
	"fmt"

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
		_ = mergeGuiPrefsConnect(configPath, connect, secret)
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

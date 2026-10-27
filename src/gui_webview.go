//go:build cgo && !nogui

package main

import (
	"fmt"

	webview "github.com/samcharles93/webview_go"
)

func openWebViewImpl(url, title, configPath string) error {
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

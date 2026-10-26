// HTTP-only baseline for binary size comparison (no WebView / CGO).
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	log.Printf("listening on http://%s/", ln.Addr())
	log.Fatal(http.Serve(ln, nil))
}

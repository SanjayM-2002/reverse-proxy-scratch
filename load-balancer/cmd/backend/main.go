// A dumb backend: logs every request with its name and echoes back who served it.
// Add ?sleep=3s to hold the connection open (handy for seeing least_connections work).
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	id := flag.Int("id", 1, "server number")
	port := flag.Int("port", 8001, "port to listen on")
	flag.Parse()

	name := fmt.Sprintf("server-%d-%d", *id, *port)
	logger := log.New(log.Writer(), "["+name+"] ", log.LstdFlags)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("%s %s from %s", r.Method, r.URL.RequestURI(), r.RemoteAddr)
		if d, err := time.ParseDuration(r.URL.Query().Get("sleep")); err == nil {
			time.Sleep(d)
		}
		fmt.Fprintf(w, "hello from %s\n", name)
	})

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	logger.Printf("listening on %s", addr)
	logger.Fatal(http.ListenAndServe(addr, nil))
}

package main

import (
	"net/http"
	"os"
	"os/signal"

	"github.com/maaxorlov1/cmd/server"
)

func main() {
	handler := server.NewRouter()

	// Start the server
	go func() {
		http.ListenAndServe("", handler)
	}()

	// Wait for an interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

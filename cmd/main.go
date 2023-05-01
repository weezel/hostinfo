package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"weezel/hostinfo/httpserver"
)

// Generated info
var (
	gitHash   string
	buildTime string
)

// Flags
var (
	showVersion bool
	endpoint    string
	listenPort  string
)

func main() {
	flag.BoolVar(&showVersion, "v", false, "Show version (git hash) and build time")
	flag.StringVar(&endpoint, "e", "", "Which HTTP route endpoint to listen i.e. http://localhost/myendpoint")
	flag.StringVar(&listenPort, "p", "8080", "Port to listen")
	flag.Parse()

	if showVersion {
		fmt.Printf("Git hash: %s, Build time: %s\n", gitHash, buildTime)
		os.Exit(0)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/%s", endpoint), httpserver.HostInfo)
	httpServ := &http.Server{
		Addr:              ":" + listenPort,
		Handler:           mux,
		ReadHeaderTimeout: 60 * time.Second,
	}

	go func() {
		if err := httpServ.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server closed")
		}
	}()
	log.Printf("Listening on port %s", listenPort)

	// Graceful shutdown for HTTP server
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	log.Println("HTTP server stopping")
	defer cancel()
	if err := httpServ.Shutdown(ctx); err != nil {
		log.Println("HTTP server stopped")
	}
}

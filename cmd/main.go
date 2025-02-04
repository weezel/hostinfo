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
	"runtime/pprof"
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
	profiling   bool
	endpoint    string
	listenPort  string
)

func enableProfiling() (*os.File, func()) {
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Panicf("Couldn't create cpu.prof file: %v", err)
	}

	if err = pprof.StartCPUProfile(f); err != nil {
		log.Panicf("Couldn't start CPU profile: %v", err)
	}

	return f, pprof.StopCPUProfile
}

func main() {
	flag.BoolVar(&showVersion, "v", false, "Show version (git hash) and build time")
	flag.StringVar(&endpoint, "e", "", "Which HTTP route endpoint to listen i.e. http://localhost/myendpoint")
	flag.StringVar(&listenPort, "p", "8080", "Port to listen")
	flag.BoolVar(&profiling, "P", false, "Enable profiling")
	flag.Parse()

	if showVersion {
		fmt.Printf("Git hash: %s, Build time: %s\n", gitHash, buildTime)
		os.Exit(0)
	}

	if profiling {
		log.Println("Profiling enabled")
		f, cpuProf := enableProfiling()
		defer func() {
			cpuProf()
			f.Close()
		}()
	}

	mux := http.NewServeMux()
	routeHandler := httpserver.NewRouteHandler()
	mux.HandleFunc(fmt.Sprintf("/%s", endpoint), routeHandler.HostInfo)
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

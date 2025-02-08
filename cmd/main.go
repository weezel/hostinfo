package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"weezel/hostinfo/httpserver"
)

// Generated info
var (
	gitHash   string
	buildTime string
)

// Flags
var (
	showVersion    bool
	profiling      bool
	pathEndoint    string
	listenPort     string
	unixSocketPath string
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
	ctx := context.Background()

	flag.BoolVar(&showVersion, "v", false, "Show version (git hash) and build time")
	flag.StringVar(&pathEndoint, "e", "", "Which HTTP route endpoint to listen i.e. http://localhost/myendpoint")
	flag.StringVar(&listenPort, "p", "", "Port to listen, default 8080")
	flag.StringVar(&unixSocketPath, "u", "", "Listen on Unix socket")
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

	if unixSocketPath != "" && listenPort != "" {
		fmt.Println("Using Unix socket and TCP socket simultaneously is not supported")
		os.Exit(1) //nolint:gocritic // Nothing to be closed at this point so exit is safe
	}

	var httpServer *httpserver.HTTPServer
	if unixSocketPath != "" {
		httpServer = httpserver.NewHTTPServer(httpserver.WithUnixSocketListener(unixSocketPath))
	} else {
		httpServer = httpserver.NewHTTPServer()
	}

	httpServer.AddRoute(fmt.Sprintf("/%s", pathEndoint), httpServer.HostInfo)
	httpServer.Start()
	defer httpServer.Stop(ctx)

	// Graceful shutdown for HTTP server
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done

	log.Println("HTTP server stopping")
}

package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"weezel/hostinfo/geoip"

	"golang.org/x/sync/errgroup"
)

const resolverTimeout = 500 * time.Millisecond

type hostInfo struct {
	SrcAddr      string        `json:"addr,omitempty"`
	SrcHostnames string        `json:"hostnames,omitempty"`
	SrcPort      string        `json:"src_port,omitempty"`
	UserAgent    string        `json:"user_agent,omitempty"`
	GeoLocation  geoip.GeoInfo `json:"geo_location,omitempty"`
}

func getClientIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}

	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}

	return IPAddress
}

func getHostname(ctx context.Context, ip string) (string, error) {
	cCtx, cancel := context.WithTimeout(ctx, resolverTimeout)
	defer cancel()

	resolver := net.Resolver{}
	hostNames, err := resolver.LookupAddr(cCtx, ip)
	if err != nil {
		return "", fmt.Errorf("failed to resolve hostname: %w", err)
	}

	return strings.Join(hostNames, ", "), nil
}

type Option func(*HTTPServer)

func WithUnixSocketListener(socketPath string) Option {
	// Remove any existing socket file
	if _, err := os.Stat(socketPath); err == nil {
		if err := os.Remove(socketPath); err != nil {
			log.Panicf("Error removing existing socket: %s", err)
		}
	}

	// Create a listener on the Unix socket
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Panicf("Error creating Unix socket: %s", err)
	}

	// Ensure the socket file is accessible by web server user and the non privileged user.
	// Forbid access for others.
	if err := os.Chmod(socketPath, 0o770); err != nil {
		log.Panicf("Error setting permissions on socket: %s", err)
	}

	return func(h *HTTPServer) {
		h.listener = listener
	}
}

func WithTCPSocketListener(host, port string) Option {
	return func(h *HTTPServer) {
		h.server.Addr = net.JoinHostPort(host, port)
	}
}

type HTTPServer struct {
	listener net.Listener
	geoData  *geoip.GeoInfo
	serveMux *http.ServeMux
	server   *http.Server
}

func NewHTTPServer(opts ...Option) *HTTPServer {
	sm := http.NewServeMux()
	h := &HTTPServer{
		geoData:  geoip.New(),
		serveMux: sm,
		server: &http.Server{
			Addr:              net.JoinHostPort("127.0.0.1", "8080"),
			Handler:           sm,
			ReadHeaderTimeout: 60 * time.Second,
		},
	}

	// Override defaults
	for _, opt := range opts {
		opt(h)
	}

	return h
}

func (h *HTTPServer) Start() {
	go func() {
		if h.listener != nil {
			log.Printf("Starting server on Unix socket %s", h.listener.Addr())
			if err := h.server.Serve(h.listener); err != nil &&
				!errors.Is(err, http.ErrServerClosed) {
				log.Print("HTTP server closed")
			}
		} else {
			log.Printf("Starting server on %s", h.server.Addr)
			if err := h.server.ListenAndServe(); err != nil &&
				!errors.Is(err, http.ErrServerClosed) {
				log.Print("HTTP server closed")
			}
		}
	}()
}

func (h *HTTPServer) Stop(ctx context.Context) {
	timeout := 5 * time.Second
	cCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.Printf("Closing HTTP server within %s", timeout)

	shutdownComplete := make(chan error, 1)
	go func() {
		shutdownComplete <- h.server.Shutdown(ctx)
		defer close(shutdownComplete)
	}()

	select {
	case <-cCtx.Done():
		h.server.Close()
	case err := <-shutdownComplete:
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Print("HTTP server closed")
				return
			}
			log.Printf("Error while shutting down the server: %v", err)
		} else {
			log.Print("HTTP server closed")
		}
	}
}

func (h *HTTPServer) AddRoute(path string, handler func(http.ResponseWriter, *http.Request)) {
	h.serveMux.HandleFunc(path, handler)
}

func (h *HTTPServer) HostInfo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()

	startTime := time.Now()

	var err error

	// Must be done before address resolve
	tmp := strings.Split(getClientIP(r), ":") // hostname:port
	info := hostInfo{}
	if len(tmp) > 1 {
		info = hostInfo{
			SrcAddr: tmp[0],
			SrcPort: tmp[1],
		}
	} else {
		info = hostInfo{
			SrcAddr: tmp[0],
			SrcPort: "0",
		}
	}

	geoLocation := h.geoData.GetGeoData(info.SrcAddr)
	eg, eCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		hostnames, err1 := getHostname(eCtx, info.SrcAddr)
		if err1 != nil {
			return fmt.Errorf("Failed to resolve hostname for IP=%s: %w", info.SrcAddr, err1)
		}
		info.SrcHostnames = hostnames

		return nil
	})
	info.UserAgent = r.Header.Get("User-Agent")
	if err = eg.Wait(); err != nil {
		log.Printf("Couldn't get hostname: %v", err)
	}

	// Intentionally after errgroup wait() call so that we don't cause races
	info.GeoLocation = <-geoLocation

	inJSON, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		log.Printf("failed to marshal into JSON: %v", err)
		fmt.Fprint(w, "Failed\n")
	}

	log.Printf("Incoming connection from: %#v, took %s\n", info, time.Since(startTime))

	fmt.Fprintf(w, "%s\n", string(inJSON))
}

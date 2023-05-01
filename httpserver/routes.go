package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
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
	ctx, cancel := context.WithTimeout(ctx, resolverTimeout)
	defer cancel()

	resolver := net.Resolver{}
	hostNames, err := resolver.LookupAddr(ctx, ip)
	if err != nil {
		return "", fmt.Errorf("failed to resolve hostname: %w", err)
	}

	return strings.Join(hostNames, ", "), nil
}

func HostInfo(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var err error
	geoData := geoip.New()
	defer geoData.Close()

	// Must be done before address resolve
	tmp := strings.Split(getClientIP(r), ":") // hostname:port
	info := hostInfo{
		SrcAddr: tmp[0],
		SrcPort: tmp[1],
	}

	geoLocation := geoData.GetGeoData(info.SrcAddr)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		hostnames, err1 := getHostname(ctx, info.SrcAddr)
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

	fmt.Fprintf(w, "%s\n", string(inJSON))
}

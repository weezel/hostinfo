package httpserver

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func suppressLogOutput() func() {
	originalOutput := log.Writer()

	log.SetOutput(io.Discard)

	// Return a function to restore the original logger output
	return func() {
		log.SetOutput(originalOutput)
	}
}

func BenchmarkHostInfo(b *testing.B) {
	b.Helper()

	if testing.Short() {
		b.Skipf("Not running benchmark tests when short is defined")
	}

	restoreLogging := suppressLogOutput()
	defer restoreLogging()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	httpServer := NewHTTPServer()
	defer httpServer.server.Close()
	httpServer.AddRoute("/hostinfo", httpServer.HostInfo)

	req := httptest.NewRequest(http.MethodGet, "/hostinfo", nil)
	req.Header.Set("X-Real-Ip", "127.0.0.1")
	req.Header.Set("User-Agent", "BenchmarkTest")

	rec := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.Run(b.Name(), func(_ *testing.B) {
			httpServer.HostInfo(rec, req)
		})
	}

	httpServer.Stop(ctx)
}

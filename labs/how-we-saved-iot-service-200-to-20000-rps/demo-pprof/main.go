package main

import (
	"log"
	"net/http"

	// Blank import registers /debug/pprof/* routes on the default ServeMux.
	// This is all you need — no extra configuration.
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
)

func main() {
	// ── 1. pprof HTTP server (separate port, never expose in prod) ────
	//
	// Gin does NOT use http.DefaultServeMux, so we spin up a second
	// plain net/http server on :6060 exclusively for pprof endpoints.
	// This is the standard pattern for Gin + pprof.
	//
	// Available endpoints after server starts:
	//   GET http://localhost:6060/debug/pprof/             ← index
	//   GET http://localhost:6060/debug/pprof/profile?seconds=30  ← CPU profile
	//   GET http://localhost:6060/debug/pprof/heap         ← heap snapshot
	//   GET http://localhost:6060/debug/pprof/goroutine    ← goroutine dump
	//   GET http://localhost:6060/debug/pprof/trace?seconds=5     ← execution trace
	go func() {
		log.Println("pprof server listening on :6060 — http://localhost:6060/debug/pprof/")
		if err := http.ListenAndServe(":6060", nil); err != nil {
			log.Fatalf("pprof server failed: %v", err)
		}
	}()

	// ── 2. Main Gin application ───────────────────────────────────────
	r := gin.Default()

	// Health check — no auth, useful as profiling baseline
	r.GET("/health", HandleHealthCheck)

	// Main telemetry endpoint — auth middleware gates it
	// Both human and machine clients hit the same URL.
	// The X-Client-Type header determines which auth path runs.
	r.POST("/v1/telemetry", AuthMiddleware(), HandleTelemetry)

	// ── 3. Start ──────────────────────────────────────────────────────
	log.Println("API server listening on :8080")
	log.Println("")
	log.Println("  POST /v1/telemetry   (X-Client-Type: human | machine)")
	log.Println("  GET  /health")
	log.Println("")
	log.Println("Human auth headers:   X-User-ID, X-Required-Permission")
	log.Println("Machine auth headers: X-Device-ID, X-Device-Token")
	log.Println("")
	log.Println("pprof: http://localhost:6060/debug/pprof/")

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}

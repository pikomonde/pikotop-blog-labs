package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// TelemetryPayload is the request body for a telemetry ping.
// For human clients this might be a dashboard data request;
// for machine clients this is a device pushing sensor data.
type TelemetryPayload struct {
	Timestamp   string             `json:"timestamp"`
	Metrics     map[string]float64 `json:"metrics"`
	Tags        map[string]string  `json:"tags"`
}

// TelemetryResponse is the standard response envelope.
type TelemetryResponse struct {
	Status    string      `json:"status"`
	RequestID string      `json:"request_id"`
	At        string      `json:"at"`
	Data      interface{} `json:"data,omitempty"`
}

// HandleTelemetry is the single endpoint that serves both human and machine clients.
// By the time we reach here, AuthMiddleware has already validated credentials.
func HandleTelemetry(c *gin.Context) {
	// ── Parse body ──────────────────────────────────────────────────
	var payload TelemetryPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		// Allow missing body for easy curl testing
		payload = TelemetryPayload{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Metrics:   map[string]float64{"ping": 1},
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// ── Build response depending on client type ──────────────────────
	clientType := c.GetHeader("X-Client-Type")

	var responseData interface{}

	switch clientType {
	case "human":
		// Return some dashboard-style summary using data already in context
		user, _ := c.Get(string(ctxKeyUser))
		u, _ := user.(*BigUser)

		name := "unknown"
		permCount := 0
		if u != nil {
			name = u.DisplayName
			permCount = len(u.Permissions)
		}

		responseData = gin.H{
			"message":          "Telemetry dashboard data fetched",
			"user_display_name": name,
			"permission_count": permCount,
			"payload_received": payload,
		}

	case "machine":
		deviceID, _ := c.Get(string(ctxKeyDeviceID))
		responseData = gin.H{
			"message":   "Telemetry data accepted",
			"device_id": deviceID,
			"payload_received": payload,
		}

	default:
		responseData = gin.H{"message": "ok"}
	}

	c.JSON(http.StatusOK, TelemetryResponse{
		Status:    "ok",
		RequestID: c.GetHeader("X-Request-ID"),
		At:        now,
		Data:      responseData,
	})
}

// HandleHealthCheck is a simple liveness probe — no auth, no Redis.
// Useful as a baseline during profiling: if this is slow, something is very wrong.
func HandleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

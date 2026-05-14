package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// ── Context keys ────────────────────────────────────────────────────────────

type contextKey string

const (
	ctxKeyUser     contextKey = "authenticated_user"
	ctxKeyDeviceID contextKey = "device_id"
)

// ── Simulated Redis (in-memory for demo) ────────────────────────────────────
//
// In production this would be a real Redis client.
// We simulate the two anti-patterns side by side:
//   humanCache  → stores the ENTIRE BigUser JSON blob (~300KB per entry)
//   machineHashes → pre-hashed bcrypt tokens for device credentials

type fakeRedis struct {
	humanCache    map[string][]byte     // userID → full JSON blob
	machineHashes map[string]string     // deviceID → bcrypt hash of token
	ttl           map[string]time.Time  // shared TTL tracking
}

func newFakeRedis() *fakeRedis {
	r := &fakeRedis{
		humanCache:    make(map[string][]byte),
		machineHashes: make(map[string]string),
		ttl:           make(map[string]time.Time),
	}

	// Pre-populate one human user in cache (the whole BigUser blob)
	user := generateMockUser("usr-demo-001")
	blob, _ := json.Marshal(user)
	r.humanCache["usr-demo-001"] = blob

	// Pre-populate machine tokens.
	// Cost 12 is common in production (and the root cause of our perf problem).
	// Lower values like 4 are faster but still visibly slow under load.
	// Use cost 12 to match realistic production behaviour.
	for _, id := range []string{"device-iot-001", "device-iot-002", "device-iot-003"} {
		hash, _ := bcrypt.GenerateFromPassword([]byte("s3cr3t-t0k3n-"+id), bcrypt.DefaultCost)
		r.machineHashes[id] = string(hash)
	}

	return r
}

// GetUser fetches the full user blob from the fake Redis cache.
func (r *fakeRedis) GetUser(userID string) (*BigUser, bool) {
	blob, ok := r.humanCache[userID]
	if !ok {
		return nil, false
	}
	var u BigUser
	if err := json.Unmarshal(blob, &u); err != nil {
		return nil, false
	}
	return &u, true
}

// CheckMachineToken runs bcrypt.CompareHashAndPassword on every call —
// this is the anti-pattern that killed the production service.
func (r *fakeRedis) CheckMachineToken(deviceID, token string) bool {
	hash, ok := r.machineHashes[deviceID]
	if !ok {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(token))
	return err == nil
}

// ── Global fake-Redis instance ───────────────────────────────────────────────

var redis = newFakeRedis()

// ── Auth Middleware ──────────────────────────────────────────────────────────
//
// Routing logic (simplified):
//   • X-Client-Type: human  → validate by userID header, fetch BigUser from Redis
//   • X-Client-Type: machine → validate device token with bcrypt on every request

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientType := c.GetHeader("X-Client-Type")

		switch strings.ToLower(clientType) {

		case "human":
			handleHumanAuth(c)

		case "machine":
			handleMachineAuth(c)

		default:
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing or unknown X-Client-Type header (human|machine)",
			})
		}
	}
}

// handleHumanAuth: the RBAC / JSON-unmarshal anti-pattern.
//
// Problem: we fetch ~300KB of JSON from Redis and unmarshal the entire
// BigUser struct just to check a single permission bit.
// pprof will show json.Unmarshal taking a large but non-catastrophic slice of CPU.
func handleHumanAuth(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	requiredPerm := c.GetHeader("X-Required-Permission") // e.g. "telemetry:read"

	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "X-User-ID header required for human clients"})
		return
	}

	// Fetch and unmarshal the entire BigUser from fake Redis
	user, ok := redis.GetUser(userID)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found in cache"})
		return
	}

	// Check the single permission we actually care about
	if requiredPerm != "" && !hasPermission(user, requiredPerm) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied", "required": requiredPerm})
		return
	}

	// Store user in context for the handler
	c.Set(string(ctxKeyUser), user)
	c.Request = c.Request.WithContext(
		context.WithValue(c.Request.Context(), ctxKeyUser, user),
	)
	c.Next()
}

// handleMachineAuth: the bcrypt-on-hot-path anti-pattern.
//
// Problem: every telemetry ping runs bcrypt.CompareHashAndPassword.
// Bcrypt cost 12 ≈ 300ms/op. At 10 concurrent requests that's 3s of CPU/s
// just for auth — before a single byte of business logic runs.
// pprof will show bcrypt consuming >90% of CPU under any meaningful load.
func handleMachineAuth(c *gin.Context) {
	deviceID := c.GetHeader("X-Device-ID")
	token := c.GetHeader("X-Device-Token")

	if deviceID == "" || token == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "X-Device-ID and X-Device-Token headers required for machine clients",
		})
		return
	}

	// ❌ Anti-pattern: bcrypt on every single telemetry request
	if !redis.CheckMachineToken(deviceID, token) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid device token"})
		return
	}

	c.Set(string(ctxKeyDeviceID), deviceID)
	c.Next()
}

// hasPermission checks if the user has a permission matching "resource:action".
func hasPermission(user *BigUser, required string) bool {
	parts := strings.SplitN(required, ":", 2)
	if len(parts) != 2 {
		return false
	}
	resource, action := parts[0], parts[1]
	for _, p := range user.Permissions {
		if p.Resource == resource && p.Action == action {
			return true
		}
	}
	return false
}

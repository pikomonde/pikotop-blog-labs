# demo-pprof

A Go service built with Gin to demonstrate two common performance anti-patterns,
and how to profile them using `pprof`.

Used as the companion demo for the article:
> **"How We Saved a High-Traffic IoT Service from 200 RPS to 20,000+ RPS"**

---

## The Two Anti-Patterns

| Path | Header | Anti-Pattern | pprof shows |
|------|--------|--------------|-------------|
| Human | `X-Client-Type: human` | Fetch + unmarshal ~300KB `BigUser` JSON from Redis on every request just to check one permission bit | `json.Unmarshal` consuming 20–30% CPU |
| Machine | `X-Client-Type: machine` | `bcrypt.CompareHashAndPassword` on every telemetry ping | `bcrypt` consuming 90%+ CPU, RPS collapses |

---

## Quickstart

```bash
# 1. Install dependencies
go mod tidy

# 2. Run the server (opens :8080 for API, :6060 for pprof)
go run .

# 3. In another terminal — make scripts executable
chmod +x scripts/*.sh
```

---

## curl Examples

### Health check (no auth)
```bash
curl http://localhost:8080/health
```

### Human client (BigUser JSON unmarshal path)
```bash
curl -X POST http://localhost:8080/v1/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Client-Type: human" \
  -H "X-User-ID: usr-demo-001" \
  -H "X-Required-Permission: telemetry:read" \
  -d '{
    "timestamp": "2026-05-06T10:00:00Z",
    "metrics": {"dashboard_load_ms": 142.5},
    "tags": {"page": "fleet-overview"}
  }'
```

### Machine client (bcrypt hot-path — ⚠ slow by design)
```bash
curl -X POST http://localhost:8080/v1/telemetry \
  -H "Content-Type: application/json" \
  -H "X-Client-Type: machine" \
  -H "X-Device-ID: device-iot-001" \
  -H "X-Device-Token: s3cr3t-t0k3n-device-iot-001" \
  -d '{
    "timestamp": "2026-05-06T10:00:00Z",
    "metrics": {"speed_kmh": 87.5, "rpm": 2340, "fuel_pct": 62.1},
    "tags": {"vehicle_id": "VH-00421", "fleet": "fleet-jakarta-north"}
  }'
```

Valid device credentials (pre-seeded):
| Device ID | Token |
|-----------|-------|
| `device-iot-001` | `s3cr3t-t0k3n-device-iot-001` |
| `device-iot-002` | `s3cr3t-t0k3n-device-iot-002` |
| `device-iot-003` | `s3cr3t-t0k3n-device-iot-003` |

---

## go-wrk Load Tests

Install go-wrk first:
```bash
go install github.com/tsliwowicz/go-wrk@latest
```

### Human path (moderate CPU — json.Unmarshal visible)
```bash
go-wrk \
  -d 30 -c 10 \
  -m POST \
  -H "Content-Type: application/json" \
  -H "X-Client-Type: human" \
  -H "X-User-ID: usr-demo-001" \
  -H "X-Required-Permission: telemetry:read" \
  -body '{"timestamp":"2026-05-06T10:00:00Z","metrics":{"ping":1},"tags":{}}' \
  http://localhost:8080/v1/telemetry
```

### Machine path (CPU saturates — bcrypt dominates flamegraph)
```bash
go-wrk \
  -d 30 -c 10 \
  -m POST \
  -H "Content-Type: application/json" \
  -H "X-Client-Type: machine" \
  -H "X-Device-ID: device-iot-001" \
  -H "X-Device-Token: s3cr3t-t0k3n-device-iot-001" \
  -body '{"timestamp":"2026-05-06T10:00:00Z","metrics":{"speed_kmh":87.5},"tags":{}}' \
  http://localhost:8080/v1/telemetry
```

Or use the provided scripts:
```bash
./scripts/hit_human.sh
./scripts/hit_machine.sh
```

---

## Profiling Workflow

**Step 1:** Start the server
```bash
go run .
```

**Step 2:** Start a load test in Terminal 2
```bash
./scripts/hit_machine.sh   # or hit_human.sh
```

**Step 3:** Capture a CPU profile in Terminal 3 (while load is running)
```bash
./scripts/run_pprof.sh
# or manually:
curl -sf "http://localhost:6060/debug/pprof/profile?seconds=30" -o cpu.pb.gz
```

**Step 4:** Generate the flamegraph
```bash
./scripts/gen_html.sh
# or manually open the interactive UI:
go tool pprof -http=:8888 pprof-output/profile_*.pb.gz
```

Then open **http://localhost:8888** → click **Flame Graph** tab.

### pprof endpoints (while server is running)
| URL | Description |
|-----|-------------|
| http://localhost:6060/debug/pprof/ | Index of all profiles |
| http://localhost:6060/debug/pprof/profile?seconds=30 | 30s CPU profile |
| http://localhost:6060/debug/pprof/heap | Current heap snapshot |
| http://localhost:6060/debug/pprof/goroutine | All goroutine stacks |
| http://localhost:6060/debug/pprof/allocs | Memory allocation profile |
| http://localhost:6060/debug/pprof/trace?seconds=5 | Execution trace |

---

## What to Expect in the Flamegraph

### Machine path (bcrypt)
```
runtime.goexit
  └─ net/http.(*conn).serve
       └─ github.com/gin-gonic/gin.(*Engine).ServeHTTP
            └─ main.AuthMiddleware.func1
                 └─ main.handleMachineAuth
                      └─ (*fakeRedis).CheckMachineToken        ← 95%+ of width
                           └─ golang.org/x/crypto/bcrypt.CompareHashAndPassword
```

### Human path (json.Unmarshal)
```
runtime.goexit
  └─ net/http.(*conn).serve
       └─ ...gin...
            └─ main.handleHumanAuth
                 └─ (*fakeRedis).GetUser
                      └─ encoding/json.Unmarshal               ← 20-30% of width
```

---

## File Structure

```
demo-pprof/
  main.go        ← Gin server, pprof registration, routes
  middleware.go  ← AuthMiddleware: human (json unmarshal) vs machine (bcrypt)
  handlers.go    ← HandleTelemetry, HandleHealthCheck
  mock.go        ← generateMockUser() → ~300KB BigUser JSON
  go.mod
  scripts/
    hit_human.sh    ← curl + go-wrk for human path
    hit_machine.sh  ← curl + go-wrk for machine path (bcrypt)
    run_pprof.sh    ← capture CPU profile during load
    gen_html.sh     ← generate flamegraph HTML/SVG from profile
  pprof-output/     ← created at runtime, gitignored
    *.pb.gz         ← captured profiles
    *_graph.svg     ← generated SVG call graphs
    *_top20.txt     ← top-N text reports
```

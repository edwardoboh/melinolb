# Load Balancing Strategies

This guide explains the available load balancing algorithms, when to use each, and how they work internally.

## Available Strategies

- [Round Robin](#round-robin)
- [Least Connections](#least-connections)

## Round Robin

### Overview

Distributes requests sequentially across all healthy backends in a circular pattern.

### Configuration

```yaml
routes:
  - id: "api"
    match:
      path: "/api"
    backend: "api_servers"
    lb: "round-robin"
```

### How It Works

```
Backends: [A, B, C]

Request 1 → Backend A
Request 2 → Backend B
Request 3 → Backend C
Request 4 → Backend A  (cycles back)
Request 5 → Backend B
Request 6 → Backend C
...
```

### Algorithm Details

1. Maintains a counter (starting at 0)
2. On each request:
   - Filter to healthy backends only
   - Increment counter atomically
   - Select: `counter % healthy_backend_count`
3. Thread-safe using atomic operations

### Best For

✅ **Backends with similar capacity**
- All backends have same CPU, memory, network
- Homogeneous infrastructure

✅ **Stateless applications**
- No session state stored in backends
- Each request is independent

✅ **Short-lived requests**
- Quick API calls (< 1 second)
- Uniform request processing time

✅ **Simple, predictable distribution**
- Equal traffic to all backends
- Easy to reason about behavior

### Not Ideal For

❌ **Long-running requests**
- File uploads/downloads
- Streaming responses
- Varying processing times

❌ **Backends with different capacity**
- Mixed instance sizes
- Different hardware specs

❌ **Connection-based protocols**
- WebSockets
- Long polling
- Server-Sent Events (SSE)

### Performance Characteristics

- **Speed**: Extremely fast (atomic increment + modulo)
- **Memory**: O(1) - single counter
- **Fairness**: Perfect distribution over time
- **Overhead**: Minimal (~nanoseconds per request)

### Example Use Cases

```yaml
# API Gateway - uniform requests
routes:
  - id: "api"
    match:
      path: "/api/v1"
    backend: "api_pool"
    lb: "round-robin"

# Static content serving
routes:
  - id: "assets"
    match:
      path: "/static"
    backend: "cdn_origins"
    lb: "round-robin"

# Microservices - stateless
routes:
  - id: "user-service"
    match:
      path: "/users"
    backend: "user_backends"
    lb: "round-robin"
```

## Least Connections

### Overview

Routes requests to the backend with the fewest active connections. Dynamically adapts to backend load.

### Configuration

```yaml
routes:
  - id: "api"
    match:
      path: "/api"
    backend: "api_servers"
    lb: "least-connections"
```

### How It Works

```
Backends: [A(2 conns), B(5 conns), C(1 conn)]

New Request → Backend C  (fewest: 1)
After: [A(2), B(5), C(2)]

New Request → Backend A  (fewest: 2, ties go to first)
After: [A(3), B(5), C(2)]

New Request → Backend C  (fewest: 2)
After: [A(3), B(5), C(3)]
```

### Algorithm Details

1. Maintains connection counter per backend
2. On each request:
   - Filter to healthy backends only
   - Find backend with minimum active connections
   - Increment its counter
   - Process request
   - Decrement counter when complete
3. Thread-safe with mutex-protected map

### Best For

✅ **Long-running requests**
- File uploads (minutes)
- Report generation (seconds to minutes)
- Video streaming

✅ **Varying request processing times**
- Mixed workload (fast + slow operations)
- Unpredictable request duration

✅ **Backends with different capacity**
- Mixed instance types (small, medium, large)
- Heterogeneous infrastructure
- Naturally directs more traffic to underutilized backends

✅ **Connection-based protocols**
- WebSockets
- HTTP/2 with multiplexing
- Long polling

### Not Ideal For

❌ **Very short requests**
- Overhead of connection tracking
- Round-robin is faster for uniform quick requests

❌ **When backend capacity isn't the bottleneck**
- If database is the constraint
- If external API is the slowdown

### Performance Characteristics

- **Speed**: Fast (map lookup + atomic operations)
- **Memory**: O(n) where n = number of backends
- **Fairness**: Adapts to actual load
- **Overhead**: Low (~microseconds per request)

### Connection Lifecycle

```
Request arrives
  ↓
Select backend with fewest connections
  ↓
Increment connection count
  ↓
Process request (proxy to backend)
  ↓
Response complete
  ↓
Decrement connection count
```

### Example Use Cases

```yaml
# File upload service
routes:
  - id: "uploads"
    match:
      path: "/upload"
    backend: "upload_servers"
    lb: "least-connections"

# WebSocket server
routes:
  - id: "websocket"
    match:
      path: "/ws"
    backend: "websocket_backends"
    lb: "least-connections"

# Report generation
routes:
  - id: "reports"
    match:
      path: "/api/reports"
    backend: "report_workers"
    lb: "least-connections"

# Mixed capacity backends
routes:
  - id: "api"
    match:
      path: "/api"
    backend:
      - "http://small-1:8080"   # 2 CPU
      - "http://medium-1:8080"  # 4 CPU
      - "http://large-1:8080"   # 8 CPU
    lb: "least-connections"  # Large instance naturally handles more
```

## Comparison Table

| Feature | Round Robin | Least Connections |
|---------|-------------|-------------------|
| **Complexity** | Simple | Moderate |
| **Performance** | Fastest | Very Fast |
| **Memory** | Minimal (O(1)) | Low (O(n)) |
| **Best Request Type** | Short, uniform | Long, varying |
| **Adapts to Load** | No | Yes |
| **Backend Capacity Aware** | No | Indirectly |
| **Connection Tracking** | No | Yes |
| **Setup Complexity** | Zero config | Zero config |

## Choosing a Strategy

### Decision Tree

```
Do your requests vary significantly in duration?
├─ Yes → Least Connections
└─ No
    └─ Are backends homogeneous and requests uniform?
        ├─ Yes → Round Robin
        └─ No → Least Connections
```

### Quick Selection Guide

**Use Round Robin when:**
- All backends are identical
- Requests complete in < 1 second
- Simplicity is priority
- Traffic is consistent

**Use Least Connections when:**
- Request duration varies (100ms to 10s+)
- Mixed backend capacity
- WebSockets or long-polling
- Upload/download operations
- Report generation
- When in doubt (it handles most cases well)

## Advanced Considerations

### Health Check Integration

Both strategies automatically exclude unhealthy backends:

```yaml
routes:
  - id: "api"
    backend: "api_servers"
    lb: "least-connections"  # or "round-robin"
    health:
      path: "/health"
      interval: 10s
      timeout: 2s
```

Unhealthy backends:
- Removed from selection pool immediately
- No requests sent to failed backends
- Automatically restored when health checks pass

### Sticky Sessions Impact

Sticky sessions override load balancing strategy:

```yaml
routes:
  - id: "api"
    backend: "api_servers"
    lb: "least-connections"
    sticky:
      enabled: true
      cookie_name: "SESSION"
      ttl: 3600
```

**Behavior:**
- First request: Uses configured strategy (round-robin or least-connections)
- Subsequent requests: Routes to same backend (ignores strategy)
- After TTL expires: Returns to using strategy

### Performance Tuning

**Round Robin:**
```yaml
# No tuning needed - inherently optimal
routes:
  - id: "api"
    backend: "api_servers"
    lb: "round-robin"
```

**Least Connections:**
```yaml
# Consider backend timeouts for accurate tracking
defaults:
  backend_read_timeout: 30s  # Shorter = faster connection release
  
routes:
  - id: "api"
    backend: "api_servers"
    lb: "least-connections"
```

## Monitoring Strategy Effectiveness

Use the metrics endpoint to observe distribution:

```bash
# Check backend health
curl http://127.0.0.1:9000/metrics
```

### What to Look For

**Round Robin:**
- Traffic should be evenly distributed
- All healthy backends should have similar request counts

**Least Connections:**
- Backends with fewer connections should receive more requests
- Distribution adapts to actual load

### Example Monitoring

```bash
# Continuous monitoring
watch -n 1 'curl -s http://127.0.0.1:9000/metrics'
```

## Future Strategies

Planned for future releases:
- **IP Hash**: Consistent routing based on client IP
- **Weighted Round Robin**: Manual traffic distribution control
- **Random**: Simple random selection
- **Least Response Time**: Route to fastest backend

## See Also

- [Configuration Reference](configuration.md#load-balancing-strategy)
- [Health Checks](features.md#health-checks)
- [Sticky Sessions](features.md#sticky-sessions)
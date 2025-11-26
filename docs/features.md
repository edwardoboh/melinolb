# Features Guide

Comprehensive guide to the load balancer's advanced features: health checks, sticky sessions, rate limiting, and retry logic.

## Table of Contents
- [Health Checks](#health-checks)
- [Sticky Sessions](#sticky-sessions)
- [Rate Limiting](#rate-limiting)
- [Retry Logic](#retry-logic)

## Health Checks

### Overview

Health checks monitor backend availability and automatically remove unhealthy backends from the load balancing pool.

### Configuration

```yaml
routes:
  - id: "api"
    match:
      path: "/api"
    backend: "api_servers"
    lb: "round-robin"
    health:
      path: "/healthz"
      interval: 10s
      timeout: 2s
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | HTTP endpoint to check on backend |
| `interval` | duration | Yes | Time between health checks |
| `timeout` | duration | Yes | Request timeout for each check |

### How It Works

1. **Periodic Checks**: Load balancer sends GET request to `backend_url + health.path` every `interval`
2. **Success Criteria**: HTTP status 2xx (200-299) = healthy
3. **Failure Criteria**: Connection error, timeout, or non-2xx status = unhealthy
4. **Automatic Recovery**: Unhealthy backends continue being checked and are restored when passing

### Health Check Flow

```
Load Balancer starts
  ↓
Initial health check (all backends)
  ↓
Every 'interval':
  ├─ For each backend:
  │   ├─ Send GET request to /health
  │   ├─ Wait up to 'timeout'
  │   ├─ Check response status
  │   └─ Update backend health status
  └─ Continue...
```

### Backend Implementation

Your backend services must implement the health endpoint:

#### Python/Flask Example

```python
from flask import Flask
app = Flask(__name__)

@app.route('/healthz')
def health():
    # Check dependencies (database, cache, etc.)
    if database_connected():
        return 'OK', 200
    else:
        return 'Database unavailable', 503

if __name__ == '__main__':
    app.run(port=3001)
```

#### Go Example

```go
http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
    // Check dependencies
    if isDatabaseHealthy() {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("Database down"))
    }
})
```

#### Node.js/Express Example

```javascript
app.get('/healthz', (req, res) => {
  // Check dependencies
  if (dbClient.isConnected()) {
    res.status(200).send('OK');
  } else {
    res.status(503).send('Database unavailable');
  }
});
```

### Health Check Strategies

#### Simple Liveness Check

```yaml
health:
  path: "/health"
  interval: 10s
  timeout: 2s
```

**Backend returns:** Always 200 if process is running

**Use case:** Basic availability monitoring

#### Dependency Check

```python
@app.route('/health')
def health():
    checks = {
        'database': check_database(),
        'cache': check_cache(),
        'external_api': check_external_api()
    }
    
    if all(checks.values()):
        return 'OK', 200
    else:
        return f'Failed: {checks}', 503
```

**Use case:** Ensure all critical dependencies are available

#### Shallow vs Deep Checks

```python
@app.route('/health/shallow')
def shallow_health():
    # Quick check - just verify process is alive
    return 'OK', 200

@app.route('/health/deep')
def deep_health():
    # Comprehensive check - verify all dependencies
    check_database()
    check_cache()
    check_disk_space()
    return 'OK', 200
```

Configure based on check type:
```yaml
# Quick checks every 5s
health:
  path: "/health/shallow"
  interval: 5s
  timeout: 1s

# Or thorough checks every 30s
health:
  path: "/health/deep"
  interval: 30s
  timeout: 5s
```

### Monitoring Health Status

#### Check Current Status

```bash
curl http://127.0.0.1:9000/metrics
```

Output:
```
route_api_healthy_backends 3
route_api_total_backends 3
route_orders_healthy_backends 1
route_orders_total_backends 2
```

#### Continuous Monitoring

```bash
watch -n 2 'curl -s http://127.0.0.1:9000/metrics | grep healthy'
```

### Tuning Health Checks

#### Aggressive (Fast Failure Detection)

```yaml
health:
  path: "/health"
  interval: 5s   # Check frequently
  timeout: 1s    # Fail fast
```

**Pros:**
- Quick failure detection (5s max)
- Minimal time serving bad backends

**Cons:**
- Higher load on backends
- More sensitive to transient issues

**Use for:** Critical services, user-facing APIs

#### Conservative (Stable)

```yaml
health:
  path: "/health"
  interval: 30s  # Check less frequently
  timeout: 5s    # Allow more time
```

**Pros:**
- Lower backend load
- Less sensitive to momentary spikes

**Cons:**
- Slower failure detection (up to 30s)
- May serve unhealthy backend briefly

**Use for:** Internal services, batch processing

### Troubleshooting Health Checks

#### All Backends Marked Unhealthy

**Problem:** `route_X_healthy_backends 0`

**Solutions:**
1. Verify health endpoint exists:
   ```bash
   curl http://backend-url:port/healthz
   ```

2. Check timeout is sufficient:
   ```yaml
   health:
     timeout: 5s  # Increase if backend is slow
   ```

3. Review backend logs for errors

4. Temporarily disable health checks to isolate:
   ```yaml
   routes:
     - id: "api"
       backend: "api_servers"
       # health: {...}  # Commented out
   ```

#### Flapping (Backend Healthy/Unhealthy/Healthy)

**Problem:** Backend status oscillating

**Solutions:**
1. Increase timeout:
   ```yaml
   health:
     timeout: 3s  # Was: 1s
   ```

2. Increase interval:
   ```yaml
   health:
     interval: 20s  # Was: 10s
   ```

3. Fix backend performance issues
4. Check network stability

### Health Check Best Practices

✅ **DO:**
- Implement lightweight health endpoints
- Check critical dependencies only
- Return quickly (< 1 second)
- Use HTTP 200 for healthy, 503 for unhealthy
- Log health check failures in backend

❌ **DON'T:**
- Make health checks expensive (complex queries)
- Check non-critical dependencies
- Return 200 when actually unhealthy
- Ignore health check errors

## Sticky Sessions

### Overview

Sticky sessions (session affinity) ensure requests from the same client are routed to the same backend server.

### Configuration

```yaml
routes:
  - id: "api"
    match:
      path: "/api"
    backend: "api_servers"
    lb: "round-robin"
    sticky:
      enabled: true
      cookie_name: "BACKEND_ID"
      ttl: 3600
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `enabled` | bool | Yes | Enable sticky sessions |
| `cookie_name` | string | Yes | Name of the cookie to use |
| `ttl` | int | Yes | Cookie lifetime in seconds |

### How It Works

1. **First Request** (no cookie):
   - Load balancer selects backend using configured strategy
   - Sets cookie: `Set-Cookie: BACKEND_ID=backend-host:port; Max-Age=3600`
   - Forwards request to selected backend

2. **Subsequent Requests** (with cookie):
   - Client includes cookie in request
   - Load balancer reads cookie value
   - Routes directly to that backend (bypasses LB strategy)

3. **After TTL Expires**:
   - Cookie expires on client
   - Next request treated as first request
   - New backend selected

### Flow Diagram

```
Client Request 1 (no cookie)
  ↓
Load Balancer
  ├─ Select backend (round-robin/least-connections)
  ├─ Set cookie: BACKEND_ID=backend-1:3001
  └─ Forward to backend-1
      ↓
Client receives: Set-Cookie: BACKEND_ID=backend-1:3001; Max-Age=3600

Client Request 2 (includes cookie: BACKEND_ID=backend-1:3001)
  ↓
Load Balancer
  ├─ Read cookie
  ├─ Extract backend: backend-1:3001
  └─ Forward to backend-1 (skip LB algorithm)
```

### Use Cases

#### Shopping Cart (In-Memory Sessions)

```yaml
routes:
  - id: "shop"
    match:
      path: "/shop"
    backend: "shop_servers"
    lb: "round-robin"
    sticky:
      enabled: true
      cookie_name: "SHOP_SESSION"
      ttl: 1800  # 30 minutes
```

#### WebSocket Connections

```yaml
routes:
  - id: "websocket"
    match:
      path: "/ws"
    backend: "ws_servers"
    lb: "least-connections"
    sticky:
      enabled: true
      cookie_name: "WS_BACKEND"
      ttl: 86400  # 24 hours
```

#### Stateful APIs

```yaml
routes:
  - id: "api"
    match:
      path: "/api"
    backend: "api_servers"
    lb: "round-robin"
    sticky:
      enabled: true
      cookie_name: "API_SESSION"
      ttl: 3600  # 1 hour
```

### Cookie Details

**Set-Cookie Header:**
```
Set-Cookie: BACKEND_ID=10.0.1.11:3001; Max-Age=3600; Path=/; HttpOnly
```

**Attributes:**
- `Max-Age`: TTL in seconds
- `Path`: `/` (applies to all paths)
- `HttpOnly`: Cannot be accessed by JavaScript (security)

### Testing Sticky Sessions

```bash
# First request - get cookie
curl -v http://localhost:8080/api 2>&1 | grep Set-Cookie
# Output: Set-Cookie: BACKEND_ID=localhost:3001; Max-Age=3600

# Subsequent request - use cookie
curl -H "Cookie: BACKEND_ID=localhost:3001" http://localhost:8080/api
# Always routes to localhost:3001

# Different cookie - different backend
curl -H "Cookie: BACKEND_ID=localhost:3002" http://localhost:8080/api
# Always routes to localhost:3002
```

### Sticky Sessions + Health Checks

**What happens if pinned backend becomes unhealthy?**

Current behavior: **Request fails**

```yaml
routes:
  - id: "api"
    backend: "api_servers"
    sticky:
      enabled: true
      cookie_name: "SESSION"
      ttl: 3600
    health:
      path: "/health"
      interval: 10s
      timeout: 2s
```

If backend-1 fails:
- Client with `SESSION=backend-1` → Request fails
- Client gets error (503 or connection error)
- Client retries → New cookie issued → Routes to healthy backend

**Future enhancement:** Automatic failover to healthy backend

### Tuning TTL

#### Short TTL (Minutes)

```yaml
sticky:
  enabled: true
  cookie_name: "SESSION"
  ttl: 300  # 5 minutes
```

**Pros:**
- Better load distribution
- Faster recovery from backend failures
- Less impact from unbalanced distribution

**Cons:**
- More frequent session migrations
- May impact user experience if state is lost

**Use for:** Light session state, public APIs

#### Long TTL (Hours/Days)

```yaml
sticky:
  enabled: true
  cookie_name: "SESSION"
  ttl: 86400  # 24 hours
```

**Pros:**
- Consistent user experience
- Less overhead (fewer cookie sets)
- Better for persistent connections

**Cons:**
- Uneven load distribution
- Longer impact from failed backends
- Stale backend associations

**Use for:** WebSockets, heavy session state, authenticated users

### Sticky Sessions vs Stateless Design

**When NOT to use sticky sessions:**

❌ **Stateless applications**
```yaml
# Don't use sticky sessions if:
# - Session state in database/Redis
# - No user-specific caching
# - True stateless microservices
routes:
  - id: "stateless-api"
    backend: "api_servers"
    lb: "round-robin"
    # sticky: {...}  # Not needed!
```

**When to use sticky sessions:**

✅ **In-memory session state**
✅ **WebSocket/SSE connections**
✅ **User-specific caching**
✅ **Legacy applications**

### Best Practices

✅ **DO:**
- Use shared storage (Redis/DB) instead when possible
- Set appropriate TTL based on session importance
- Monitor backend health
- Implement session migration in application

❌ **DON'T:**
- Use sticky sessions as substitute for proper session management
- Set extremely long TTL (weeks/months)
- Rely on sticky sessions for critical state
- Ignore backend failures

## Rate Limiting

### Overview

Rate limiting protects backends from being overwhelmed by limiting requests per second.

### Configuration

```yaml
routes:
  - id: "api"
    match:
      path: "/api"
    backend: "api_servers"
    lb: "round-robin"
    rate_limit:
      enabled: true
      requests_per_second: 100
      burst: 200
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `enabled` | bool | Yes | Enable rate limiting |
| `requests_per_second` | int | Yes | Sustained request rate |
| `burst` | int | Yes | Maximum burst capacity |

### Token Bucket Algorithm

Rate limiting uses the **token bucket algorithm**:

1. **Bucket**: Holds tokens (max = `burst`)
2. **Tokens**: Added at `requests_per_second` rate
3. **Request**: Consumes one token
4. **Allow**: If token available
5. **Reject**: If bucket empty (429 status)

### How It Works

```
Initial State:
  Bucket: [🪙🪙🪙🪙🪙...] (200 tokens, full burst)

100 requests arrive simultaneously:
  Bucket: [🪙🪙🪙...] (100 tokens consumed, 100 remain)
  Status: All 100 allowed

100 more requests arrive immediately:
  Bucket: [] (100 tokens consumed, bucket empty)
  Status: All 100 allowed

Next request (bucket empty):
  Status: 429 Too Many Requests

After 1 second:
  Bucket: [🪙🪙🪙...] (100 tokens refilled at 100/sec rate)
  Status: Next 100 requests allowed
```

### Configuration Examples

#### Strict Rate Limiting

```yaml
rate_limit:
  enabled: true
  requests_per_second: 10
  burst: 15
```

**Behavior:**
- Sustained: 10 requests/second
- Can handle bursts up to 15 requests
- Very protective of backends

**Use for:** External/public APIs, preventing abuse

#### Lenient Rate Limiting

```yaml
rate_limit:
  enabled: true
  requests_per_second: 1000
  burst: 2000
```

**Behavior:**
- Sustained: 1000 requests/second
- Can handle large bursts (2000 requests)
- Allows traffic spikes

**Use for:** Internal APIs, trusted clients

#### Burst-Focused

```yaml
rate_limit:
  enabled: true
  requests_per_second: 50
  burst: 500
```

**Behavior:**
- Moderate sustained rate
- Large burst capacity (10x)
- Absorbs traffic spikes well

**Use for:** APIs with bursty traffic patterns

### Response to Rate-Limited Requests

When limit exceeded:

**HTTP Response:**
```
HTTP/1.1 429 Too Many Requests
Content-Type: text/plain

Too Many Requests
```

**Client should:**
- Implement exponential backoff
- Retry after delay
- Respect rate limits

### Testing Rate Limiting

```bash
# Send burst of requests
for i in {1..50}; do
  curl -w "%{http_code}\n" -s http://localhost:8080/api -o /dev/null
done

# Expected output (with rps:10, burst:20):
# 200 (first 20 allowed - burst)
# 200
# ...
# 200
# 429 (rejected - burst exhausted)
# 429
# ...
```

### Per-Route vs Global

Rate limiting is **per-route**:

```yaml
routes:
  - id: "public-api"
    match:
      path: "/api/public"
    backend: "api_servers"
    rate_limit:
      enabled: true
      requests_per_second: 10  # 10/sec for this route
      burst: 20

  - id: "admin-api"
    match:
      path: "/api/admin"
    backend: "api_servers"
    rate_limit:
      enabled: true
      requests_per_second: 100  # 100/sec for this route
      burst: 200
```

Each route has independent rate limits.

### Monitoring Rate Limits

**Enable debug logging:**
```yaml
logging:
  level: debug
```

**Logs show:**
- Allowed requests
- Rejected requests (429)
- Token bucket state

### Tuning Guidelines

| Scenario | RPS | Burst | Reasoning |
|----------|-----|-------|-----------|
| Public API | 10-50 | 2x RPS | Prevent abuse |
| Internal API | 100-1000 | 2-5x RPS | Allow legitimate traffic |
| High-traffic | 1000+ | 2x RPS | Scale with capacity |
| Bursty traffic | 50-100 | 10x RPS | Absorb spikes |

### Limitations

Current implementation:
- ⚠️ **Global per route** (not per-client)
- ⚠️ All clients share the same bucket
- ⚠️ No IP-based or user-based limiting

**Future:** Per-client rate limiting

### Best Practices

✅ **DO:**
- Set conservative limits initially
- Monitor 429 responses
- Adjust based on backend capacity
- Document limits for API consumers

❌ **DON'T:**
- Set limits too low (poor UX)
- Set limits too high (no protection)
- Ignore 429 error rates
- Apply same limits to all routes

## Retry Logic

### Overview

Automatically retry failed requests to improve reliability.

### Configuration

```yaml
routes:
  - id: "api"
    match:
      path: "/api"
    backend: "api_servers"
    lb: "round-robin"
    retry:
      enabled: true
      max_retries: 2
      per_try_timeout: 5s
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `enabled` | bool | Yes | Enable retry logic |
| `max_retries` | int | Yes | Maximum retry attempts |
| `per_try_timeout` | duration | Yes | Timeout for each attempt |

### How It Works

```
Request arrives
  ↓
Attempt 1: Forward to backend
  ├─ Success → Return response
  └─ Failure (timeout, 5xx, connection error)
      ↓
  Attempt 2: Forward to backend (may be different)
      ├─ Success → Return response
      └─ Failure
          ↓
      Attempt 3: Forward to backend
          ├─ Success → Return response
          └─ Failure → Return error to client
```

### Retry Conditions

Requests are retried on:
- ✅ Connection errors (backend unreachable)
- ✅ Timeouts (backend not responding)
- ✅ HTTP 5xx errors (if `defaults.retry_on_5xx: true`)

Requests are NOT retried on:
- ❌ HTTP 2xx (success)
- ❌ HTTP 3xx (redirect)
- ❌ HTTP 4xx (client error)
- ❌ HTTP 5xx (if `defaults.retry_on_5xx: false`)

### Configuration Examples

#### Conservative Retries

```yaml
defaults:
  retry_on_5xx: false

routes:
  - id: "api"
    backend: "api_servers"
    retry:
      enabled: true
      max_retries: 1
      per_try_timeout: 3s
```

**Behavior:**
- Only retry connection errors/timeouts
- Single retry attempt
- Quick timeout per attempt

**Use for:** Idempotent GET requests

#### Aggressive Retries

```yaml
defaults:
  retry_on_5xx: true
  
routes:
  - id: "api"
    backend: "api_servers"
    retry:
      enabled: true
      max_retries: 3
      per_try_timeout: 10s
```

**Behavior:**
- Retry on 5xx errors too
- Multiple retry attempts
- Longer timeout per attempt

**Use for:** Critical read operations

### Retry + Load Balancing

Retries work with load balancing strategies:

**Round Robin:**
```
Request fails on backend-1
  ↓
Retry selects backend-2 (next in rotation)
  ↓
If fails, retry selects backend-3
```

**Least Connections:**
```
Request fails on backend-1
  ↓
Retry selects backend with fewest connections
  ↓
May be same or different backend
```

### Idempotency Considerations

⚠️ **Critical:** Only retry idempotent operations

**Safe to retry:**
- ✅ GET requests (read-only)
- ✅ HEAD requests
- ✅ OPTIONS requests
- ✅ Idempotent POST (with idempotency keys)

**Dangerous to retry:**
- ❌ POST (creates resources)
- ❌ PUT (updates)
- ❌ DELETE (removes)
- ❌ Any non-idempotent operation

**Configuration by method:**
```yaml
# GET - safe to retry
routes:
  - id: "read-api"
    match:
      path: "/api"
      methods: ["GET"]
    backend: "api_servers"
    retry:
      enabled: true
      max_retries: 3

# POST/PUT/DELETE - no retries
  - id: "write-api"
    match:
      path: "/api"
      methods: ["POST", "PUT", "DELETE"]
    backend: "api_servers"
    # retry: {...}  # Omit retries
```

### Timeout Calculation

**Total request timeout:**
```
Total = (max_retries + 1) × per_try_timeout

Example:
max_retries: 2
per_try_timeout: 5s
Total: (2 + 1) × 5s = 15s maximum
```

Configure client timeouts accordingly:
```
Client timeout > Load balancer total timeout
```

### Best Practices

✅ **DO:**
- Enable for GET requests
- Set reasonable retry counts (1-3)
- Use shorter timeouts per attempt
- Monitor retry rates
- Implement idempotency in backends

❌ **DON'T:**
- Retry non-idempotent operations
- Set excessive retry counts (> 5)
- Use same timeout as non-retry requests
- Retry on 4xx errors
- Ignore retry amplification effects

### Monitoring Retries

Enable debug logging to see retries:

```yaml
logging:
  level: debug
```

**Log output:**
```
[debug] Attempt 1 failed: connection refused
[debug] Retrying request (attempt 2/3)
[debug] Attempt 2 succeeded
```

## See Also

- [Configuration Reference](configuration.md)
- [Load Balancing Strategies](load-balancing.md)
- [Operations Guide](operations.md)
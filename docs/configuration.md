# Configuration Reference

Complete reference for all configuration options in the load balancer.

## Table of Contents
- [Configuration File Format](#configuration-file-format)
- [Service Configuration](#service-configuration)
- [TLS Configuration](#tls-configuration)
- [Logging Configuration](#logging-configuration)
- [Admin Interface](#admin-interface)
- [Default Settings](#default-settings)
- [Backend Pools](#backend-pools)
- [Routes](#routes)

## Configuration File Format

The load balancer uses YAML format for configuration. The main sections are:
```yaml
service:      # Service identification and listening address
tls:          # TLS/SSL configuration
logging:      # Logging settings
admin:        # Admin interface configuration
defaults:     # Default timeouts and retry settings
backends:     # Named backend pools
routes:       # Routing rules
```

## Service Configuration

Defines basic service information and the main listening address.
```yaml
service:
  name: "edge-proxy"        # Service name for logging/identification
  env: "production"         # Environment (production, staging, dev)
  listen: "0.0.0.0:8080"   # Address and port to listen on
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Identifier for the service in logs |
| `env` | string | Yes | Environment name (production, staging, dev) |
| `listen` | string | Yes | IP:PORT format. Use `0.0.0.0` for all interfaces |

### Examples
```yaml
# Listen on all interfaces, port 8080
service:
  name: "my-proxy"
  env: "production"
  listen: "0.0.0.0:8080"

# Listen on localhost only
service:
  name: "dev-proxy"
  env: "development"
  listen: "127.0.0.1:8080"

# Listen on specific IP
service:
  name: "internal-proxy"
  env: "staging"
  listen: "10.0.1.50:8080"
```

## TLS Configuration

Enable HTTPS/TLS termination at the load balancer.
```yaml
tls:
  enabled: true
  cert_path: "/etc/ssl/certs/server.crt"
  key_path: "/etc/ssl/private/server.key"
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | Yes | Set to `true` to enable TLS |
| `cert_path` | string | If enabled | Path to TLS certificate file (PEM format) |
| `key_path` | string | If enabled | Path to TLS private key file (PEM format) |

### Notes

- When enabled, the service listens on HTTPS instead of HTTP
- Certificate and key files must be readable by the process
- Use Let's Encrypt or your CA for production certificates
- Supports TLS 1.2 and TLS 1.3

### Examples
```yaml
# Disabled TLS (HTTP only)
tls:
  enabled: false
  cert_path: ""
  key_path: ""

# Enabled with Let's Encrypt certificates
tls:
  enabled: true
  cert_path: "/etc/letsencrypt/live/example.com/fullchain.pem"
  key_path: "/etc/letsencrypt/live/example.com/privkey.pem"
```

## Logging Configuration

Control logging verbosity and output destinations.
```yaml
logging:
  level: info              # debug | info | warn | error
  access_log: "-"          # "-" for stdout, or file path
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `level` | string | `info` | Minimum log level to output |
| `access_log` | string | `"-"` | Access log destination |

### Log Levels

| Level | Description | Use Case |
|-------|-------------|----------|
| `debug` | Verbose logging including backend selection | Development, troubleshooting |
| `info` | General operational information | Normal operation |
| `warn` | Warning messages only | Production with selective logging |
| `error` | Error messages only | Production, minimal logging |

### Access Log Destinations

- `"-"`: Write to stdout (default)
- `/path/to/file.log`: Write to specified file
- File is created if it doesn't exist
- Log rotation should be handled externally (e.g., logrotate)

### Access Log Format
```
METHOD PATH STATUS DURATION BACKEND
GET /api/users 200 15ms backend-1:3001
POST /api/orders 201 42ms backend-2:4001
GET /api/payments 503 2ms -
```
### Examples

```yaml
# Development: debug logs to stdout
logging:
  level: debug
  access_log: "-"

# Production: info logs, file-based access log
logging:
  level: info
  access_log: "/var/log/loadbalancer/access.log"

# Minimal logging
logging:
  level: error
  access_log: "/var/log/loadbalancer/access.log"
```

## Admin Interface

Expose operational endpoints for monitoring and health checks.

```yaml
admin:
  enabled: true
  listen: "127.0.0.1:9000"
  health_endpoint: "/healthz"
  metrics_endpoint: "/metrics"
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable/disable admin server |
| `listen` | string | - | Admin server address |
| `health_endpoint` | string | `"/healthz"` | URL path for health checks |
| `metrics_endpoint` | string | `"/metrics"` | URL path for metrics |

### Security Note

**Always bind admin interface to localhost (`127.0.0.1`) in production** to prevent external access to operational endpoints.

### Endpoints

#### Health Check: `GET /healthz`

Returns HTTP 200 if the load balancer is running.

```bash
curl http://127.0.0.1:9000/healthz
# Response: OK
```

**Use cases:**
- Container orchestration health checks (Kubernetes, Docker)
- Monitoring systems (Nagios, Prometheus)
- Load balancer health checks (upstream LB checking this LB)

#### Metrics: `GET /metrics`

Returns metrics about backend health and routing status.

```bash
curl http://127.0.0.1:9000/metrics
```

**Response format:**
```
route_users_healthy_backends 2
route_users_total_backends 2
route_orders_healthy_backends 1
route_orders_total_backends 2
```

### Examples

```yaml
# Enabled for production (localhost only)
admin:
  enabled: true
  listen: "127.0.0.1:9000"
  health_endpoint: "/healthz"
  metrics_endpoint: "/metrics"

# Disabled for minimal setup
admin:
  enabled: false

# Custom endpoints
admin:
  enabled: true
  listen: "127.0.0.1:8888"
  health_endpoint: "/health"
  metrics_endpoint: "/_metrics"
```

## Default Settings

Global timeout and retry settings applied to all routes unless overridden.

```yaml
defaults:
  read_timeout: 30s
  write_timeout: 30s
  backend_connect_timeout: 3s
  backend_read_timeout: 10s
  retry_on_5xx: false
  max_retries: 2
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `read_timeout` | duration | `30s` | Maximum time to read incoming request |
| `write_timeout` | duration | `30s` | Maximum time to write response |
| `backend_connect_timeout` | duration | `3s` | Timeout for establishing backend connections |
| `backend_read_timeout` | duration | `10s` | Timeout for reading from backend |
| `retry_on_5xx` | bool | `false` | Whether to retry requests on 5xx errors |
| `max_retries` | int | `2` | Maximum retry attempts |

### Duration Format

Durations are specified as strings with units:
- `s`: seconds (`30s`)
- `m`: minutes (`5m`)
- `h`: hours (`1h`)
- `ms`: milliseconds (`500ms`)
- Combinations: `1h30m`, `2m30s`

### Timeout Flow

```
Client Request → [read_timeout] → Load Balancer → [backend_connect_timeout] → Backend
                                                  → [backend_read_timeout] → Response
Client ← [write_timeout] ← Load Balancer ← Backend
```

### Examples

```yaml
# Aggressive timeouts for fast APIs
defaults:
  read_timeout: 10s
  write_timeout: 10s
  backend_connect_timeout: 1s
  backend_read_timeout: 5s
  retry_on_5xx: true
  max_retries: 2

# Lenient timeouts for slow operations
defaults:
  read_timeout: 60s
  write_timeout: 60s
  backend_connect_timeout: 5s
  backend_read_timeout: 30s
  retry_on_5xx: false
  max_retries: 0
```

## Backend Pools

Define named groups of backend servers that routes can reference.

```yaml
backends:
  users:
    - "http://10.0.1.11:3001"
    - "http://10.0.1.12:3001"
    - "http://10.0.1.13:3001"
  orders:
    - "http://10.0.2.21:4001"
    - "http://10.0.2.22:4001"
```

### Format

```yaml
backends:
  <pool-name>:
    - "<backend-url>"
    - "<backend-url>"
    ...
```

### Backend URL Format

- Protocol: `http://` or `https://`
- Host: IP address, hostname, or domain
- Port: Required
- Examples:
  - `http://localhost:3000`
  - `http://192.168.1.10:8080`
  - `https://api.internal.example.com:443`

### Usage

Named pools can be referenced by multiple routes:

```yaml
backends:
  api_servers:
    - "http://api-1:8080"
    - "http://api-2:8080"

routes:
  - id: "users"
    match:
      path: "/api/users"
    backend: "api_servers"  # References the pool

  - id: "orders"
    match:
      path: "/api/orders"
    backend: "api_servers"  # Same pool reused
```

### Benefits

- **DRY principle**: Define once, reference multiple times
- **Maintainability**: Update backends in one place
- **Readability**: Named pools are self-documenting

### Examples

```yaml
# Simple pool
backends:
  web:
    - "http://localhost:3000"

# Multiple environments
backends:
  prod_api:
    - "http://10.0.1.11:8080"
    - "http://10.0.1.12:8080"
    - "http://10.0.1.13:8080"
  staging_api:
    - "http://10.0.2.11:8080"
    - "http://10.0.2.12:8080"

# Service-based organization
backends:
  user_service:
    - "http://users-1.internal:3001"
    - "http://users-2.internal:3001"
  order_service:
    - "http://orders-1.internal:4001"
    - "http://orders-2.internal:4001"
  payment_service:
    - "http://payments-1.internal:5001"
```

## Routes

Routes define how incoming requests are matched and forwarded to backends.

```yaml
routes:
  - id: "users-api"
    match:
      host: "api.example.com"
      path: "/api/users"
      methods: ["GET", "POST"]
    backend: "users"
    lb: "round-robin"
```

### Route Structure

```yaml
routes:
  - id: "<unique-identifier>"
    match:
      host: "<hostname>"        # Optional
      path: "<path-prefix>"     # Optional
      methods: [...]            # Optional
    backend: "<pool-or-url>"
    lb: "<strategy>"
    sticky: {...}               # Optional
    health: {...}               # Optional
    retry: {...}                # Optional
    rate_limit: {...}           # Optional
```

### Routing Logic

1. Routes are evaluated **top to bottom**
2. **First matching route wins**
3. Place more specific routes first
4. Use `path: "/"` as catch-all (place last)

### Match Configuration

At least one match criterion must be specified.

```yaml
match:
  host: "api.example.com"     # Optional: Match Host header
  path: "/api/users"          # Optional: Match URL path (prefix)
  methods: ["GET", "POST"]    # Optional: Match HTTP methods
```

#### Host Matching

- Matches the HTTP `Host` header
- Case-insensitive
- If omitted, matches all hosts

```yaml
# Match specific host
match:
  host: "api.example.com"

# Match any host (omit field)
match:
  path: "/api"
```

#### Path Matching

- **Prefix matching**: `/api` matches `/api`, `/api/users`, `/api/orders`, etc.
- Case-sensitive
- Always starts with `/`

```yaml
# Match /api and all sub-paths
match:
  path: "/api"

# Match root and all paths
match:
  path: "/"

# Match exact path (use specific path)
match:
  path: "/api/v1/users/profile"
```

#### Method Matching

- Array of HTTP methods
- Case-sensitive (use uppercase)
- If omitted, matches all methods

```yaml
# Match specific methods
match:
  methods: ["GET", "POST"]

# Match single method
match:
  methods: ["POST"]

# Match all methods (omit field)
match:
  path: "/api"
```

### Backend Configuration

Three ways to specify backends:

#### 1. Named Pool Reference

```yaml
backend: "users"  # References backends.users
```

#### 2. Inline Single Backend

```yaml
backend: "http://localhost:5000"
```

#### 3. Inline Multiple Backends

```yaml
backend:
  - "http://10.0.1.11:3001"
  - "http://10.0.1.12:3001"
```

### Load Balancing Strategy

```yaml
lb: "round-robin"  # or "least-connections"
```

See [Load Balancing Strategies](load-balancing.md) for details.

### Complete Route Example

```yaml
routes:
  - id: "production-api"
    match:
      host: "api.example.com"
      path: "/api/v1"
      methods: ["GET", "POST", "PUT", "DELETE"]
    backend: "api_pool"
    lb: "least-connections"
    sticky:
      enabled: true
      cookie_name: "SESSION_ID"
      ttl: 3600
    health:
      path: "/health"
      interval: 10s
      timeout: 2s
    retry:
      enabled: true
      max_retries: 2
      per_try_timeout: 5s
    rate_limit:
      enabled: true
      requests_per_second: 100
      burst: 200
```

### Route Examples

```yaml
# Simple route
routes:
  - id: "api"
    match:
      path: "/api"
    backend: "http://localhost:3000"
    lb: "round-robin"

# Host-based routing
routes:
  - id: "api-host"
    match:
      host: "api.example.com"
      path: "/"
    backend: "api_servers"
    lb: "round-robin"

  - id: "admin-host"
    match:
      host: "admin.example.com"
      path: "/"
    backend: "admin_servers"
    lb: "round-robin"

# Method-specific routes
routes:
  - id: "read-api"
    match:
      path: "/api"
      methods: ["GET"]
    backend: "read_replicas"
    lb: "round-robin"

  - id: "write-api"
    match:
      path: "/api"
      methods: ["POST", "PUT", "DELETE"]
    backend: "primary_servers"
    lb: "least-connections"

# Catch-all fallback
routes:
  - id: "users"
    match:
      path: "/api/users"
    backend: "user_service"
    lb: "round-robin"

  - id: "orders"
    match:
      path: "/api/orders"
    backend: "order_service"
    lb: "round-robin"

  - id: "default"
    match:
      path: "/"
    backend: "http://frontend:8080"
    lb: "round-robin"
```

## Advanced Route Features

For detailed information on these features, see:
- [Sticky Sessions](features.md#sticky-sessions)
- [Health Checks](features.md#health-checks)
- [Retry Configuration](features.md#retry-logic)
- [Rate Limiting](features.md#rate-limiting)

### Quick Reference

```yaml
# Sticky Sessions
sticky:
  enabled: true
  cookie_name: "BACKEND_ID"
  ttl: 3600

# Health Checks
health:
  path: "/healthz"
  interval: 10s
  timeout: 2s

# Retry Logic
retry:
  enabled: true
  max_retries: 2
  per_try_timeout: 5s

# Rate Limiting
rate_limit:
  enabled: true
  requests_per_second: 100
  burst: 200
```

## Configuration Validation

The load balancer validates configuration on startup:

- ✅ Valid YAML syntax
- ✅ All required fields present
- ✅ Valid backend URLs
- ✅ Positive timeout values
- ✅ Valid duration formats
- ✅ Non-empty route backends
- ✅ Unique route IDs (recommended)

**Startup will fail with clear error messages if validation fails.**

## See Also

- [Getting Started Guide](getting-started.md) - Step-by-step tutorial
- [Load Balancing Strategies](load-balancing.md) - Algorithm details
- [Features Guide](features.md) - Health checks, rate limiting, sticky sessions
- [Operations Guide](operations.md) - Monitoring and troubleshooting
- [Examples Directory](../examples/) - Ready-to-use configurations
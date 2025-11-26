# Getting Started

This guide will walk you through setting up your first load balancer, from basic configuration to production-ready deployment.

## Prerequisites

- Load balancer binary installed (see main README)
- Basic understanding of HTTP/HTTPS
- Backend services to load balance (or use our test setup)

## Step 1: Your First Load Balancer

Let's start with the simplest possible configuration.

### Create config.yaml

```yaml
service:
  name: "my-first-proxy"
  env: "development"
  listen: "0.0.0.0:8080"

logging:
  level: info
  access_log: "-"

routes:
  - id: "default"
    match:
      path: "/"
    backend: "http://localhost:3000"
    lb: "round-robin"
```

### Start the Load Balancer

```bash
melinolb config.yaml
```

You should see:
```
[info] Starting my-first-proxy (development) on 0.0.0.0:8080
```

### Test It

```bash
curl http://localhost:8080
```

## Step 2: Adding Multiple Backends

Let's add multiple backend servers and see load balancing in action.

### Update config.yaml

```yaml
service:
  name: "multi-backend-proxy"
  env: "development"
  listen: "0.0.0.0:8080"

logging:
  level: debug  # Enable debug to see backend selection
  access_log: "-"

backends:
  app_servers:
    - "http://localhost:3001"
    - "http://localhost:3002"
    - "http://localhost:3003"

routes:
  - id: "app"
    match:
      path: "/"
    backend: "app_servers"
    lb: "round-robin"
```

### Start Test Backends

```bash
# Terminal 1
python3 -m http.server 3001

# Terminal 2
python3 -m http.server 3002

# Terminal 3
python3 -m http.server 3003
```

### Start Load Balancer

```bash
./melinolb config.yaml
```

### Watch Load Distribution

```bash
# Send multiple requests
for i in {1..9}; do
  curl -s http://localhost:8080 > /dev/null
done
```

Check the debug logs - you'll see requests distributed evenly across all three backends.

## Step 3: Path-Based Routing

Route different paths to different backend services.

### config.yaml

```yaml
service:
  name: "api-gateway"
  env: "development"
  listen: "0.0.0.0:8080"

logging:
  level: info
  access_log: "-"

backends:
  users:
    - "http://localhost:3001"
    - "http://localhost:3002"
  orders:
    - "http://localhost:4001"
    - "http://localhost:4002"

routes:
  - id: "users-api"
    match:
      path: "/api/users"
    backend: "users"
    lb: "round-robin"

  - id: "orders-api"
    match:
      path: "/api/orders"
    backend: "orders"
    lb: "round-robin"

  - id: "default"
    match:
      path: "/"
    backend: "http://localhost:5000"
    lb: "round-robin"
```

### Test Routing

```bash
# Routes to users backend
curl http://localhost:8080/api/users

# Routes to orders backend
curl http://localhost:8080/api/orders

# Routes to default backend
curl http://localhost:8080/
```

## Step 4: Health Checks

Add health monitoring to automatically remove failing backends.

### Update config.yaml

```yaml
service:
  name: "health-aware-proxy"
  env: "development"
  listen: "0.0.0.0:8080"

logging:
  level: info
  access_log: "-"

admin:
  enabled: true
  listen: "127.0.0.1:9000"
  health_endpoint: "/healthz"
  metrics_endpoint: "/metrics"

backends:
  app:
    - "http://localhost:3001"
    - "http://localhost:3002"
    - "http://localhost:3003"

routes:
  - id: "app"
    match:
      path: "/"
    backend: "app"
    lb: "round-robin"
    health:
      path: "/health"
      interval: 5s
      timeout: 2s
```

### Add Health Endpoint to Backends

Your backends need to implement a health endpoint. Here's a simple Python example:

```python
# health_server.py
from http.server import HTTPServer, BaseHTTPRequestHandler

class HealthHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b'OK')
        else:
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b'Hello from backend!')

if __name__ == '__main__':
    import sys
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 3001
    server = HTTPServer(('localhost', port), HealthHandler)
    print(f'Server running on port {port}')
    server.serve_forever()
```

### Start and Test

```bash
# Start backends
python3 health_server.py 3001 &
python3 health_server.py 3002 &
python3 health_server.py 3003 &

# Start load balancer
./melinolb config.yaml

# Check metrics
curl http://127.0.0.1:9000/metrics
```

You should see:
```
route_app_healthy_backends 3
route_app_total_backends 3
```

### Simulate Backend Failure

```bash
# Kill one backend
pkill -f "3001"

# Wait 5-10 seconds for health check to detect failure
sleep 10

# Check metrics again
curl http://127.0.0.1:9000/metrics
```

Now you'll see:
```
route_app_healthy_backends 2
route_app_total_backends 3
```

Requests will only go to healthy backends!

## Step 5: Rate Limiting

Protect your backends from overload.

### Update config.yaml

```yaml
routes:
  - id: "api"
    match:
      path: "/api"
    backend: "app"
    lb: "round-robin"
    rate_limit:
      enabled: true
      requests_per_second: 10
      burst: 20
```

### Test Rate Limiting

```bash
# Send rapid requests
for i in {1..30}; do
  curl -w "%{http_code}\n" -s http://localhost:8080/api -o /dev/null
done
```

You'll see a mix of `200` (success) and `429` (Too Many Requests).

## Step 6: Production Configuration

Here's a production-ready configuration with all features:

### config.yaml

```yaml
service:
  name: "production-lb"
  env: "production"
  listen: "0.0.0.0:443"

tls:
  enabled: true
  cert_path: "/etc/ssl/certs/server.crt"
  key_path: "/etc/ssl/private/server.key"

logging:
  level: warn
  access_log: "/var/log/loadbalancer/access.log"

admin:
  enabled: true
  listen: "127.0.0.1:9000"
  health_endpoint: "/healthz"
  metrics_endpoint: "/metrics"

defaults:
  read_timeout: 30s
  write_timeout: 30s
  backend_connect_timeout: 3s
  backend_read_timeout: 10s
  retry_on_5xx: true
  max_retries: 2

backends:
  api:
    - "http://10.0.1.11:8080"
    - "http://10.0.1.12:8080"
    - "http://10.0.1.13:8080"

routes:
  - id: "api"
    match:
      host: "api.example.com"
      path: "/"
    backend: "api"
    lb: "least-connections"
    health:
      path: "/health"
      interval: 10s
      timeout: 2s
    rate_limit:
      enabled: true
      requests_per_second: 1000
      burst: 2000
    retry:
      enabled: true
      max_retries: 2
      per_try_timeout: 5s
```

## Common Patterns

### Development Setup

```yaml
service:
  listen: "127.0.0.1:8080"  # Localhost only

logging:
  level: debug               # Verbose logging

routes:
  - id: "dev"
    match:
      path: "/"
    backend: "http://localhost:3000"
    lb: "round-robin"
```

### Staging Environment

```yaml
service:
  name: "staging-lb"
  env: "staging"
  listen: "0.0.0.0:8080"

logging:
  level: info
  access_log: "/var/log/staging-lb.log"

admin:
  enabled: true
  listen: "127.0.0.1:9000"
  health_endpoint: "/healthz"
  metrics_endpoint: "/metrics"

backends:
  api:
    - "http://staging-api-1:8080"
    - "http://staging-api-2:8080"

routes:
  - id: "api"
    match:
      path: "/"
    backend: "api"
    lb: "round-robin"
    health:
      path: "/health"
      interval: 10s
      timeout: 2s
```

### Production Multi-Service

```yaml
service:
  name: "prod-gateway"
  env: "production"
  listen: "0.0.0.0:443"

tls:
  enabled: true
  cert_path: "/etc/ssl/certs/cert.pem"
  key_path: "/etc/ssl/private/key.pem"

logging:
  level: warn
  access_log: "/var/log/gateway/access.log"

admin:
  enabled: true
  listen: "127.0.0.1:9000"
  health_endpoint: "/healthz"
  metrics_endpoint: "/metrics"

backends:
  users:
    - "http://users-1:3001"
    - "http://users-2:3001"
  orders:
    - "http://orders-1:4001"
    - "http://orders-2:4001"

routes:
  - id: "users"
    match:
      path: "/api/users"
    backend: "users"
    lb: "least-connections"
    health:
      path: "/health"
      interval: 10s
      timeout: 2s

  - id: "orders"
    match:
      path: "/api/orders"
    backend: "orders"
    lb: "least-connections"
    health:
      path: "/health"
      interval: 10s
      timeout: 2s
```

## Next Steps

Now that you've got the basics, explore:

- [Configuration Reference](configuration.md) - Complete config options
- [Load Balancing Strategies](load-balancing.md) - Choose the right algorithm
- [Features Guide](features.md) - Deep dive into advanced features
- [Operations Guide](operations.md) - Monitoring and troubleshooting
- [Examples](../examples/) - More configuration patterns

## Troubleshooting

### Load balancer won't start

**Check:** Configuration syntax
```bash
# Validate YAML syntax
yamllint config.yaml
```

**Check:** Port already in use
```bash
# On Linux/Mac
lsof -i :8080

# Change port in config if needed
```

### All requests return 503

**Check:** Backend URLs are correct and reachable
```bash
curl http://localhost:3001
```

**Check:** Backends are running
```bash
ps aux | grep python
```

**Check:** Health checks (if enabled)
```bash
curl http://127.0.0.1:9000/metrics
```

### Backends not receiving traffic

**Check:** Route path matching
```bash
# Enable debug logging to see routing decisions
logging:
  level: debug
```

**Check:** Health check endpoint
```bash
# Test health endpoint directly
curl http://localhost:3001/health
```

## Getting Help

- Review the [Configuration Reference](configuration.md)
- Check [Examples](../examples/) for similar use cases
- See [Operations Guide](operations.md) for common issues
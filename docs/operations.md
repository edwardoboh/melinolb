# Operations Guide

Complete guide for operating, monitoring, and troubleshooting the load balancer in production.

## Table of Contents
- [Monitoring](#monitoring)
- [Logging](#logging)
- [Health Monitoring](#health-monitoring)
- [Common Operations](#common-operations)
- [Troubleshooting](#troubleshooting)
- [Performance Tuning](#performance-tuning)
- [Security](#security)

## Monitoring

### Admin Endpoints

The admin interface exposes operational endpoints:

```yaml
admin:
  enabled: true
  listen: "127.0.0.1:9000"
  health_endpoint: "/healthz"
  metrics_endpoint: "/metrics"
```

### Health Check Endpoint

**Purpose:** Verify load balancer is running

```bash
curl http://127.0.0.1:9000/healthz
```

**Response:**
```
OK
```

**HTTP Status:**
- `200 OK`: Load balancer is healthy
- No response: Load balancer is down

**Use cases:**
- Kubernetes liveness probe
- Docker health check
- Monitoring systems (Nagios, Prometheus)
- Upstream load balancer health checks

**Kubernetes example:**
```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 9000
  initialDelaySeconds: 10
  periodSeconds: 5
```

### Metrics Endpoint

**Purpose:** Backend health and routing statistics

```bash
curl http://127.0.0.1:9000/metrics
```

**Response format:**
```
route_users_healthy_backends 2
route_users_total_backends 2
route_orders_healthy_backends 1
route_orders_total_backends 2
route_payments_healthy_backends 0
route_payments_total_backends 1
```

**Metrics explained:**
- `route_<id>_healthy_backends`: Number of healthy backends
- `route_<id>_total_backends`: Total configured backends

### Continuous Monitoring

```bash
# Watch metrics every 2 seconds
watch -n 2 'curl -s http://127.0.0.1:9000/metrics'

# Monitor specific route
watch -n 1 'curl -s http://127.0.0.1:9000/metrics | grep users'

# Check health continuously
while true; do
  curl -s http://127.0.0.1:9000/healthz
  sleep 5
done
```

### Integration with Monitoring Systems

#### Prometheus

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'loadbalancer'
    static_configs:
      - targets: ['127.0.0.1:9000']
    metrics_path: '/metrics'
```

#### Nagios

```bash
#!/bin/bash
# check_loadbalancer.sh
status=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:9000/healthz)
if [ "$status" == "200" ]; then
  echo "OK - Load balancer is healthy"
  exit 0
else
  echo "CRITICAL - Load balancer is down"
  exit 2
fi
```

## Logging

### Log Levels

Configure verbosity based on environment:

```yaml
logging:
  level: info  # debug | info | warn | error
```

| Level | Output | Use Case |
|-------|--------|----------|
| `debug` | All events | Development, troubleshooting |
| `info` | Operational info | Normal production |
| `warn` | Warnings only | Quiet production |
| `error` | Errors only | Minimal logging |

### Access Logs

**Format:**
```
METHOD PATH STATUS DURATION BACKEND
GET /api/users 200 15ms backend-1:3001
POST /api/orders 201 42ms backend-2:4001
GET /api/payments 503 2ms -
```

**Configuration:**
```yaml
logging:
  access_log: "/var/log/loadbalancer/access.log"  # or "-" for stdout
```

### Log Rotation

**Using logrotate (Linux):**

```
# /etc/logrotate.d/loadbalancer
/var/log/loadbalancer/*.log {
    daily
    rotate 7
    compress
    delaycompress
    notifempty
    missingok
    postrotate
        systemctl reload loadbalancer
    endscript
}
```

### Structured Logging

Current format is plain text. For JSON logging (future):

```json
{
  "timestamp": "2025-01-15T10:30:45Z",
  "level": "info",
  "method": "GET",
  "path": "/api/users",
  "status": 200,
  "duration_ms": 15,
  "backend": "backend-1:3001"
}
```

## Health Monitoring

### Checking Backend Health

```bash
# Get current health status
curl http://127.0.0.1:9000/metrics | grep healthy

# Output:
# route_api_healthy_backends 3
# route_api_total_backends 3
```

### Interpreting Health Metrics

**All backends healthy:**
```
route_api_healthy_backends 3
route_api_total_backends 3
```
✅ Status: Good

**Some backends unhealthy:**
```
route_api_healthy_backends 1
route_api_total_backends 3
```
⚠️ Status: Degraded (investigate)

**All backends unhealthy:**
```
route_api_healthy_backends 0
route_api_total_backends 3
```
🚨 Status: Critical (immediate action required)

### Setting Up Alerts

**Alert conditions:**
1. **Zero healthy backends** → Critical alert
2. **< 50% healthy backends** → Warning alert
3. **Load balancer down** → Critical alert

**Prometheus alert example:**
```yaml
groups:
  - name: loadbalancer
    rules:
      - alert: NoHealthyBackends
        expr: route_api_healthy_backends == 0
        for: 1m
        annotations:
          summary: "No healthy backends for route {{ $labels.route }}"
          
      - alert: DegradedBackends
        expr: route_api_healthy_backends / route_api_total_backends < 0.5
        for: 5m
        annotations:
          summary: "Less than 50% backends healthy"
```

## Common Operations

### Adding Backends

1. Update `config.yaml`:
```yaml
backends:
  api:
    - "http://existing-1:3001"
    - "http://existing-2:3002"
    - "http://new-backend:3003"  # Added
```

2. Restart load balancer:
```bash
systemctl restart loadbalancer
```

3. Verify new backend is healthy:
```bash
curl http://127.0.0.1:9000/metrics
```

### Removing Backends

1. Update `config.yaml` (remove backend URL)
2. Restart load balancer
3. Verify metrics show correct backend count

### Changing Configuration

1. Edit `config.yaml`
2. Validate syntax:
```bash
yamllint config.yaml
```
3. Restart load balancer
4. Check logs for startup errors
5. Verify metrics

### Graceful Shutdown

**Systemd service:**
```bash
systemctl stop loadbalancer
```

**Manual:**
```bash
# Send SIGTERM
kill -TERM <pid>

# Load balancer will:
# 1. Stop accepting new connections
# 2. Complete in-flight requests
# 3. Shutdown gracefully
```

### Zero-Downtime Updates

**Using multiple instances:**

1. Start new instance with updated config on different port:
```bash
# New config listens on :8081
./loadbalancer config-new.yaml &
```

2. Update upstream (nginx/DNS) to point to new instance

3. Wait for old instance to drain connections

4. Stop old instance:
```bash
kill -TERM <old-pid>
```

## Troubleshooting

### All Requests Return 503

**Symptom:** Every request fails with 503 Service Unavailable

**Possible causes:**

1. **No healthy backends**
   ```bash
   # Check metrics
   curl http://127.0.0.1:9000/metrics | grep healthy
   ```
   
   **Solution:** Fix backends or health check configuration

2. **Backend URLs incorrect**
   ```yaml
   # Check config
   backends:
     api:
       - "http://localhost:3001"  # Should be correct URL
   ```
   
   **Test backends directly:**
   ```bash
   curl http://localhost:3001
   ```

3. **Backends not running**
   ```bash
   # Check if backends are up
   ps aux | grep backend
   netstat -tlnp | grep 3001
   ```

### Some Backends Not Receiving Traffic

**Symptom:** Uneven distribution, some backends idle

**Possible causes:**

1. **Backends marked unhealthy**
   ```bash
   curl http://127.0.0.1:9000/metrics
   # Check if all backends are healthy
   ```

2. **Sticky sessions enabled**
   ```yaml
   # Sticky sessions may cause apparent unevenness
   sticky:
     enabled: true
   ```
   
   **Expected:** Clients stick to same backend

3. **Health check failing**
   ```bash
   # Test health endpoint directly
   curl http://backend-url:port/health
   ```

### High Latency

**Symptom:** Requests taking longer than expected

**Investigation steps:**

1. **Check backend response times:**
   ```bash
   # Enable debug logging
   logging:
     level: debug
   ```
   
   Look for slow backends in logs

2. **Check timeout settings:**
   ```yaml
   defaults:
     backend_read_timeout: 10s  # May be too long
   ```

3. **Backend overloaded:**
   - Check backend CPU/memory
   - Consider adding more backends
   - Enable rate limiting

4. **Network issues:**
   ```bash
   # Test network latency to backends
   ping backend-host
   traceroute backend-host
   ```

### Rate Limiting Too Aggressive

**Symptom:** Legitimate requests returning 429

**Solutions:**

1. **Increase rate limit:**
   ```yaml
   rate_limit:
     requests_per_second: 200  # Was: 100
     burst: 400  # Was: 200
   ```

2. **Increase burst capacity:**
   ```yaml
   rate_limit:
     requests_per_second: 100
     burst: 500  # Allow larger bursts
   ```

3. **Disable for testing:**
   ```yaml
   rate_limit:
     enabled: false
   ```

### Memory Usage Growing

**Symptom:** Load balancer memory increasing over time

**Investigation:**

1. **Check connection counts** (least-connections mode):
   - May indicate connection leaks in backends
   - Backends not closing connections properly

2. **Review timeout settings:**
   ```yaml
   defaults:
     backend_read_timeout: 30s  # Consider reducing
   ```

3. **Monitor with:**
   ```bash
   # Memory usage
   ps aux | grep loadbalancer
   
   # Open connections
   netstat -an | grep :8080 | wc -l
   ```

### Configuration Errors

**Symptom:** Load balancer won't start

**Common errors:**

1. **Invalid YAML syntax:**
   ```bash
   yamllint config.yaml
   ```

2. **Invalid backend URLs:**
   ```
   Error: invalid backend URL "localhost:3000": missing protocol
   ```
   
   **Fix:** Use `http://localhost:3000`

3. **Port already in use:**
   ```
   Error: listen tcp :8080: bind: address already in use
   ```
   
   **Fix:** Change port or stop conflicting process

## Performance Tuning

### Timeout Optimization

**For fast APIs (< 1s response time):**
```yaml
defaults:
  read_timeout: 10s
  write_timeout: 10s
  backend_connect_timeout: 1s
  backend_read_timeout: 5s
```

**For slow operations (uploads, reports):**
```yaml
defaults:
  read_timeout: 60s
  write_timeout: 60s
  backend_connect_timeout: 5s
  backend_read_timeout: 30s
```

### Health Check Tuning

**For stable backends:**
```yaml
health:
  path: "/health"
  interval: 30s  # Check less frequently
  timeout: 5s
```

**For critical services:**
```yaml
health:
  path: "/health"
  interval: 5s   # Check frequently
  timeout: 1s
```

### Load Balancing Strategy

**For uniform, short requests:**
```yaml
lb: "round-robin"  # Fastest, simplest
```

**For varying request duration:**
```yaml
lb: "least-connections"  # Adapts to load
```

### Rate Limiting

**Tune based on backend capacity:**
```yaml
rate_limit:
  # Start conservative
  requests_per_second: 100
  burst: 200
  
  # Increase based on monitoring
  # requests_per_second: 500
  # burst: 1000
```

## Security

### Admin Interface

**Always bind to localhost in production:**
```yaml
admin:
  enabled: true
  listen: "127.0.0.1:9000"  # NOT 0.0.0.0!
```

**Why:** Prevents external access to operational endpoints

### TLS Configuration

**Use strong TLS settings:**
```yaml
tls:
  enabled: true
  cert_path: "/etc/ssl/certs/cert.pem"
  key_path: "/etc/ssl/private/key.pem"
```

**File permissions:**
```bash
# Certificate can be world-readable
chmod 644 /etc/ssl/certs/cert.pem

# Private key must be restricted
chmod 600 /etc/ssl/private/key.pem
chown loadbalancer:loadbalancer /etc/ssl/private/key.pem
```

### Rate Limiting as DDoS Protection

```yaml
routes:
  - id: "public-api"
    match:
      path: "/api"
    backend: "api_servers"
    rate_limit:
      enabled: true
      requests_per_second: 100  # Limit abuse
      burst: 200
```

### Logging Sensitive Data

**Avoid logging:**
- Authorization headers
- API keys
- Passwords
- Session tokens

Current implementation logs paths and status codes only (safe).

## Best Practices

### Production Checklist

- [ ] TLS enabled with valid certificates
- [ ] Admin interface on localhost only
- [ ] Health checks configured
- [ ] Appropriate timeouts set
- [ ] Rate limiting enabled for public routes
- [ ] Access logs configured
- [ ] Log rotation set up
- [ ] Monitoring alerts configured
- [ ] Backup configuration files
- [ ] Document custom settings

### Monitoring Checklist

- [ ] Health endpoint monitored
- [ ] Metrics endpoint scraped
- [ ] Alert on zero healthy backends
- [ ] Alert on load balancer down
- [ ] Access logs reviewed regularly
- [ ] 429 rate limit errors tracked

### Maintenance Checklist

- [ ] Review and rotate logs monthly
- [ ] Update TLS certificates before expiry
- [ ] Review and tune timeouts quarterly
- [ ] Test backup/recovery procedures
- [ ] Document configuration changes
- [ ] Keep load balancer updated

## See Also

- [Configuration Reference](configuration.md)
- [Getting Started Guide](getting-started.md)
- [Features Guide](features.md)
- [Examples Directory](../examples/)
# Melino Load Balancer
A lightweight, high-performance HTTP/HTTPS load balancer written in Go.

## Features

- 🔄 Multiple load balancing strategies (round-robin, least-connections)
- 💚 Active health checking
- 🔒 TLS/SSL termination
- 📊 Built-in metrics and monitoring
- 🎯 Path-based and host-based routing
- 🍪 Sticky sessions support
- 🚦 Rate limiting
- 📝 Comprehensive logging

## Installation

### Quick Install (Linux/macOS)
```bash
curl -fsSL https://raw.githubusercontent.com/edwardoboh/melinolb/main/install.sh | sh
```

### Manual Installation
Download the latest release for your platform from [releases page](https://github.com/edwardoboh/melinolb/releases/latest):
```bash
# Linux/macOS
wget https://github.com/edwardoboh/melinolb/releases/download/v1.0.0/melinolb_1.0.0_linux_amd64.tar.gz
tar -xzf melinolb_1.0.0_linux_amd64.tar.gz
sudo mv melinolb /usr/local/bin/

# Verify installation
melinolb --version
```

### Using Go
```bash
go install github.com/edwardoboh/melinolb/cmd@latest
```

## Quick Start

Create a `config.yaml`:
```yaml
service:
  name: "my-proxy"
  listen: "0.0.0.0:8080"

routes:
  - id: "api"
    match:
      path: "/api"
    backend:
      - "http://localhost:3001"
      - "http://localhost:3002"
    lb: "round-robin"
```

Run the load balancer:
```bash
melinolb config.yaml
```

## Documentation

- **[Configuration Reference](docs/configuration.md)** - Complete configuration guide
- **[Getting Started](docs/getting-started.md)** - Step-by-step tutorial
- **[Load Balancing Strategies](docs/load-balancing.md)** - Algorithm details
- **[Features Guide](docs/features.md)** - Health checks, rate limiting, sticky sessions
- **[Operations Guide](docs/operations.md)** - Monitoring and troubleshooting

## Examples

See the [`examples/`](examples/) directory for common configuration patterns:
- [Simple proxy](examples/simple.yaml)
- [Production setup](examples/production.yaml)
- [Multi-service gateway](examples/multi-service.yaml)
- [High-traffic configuration](examples/high-traffic.yaml)

## Contributing

We welcome contributions! Here's how you can help:

### Reporting Issues

- Check existing issues before creating a new one
- Provide clear reproduction steps
- Include configuration files (sanitize sensitive data)
- Share error logs and version information

### Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Test thoroughly with various configurations
5. Commit with clear messages (`git commit -m 'feat: add amazing feature'`)
6. Push to your branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Development Guidelines

- Follow existing code style and conventions
- Add tests for new features
- Update documentation for configuration changes
- Add examples for new functionality

### Areas We Need Help

- 🐛 Bug fixes and testing
- 📝 Documentation improvements
- 🌟 New load balancing strategies
- 🔧 Performance optimizations
- 🧪 Test coverage
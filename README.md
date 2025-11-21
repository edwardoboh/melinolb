# Melino Load Balancer
Light weight Load Balancer with auto adjusting load balancing algorithms to ensure equal traffic distributions amongst backend servers.

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
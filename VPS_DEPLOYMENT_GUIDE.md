# Luna IoT Server VPS Deployment Guide

## 🚀 Quick VPS Setup

### 1. **Server Requirements**
- **OS**: Ubuntu 20.04+ or CentOS 7+
- **RAM**: Minimum 1GB (Recommended 2GB+)
- **Storage**: Minimum 10GB
- **Ports**: 8080 (HTTP), 5000 (TCP for IoT devices)

### 2. **VPS Setup Steps**

#### Install Dependencies
```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Go 1.21+
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Git
sudo apt install git -y

# Install PostgreSQL (if using local database)
sudo apt install postgresql postgresql-contrib -y
```

#### Clone and Setup Project
```bash
# Clone repository
git clone <your-repo-url>
cd luna_iot_server

# Install Go dependencies
go mod download

# Create .env file
cp config.example.env .env
nano .env  # Edit with your VPS configuration
```

### 3. **Environment Configuration**

Create `.env` file with your VPS settings:
```bash
# Database Configuration (using your existing remote DB)
DB_HOST=84.247.131.246
DB_PORT=5433
DB_USER=luna
DB_PASSWORD=Luna@#$321
DB_NAME=luna_iot
DB_SSL_MODE=disable

# Server Ports
HTTP_PORT=8080
TCP_PORT=5000

# Logging
LOG_LEVEL=info
LOG_HTTP=false
```

### 4. **Firewall Configuration**

```bash
# Allow HTTP API port
sudo ufw allow 8080/tcp

# Allow TCP for IoT devices  
sudo ufw allow 5000/tcp

# Allow SSH (if not already)
sudo ufw allow 22/tcp

# Enable firewall
sudo ufw enable
```

### 5. **Build and Run**

```bash
# Build the application
go build -o luna_server main.go

# Make executable
chmod +x luna_server

# Test run (foreground)
./luna_server
```

### 6. **Create Systemd Service** (Recommended)

Create service file:
```bash
sudo nano /etc/systemd/system/luna-iot.service
```

Add this content:
```ini
[Unit]
Description=Luna IoT Server
After=network.target
Wants=network.target

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=/home/ubuntu/luna_iot_server
ExecStart=/home/ubuntu/luna_iot_server/luna_server
Restart=always
RestartSec=10
Environment=PATH=/usr/local/go/bin:/usr/bin:/bin
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

Enable and start service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable luna-iot
sudo systemctl start luna-iot
sudo systemctl status luna-iot
```

### 7. **Frontend Configuration**

Update your frontend to connect to VPS:

**Option A**: Environment Variable
```bash
# In luna_iot_frontend/.env.local
VITE_API_BASE_URL=http://YOUR_VPS_IP:8080
VITE_WS_URL=ws://YOUR_VPS_IP:8080/ws
```

**Option B**: Update config directly
In `luna_iot_frontend/src/config/api.ts`:
```typescript
const VPS_CONFIG = {
  HOST: 'YOUR_VPS_IP',  // Replace with your actual VPS IP
  HTTP_PORT: '8080',
  WS_PORT: '8080'
};
```

## 🔧 Troubleshooting

### Check Server Status
```bash
# Check if service is running
sudo systemctl status luna-iot

# View logs
sudo journalctl -u luna-iot -f

# Check if ports are open
sudo netstat -tlnp | grep :8080
sudo netstat -tlnp | grep :5000
```

### Test API Connection
```bash
# Test from VPS itself
curl http://localhost:8080/health

# Test from external
curl http://YOUR_VPS_IP:8080/health
```

### Common Issues & Solutions

1. **Port not accessible from outside**
   ```bash
   # Check firewall
   sudo ufw status
   
   # Check if service is binding to all interfaces
   sudo netstat -tlnp | grep :8080
   # Should show 0.0.0.0:8080, not 127.0.0.1:8080
   ```

2. **Database connection fails**
   ```bash
   # Test database connection
   psql -h 84.247.131.246 -p 5433 -U luna -d luna_iot
   ```

3. **Service won't start**
   ```bash
   # Check logs for errors
   sudo journalctl -u luna-iot --no-pager
   
   # Test manual start
   cd /home/ubuntu/luna_iot_server
   ./luna_server
   ```

## 📱 Device Connection

Your IoT devices should connect to:
```
TCP Host: YOUR_VPS_IP
TCP Port: 5000
Protocol: GT06
```

## 🌐 API Access

Your frontend and API clients should use:
```
HTTP API: http://YOUR_VPS_IP:8080
WebSocket: ws://YOUR_VPS_IP:8080/ws
Health Check: http://YOUR_VPS_IP:8080/health
```

## 🔐 Security Recommendations

1. **Use HTTPS in production** (setup SSL certificate)
2. **Restrict database access** to your VPS IP only
3. **Use strong passwords** for all accounts
4. **Keep system updated** regularly
5. **Monitor logs** for suspicious activity

## 📊 Monitoring

```bash
# Monitor system resources
htop

# Monitor logs in real-time
sudo journalctl -u luna-iot -f

# Check database connections
sudo netstat -an | grep :5433
``` 
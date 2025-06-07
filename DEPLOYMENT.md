# Luna IoT Server Deployment Guide

This guide covers deploying the unified Luna IoT Server in production environments.

## 🏗️ Architecture Overview

The Luna IoT Server runs as a single process that manages:
- **TCP Server** (Port 5000): Handles IoT device connections
- **HTTP Server** (Port 8080): Provides REST API access
- **Database Connection**: Shared PostgreSQL connection pool
- **Device Control**: Real-time command sending to connected devices

## 🚀 Production Deployment

### 1. System Requirements

**Minimum:**
- CPU: 2 cores
- RAM: 2GB
- Storage: 10GB SSD
- OS: Linux/Windows Server

**Recommended:**
- CPU: 4+ cores
- RAM: 4GB+
- Storage: 50GB+ SSD
- OS: Ubuntu 20.04+ / Windows Server 2019+

### 2. Database Setup

```bash
# Install PostgreSQL
sudo apt update
sudo apt install postgresql postgresql-contrib

# Create database and user
sudo -u postgres psql
CREATE DATABASE luna_iot;
CREATE USER luna_user WITH PASSWORD 'secure_password_here';
GRANT ALL PRIVILEGES ON DATABASE luna_iot TO luna_user;
\q
```

### 3. Application Deployment

```bash
# Clone repository
git clone <your-repo-url>
cd luna_iot_server

# Copy configuration
cp config.example.env .env
nano .env  # Edit with your settings

# Build application
go build -o luna_server main.go

# Make executable
chmod +x luna_server
```

### 4. Environment Configuration

Edit `.env` file:
```env
# Production Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=luna_user
DB_PASSWORD=secure_password_here
DB_NAME=luna_iot
DB_SSL_MODE=require

# Production Ports
HTTP_PORT=8080
TCP_PORT=5000

# Production Settings
LOG_LEVEL=info
MAX_TCP_CONNECTIONS=5000
```

### 5. Systemd Service (Linux)

Create `/etc/systemd/system/luna-iot.service`:
```ini
[Unit]
Description=Luna IoT Server
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=luna
Group=luna
WorkingDirectory=/opt/luna-iot
ExecStart=/opt/luna-iot/luna_server
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/luna-iot

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable luna-iot
sudo systemctl start luna-iot
sudo systemctl status luna-iot
```

### 6. Windows Service

Use NSSM (Non-Sucking Service Manager):
```cmd
# Download NSSM from https://nssm.cc/
nssm install "Luna IoT Server" "C:\luna-iot\luna_server.exe"
nssm set "Luna IoT Server" AppDirectory "C:\luna-iot"
nssm start "Luna IoT Server"
```

## 🔒 Security Considerations

### 1. Firewall Configuration

```bash
# Allow HTTP API (adjust as needed)
sudo ufw allow 8080/tcp

# Allow TCP for IoT devices
sudo ufw allow 5000/tcp

# Enable firewall
sudo ufw enable
```

### 2. Database Security

```sql
-- Create read-only user for monitoring
CREATE USER luna_readonly WITH PASSWORD 'readonly_password';
GRANT CONNECT ON DATABASE luna_iot TO luna_readonly;
GRANT USAGE ON SCHEMA public TO luna_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO luna_readonly;
```

### 3. SSL/TLS (Recommended)

Use a reverse proxy like Nginx for HTTPS:
```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 📊 Monitoring & Logging

### 1. Log Management

```bash
# View logs (systemd)
sudo journalctl -u luna-iot -f

# Log rotation
sudo nano /etc/logrotate.d/luna-iot
```

### 2. Health Monitoring

```bash
# Simple health check script
#!/bin/bash
curl -f http://localhost:8080/health || exit 1
```

### 3. Database Monitoring

```sql
-- Monitor active connections
SELECT count(*) FROM pg_stat_activity WHERE datname = 'luna_iot';

-- Monitor table sizes
SELECT schemaname,tablename,pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size 
FROM pg_tables WHERE schemaname = 'public';
```

## 🔄 Backup & Recovery

### 1. Database Backup

```bash
# Daily backup script
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
pg_dump -h localhost -U luna_user luna_iot > /backups/luna_iot_$DATE.sql
find /backups -name "luna_iot_*.sql" -mtime +7 -delete
```

### 2. Application Backup

```bash
# Backup configuration and binary
tar -czf luna_iot_backup_$(date +%Y%m%d).tar.gz \
    luna_server .env config/ logs/
```

## 🚀 Scaling Considerations

### 1. Horizontal Scaling

For high-traffic deployments:
- Use a load balancer for HTTP API
- Consider database read replicas
- Implement connection pooling

### 2. Performance Tuning

```env
# Increase connection limits
MAX_TCP_CONNECTIONS=10000

# Database connection pooling
DB_MAX_OPEN_CONNS=100
DB_MAX_IDLE_CONNS=10
```

## 🔧 Troubleshooting

### Common Issues

1. **Port conflicts**: Change ports in `.env`
2. **Database connection**: Check PostgreSQL service
3. **Permission errors**: Verify file permissions
4. **Memory issues**: Monitor with `htop` or Task Manager

### Debug Mode

```env
LOG_LEVEL=debug
```

### Health Checks

```bash
# API health
curl http://localhost:8080/health

# TCP port check
telnet localhost 5000

# Database connection
psql -h localhost -U luna_user -d luna_iot -c "SELECT 1;"
```

## 📈 Performance Metrics

Monitor these key metrics:
- **HTTP Response Time**: < 100ms for API calls
- **TCP Connections**: Number of active IoT devices
- **Database Performance**: Query execution time
- **Memory Usage**: Should be stable over time
- **CPU Usage**: Should be < 50% under normal load

## 🎯 Production Checklist

- [ ] Database properly configured and secured
- [ ] Environment variables set correctly
- [ ] Firewall rules configured
- [ ] SSL/TLS enabled (if needed)
- [ ] Monitoring and logging set up
- [ ] Backup strategy implemented
- [ ] Service auto-start configured
- [ ] Health checks working
- [ ] Performance tested with expected load
- [ ] Documentation updated for your environment

Your Luna IoT Server is now ready for production! 🎉 
#!/bin/bash

echo "🚀 Deploying Hardcoded Firebase Solution"
echo "========================================"
echo ""

# Stop the service
echo "⏹️  Stopping luna_iot_server service..."
sudo systemctl stop luna_iot_server
sleep 2

# Kill any remaining processes
echo "🔍 Killing any remaining processes..."
sudo pkill -f "luna_iot_server" || true
sleep 2

# Navigate to project directory
echo "📁 Navigating to project directory..."
cd /home/luna/Luna-IOT-Server

# Build the new binary with hardcoded Firebase
echo "🔨 Building new binary with hardcoded Firebase credentials..."
go build -o luna_iot_server

if [ $? -eq 0 ]; then
    echo "✅ Binary built successfully"
    echo "📄 Binary size: $(ls -lh luna_iot_server | awk '{print $5}')"
else
    echo "❌ Build failed!"
    exit 1
fi

# Set permissions
echo "🔐 Setting permissions..."
chmod +x luna_iot_server
chown luna:luna luna_iot_server

# Start the service
echo "▶️  Starting luna_iot_server service..."
sudo systemctl start luna_iot_server

# Wait for service to start
echo "⏳ Waiting for service to start..."
sleep 5

# Check status
echo "📊 Service status:"
sudo systemctl status luna_iot_server --no-pager

# Show recent logs
echo ""
echo "📋 Recent logs:"
sudo journalctl -u luna_iot_server --no-pager -n 20

echo ""
echo "🎉 Hardcoded Firebase deployment completed!"
echo ""
echo "📋 Look for these SUCCESS messages in the logs:"
echo "✅ Using hardcoded Firebase credentials:"
echo "✅ Firebase initialized successfully using hardcoded credentials with OAuth2"
echo "✅ Messaging client created successfully"
echo ""
echo "📋 If you see these messages, Firebase is working!"
echo "📋 Test your API: POST http://84.247.131.246:8080/api/v1/admin/notification-management/5/send"
echo ""
echo "📋 To monitor logs in real-time:"
echo "sudo journalctl -u luna_iot_server -f" 
#!/bin/bash

echo "🚀 Deploying Firebase Fix"
echo "========================"
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

# Build the new binary
echo "🔨 Building new binary..."
go build -o luna_iot_server

if [ $? -eq 0 ]; then
    echo "✅ Binary built successfully"
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
echo "🎉 Deployment completed!"
echo ""
echo "📋 If you still get 404 errors, the issue is:"
echo "❌ Your Firebase project 'luna-iot-b5cdd' doesn't exist"
echo "❌ Or your service account doesn't have access"
echo ""
echo "📋 To fix this, you need to:"
echo "1. Go to https://console.firebase.google.com/"
echo "2. Create a new project or check if 'luna-iot-b5cdd' exists"
echo "3. Enable Cloud Messaging"
echo "4. Generate a new service account key"
echo "5. Update the hardcoded credentials in config/firebase.go"
echo ""
echo "📋 To monitor logs in real-time:"
echo "sudo journalctl -u luna_iot_server -f" 
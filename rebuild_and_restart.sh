#!/bin/bash

echo "🔧 Rebuilding Luna IoT Server..."
echo ""

# Stop the service
echo "⏹️  Stopping luna_iot_server service..."
sudo systemctl stop luna_iot_server

# Build the new binary
echo "🔨 Building new binary..."
go build -o luna_iot_server

if [ $? -eq 0 ]; then
    echo "✅ Binary built successfully"
else
    echo "❌ Build failed"
    exit 1
fi

# Start the service
echo "▶️  Starting luna_iot_server service..."
sudo systemctl start luna_iot_server

# Check service status
echo "📊 Checking service status..."
sudo systemctl status luna_iot_server

echo ""
echo "📋 To view logs:"
echo "sudo journalctl -u luna_iot_server -f"
echo ""
echo "📋 To check if Firebase is working, look for:"
echo "✅ Firebase initialized successfully using service account file with OAuth2"
echo ""
echo "📋 If you see Firebase initialization logs, try sending a notification again." 
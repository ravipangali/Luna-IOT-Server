#!/bin/bash

echo "🔧 Complete Firebase Fix for Luna IoT Server"
echo "=============================================="
echo ""

# Step 1: Stop the service
echo "⏹️  Step 1: Stopping luna_iot_server service..."
sudo systemctl stop luna_iot_server
sleep 2

# Step 2: Check if any processes are still running
echo "🔍 Step 2: Checking for any remaining processes..."
if pgrep -f "luna_iot_server" > /dev/null; then
    echo "⚠️  Found running processes, killing them..."
    sudo pkill -f "luna_iot_server"
    sleep 2
else
    echo "✅ No running processes found"
fi

# Step 3: Navigate to the project directory
echo "📁 Step 3: Navigating to project directory..."
cd /home/luna/Luna-IOT-Server

# Step 4: Check if service account file exists
echo "📋 Step 4: Checking Firebase service account file..."
if [ -f "firebase-service-account.json" ]; then
    echo "✅ Service account file found"
    echo "📄 File size: $(ls -lh firebase-service-account.json | awk '{print $5}')"
else
    echo "❌ Service account file not found!"
    echo "Please ensure firebase-service-account.json is in /home/luna/Luna-IOT-Server/"
    exit 1
fi

# Step 5: Build the new binary
echo "🔨 Step 5: Building new binary with fixed Firebase configuration..."
go build -o luna_iot_server

if [ $? -eq 0 ]; then
    echo "✅ Binary built successfully"
    echo "📄 Binary size: $(ls -lh luna_iot_server | awk '{print $5}')"
else
    echo "❌ Build failed!"
    exit 1
fi

# Step 6: Set proper permissions
echo "🔐 Step 6: Setting proper permissions..."
chmod +x luna_iot_server
chown luna:luna luna_iot_server

# Step 7: Start the service
echo "▶️  Step 7: Starting luna_iot_server service..."
sudo systemctl start luna_iot_server

# Step 8: Wait a moment for the service to start
echo "⏳ Step 8: Waiting for service to start..."
sleep 5

# Step 9: Check service status
echo "📊 Step 9: Checking service status..."
sudo systemctl status luna_iot_server --no-pager

# Step 10: Show recent logs
echo ""
echo "📋 Step 10: Recent logs (last 20 lines):"
sudo journalctl -u luna_iot_server --no-pager -n 20

echo ""
echo "🎉 Firebase fix completed!"
echo ""
echo "📋 To monitor logs in real-time:"
echo "sudo journalctl -u luna_iot_server -f"
echo ""
echo "📋 Look for these success messages in the logs:"
echo "✅ Found Firebase service account file at:"
echo "✅ Service account file validation passed:"
echo "✅ Firebase initialized successfully using service account file with OAuth2"
echo ""
echo "📋 If you see these messages, try sending a notification again!"
echo "The 404 error should be resolved." 
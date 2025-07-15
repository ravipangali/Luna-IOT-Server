#!/bin/bash

echo "🔍 Checking Luna IoT Server Status..."
echo ""

# Check if service exists
echo "📋 Service Status:"
sudo systemctl status luna_iot_server --no-pager

echo ""
echo "📋 Recent Logs (last 50 lines):"
sudo journalctl -u luna_iot_server --no-pager -n 50

echo ""
echo "📋 Checking if binary exists:"
if [ -f "/home/luna/Luna-IOT-Server/luna_iot_server" ]; then
    echo "✅ Binary exists"
    ls -la /home/luna/Luna-IOT-Server/luna_iot_server
else
    echo "❌ Binary not found"
fi

echo ""
echo "📋 Checking if service account file exists:"
if [ -f "/home/luna/Luna-IOT-Server/firebase-service-account.json" ]; then
    echo "✅ Service account file exists"
    ls -la /home/luna/Luna-IOT-Server/firebase-service-account.json
else
    echo "❌ Service account file not found"
fi

echo ""
echo "📋 Current working directory of service:"
sudo systemctl show luna_iot_server --property=WorkingDirectory --no-pager 
#!/bin/bash
# Wait for config server to be ready
echo "Waiting for Vespa config server..."
until curl -s http://localhost:19071/state/v1/health 2>/dev/null | grep -q '"up"'; do
    sleep 2
done
echo "Config server is ready"

# Deploy the application
echo "Deploying Vespa application..."
cd /opt/vespa-app

# Create a zip package
zip -r /tmp/app.zip . 2>/dev/null

# Deploy using REST API
curl -s -X POST -H "Content-Type: application/zip" \
    --data-binary @/tmp/app.zip \
    http://localhost:19071/application/v2/tenant/default/prepareandactivate

echo "Application deployed"

# Wait for application to be ready
echo "Waiting for application to be ready..."
sleep 10

# Check document API
until curl -s http://localhost:8080/state/v1/health 2>/dev/null | grep -q '"up"'; do
    sleep 2
done
echo "Vespa is ready for documents"

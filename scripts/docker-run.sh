#!/bin/bash

# Tybalt Docker/Podman Build and Run Script with HTTPS Support

# Default to docker, but allow override
CONTAINER_ENGINE=${CONTAINER_ENGINE:-docker}

# Check if podman is available and docker is not
if ! command -v docker &> /dev/null && command -v podman &> /dev/null; then
    CONTAINER_ENGINE=podman
fi

echo "Using container engine: $CONTAINER_ENGINE"

# Parse command line arguments
DOMAIN=""
HTTP_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --domain)
            DOMAIN="$2"
            shift 2
            ;;
        --http-only)
            HTTP_ONLY=true
            shift
            ;;
        --help)
            echo "Usage: $0 [--domain yourdomain.com] [--http-only]"
            echo ""
            echo "Options:"
            echo "  --domain DOMAIN    Enable HTTPS with Let's Encrypt for specified domain"
            echo "  --http-only        Run in HTTP-only mode (default if no domain specified)"
            echo "  --help             Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                              # HTTP only on port 8080"
            echo "  $0 --domain example.com         # HTTPS with Let's Encrypt"
            echo "  $0 --http-only                  # Force HTTP only"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Create data directory for database persistence
mkdir -p ./data

# Build the image
echo "Building tybalt image..."
$CONTAINER_ENGINE build -t tybalt .

if [ $? -eq 0 ]; then
    echo "Build successful!"
    
    # Prepare run command
    RUN_CMD="$CONTAINER_ENGINE run -d --name tybalt"
    
    if [ "$HTTP_ONLY" = true ] || [ -z "$DOMAIN" ]; then
        echo "Starting tybalt container (HTTP only)..."
        RUN_CMD="$RUN_CMD -p 8080:8080"
        ENV_VARS=""
    else
        echo "Starting tybalt container with HTTPS for domain: $DOMAIN"
        echo "⚠️  Make sure:"
        echo "   - Domain $DOMAIN points to this server's public IP"
        echo "   - Ports 80 and 443 are accessible from the internet"
        echo "   - No other services are using ports 80/443"
        echo ""
        RUN_CMD="$RUN_CMD -p 80:80 -p 443:443 -p 8080:8080"
        ENV_VARS="-e DOMAIN=$DOMAIN"
    fi
    
    # Complete the run command
    RUN_CMD="$RUN_CMD -v $(pwd)/data:/pb/pb_data $ENV_VARS --restart unless-stopped tybalt"
    
    # Execute the run command
    eval $RUN_CMD
    
    if [ $? -eq 0 ]; then
        echo "Container started successfully!"
        
        if [ -z "$DOMAIN" ] || [ "$HTTP_ONLY" = true ]; then
            echo "Tybalt is now running at http://localhost:8080"
        else
            echo "Tybalt is now running at:"
            echo "  HTTP:  http://$DOMAIN (redirects to HTTPS)"
            echo "  HTTPS: https://$DOMAIN"
            echo "  Local: http://localhost:8080"
            echo ""
            echo "Note: Let's Encrypt certificate generation may take a few minutes"
        fi
        
        echo ""
        echo "Useful commands:"
        echo "  View logs:    $CONTAINER_ENGINE logs tybalt"
        echo "  Stop:         $CONTAINER_ENGINE stop tybalt"
        echo "  Remove:       $CONTAINER_ENGINE rm tybalt"
        echo "  View status:  $CONTAINER_ENGINE ps"
    else
        echo "Failed to start container"
        exit 1
    fi
else
    echo "Build failed"
    exit 1
fi 
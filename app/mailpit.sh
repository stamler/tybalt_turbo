#!/bin/bash

# Check if mailpit is installed, and install it if not
if ! command -v mailpit &> /dev/null; then
    echo "Mailpit could not be found. Installing Mailpit..."
    brew install mailpit
fi

# Check if Mailpit is running
if ! pgrep -x "mailpit" > /dev/null; then
    echo "Mailpit is not running. Starting Mailpit..."
    /opt/homebrew/opt/mailpit/bin/mailpit --smtp-auth-accept-any --smtp-auth-allow-insecure
fi

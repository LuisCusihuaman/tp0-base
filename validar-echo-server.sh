#!/bin/bash

# Define the server container name, network name, and test message
SERVER_CONTAINER_NAME="server"
NETWORK_NAME="tp0_testing_net"
TEST_MESSAGE="success"

# Run a BusyBox container to send the test message and capture the response
RESPONSE=$(docker run --rm \
    --network $NETWORK_NAME \
    busybox:latest \
    sh -c "echo '$TEST_MESSAGE' | nc $SERVER_CONTAINER_NAME 12345")

# Check if the response matches the test message
if [ "$RESPONSE" = "$TEST_MESSAGE" ]; then
    echo "action: test_echo_server | result: success"
else
    echo "action: test_echo_server | result: fail"
fi

#!/bin/bash

# Parameter verification
if [ $# -ne 2 ]; then
  echo "Usage: $0 <output-file-name> <number-of-clients>"
  exit 1
fi

# Assigning parameters to variables
output_file=$1
num_clients=$2

# Initial definition of the output file
cat <<EOL > $output_file
version: "3.9"
name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
      - LOGGING_LEVEL=DEBUG
    networks:
      - testing_net
EOL

# Dynamic generation of clients if num_clients is greater or equal to 1
if [ "$num_clients" -ge 1 ]; then
  for i in $(seq 1 $num_clients)
  do
    cat <<EOL >> $output_file
  client$i:
    container_name: client$i
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=$i
      - CLI_LOG_LEVEL=DEBUG
    networks:
      - testing_net
    depends_on:
      - server
EOL
  done
else
  echo "Skipping client generation as the number of clients is less than or equal to 0."
fi

# Network definition
cat <<EOL >> $output_file
networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
EOL

# Confirmation message
echo "$output_file file generated with $num_clients clients."

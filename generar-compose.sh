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
      - NUM_AGENCIES=$num_clients
    volumes:
      - ./server/config.ini:/server/config.ini:ro
    networks:
      - testing_net
EOL

# Client details
client1="Santiago Lionel,Lorca,30904465,1999-03-17,7574"
client2="Luciana Valentina,Moreno,32567980,2000-06-21,8901"
client3="Mateo Rodrigo,Pérez,31234567,1998-12-03,4562"
client4="Catalina Sofía,Gómez,31098765,1997-05-14,1238"
client5="Juan Manuel,Ruiz,31876543,1996-11-08,2345"

# Dynamic generation of clients if num_clients is greater or equal to 1
if [ "$num_clients" -ge 1 ]; then
  for i in $(seq 1 $num_clients)
  do
    client_index=$(( (i-1) % 5 + 1 ))

    case $client_index in
      1) details=$client1 ;;
      2) details=$client2 ;;
      3) details=$client3 ;;
      4) details=$client4 ;;
      5) details=$client5 ;;
    esac

    # Splitting the client details
    nombre=$(echo $details | cut -d',' -f1)
    apellido=$(echo $details | cut -d',' -f2)
    documento=$(echo $details | cut -d',' -f3)
    nacimiento=$(echo $details | cut -d',' -f4)
    numero=$(echo $details | cut -d',' -f5)

    cat <<EOL >> $output_file
  client$i:
    container_name: client$i
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=$i
      - NOMBRE=$nombre
      - APELLIDO=$apellido
      - DOCUMENTO=$documento
      - NACIMIENTO=$nacimiento
      - NUMERO=$numero
    volumes:
      - ./client/config.yaml:/config.yaml:ro
      - ./client/dataset.zip:/dataset.zip:ro
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

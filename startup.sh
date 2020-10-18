#!/bin/bash

if [ ! -f config.yml ]
then
    echo "Creating config file"
    cp example-config.yml config.yml
fi

if [ "$1" == "fakepowerwall" ]
then
    echo "Setting up fake powerwall docker-compose file"
    cp example-docker-compose.fake-powerwall.yml docker-compose.yml
    echo "Setting up fake powerwall environment variables"
    echo "INVERTER_HOST=${INVERTER_HOST}" > .env.fakepowerwall
    echo "INVERTER_TYPE=${INVERTER_TYPE}" >> .env.fakepowerwall
else
    echo "Setting up docker-compose file"
    cp example-docker-compose.yml docker-compose.yml
fi

echo "Attempting to start controller"
docker-compose up -d --remove-orphans

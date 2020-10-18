#!/bin/bash

if [ ! -f config.yml ]; then
    echo "Creating config file"
    cp example-config.yml config.yml
fi
if [ ! -f docker-compose.fake-powerwall.yml ] && [ "$1" == "fakepowerwall" ]; then
    echo "Setting up fake powerwall docker-compose override file"
    cp example-docker-compose.fake-powerwall.yml docker-compose.fake-powerwall.yml
fi

if [ "$1" == "fakepowerwall" ]; then
    echo "Attempting to start controller with fake powerwall support"
    docker-compose -f docker-compose.yml -f docker-compose.fake-powerwall.yml up -d --remove-orphans
else
    echo "Attempting to start controller"
    docker-compose up -d --remove-orphans
fi
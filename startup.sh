#!/bin/bash

usage() {
  echo "Usage: ./startup.sh -p fakepowerwall -i http://192.168.1.50 -t fronius"
  echo "WARNING: specifying -p fakepowerwall without options -i or -t will result in errors"
  echo "Options:"
  echo "  -p fakepowerwall"
  echo "  -i <inverter_host>"
  echo "  -t <inverter_type>"
  exit 1
}

if [[ ! $@ =~ ^\-.+ ]]
then
  usage
fi

while getopts ":p:i:t:h:" opt; do
  case ${opt} in
    p ) # process option p
      FAKEPOWERWALL=$OPTARG;;
    t ) # process option d
      INVERTER_TYPE=$OPTARG;;
    i ) # process option s
      INVERTER_HOST=$OPTARG;;
    h )
      usage;;
    *)
      usage;;
  esac
done

if [ ! -f config.yml ]
then
    echo "Creating config file"
    cp example-config.yml config.yml
fi

if [ "$FAKEPOWERWALL" == "fakepowerwall" ]
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

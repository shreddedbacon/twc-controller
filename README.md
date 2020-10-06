# Tesla Wall Connector Controller

> Disclaimer: Use this at your own risk, it is not fully functional and under heavy development at the moment.

### Requirements

- [Docker](https://docs.docker.com/get-docker/)
- [Docker-compose](https://docs.docker.com/compose/install/)
- RaspberryPi B+ 512MB or greater
- [USB RS485 device](https://www.ebay.com.au/itm/USB-To-RS485-Converter-Module-USB-To-TTL-RS485-Dual-Function-Dual-Protection/392923867548)

### Wiring

Recommened powering your RaspberryPi from an external power source rather than directly from the Tesla Wall Connector (TWC). This allows you to perform maintenance on the TWC without having to restart the RaspberryPI. It is possible to power the RaspberryPi directly from the TWC, but it is not recommended unless you know what you're doing.

Before wiring up the USB RS485 into the TWC, make sure the TWC is powered off.
The wiring into the TWC is pretty straight forward, you need to connect D+ on the USB device to D+ on the TWC, same for D- to D-.
You need to rotate the rotary dial to position F to put it into the secondary mode.

Power it back on when you're done. The TWC will probably be in a red or broken state until the controller is turned on.
You may need to press and hold the reset button for a few seconds on the side of the TWC after starting the controller though.

## Installation

The preferred installation method is using docker and docker-compose.

Using HypriotOS as the preferred operating system as it comes pre-bundled with docker. You can use the `flash` tool [here](https://github.com/hypriot/flash) to install the HypriotOS to an SD card to install in your RaspberryPi.

### Clone and Config

First step is to clone the repository and then edit the initial configuration for your setup.

```
# clone the repo
git clone https://github.com/shreddedbacon/twc-controller && cd twc-controller

# copy, then edit the config.yml
cp example-config.yml config.yml
```

You may need to edit `config.yml` to change the serial port device path, this depends on your setup.
If you're using a USB RS-485 serial adapter on a RasperryPi, it will typically be `/dev/ttyUSB0` and should not need adjusting.

### Configuring With Powerwall

Once you have set up the `config.yml` file to point to your local IP for the powerwall (under `powerwall: x` in config.yml), you can run the controller.

If you want to change the port that the controller listens on, edit the `docker-compose.yml` file from `8080:8080` to `<customport>:8080`

```
docker-compose up -d
```

Once running, visit the IP address of your RaspberryPi in the browser with the configured port, eg: http://192.168.1.25:8080

### Configuring With No Powerwall

If you don't have a powerwall, but you do have a solar system, you can still use this controller as long as your inverter is supported.

Currently [fake-powerwall](https://github.com/shreddedbacon/fake-powerwall) only supports Fronius inverters, but support for others can be built in (feel free to submit PRs/issues)

To start the controller with fake-powerwall support, copy the following file

```
cp example-docker-compose.fake-powerwall.yml docker-compose.fake-powerwall.yml
```

Then edit `docker-compose.fake-powerwall.yml` and change `INVERTER_HOST` to the IP or hostname for your inverter.

```
# start the controller using docker-compose, with the bundled fake-powerwall
docker-compose -f docker-compose.yml -f docker-compose.fake-powerwall.yml up -d
```

### Configuring With No Powerwall/Solar

If you have no powerwall, or solar, you can still use the controller.

You need to edit the `config.yml` and change

```
enablePowerwall: true
# to
enablePowerwall: false
```

This will allow you to control the TWC using the in-built controller API from any other external automation system/
(@TODO: document the API usage)

## Building From Source

If you want to build it from source, you can do so. You will need a few things, see the instructions below.

### Requirements

- Go 1.14+ [here](https://golang.org/doc/install)
- [go-bindata](https://github.com/go-bindata/go-bindata)

### Build - GO

```
go generate -v ./...
go build -o controller main.go
```

### Build - Docker

```
docker build -t shreddedbacon/twc-controller:arm32v6-rpi-${VERSION} .
```

### Run

```
./controller
```

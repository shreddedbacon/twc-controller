# Tesla Wall Connector Controller

> Disclaimer: Use this at your own risk, it is not fully functional and under heavy development at the moment.

### Requirements

- [Docker](https://docs.docker.com/get-docker/)
- [Docker-compose](https://docs.docker.com/compose/install/)
- RaspberryPi B+ 512MB or greater
- [USB RS485 device](https://www.ebay.com.au/itm/USB-To-RS485-Converter-Module-USB-To-TTL-RS485-Dual-Function-Dual-Protection/392923867548)

## Installation

The preferred installation method is using docker and docker-compose.

Using HypriotOS as the preferred operating system as it comes pre-bundled with docker. You can use the `flash` tool [here](https://github.com/hypriot/flash) to install the HypriotOS to an SD card to install in your RaspberryPi.

### Installation

```
# clone the repo
git clone https://github.com/shreddedbacon/twc-controller && cd twc-controller

# copy, then edit the config.yml
cp example-config.yml config.yml
```

You may need to edit `config.yml` to change the serial port device path, this depends on your setup.
If you're using a USB RS-485 serial adapter on a RasperryPi, it will typically be `/dev/ttyUSB0` and should not need adjusting.

### Run - With Powerwall

Once you have set up the `config.yml` file to point to your local IP for the powerwall (under `powerwall: x` in config.yml), you can run the controller.

If you want to change the port that the controller listens on, edit the `docker-compose.yml` file from `8080:8080` to `<customport>:8080`

```
docker-compose up -d
```

Once running, visit the IP address of your RaspberryPi in the browser with the defined port, eg: http://192.168.1.25:8080

#### No Powerwall?

If you don't have a powerwall, you can still use this controller as long as your inverter is supported. Currently [fake-powerwall](https://github.com/shreddedbacon/fake-powerwall) only supports Fronius inverters, but support for others can be built in (feel free to submit PRs)

To start the controller with fake-powerwall support, you need to edit the `config.yml` file and change

```
powerwall: http://192.168.1.50
# to
powerwall: http://fake-powerwall:8080
```

Then copy the following

```
cp example-docker-compose.fake-powerwall.yml docker-compose.fake-powerwall.yml
```

Edit `docker-compose.fake-powerwall.yml` and change `INVERTER_HOST` to the IP or hostname for your inverter.

```
# start the controller using docker-compose
docker-compose -f docker-compose.yml -f docker-compose.fake-powerwall.yml up -d
```

## Building

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

# Tesla Wall Connector Controller

> Disclaimer: Use this at your own risk, it is not fully functional and under heavy development at the moment.

## Requirements

- [Docker](https://docs.docker.com/get-docker/)
- [Docker-compose](https://docs.docker.com/compose/install/)
- RaspberryPi B+ 512MB or greater (preferable RaspberryPi 3B with WIFI)
- [USB RS485 device](https://www.ebay.com.au/itm/USB-To-RS485-Converter-Module-USB-To-TTL-RS485-Dual-Function-Dual-Protection/392923867548)

## Wiring

Recommened powering your RaspberryPi from an external power source rather than directly from the Tesla Wall Connector (TWC). This allows you to perform maintenance on the TWC without having to restart the RaspberryPi. It is possible to power the RaspberryPi directly from the TWC, but it is not recommended unless you know what you're doing.

Before wiring up the USB RS485 into the TWC, make sure the TWC is powered off.
The wiring into the TWC is pretty straight forward, you need to connect D+ on the USB device to D+ on the TWC, same for D- to D-.
You need to rotate the rotary dial to position F to put it into the secondary mode.

Power it back on when you're done. The TWC will probably be in a red or broken state until the controller is turned on.
You may need to press and hold the reset button for a few seconds on the side of the TWC after starting the controller though.

## Installation

The preferred installation method is using docker and docker-compose. But bundled with `cloud-init` that HypriotOS offers makes this simpler, see the following `Mac / Linux Users` and `Windows Users` sections on installing using HypriotOS.

Using HypriotOS as the preferred operating system as it comes pre-bundled with docker. You can use the `flash` tool [here](https://github.com/hypriot/flash) to install the HypriotOS to an SD card to install in your RaspberryPi.

### Mac / Linux Users

> Note: if you don't have a powerwall, and want to use the `fake-powerwall` service, this is not currently supported by this method of installation (it is coming)

You can use `flash` tool above to install HypriotOS with the accompanying user-data.yml onto the SD card.

```
# first copy the example-user-data.yml file
cp example-user-data.yml user-data.yml
```

Edit `user-data.yml` and change the following:

* `YOUR_WIFI_SSID` - change this to your WiFi SSID
* `YOUR_WIFI_PSK_PASSWORD` - change this to your WiFi Password

The default user that gets created in the operating system is `tesla` with the password `tesla`, you can change this by editing the `plain_text_passwd` in the `user-data.yml` file to the password you want to use.

```
flash --userdata user-data.yml \
    https://github.com/hypriot/image-builder-rpi/releases/download/v1.12.0/hypriotos-rpi-v1.12.0.img.zip
```

Once done, simply insert the SD card into the RaspberryPi and power it up, if the WiFi is configured correctly in `user-data.yml` the installer script will download the required images and start the containers. This can take a few minutes to do.

Once running, visit the IP address of your RaspberryPi in the browser with the configured port (default is 8080), eg: http://192.168.1.25:8080

### Windows Users

> Note: if you don't have a powerwall, and want to use the `fake-powerwall` service, this is not currently supported by this method of installation (it is coming)

There is a blog post here on the Hypriot [https://blog.hypriot.com/getting-started-with-docker-and-windows-on-the-raspberry-pi/](https://blog.hypriot.com/getting-started-with-docker-and-windows-on-the-raspberry-pi/) that explains how to flash the image to your SD card.

* Download the Hypriot Docker SD card image
* Flash the downloaded image to your SD card

Once the image is flashed, load the SD card in your computer and copy the contents of `example-user-data.yml` and overwrite the entire contents of the `user-data` file in the `HypriotOS` partition.

Change the following in the `user-data` file:

* `YOUR_WIFI_SSID` - change this to your WiFi SSID
* `YOUR_WIFI_PSK_PASSWORD` - change this to your WiFi Password

The default user that gets created in the operating system is `tesla` with the password `tesla`, you can change this by editing the `plain_text_passwd` in the `user-data` file to the password you want to use.

Once done, simply insert the SD card into the RaspberryPi and power it up, if the WiFi is configured correctly in `user-data` the installer script will download the required images and start the containers. This can take a few minutes to do.

Once running, visit the IP address of your RaspberryPi in the browser with the configured port (default is 8080), eg: http://192.168.1.25:8080

## Usage

### Main Page

The main page is where all the connected TWCs are listed, and also allows for setting the available amps directly.

Clicking on the ID button of a connected TWC will load the info page for that TWC

![main page](https://github.com/shreddedbacon/twc-controller/blob/main/docs/screenshots/home.png)

### TWC Info Page

This page lists the information being provided from the TWC itself.

This info includes things like:
* Connected vehicles VIN
* Plug state
* Charging state
* Current usage
* Each electrical phase and its usage

![twc info page](https://github.com/shreddedbacon/twc-controller/blob/main/docs/screenshots/twcinfo.png)

### Powerwall Page

The powerwall page is where you can configure the TWC to utilise Solar power to adjust the amperage that the car sees.

This allows you to charge down to 6A in 3Phase or 1Phase setups, all the way up to 16A(3Phase) or 32A(1Phase), as long as your solar system is capable of it.

Setting `Enable Powerwall Monitoring` to true will tell the controller to talk to the powerwall defined in the configuration. 

> If you don't have a powerwall, there is the[fake-powerwall](https://github.com/shreddedbacon/fake-powerwall) project that provides a powerwall like api for some solar inverters

![powerwall page](https://github.com/shreddedbacon/twc-controller/blob/main/docs/screenshots/powerwall.png)

### Accounts Page

A problem that occurs with using the TWC in this way is that there is no smooth way to stop the car from charging, so we use the Tesla API to do that. Without entering any credentials, when the controller tells a TWC to stop charging, the car will get into a funky state that requires you to go and unplug, then re-plug the car in for it it start charging again.

Recommend creating a secondary account in the Tesla account portal, and using that account to sign in to the TWC controller, the credentials aren't stored by the controller, so if you restart you will need to sign in again in the future, the credentias also do not automatically renew, so make sure you make note of the expiration date so you can log in again later on.

No other functions are performed against the Tesla API except for waking the car up to tell the car to stop or start charging.

![accounts page](https://github.com/shreddedbacon/twc-controller/blob/main/docs/screenshots/accounts.png)

### Settings Page

Settings page is where you can set the minimum and maximum amperages to to provide to the TWC(s), and also to configure the voltage and phases values that are used to calculate watts and amps to charge with.

![settings page](https://github.com/shreddedbacon/twc-controller/blob/main/docs/screenshots/settings.png)

### SSH Access

Bundled with the controller is a web based SSH service called shellinabox, you can access this on port 4200, eg: http://192.168.1.25:4200

The default credentials for SSH access are:

* Username: tesla
* Password: tesla

### Advanced Users

#### Clone and Config

First step is to clone the repository and then edit the initial configuration for your setup.

```
# clone the repo
git clone https://github.com/shreddedbacon/twc-controller && cd twc-controller

# copy, then edit the config.yml
cp example-config.yml config.yml
```

You may need to edit `config.yml` to change the serial port device path, this depends on your setup.
If you're using a USB RS-485 serial adapter on a RasperryPi, it will typically be `/dev/ttyUSB0` and should not need adjusting.

#### Configuring With Powerwall

Once you have set up the `config.yml` file to point to your local IP for the powerwall (under `powerwall: x` in config.yml), you can run the controller.

If you want to change the port that the controller listens on, edit the `docker-compose.yml` file from `8080:8080` to `<customport>:8080`

```
docker-compose up -d
```

Once running, visit the IP address of your RaspberryPi in the browser with the configured port, eg: http://192.168.1.25:8080

#### Configuring With No Powerwall

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

#### Configuring With No Powerwall/Solar

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


## Troubleshooting

### RaspberryPI Boot up seems stuck

Sometimes the first boot of the controller fails and even after 10 minutes the controller UI is not available, or the RaspberryPi is not reachable on the network

This could be related to two things:

* Incorrect WiFi SSID/Password
  * Check you've got the correct WiFi settings configured.
* First boot failed
  * If the first boot failed (it happens :() then try restarting the RaspberryPi (power cycle it) and see if it starts up. Sometimes cloud-init in HypriotOS doesn't run properly and a restart seems to fix it.


## Acknowledgements
* Heavily inspired by https://github.com/dracoventions/TWCManager
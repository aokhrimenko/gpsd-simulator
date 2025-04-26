# gpsd-simulator

## About

The goal of this project is to provide a simple way to simulate GPS data generation in gpsd format. 
Application provides 
- a gpsd-like server
- a simple web interface to define the route and run/pause the simulation

The interface as simple as possible. All you need is to define the route by clicking on the starting and on the ending points, 
and click on the "Run simulation" button. The simulation will start and the gpsd server will start sending the data to the client.
If you then click on the "Pause simulation" button, the simulation will be paused and the gpsd server will continue sending
the latest point, but with speed set to zero. When you click on the "Run simulation" button again, the simulation will continue
from the last point.

![Demo](docs/demo.gif)

## Features

### Web interface

- [x] Define route by clicking on the starting and on the ending points
- [x] Run/pause simulation

### GPSD server

Commands:
- [x] WATCH without any parameter processing

TPV report:
- [x] Time
- [x] Latitude
- [x] Longitude
- [x] Altitude
- [x] Speed
- [x] Track
- [ ] Device customization
- [ ] Mode customization

## Installation

Download the latest [releases](https://github.com/aokhrimenko/gpsd-simulator/releases) for your platform.
Under the MacOsX before run the binary you have to remove the quarantine attribute:
```shell
xattr -d com.apple.quarantine gpsd-simulator
```

## Usage
```shell
gpsd-simulator
```
- gpsd port - port for gpsd-simulator server
- http port - port for the HTTP server of the UI

After running the application, open the browser and navigate to the `http://localhost:8881` to open the UI.
Also, don't forget to point your application which is intended to work with gpsd to the `localhost:2947`.

Default ports could be changed with command line arguments:
```shell
gpsd-simulator --gpsd-port 2947 --webui-port 8881
```

Additional debug information could be enabled with the `-d` flag, or even more debug information with `-v` flag.

## Credits

In this project the following libraries/products are used:

- [Leaflet](https://leafletjs.com/)
- [Leaflet Routing Machine](http://www.liedman.net/leaflet-routing-machine/)
- [Open Elevation API](https://open-elevation.com/)
- [gpsd](https://gpsd.gitlab.io/gpsd/index.html) not used directly, but the data format is used

# myq-teslamate-geofence
A lightweight portable app that uses the MQTT broker from TeslaMate to track your Tesla's location and operate your MyQ garage door based on whether you're arriving or leaving.

<!-- TOC -->

- [myq-teslamate-geofence](#myq-teslamate-geofence)
  - [Prerequisite](#prerequisite)
  - [How to use](#how-to-use)
    - [Docker](#docker)
    - [Portable App](#portable-app)
  - [Notes](#notes)
    - [Serials](#serials)
    - [Geofences](#geofences)
    - [Run as a Service](#run-as-a-service)
    - [Supported Environment Variables](#supported-environment-variables)
  - [Known Issues](#known-issues)

<!-- /TOC -->

## Prerequisite
This app uses the MQTT broker bundled with [TeslaMate](https://github.com/adriankumpf/teslamate). You must be running TeslaMate and have the MQTT broker exposed for consumption to use this app. TeslaMate has many other features that make it more than worthwhile to use in addition to this app.

## How to use
### Docker
There is now a docker image available and will be the only supported release type going forward. You will still need to download the [config.example.yml](https://github.com/brchri/myq-teslamate-geofence/blob/main/config.example.yml) file and edit it appropriately, and then mount it to the container at runtime. For example:

```bash
docker run \
  -e MYQ_EMAIL=my_email@address.com \ # optional, can also be saved in the config.yml file
  -e MYQ_PASS=my_super_secret_pass \ # optional, can also be saved in the config.yml file
  -v /etc/myq-teslamate-geofence/config.yml:/app/config/config.yml:ro \ # required, config file volume
  -v /etc/timezone:/etc/timezone:ro \ # optional, timezone file to set timezone based on host machine
  -v /etc/localtime:/etc/localtime:ro \ # optional, localtime file to set time based on host machine
  brchri/myq-teslamate-geofence:0.1.0
```

Or you can use a docker compose file like this:

```yaml
version: '3.8'
services:

  myq-teslamate-geofence:
    image: brchri/myq-teslamate-geofence:0.1.0
    container_name: myq-teslamate-geofence
    environment:
      - MYQ_EMAIL=my_email@address.com # optional, can also be saved in the config.yml file
      - MYQ_PASS=my_super_secret_pass # optional, can also be saved in the config.yml file
    volumes:
      - /etc/myq-teslamate-geofence/config.yml:/app/config/config.yml:ro # required, config file volume
      - /etc/timezone:/etc/timezone:ro # timezone file to set timezone based on host machine
      - /etc/localtime:/etc/localtime:ro # localtime file to set time based on host machine
    restart: unless-stopped
```

### Portable App
Deprecated after release `v0.1.0`. Please refer to the [Docker](#docker) instructions for more recent versions. For earlier versions, continue on.

Download the release zip containing the binary and sample `config.example.yml` file. Edit the yml file to have the settings appropriate for your use case (see Notes section below for more info).

Open a terminal (e.g. bash on linux or cmd/powershell on Windows), `cd` to the directory containing the downloaded binary, and execute it with a `-c` flag to point to your config file. Here's an example:

`myq-teslamate-geofence -c /etc/myq-teslamate-geofence/config.yml`

You can also set `CONFIG_FILE` environment variable to pass the config file directory:

```bash
export CONFIG_FILE=/etc/myq-teslamate-geofence/config.yml
myq-teslamate-geofence
```

## Notes

### Serials
The serial displayed in your MyQ app may not be the serial used to control your door (e.g. it may be the hub rather than the opener). You can run this app with the `-d` flag to list your device serials and pick the appropriate one (listed with `type: garagedooropener`). Example:

Docker image:

```shell
docker run --rm \
  -e MYQ_EMAIL=myq@example.com \
  -e MYQ_PASS=supersecretpass \
  brchri/myq-teslamate-geofence:0.1.0 \
  myq-teslamate-geofence -d
```

Portable app:

`MYQ_EMAIL=myq@example.com MYQ_PASS=supersecretpass myq-teslamate-geofence -d`

### Geofences
There are separate geofences for opening the garage and closing it. This is to facilitate closing the garage more immediately when leaving, but opening it sooner so it's already open when you arrive. This is useful due to delays in receiving positional data from the Tesla API. The recommendation is to set a larger value for `open_radius` and a smaller one for `close_radius`, but this is up to you.

### Run as a Service
This is only relevant for the [Portable App](#portable-app) installation, which has been deprecated.

You can run the portable app as a service, and there is a sample systemd service file in the root of the repo. Instructions for how to use the service file are outside the scope of this README, but there is ample documentation online.

### Supported Environment Variables
The following environment variables are supported:
```bash
CONFIG_FILE=<path> # path to config file, can be used instead of -c flag
MYQ_EMAIL=<string> # this can be set instead of setting these values in the config.yml file
MYQ_PASS=<string> # this can be set instead of setting these values in the config.yml file
DEBUG=<bool> # prints more verbose messages
TESTING=<bool> # will not actually operate the garage door
```

## Known Issues
* ~~Currently this only works with one vehicle. It is set up to work with multiple, but it hangs when receiving broker messages from MQTT for some reason. I haven't yet had time to dig into this.~~
  * This should be fixed as of v0.0.3

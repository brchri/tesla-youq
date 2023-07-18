# Tesla-YouQ
A lightweight app that will operate your MyQ connected garage doors based on the location of your Tesla vehicles, automatically closing when you leave, and opening when you return. Supports multiple vehicles and MyQ devices.

<!-- TOC -->

- [Tesla-YouQ](#tesla-youq)
  - [Prerequisite](#prerequisite)
  - [How to use](#how-to-use)
    - [Docker](#docker)
    - [Supported Environment Variables](#supported-environment-variables)
    - [Portable App](#portable-app)
  - [Notes](#notes)
    - [Serials](#serials)
    - [Geofence Radii](#geofence-radii)
    - [Custom Geofence vs TeslaMate Geofence](#custom-geofence-vs-teslamate-geofence)
    - [Triggers](#triggers)
  - [Credits](#credits)

<!-- /TOC -->

## Prerequisite
This app uses the MQTT broker bundled with [TeslaMate](https://github.com/adriankumpf/teslamate). You must be running TeslaMate and have the MQTT broker exposed for consumption to use this app. TeslaMate has done a lot of work in scraping API data while minimizing vampire drain on vehicles from API requests, and TeslaMate has many other features that make it more than worthwhile to use in addition to this app.

## How to use
### Docker
There is now a docker image available and will be the only supported release type going forward. You will need to download the [config.example.yml](https://github.com/brchri/tesla-youq/blob/main/config.example.yml) file and edit it appropriately, and then mount it to the container at runtime. For example:

```bash
docker run \
  -e MYQ_EMAIL=my_email@address.com \ # optional, can also be saved in the config.yml file
  -e MYQ_PASS=my_super_secret_pass \ # optional, can also be saved in the config.yml file
  -e TZ=America/New_York \ # optional, sets timezone for container
  -v /etc/tesla-youq/config.yml:/app/config/config.yml:ro \ # required, mounts config file into container
  brchri/tesla-youq:latest
```

Or you can use a docker compose file like this:

```yaml
version: '3.8'
services:

  tesla-youq:
    image: brchri/tesla-youq:latest
    container_name: tesla-youq
    environment:
      - MYQ_EMAIL=my_email@address.com # optional, can also be saved in the config.yml file
      - MYQ_PASS=my_super_secret_pass # optional, can also be saved in the config.yml file
      - TZ=America/New_York # optional, sets timezone for container
    volumes:
      - /etc/tesla-youq/config.yml:/app/config/config.yml:ro # required, mounts config file into container
    restart: unless-stopped
```

### Supported Environment Variables
The following Docker environment variables are supported but not required.
| Variable Name | Type | Description |
| ------------- | ---- | ----------- |
| `CONFIG_FILE` | String (Filepath) | Path to config file within container |
| `MYQ_EMAIL` | String | Email to authenticate to MyQ account. Can be used instead of setting `myq_email` in the `config.yml` file |
| `MYQ_PASS` | String | Password to authenticate to MyQ account. Can be used instead of setting `myq_pass` in the `config.yml` file |
| `MQTT_USER` | String | User to authenticate to MQTT broker. Can be used instead of setting `mqtt_user` in the `config.yml` file |
| `MQTT_PASS` | String | Password to authenticate to MQTT broker. Can be used instead of setting `mqtt_pass` in the `config.yml` file |
| `DEBUG` | Bool | Increases output verbosity |
| `TESTING` | Bool | Will perform all functions *except* actually operating garage door, and will just output operation *would've* happened |
| `TZ` | String | Sets timezone for container |

### Portable App
Deprecated after release `v0.1.0`. Please refer to the [Docker](#docker) instructions for more recent versions. For earlier versions, continue on.

Download the release zip containing the binary and sample `config.example.yml` file. Edit the yml file to have the settings appropriate for your use case (see Notes section below for more info).

Open a terminal (e.g. bash on linux or cmd/powershell on Windows), `cd` to the directory containing the downloaded binary, and execute it with a `-c` flag to point to your config file. Here's an example:

`tesla-youq -c /etc/tesla-youq/config.yml`

You can also set `CONFIG_FILE` environment variable to pass the config file directory:

```bash
export CONFIG_FILE=/etc/tesla-youq/config.yml
tesla-youq
```

## Notes

### Serials
The serial displayed in your MyQ app may not be the serial used to control your door (e.g. it may be the hub rather than the opener). You can run this app with the `-d` flag to list your device serials and pick the appropriate one (listed with `type: garagedooropener`). Example:

Docker image:

```shell
docker run --rm \
  -e MYQ_EMAIL=myq@example.com \
  -e MYQ_PASS=supersecretpass \
  brchri/tesla-youq:latest \
  tesla-youq -d
```

Portable app:

`MYQ_EMAIL=myq@example.com MYQ_PASS=supersecretpass tesla-youq -d`

### Geofence Radii
There are separate geofence radii for opening and closing a garage. This is to facilitate closing the garage more immediately when leaving, but opening it sooner so it's already open when you arrive. This is useful due to delays in receiving positional data from the Tesla API. The recommendation is to set a larger value for `open_radius` and a smaller one for `close_radius`, but this is up to you.

### Custom Geofence vs TeslaMate Geofence
As of `v0.1.2`, you can either define your own geofence triggers in the `config.yml` file (using `location`, `close_radius`, and `open_radius` parameters), or you can reference the geofences that can be created directly through the TeslaMate web UI (using `trigger_close_geofence` and `trigger_open_geofence` parameters). However, it should be noted that if using the Geofences defined in the TeslaMate web UI, Tesla-YouQ will subscribe to the Geofence topic for updates to trigger garage door actions. This topic does not appear to be updated in realtime, but rather in a polling manner, which can cause significant delays in updates that trigger a garage close action. It also does not (currently) allow for overlapping geofence definitions. It is therefore recommended to define your own geofences in the `config.yml` file rather than relying on TeslaMate's geofencing feature. See [this PR](https://github.com/brchri/tesla-youq/pull/12) for more detailed observations.

### Triggers
Tesla-YouQ allows you to define separate geofence radii for open and close triggers. This means that there can be an overlap where, for example, you're leaving home and cross the "close door" threshold, which shares a space with the "open door" zone for when you return, as the "open door" zone is larger since you want the garage to start opening sooner so it's ready when you arrive. To account for this and remove the possibility of flapping, The close door event will only trigger when you go from *inside* the "close door" geofence and move *outside* it, indicating you are moving away from the garage. Conversely, the open door event will only trigger when you go from *outside* the "open door" geofence and move *inside* it.

There is also a configurable "operation cooldown" property in the `global` section of the `config.yml` file, which will prevent any operations against a recently operated door until the number of minutes specified has passed. This can be set to 0 if you don't want any operation cooldown.

## Credits
* [TeslaMate](https://github.com/adriankumpf/teslamate)
* [MyQ API Go Package](https://github.com/joeshaw/myq)

# Tesla-YouQ
A lightweight app that will operate your MyQ connected garage doors based on the location of your Tesla vehicles, automatically closing when you leave, and opening when you return. Supports multiple vehicles and MyQ devices.

<!-- TOC -->

- [Tesla-YouQ](#tesla-youq)
  - [Prerequisite](#prerequisite)
  - [How to use](#how-to-use)
    - [Docker](#docker)
    - [Portable App](#portable-app)
  - [Notes](#notes)
    - [Serials](#serials)
    - [Geofences](#geofences)
    - [Supported Environment Variables](#supported-environment-variables)
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
  -v /etc/tesla-youq/config.yml:/app/config/config.yml:ro \ # required, config file volume
  brchri/tesla-youq:0.1.1
```

Or you can use a docker compose file like this:

```yaml
version: '3.8'
services:

  tesla-youq:
    image: brchri/tesla-youq:0.1.1
    container_name: tesla-youq
    environment:
      - MYQ_EMAIL=my_email@address.com # optional, can also be saved in the config.yml file
      - MYQ_PASS=my_super_secret_pass # optional, can also be saved in the config.yml file
      - TZ=America/New_York # optional, sets timezone for container
    volumes:
      - /etc/tesla-youq/config.yml:/app/config/config.yml:ro # required, config file volume
    restart: unless-stopped
```

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
  brchri/tesla-youq:0.1.1 \
  tesla-youq -d
```

Portable app:

`MYQ_EMAIL=myq@example.com MYQ_PASS=supersecretpass tesla-youq -d`

### Geofences
There are separate geofences for opening and closing a garage. This is to facilitate closing the garage more immediately when leaving, but opening it sooner so it's already open when you arrive. This is useful due to delays in receiving positional data from the Tesla API. The recommendation is to set a larger value for `open_radius` and a smaller one for `close_radius`, but this is up to you.

### Supported Environment Variables
| Variable Name | Type | Description | Default |
| ------------- | ---- | ----------- | ------- |
| CONFIG_FILE | path (string) | Path to config file within container | `/app/config/config.yml` |
| MYQ_EMAIL | string | Email to authenticate to MyQ account. Can be used instead of setting in `config.yml` file | `''` |
| MYQ_PASS | string | Password to authenticate to MyQ account. Can be used instead of setting in `config.yml` file | `''` |
| MQTT_USER | string | User to authenticate to MQTT broker. Can be used instead of setting in `config.yml` file | `''` |
| MQTT_PASS | string | Password to authenticate to MQTT broker. Can be used instead of setting in `config.yml` file | `''` |
| DEBUG | bool | Increases output verbosity | `false` |
| TESTING | bool | Will perform all functions *except* actually operating garage door, and will just output operation *would've* happened | `false` |
| TZ | string | Sets timezone for container | `Etc/UTC` |

## Credits
* [TeslaMate](https://github.com/adriankumpf/teslamate)
* [MyQ API Go Package](https://github.com/joeshaw/myq)
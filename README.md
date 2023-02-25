# myq-teslamate-geofence
A lightweight portable app that uses the MQTT broker from TeslaMate to watch geofence event changes and open or close the garage door accordingly

## How to use
Download the release zip containing the binary and sample config.example.yml file. Edit the yml file to have the settings appropriate for your use case.

Note that there are separate geofences for opening the garage and closing it. This is to facilitate closing the garage more immediately when leaving, but opening it sooner so it's already open when you arrive. This is useful due to delays in receiving positional data from the Tesla API. The recommendation is to set a larger `geo_radius` for `garage_open_geofence` and a smaller one for `garage_close_geofence`, but this is up to you.

Open a terminal (e.g. bash on linux or cmd/powershell on Windows), `cd` to the directory containing the downloaded binary, and execute it with a `-c` flag to point to your config file. Here's an example:

`myq-teslamate-geofence -c /etc/myq-teslamate-geofence/config.yml`

You can also set `CONFIG_FILE` environment variable to pass the config file directory:

```bash
export CONFIG_FILE=/etc/myq-teslamate-geofence/config.yml
myq-teslamate-geofence
```

You can also run this as a service, and there is a sample systemd service file in the root of the repo. Instructions for how to use the service file are outside the scope of this README, but there is ample documentation online.

## Notes
The following environment variables are supported:
```bash
CONFIG_FILE=<path> # path to config file, can be used instead of -c flag
MYQ_EMAIL=<string> # this can be set instead of setting these values in the config.yml file
MYQ_PASS=<string> # this can be set instead of setting these values in the config.yml file
DEBUG=<bool> # prints more verbose messages
TESTING=<bool> # will not actually operate the garage door
```

## Known Issues
* Currently this only works with one vehicle. It is set up to work with multiple, but it hangs when receiving broker messages from MQTT for some reason. I haven't yet had time to dig into this.

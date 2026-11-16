# OpenVPN config file loader

## Description

The program starts a http server, listening on the `WEB_PORT` you have set on in your `.env`file. Starts `openvpn` with initial config, which you place in `VPN_CONFIG_FILE`. The purpose is to have an API that can list and load preset ovpn configs instead of the default OpenVPN management interface. The loader prevents relyant services from accessing the internet, unless is through tun0.

## Setup

In your `.env` file:
- `VPN_CONFIG_FILE` - the name of the initial config file you have provided.
- `VPN_CONFIG_USERNAME` - your VPN username.
- `VPN_CONFIG_PASSWORD` - your VPN password.
- `LOG_LEVEL` - the log levelyou wnat the web server to report. Valid are `DEBUG`, `ERROR`, `INFO` and `WARNING`. Equivalent to slog.
- `WEB_PORT` - the port you want to access the vpn loader application.

**Replace the `wait` service in `docker-compose.yml` file, with the service you want to be routed through the vpn. Do not forget to set `network_mode` and `depends_on`.**

## Interface

It is at this ip.

```
http://<the host you are running it on>:<WEB_PORT>/
```

For example:
```
http://localhost:8080
```

Click one of the load buttons, and it loads the config you selected. The design is ai slop.

## Disclamer

At this point, the server does not support any authentication. It is ment to run on a local network.

Check thouh what external ip the other services are exposed with `docker exec vpnloader-wait-1 wget -qO- https://ifconfig.me`.

If it fails, and at some point, your services are exposed though your own IP, the author does not bear responsibilitiy. Althought, the IP tables should prevent that.

Any input is welcome.

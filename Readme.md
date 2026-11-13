# OpenVPN config file loader

## Description

The program starts a http server, listenning to two endpoints `/config` and `/load`, and starts `openvpn` with initial config. The purpose is to have an API that can list and load preset ovpn configs instead of the default OpenVPN management interface.

### `/configs`

Lists the configs situated in `./conf.d/` folder. The configs should be `.ovpn`.

```
curl http://<the host you are running it on>:8080/configs
```

### `/load`

Loads a config, by copying it from the `./conf.d` folder and moving it in `./conf/` directory. Creates the directory if not existant. Also adds `auth-user-pass ./auth.txt` at the end of the copied config file.

```
curl -d "config=<the name of the config file>" http://<the host you are running it on>:8080/load
```

## Front end

For those who can only click buttons, from those who can't be bothered with FE.

```
http://<the host you are running it on>/
```
Click one of the load buttons, and it loads the config you selected. If I get extremely bored, and I can't do anything else I will improve it, but first I will try to sit that thought off.

## Disclamer

At this point, the server does not support any authentication. It is ment to run on a local network.

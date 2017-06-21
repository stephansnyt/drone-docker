# drone-gdm

[![Build Status]]
[![Go Doc]]
[![Go Report]]
[![Join the chat at https://gitter.im/drone/drone](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/drone/drone)

Drone plugin can be used to drive [Google Cloud Deployment Manager](https://cloud.google.com/deployment-manager/)

## Build

Build the binary with the following commands:

```
go build
```

## Docker

Build the docker image with the following commands:

```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo
docker build --rm=true -t plugins/gdm .
```

Please note incorrectly building the image for the correct x64 linux and with
GCO disabled will result in an error when running the Docker image:

```
docker: Error response from daemon: Container command
'/bin/drone-gdm' not found or does not exist..
```

## Usage

Execute from the working directory:

```
docker run --rm \
  -e PLUGIN_DEPLOYMENT=test-deployment \
  -e PLUGIN_VARS='{"port":80,"path":"/health"}' \
  -e TOKEN=$(<${gcp_service_account_json}) \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  --privileged \
  plugins/gdm --dry-run
```

This uses `.gdm.yml` given in this repo as an example.

Note that the first time this runs, you must `create` the deployment out-of-band, or else run it with `-e PLUGIN_ACTION=create`.

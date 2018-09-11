#!/bin/bash -ex

unbind() {
  cf unbind-service $1 pubsub
  while cf service pubsub | grep -q $1; do
    sleep 2
  done
}

unbind web-app
unbind worker-app

cf d web-app -f
cf d worker-app -f

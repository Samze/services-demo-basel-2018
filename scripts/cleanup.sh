#!/bin/bash -ex

unbind() {
  cf unbind-service $1 $2
  while cf service $2 | grep -q $1; do
    sleep 2
  done
}

# broker does not support per service unbinds in parallel
unbind web-app pubsub
unbind worker-app pubsub

unbind web-app postgresql
unbind worker-app postgresql

cf d web-app -f
cf d worker-app -f

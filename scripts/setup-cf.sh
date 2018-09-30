#!/bin/bash -ex

wait_on_bind_completion() {
  while cf service pubsub | grep $1 | grep -q "in progress"; do
    sleep 2
  done
}

create_service_if_not_present() {
  if cf service "$3"; then
    return
  fi

  cf create-service "$@"

  while cf service "$3" | grep -q "in progress"; do
    sleep 2
  done

  if cf service "$3" | grep "failed"; then
    echo "service $3 creation failed"
    exit 1
  fi
}

create_service_if_not_present watson-vision-combined lite vision
create_service_if_not_present cloud-pubsub beta pubsub -c '{"topicId": "cf_topic"}'
create_service_if_not_present azure-postgresql-9-6 basic postgresql -c '{ "resourceGroup" : "basel2018", "location" : "uksouth", "firewallRules" : [ { "name": "AllowAll", "startIPAddress": "0.0.0.0", "endIPAddress" : "255.255.255.255" } ] }'

pushd web-app
  make build-linux
  cf push web-app --no-start

  cf bind-service web-app pubsub -c '{"createServiceAccount": true, "roles": ["roles/pubsub.publisher", "roles/pubsub.viewer"], "serviceAccount": "pubsub-test-publisher" }'
  wait_on_bind_completion web-app

  cf bind-service web-app postgresql

  cf restart web-app
popd

pushd worker-app
  make build-linux
  cf push worker-app --no-start

  cf bind-service worker-app pubsub -c '{"createServiceAccount": true, "serviceAccount": "pubsub-test-subscriber", "subscription": {"subscriptionId": "subscription"}, "roles":["roles/pubsub.subscriber", "roles/pubsub.viewer"]}'
  wait_on_bind_completion worker-app

  cf bind-service worker-app postgresql
  cf bind-service worker-app vision

  cf restart worker-app
popd



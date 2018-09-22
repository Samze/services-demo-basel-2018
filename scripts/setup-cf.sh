#!/bin/bash -ex

wait_on_bind_completion() {
  while cf service pubsub | grep $1 | grep -q "in progress"; do
    sleep 2
  done
}

pushd web-app
  cf push web-app --no-start

  cf bind-service web-app pubsub -c '{"createServiceAccount": true, "roles": ["roles/pubsub.publisher", "roles/pubsub.viewer"], "serviceAccount": "pubsub-test-publisher" }'
  wait_on_bind_completion web-app

  cf bind-service web-app postgresql

  cf restart web-app
popd

pushd worker-app
  cf push worker-app --no-start

  cf bind-service worker-app pubsub -c '{"createServiceAccount": true, "serviceAccount": "pubsub-test-subscriber", "subscription": {"subscriptionId": "subscription"}, "roles":["roles/pubsub.subscriber", "roles/pubsub.viewer"]}'
  wait_on_bind_completion worker-app

  cf bind-service worker-app postgresql

  cf restart worker-app
popd



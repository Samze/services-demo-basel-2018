#!/bin/bash -x

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

svcat provision vision --class watson-vision-combined --plan standard-rc
svcat provision pubsub --class cloud-pubsub --plan beta --param topicId=k8s_topic
svcat provision postgresql --class azure-postgresql-9-6 --plan basic --params-json '{ "location": "eastus", "resourceGroup": "demo", "firewallRules" : [ { "name": "AllowAll", "startIPAddress": "0.0.0.0", "endIPAddress" : "255.255.255.255" } ] }' --wait

svcat bind postgresql --name postgresql-binding
svcat bind vision --name vision-binding
svcat bind pubsub --name pub-binding --params-json '{ "createServiceAccount": true, "roles": ["roles/pubsub.publisher", "roles/pubsub.viewer"], "serviceAccount": "pubsub-test" }' --wait
svcat bind pubsub --name sub-binding --params-json '{ "createServiceAccount": true, "roles": ["roles/pubsub.subscriber", "roles/pubsub.viewer"], "serviceAccount": "pubsub-subscriber", "subscription": {"subscriptionId": "k8s-subscription"} }' --wait

kubectl create -f "$DIR/../k8s/web-app.yaml"
kubectl create -f "$DIR/../k8s/worker-app.yaml"

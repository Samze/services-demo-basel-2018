#!/bin/bash -ex

svcat provision postgresql --class azure-postgresql-9-6 --plan basic --params-json '{ "location": "eastus", "resourceGroup": "demo", "firewallRules" : [ { "name": "AllowAll", "startIPAddress": "0.0.0.0", "endIPAddress" : "255.255.255.255" } ] }'
svcat provision vision --class watson-vision-combined --plan standard-rc
svcat provision pubsub --class cloud-pubsub --plan beta

svcat bind postgresql --name postgresql-binding
svcat bind vision --name vision-binding
svcat bind pubsub --name publisher-binding --params-json '{ "createServiceAccount": true, "roles": ["roles/pubsub.publisher", "roles/pubsub.viewer"], "serviceAccount": "pubsub-test" }'

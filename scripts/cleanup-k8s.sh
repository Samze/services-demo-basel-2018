#!/bin/bash -x

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

kubectl delete -f "$DIR/../k8s/web-app.yaml"
kubectl delete -f "$DIR/../k8s/worker-app.yaml"

svcat unbind --name vision-binding --wait
svcat unbind --name postgresql-binding --wait
svcat unbind --name pub-binding --wait
svcat unbind --name sub-binding --wait

svcat deprovision vision
svcat deprovision pubsub
svcat deprovision postgresql --wait




#!/usr/bin/env bash

########################
# include the magic
########################
. ~/workspace/demo-magic/demo-magic.sh


########################
# Configure the options
########################

#
# speed at which to simulate typing. bigger num = faster
#
# TYPE_SPEED=20

#
# custom prompt
#
# see http://www.tldp.org/HOWTO/Bash-Prompt-HOWTO/bash-prompt-escape-sequences.html for escape sequences
#
DEMO_PROMPT="${GREEN}\h ➜ ${CYAN}\w "

# hide the evidence
clear
cd ~/go/src/github.com/Samze/services-demo-basel-2018 || return
svcat unbind --name vision-binding --wait > /dev/null 2>&1
svcat deprovision vision --wait > /dev/null 2>&1
svcat delete -f k8s/web-app.yaml -f k8s/worker-app.yaml > /dev/null 2>&1
# svcat unbind --name db-binding > /dev/null 2>&1


# k8s demo
pe "svcat get classes"
pe "clear"
pe "svcat describe class watson-vision-combined"
pe "svcat provision vision --class watson-vision-combined --plan standard-rc"
pe "svcat describe instance vision"
pe "svcat bind vision --name vision-binding"
pe "svcat describe binding vision-binding --show-secrets"
pe "svcat get instances"
pe "svcat get bindings"
pe "vim k8s/worker-app.yaml"
pe "kubectl create -f k8s/worker-app.yaml -f k8s/web-app.yaml"
pe "kubectl get deployments"
pe "kubectl get services --field-selector metadata.name=web-app-service"

cmd

# show a prompt so as not to reveal our true nature after
# the demo has concluded
p ""

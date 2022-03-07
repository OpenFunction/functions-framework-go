#!/bin/bash

data_result=$(KUBECONFIG=/tmp/e2e-k8s.config kubectl logs --tail=2 -l app="pubsub-subscriber" -c sub | grep Data | awk '{ print $8 }' | yq -P '.' -)
plugin_result=$(KUBECONFIG=/tmp/e2e-k8s.config kubectl logs --tail=2 -l app="pubsub-subscriber" -c sub | grep plugin | awk '{ print $8 }' | yq -P '.' -)

echo "$data_result"
echo "$plugin_result"

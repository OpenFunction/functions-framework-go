#!/bin/bash

data_result=$(KUBECONFIG=/tmp/e2e-k8s.config kubectl logs --tail=2 -l app="bindings-target" -c target | grep Data | awk '{ print $8 }' | yq -P '.' -)
plugin_result=$(KUBECONFIG=/tmp/e2e-k8s.config kubectl logs --tail=2 -l app="bindings-target" -c target | grep plugin | awk '{ print $8 }' | yq -P '.' -)

echo "$data_result"
echo "$plugin_result"

#!/bin/bash

url=http://$1
while true; do
  st=$(curl -s -o /dev/null -w "%{http_code}" "$url" -H "Content-Type: application/cloudevents+json" -d '{"specversion":"1.0","type":"dev.knative.samples.helloworld","source":"dev.knative.samples/helloworldsource","id":"536808d3-88be-4077-9d7a-a3f162705f79","data":{"data":"hello"}}')
  if [ "$st" -eq 200 ]; then
    data_result=$(KUBECONFIG=/tmp/e2e-k8s.config kubectl logs --tail=2 -l app="sync-http-variables" -c http | grep Data | awk '{ print $8 }' | yq -P '.' -)
    plugin_result=$(KUBECONFIG=/tmp/e2e-k8s.config kubectl logs --tail=2 -l app="sync-http-variables" -c http | grep plugin | awk '{ print $8 }' | yq -P '.' -)
    break
  else
    sleep 1
    continue
  fi
done

echo "$data_result"
echo "$plugin_result"

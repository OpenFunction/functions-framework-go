#!/bin/bash

url=http://$1
while true; do
  st=$(curl -s -o /dev/null -w "%{http_code}" "$url" -H "Ce-Specversion: 1.0" -H "Ce-Type: io.openfunction.samples.helloworld" -H "Ce-Source: io.openfunction.samples/helloworldsource" -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" -H "Content-Type: application/json" -d 'hello')
  if [ "$st" -eq 200 ]; then
    data_result=$(curl -X POST -H "Content-type: application/json" -H "Accept: application/json" -s "$url" -H "Ce-Specversion: 1.0" -H "Ce-Type: io.openfunction.samples.helloworld" -H "Ce-Source: io.openfunction.samples/helloworldsource" -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" -H "Content-Type: application/json" -d 'hello' | yq -P ".")
    plugin_result=$(KUBECONFIG=/tmp/e2e-k8s.config kubectl logs --tail=1 -l app="sync-http-variables" -c http | grep plugin | awk '{ print $8 }' | yq -P '.' -)
    break
  else
    sleep 1
    continue
  fi
done

echo "$data_result"
echo "$plugin_result"

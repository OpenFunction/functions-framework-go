#!/bin/bash

url=http://$1
while true; do
  st=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$url")
  if [ "$st" -eq 200 ]; then
    data_result=$(curl -X POST -H "Content-type: application/json" -H "Accept: application/json" -s "$url" -d 'hello' | yq -P ".")
    plugin_result=$(KUBECONFIG=/tmp/e2e-k8s.config kubectl logs --tail=1 -l app="sync-http-variables" -c http | awk '{ print $8 }' | yq -P '.' -)
    break
  else
    sleep 1
    continue
  fi
done

echo "$data_result"
echo "$plugin_result"

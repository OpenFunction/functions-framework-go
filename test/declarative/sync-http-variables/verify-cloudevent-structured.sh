#!/bin/bash

url=http://$1
while true; do
  st=$(curl -s -o /dev/null -w "%{http_code}" "$url" -H "Content-Type: application/json" -d '{"specversion" : "1.0", "type" : "example.com.cloud.event", "source" : "https://example.com/cloudevents/pull", "subject" : "123", "id" : "A234-1234-1234", "time" : "2018-04-05T17:31:00Z", "data" : "hello"}')
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

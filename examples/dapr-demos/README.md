# Dapr examples

In order to follow these examples, you need to [install Dapr](https://docs.dapr.io/getting-started/install-dapr-selfhost/).

## Bindings via gRPC

This input source will be executed every 2s (Refer to [cron.yaml](../config/cron.yaml)).

Prepare a context as follows, name it `input.json`. (You can refer to [types.go](../../openfunction-context/types.go) to learn more about the OpenFunction Context)

>This indicates that the input of the function is a Dapr  Component with parameters are:
>
>`spec.type` is "bindings.*" derived from `input.in_type`
>
>`metadata.name` is "cron_input" derived from `input.name`
>
>`app-protocol` is "gRPC" derived from `protocol`
>
>`app-port` is "50001" derived from `port`

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "protocol": "gRPC",
  "port": "50001",
  "input": {
    "name": "cron_input",
    "enabled": true,
    "in_type": "bindings"
  },
  "outputs": {
    "enabled": false
  },
  "runtime": "Dapr"
}
```

Create an environment variable `FUNC_CONTEXT` and assign the above context to it.

```shell
export FUNC_CONTEXT='{"name":"MyFunc","version":"v1","request_id":"a0f2ad8d-5062-4812-91e9-95416489fb01","protocol":"gRPC","port":"50001","input":{"name":"cron_input","enabled":true,"in_type":"bindings"},"outputs":{"enabled":false},"runtime":"Dapr"}'
```

Start the service and watch the logs.

```shell
dapr run --app-id serving_function \
    --app-protocol grpc \
    --app-port 50001 \
    --components-path ../config \
    go run ../serving/grpc/bindings_without_output.go
```

<details>
<summary>View detailed logs.</summary>


```shell
ℹ️  Starting Dapr with id serving_function. HTTP Port: 33787. gRPC Port: 3500

ℹ️  Updating metadata for app command: go run ../serving/grpc/main.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == 2021/06/01 16:17:46 Function serving grpc: listening on port 50001

== APP == 2021/06/07 19:49:26 binding - Data:, Meta:map[readTimeUTC:2021-06-07 11:49:26.000470756 +0000 UTC timeZone:Local]

== APP == 2021/06/07 19:49:28 binding - Data:, Meta:map[readTimeUTC:2021-06-07 11:49:28.000193993 +0000 UTC timeZone:Local]

== APP == 2021/06/07 19:49:30 binding - Data:, Meta:map[readTimeUTC:2021-06-07 11:49:30.000218353 +0000 UTC timeZone:Local]
```

</details>

### With output

We need to prepare an output target first.

```shell
dapr run --app-id output \
    --app-protocol http \
    --app-port 7489 \
    --dapr-http-port 7490 \
    go run ../outputs/main.go
```

This will generate two available targets, one for access through Dapr's proxy address and another for direct access through the app serving address.

> Simple test with execution `curl -X POST -H "ContentType: application/json" -d '{"Hello": "World"}' <urlPath>`
>
> `urlPath` refer to follows.

```
via Dapr: http://localhost:7490/v1.0/invoke/output_demo/method/echo
via App: http://localhost:7489/echo
```

In this example, the proxy address of Dapr will be used as the target of output.

>Here we have defined only one output, which will be called `item` in the following
>
>`app-id` is "echo" derived from the key of `item`
>
>Dapr component type is "bindings" derived from `item.out_type` while its params are in `item.params`. Refer to [Dapr components reference](https://docs.dapr.io/reference/components-reference/).

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "protocol": "gRPC",
  "port": "50001",
  "input": {
    "name": "cron_input",
    "enabled": true,
    "in_type": "bindings"
  },
  "outputs": {
    "enabled": true,
    "output_objects": {
      "echo": {
        "out_type": "bindings",
        "params": {
          "operation": "create",
          "metadata": "{\"path\": \"/echo\", \"Content-Type\": \"application/json; charset=utf-8\"}"
        }
      }
    }
  },
  "runtime": "Dapr"
}
```

Create an environment variable `FUNC_CONTEXT` and assign the above context to it.

```shell
export FUNC_CONTEXT='{"name":"MyFunc","version":"v1","request_id":"a0f2ad8d-5062-4812-91e9-95416489fb01","protocol":"gRPC","port":"50001","input":{"name":"cron_input","enabled":true,"in_type":"bindings"},"outputs":{"enabled":true,"output_objects":{"echo":{"out_type":"bindings","params":{"operation":"create","metadata":"{\"path\": \"/echo\", \"Content-Type\": \"application/json; charset=utf-8\"}"}}}},"runtime":"Dapr"}'
```

Start the service and watch the logs.

```shell
dapr run --app-id serving_function \
    --app-protocol grpc \
    --app-port 50001 \
    --components-path ../config \
    go run ../serving/grpc/bindings_with_output.go
```

The logs of user function is ...

<details>
<summary>View detailed logs.</summary>

```shell
ℹ️  Starting Dapr with id serving_function. HTTP Port: 45509. gRPC Port: 3500

== APP == 2021/06/07 19:50:11 binding - Data:, Meta:map[readTimeUTC:2021-06-07 11:50:11.000105185 +0000 UTC timeZone:Local]

== APP == 2021/06/07 19:50:13 binding - Data:, Meta:map[readTimeUTC:2021-06-07 11:50:13.000938636 +0000 UTC timeZone:Local]
```

</details>

And the logs of output target app is ...

<details>
<summary>View detailed logs.</summary>

```shell
ℹ️  Starting Dapr with id output_demo. HTTP Port: 7490. gRPC Port: 43973

ℹ️  Updating metadata for app command: go run ../outputs/main.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == 2021/06/07 19:50:11 Receive a message:

== APP == 2021/06/07 19:50:11 Hello

== APP == 2021/06/07 19:50:13 Receive a message:

== APP == 2021/06/07 19:50:13 Hello
```

</details>

## Bindings via HTTP

This input source will be executed every 2s (Refer to [cron.yaml](../config/cron.yaml)).

Prepare a context as follows, name it `input.json`. (You can refer to [types.go](../../openfunction-context/types.go) to learn more about the OpenFunction Context)

>This indicates that the input of the function is a Dapr  Component with parameters are:
>
>`spec.type` is "bindings.*" derived from `input.in_type`
>
>`metadata.name` is "cron_input" derived from `input.name`
>
>`app-protocol` is "gRPC" derived from `protocol`
>
>`app-port` is "50001" derived from `port`

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "protocol": "HTTP",
  "port": "8080",
  "input": {
    "name": "cron_input",
    "enabled": true,
    "in_type": "bindings"
  },
  "outputs": {
    "enabled": false
  },
  "runtime": "Dapr"
}
```

Create an environment variable `FUNC_CONTEXT` and assign the above context to it.

```shell
export FUNC_CONTEXT='{"name":"MyFunc","version":"v1","request_id":"a0f2ad8d-5062-4812-91e9-95416489fb01","protocol":"HTTP","port":"8080","input":{"name":"cron_input","enabled":true,"in_type":"bindings"},"outputs":{"enabled":false},"runtime":"Dapr"}'
```

Start the service and watch the logs.

```shell
dapr run --app-id serving_function \
    --app-protocol http \
    --app-port 8080 \
    --components-path ../config \
    go run ../serving/http/bindings_without_output.go
```

<details>
<summary>View detailed logs.</summary>

```shell
ℹ️  Starting Dapr with id serving_function. HTTP Port: 44947. gRPC Port: 44865

ℹ️  Updating metadata for app command: go run ../serving/http/main.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == 2021/06/07 19:55:55 Function serving http: listening on port 8080

== APP == 2021/06/07 19:55:57 binding - Data:, Header:map[Content-Length:[0] Content-Type:[application/json] Readtimeutc:[2021-06-07 11:55:57.000845069 +0000 UTC] Timezone:[Local] Traceparent:[00-3203ea992fd47f6d16e31f5bcf5e9219-4072f69a23c769f9-01] User-Agent:[fasthttp]]

== APP == 2021/06/07 19:55:59 binding - Data:, Header:map[Content-Length:[0] Content-Type:[application/json] Readtimeutc:[2021-06-07 11:55:59.000358274 +0000 UTC] Timezone:[Local] Traceparent:[00-8a0942fd787dd7f700447be9871e47df-99ceab9ecd29d858-01] User-Agent:[fasthttp]]

== APP == 2021/06/07 19:56:01 binding - Data:, Header:map[Content-Length:[0] Content-Type:[application/json] Readtimeutc:[2021-06-07 11:56:01.000760386 +0000 UTC] Timezone:[Local] Traceparent:[00-09e809c75b7c60db6b2799d43356cf60-f22a61a2778c46b8-01] User-Agent:[fasthttp]]
```
</details>

### With output

We need to prepare an output target first.

```shell
dapr run --app-id output_demo \
    --app-protocol http \
    --app-port 7489 \
    --dapr-http-port 7490 \
    go run ../outputs/main.go
```

This will generate two available targets, one for access through Dapr's proxy address and another for direct access through the app serving address.

```
via Dapr: http://localhost:7490/v1.0/invoke/output_demo/method/echo
via App: http://localhost:7489/echo
```

In this example, the proxy address of Dapr will be used as the target of output.

>Here we have defined only one output, which will be called `item` in the following
>
>`app-id` is "echo" derived from the key of `item`
>
>Dapr component type is "bindings" derived from `item.out_type` while its params are in `item.params`. Refer to [Dapr components reference](https://docs.dapr.io/reference/components-reference/).

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "protocol": "HTTP",
  "port": "8080",
  "input": {
    "name": "cron_input",
    "enabled": true,
    "in_type": "bindings"
  },
  "outputs": {
    "enabled": true,
    "output_objects": {
      "echo": {
        "out_type": "bindings",
        "params": {
          "operation": "create",
          "metadata": "{\"path\": \"/echo\", \"Content-Type\": \"application/json; charset=utf-8\"}"
        }
      }
    }
  },
  "runtime": "Dapr"
}
```

Create an environment variable `FUNC_CONTEXT` and assign the above context to it.

```shell
export FUNC_CONTEXT='{"name":"MyFunc","version":"v1","request_id":"a0f2ad8d-5062-4812-91e9-95416489fb01","protocol":"HTTP","port":"8080","input":{"name":"cron_input","enabled":true,"in_type":"bindings"},"outputs":{"enabled":true,"output_objects":{"echo":{"out_type":"bindings","params":{"operation":"create","metadata":"{\"path\": \"/echo\", \"Content-Type\": \"application/json; charset=utf-8\"}"}}}},"runtime":"Dapr"}'
```

Start the service and watch the logs.

```shell
dapr run --app-id serving_function \
    --app-protocol http \
    --app-port 8080 \
    --components-path ../config \
    go run ../serving/http/bindings_with_output.go
```

The logs of user function is ...

<details>
<summary>View detailed logs.</summary>

```shell
ℹ️  Starting Dapr with id serving_function. HTTP Port: 44075. gRPC Port: 36069

ℹ️  Updating metadata for app command: go run ../serving/http/main.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == 2021/06/07 20:02:27 Function serving http: listening on port 8080

== APP == 2021/06/07 20:02:29 binding - Data:, Header:map[Content-Length:[0] Content-Type:[application/json] Readtimeutc:[2021-06-07 12:02:29.001071416 +0000 UTC] Timezone:[Local] Traceparent:[00-081b5bf5e34f229be6c3da5d95443b36-27bb3f9ef90a2b3c-01] User-Agent:[fasthttp]]

== APP == 2021/06/07 20:02:29 Send hello world to output_demo

== APP == 2021/06/07 20:02:31 binding - Data:, Header:map[Content-Length:[0] Content-Type:[application/json] Readtimeutc:[2021-06-07 12:02:31.000956037 +0000 UTC] Timezone:[Local] Traceparent:[00-68a0976e43a9740ae1e80731303b13f0-6180d624bf2ca303-01] User-Agent:[fasthttp]]

== APP == 2021/06/07 20:02:31 Send hello world to output_demo
```

</details>

And the logs of output target app is ...

<details>
<summary>View detailed logs.</summary>

```shell
ℹ️  Starting Dapr with id output_demo. HTTP Port: 7490. gRPC Port: 38851

ℹ️  Updating metadata for app command: go run ../outputs/main.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == 2021/06/07 20:02:29 Receive a message:

== APP == 2021/06/07 20:02:29 hello world

== APP == 2021/06/07 20:02:31 Receive a message:

== APP == 2021/06/07 20:02:31 hello world
```

</details>

## Pub/Sub via gRPC

### Subscriber

Prepare a context as follows, name it `input.json`. (You can refer to [types.go](../../openfunction-context/types.go) to learn about the OpenFunction Context)

>This indicates that the input of the function is a Dapr  Component with parameters are:
>
>`spec.type` is "pubsub.*" derived from `input.in_type`
>
>`metadata.name` is "msg" derived from `input.name`
>
>pubsub topic is "my_topic" derived from `input.pattern`
>
>`app-protocol` is "gRPC" derived from `protocol`
>
>`app-port` is "60011" derived from `port`

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "protocol": "gRPC",
  "port": "60011",
  "input": {
    "enabled": true,
    "name": "msg",
    "pattern": "my_topic",
    "in_type": "pubsub"
  },
  "outputs": {
    "enabled": false
  },
  "runtime": "Dapr"
}
```

Create an environment variable `FUNC_CONTEXT` and assign the above context to it.

```shell
export FUNC_CONTEXT='{"name":"MyFunc","version":"v1","request_id":"a0f2ad8d-5062-4812-91e9-95416489fb01","protocol":"gRPC","port":"60011","input":{"enabled":true,"name":"msg","pattern":"my_topic","in_type":"pubsub"},"outputs":{"enabled":false},"runtime":"Dapr"}'
```

Start the service and watch the logs.

```shell
dapr run --app-id subscriber \
    --app-protocol grpc \
    --app-port 60011 \
    --components-path ../config \
    go run ../serving/grpc/subscriber.go
```

### Producer

You also need a definition of producer.

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "protocol": "gRPC",
  "port": "60012",
  "input": {
    "enabled": false
  },
  "outputs": {
    "enabled": true,
    "output_objects": {
      "msg": {
        "pattern": "my_topic",
        "out_type": "pubsub"
      }
    }
  },
  "runtime": "Dapr"
}
```

Create an environment variable `FUNC_CONTEXT` and assign the above context to it.

```shell
export FUNC_CONTEXT='{"name":"MyFunc","version":"v1","request_id":"a0f2ad8d-5062-4812-91e9-95416489fb01","protocol":"gRPC","port":"60012","input":{"enabled":false},"outputs":{"enabled":true,"output_objects":{"msg":{"pattern":"my_topic","out_type":"pubsub"}}},"runtime":"Dapr"}'
```

Start the service with another terminal to publish message.

```shell
dapr run --app-id producer \
    --app-protocol grpc \
    --app-port 60012 \
    --components-path ../config \
    go run ../client/grpc/producer.go
```

<details>
<summary>View detailed producer logs.</summary>

```shell
ℹ️  Starting Dapr with id producer. HTTP Port: 38271. gRPC Port: 44777

ℹ️  Updating metadata for app command: go run ../client/grpc/producer.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == subscription name: msg

== APP == number of publishers: 1

== APP == publish frequency: 1s

== APP == log frequency: 3s

== APP == publish delay: 10s

== APP == 2021/06/07 12:11:52 Function serving grpc: listening on port 60012

== APP == dapr client initializing for: 127.0.0.1:44777

== APP ==          1 published,   0/sec,   0 errors

== APP ==          4 published,   0/sec,   0 errors

== APP ==          7 published,   0/sec,   0 errors

== APP ==         10 published,   0/sec,   0 errors

```
</details>

<details>
<summary>View detailed subscriber logs.</summary>

```shell
ℹ️  Starting Dapr with id subscriber. HTTP Port: 43077. gRPC Port: 39685

ℹ️  Updating metadata for app command: go run ../serving/grpc/subscriber.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == dapr client initializing for: 127.0.0.1:39685

== APP == 2021/06/07 11:48:39 Function serving grpc: listening on port 60011

== APP == 2021/06/07 12:08:33 event - PubsubName:msg, Topic:my_topic, ID:83175279-1d04-49cf-8e36-7ce1b52aa42b, Data: {"id":"p1-c8e61eef-ae10-49c7-a505-f0ae108d7049","data":"Snd2TEdkQkNwRzRRUzNXd0R1ME9aaGZuMVNWTTZkeEk0QzN5YTN5ZHBLOEpSWVdFMlFmTkVsTGp2dXhZUUhKeUZrRUtER0hadmpybVVXOWVnZHJCdjFkWDhOM1hhM09GajQwZUVqZzlOYzY5RE44akU0VWpHRGJ4aFdidkFnTzd1aE5VNFVWVlRMMXlYVzMxZ3dXN2hhZGNySW9VaFhET1BaYlZNQkVhWGVTdFBvZVM1UHE4MG9BRUd3R0lLZXhRcDRrWmJ4dVByWHRWMHJMaEhMMkNtQ2dQQk84eThoVVhObXkzU29kdWE5ZGxBVEdnRlN3Q0RRa3VZZVIyMGZwVg==","sha":"\u0017\ufffd`\ufffd\ufffd_e\ufffd\u0010A]\ufffdܧ\ufffd\ufffdc\ufffd\ufffd\u00083\ufffdVܬ(\ufffd\ufffd\u000e\u003e\ufffd","time":1623038913}

== APP == 2021/06/07 12:09:37 event - PubsubName:msg, Topic:my_topic, ID:1826c8df-a87a-49da-b27c-a05c14c64532, Data: {"id":"p1-8760d362-ebd5-42d9-a327-8c44100bdad6","data":"RVJoZkZZZjdlelpEalE3WXdZV3g0VlE4Uklnd0tlcnNCV2NocHFDVXFvb2JLWEh1OVJPYjNCa3BWN3hkTk04RVZ6V2RnUmZFOUpZSDFnMGdrUGV2QkpteFRpdVhtVWpPb1FTanZQTzJsYU01bzFBb1ExUkFZNklBSFNOYmdxalRrRHZ1dFlFRFM0bURCNVNRdmdMTUlHbGNIMVBjRlJPenhjRHBrc2dDSGZPMkc2Qk5LaVB5U1VsR1Q4TFRUS3E3UUd2dGVrRjAzOGlYVFJQWXZERGt3eU5ycVJMdkZyQlpSRUJIR1JIMmFTT21uQnRlbFQ3QXNOQnF2WXRlZEh3MQ==","sha":"\ufffd[\ufffd\ufffd\"#\u001b7\ufffd\ufffd!\ufffd\ufffd-\ufffd\ufffd\ufffdP\ufffd\ufffd;\ufffd^u\ufffd\ufffd\u0002\ufffd\ufffd\ufffd\ufffd","time":1623038977}

== APP == 2021/06/07 12:10:16 event - PubsubName:msg, Topic:my_topic, ID:4deb3b30-31fd-47e2-899a-e5884647e73b, Data: {"id":"p1-9350ee5e-8344-413e-a719-ac2c65f3078e","data":"TVN3Z1RjeHVIVVlPazRsYUhEakl1YzFoT2ppWFZOaHFXSzNiRDdxcEd0enE1OEkxR2lSTGdZaEZYUnhhalRrQUxBbFE2SGNUSVRyenBtOUtFZ1R5VmU4NVZCU0JVanRyaTkyMGtnbmx0eEtKUjJDdzEyWHZxMzZhMGpqWFhRM0JDWm01aHJ3c1hWR0hMbElxdElsc1JxRVdVd0tFdEp4eGJFeW5BYmh2OGNiV0ZCVVI3bkcxeDhpWDgxTTkxYjQ4a3VqdlVUTGFPMlJZSXRyUkdGUE1BS2hQSjlWOW9xUVBkWTc4SjFYV0NJTUNkOHcxNU9CWUlGeTdIQnFSc3VLMg==","sha":"ԓ\u0014\ufffdC\ufffd \ufffd\ufffd\ufffd*w\ufffdE3i\ufffd(ƈ\ufffdPe_\ufffd\ufffd\u0026\u0010\ufffd\ufffd\ufffdb","time":1623039016}

== APP == 2021/06/07 12:10:17 event - PubsubName:msg, Topic:my_topic, ID:f98bde5d-305f-4caa-a6c4-a514652a91f5, Data: {"id":"p1-360fdf2b-9e38-4a2f-978d-0b8dd5e2e6e0","data":"VHhMazF3R0hxODVYZ210aEN0RkJzVjBsbURpZWY4RnBnZ0owZklsajRETmJxbE9mdzV6Uk8wRk5oR2NjcFFjb3NZSXd3cFVhT2xpaGVTRngwYTJlWGp1TzhRd3ZvREhnUEZ2Q2ZCWlVET1FqTTJBaW14TzlXRzVpT2dvYWJteWlEMkkydEZoQ2VpbE11S3NsZmdhRkNCbnBtT2hOYk9VdndwTTlJQjFTNmROdEl5QnJyVzl2UHB0WHczVXVsOHI2RnppWGxGblViTjhXR1dMSUNiTGVIUENoMk85VmtaN1VZZUc5N2NaSXVQamUzbVdnaG5MZG4weks3aG00dHNnVQ==","sha":"\ufffd\ufffdZ\u0006\u001e\u001f\ufffd\ufffd\u000f_\ufffd\ufffd\ufffd\ufffdড়j~\ufffdc?_\ufffd,\ufffd\ufffd\u001f\ufffdk\ufffd\ufffd","time":1623039017}

== APP == 2021/06/07 12:10:18 event - PubsubName:msg, Topic:my_topic, ID:552a1466-983d-420b-a144-efeb78897f78, Data: {"id":"p1-bfdee206-38ae-461f-8e25-68f48d1518b1","data":"THFsQ2xxRkxMakpaZ01oVHk4dmZPUmxDdFRBODVQWk9DMkxhMEVYeWR2dXh0WEltOE9LNXZDZ3lTcHVVUUdCamUwMDNSWXE3V1FINTRKT0ViVUg3NU16ZnVrZFBIR2xjZDdVRnNZbkNBeVZpeEJPVnBFTno4YUJHMjBSZG1TNk1IUGlGcTdBZGpkbVNUd0dPclc1U3NRNjNyRXhqTTJLOFZhU1dGZWhqTXdvUGpzNHFCSmNuMVJBcU9EV0FxR0pXQWVFdHFTaGdHaE5FRXNZdTNPeVlEWTJURGRsSVRQOG1YRTRSOEJKNjJnRXUyTFQ2VUlPbEx0dGFEbXhqVm9DcQ==","sha":"\ufffdk/OPE\u0018+ޒ\ufffd\ufffdq\ufffdO\ufffd\ufffd\t\ufffd\u0019-y`\u0012X\ufffdցфbp","time":1623039018}
```
</details>

## Service invocation via gRPC

### Server

Prepare  a context as follows, name it `input.json`. (You can refer to [types.go](../../openfunction-context/types.go) to learn about the OpenFunction Context)

>This indicates that the input of the function is a Dapr  Component with parameters are:
>
>type is "invoke.*" derived from `input.in_type`
>
>name is "echo" derived from `input.name`
>
>`app-protocol` is "gRPC" derived from `protocol`
>
>`app-port` is "50001" derived from `port`

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "protocol": "gRPC",
  "port": "50001",
  "input": {
    "name": "echo",
    "enabled": true,
    "in_type": "invoke",
    "pattern": "print"
  },
  "outputs": {
    "enabled": false
  },
  "runtime": "Dapr"
}
```

Create an environment variable `FUNC_CONTEXT` and assign the above context to it.

```shell
export FUNC_CONTEXT='{"name":"MyFunc","version":"v1","request_id":"a0f2ad8d-5062-4812-91e9-95416489fb01","protocol":"gRPC","port":"50001","input":{"name":"echo","enabled":true,"in_type":"invoke","pattern":"print"},"outputs":{"enabled":false},"runtime":"Dapr"}'
```

Start the service and watch the logs.
```shell
dapr run --app-id server \
    --app-protocol grpc \
    --app-port 50001 \
    go run ../serving/grpc/server.go
```

### Client

You also need a definition of client.

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "protocol": "gRPC",
  "port": "50002",
  "input": {
    "enabled": false
  },
  "outputs": {
    "enabled": true,
    "output_objects": {
      "server": {
        "out_type": "invoke",
        "pattern": "print",
        "params": {
          "method": "post"
        }
      }
    }
  },
  "runtime": "Dapr"
}
```

Create an environment variable `FUNC_CONTEXT` and assign the above context to it.

```shell
export FUNC_CONTEXT='{"name":"MyFunc","version":"v1","request_id":"a0f2ad8d-5062-4812-91e9-95416489fb01","protocol":"gRPC","port":"50002","input":{"enabled":false},"outputs":{"enabled":true,"output_objects":{"server":{"out_type":"invoke","pattern":"print","params":{"method":"post"}}}},"runtime":"Dapr"}'
```

Start the client to post request.

```shell
dapr run --app-id client \
    --app-protocol grpc \
    go run ../client/grpc/client.go
```

<details>
<summary>View detailed logs.</summary>

```shell
ℹ️  Starting Dapr with id serving_function. HTTP Port: 40441. gRPC Port: 3500

ℹ️  Updating metadata for app command: go run ../serving/grpc/main.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == 2021/06/07 23:29:21 Function serving grpc: listening on port 50001

== APP == 2021/06/07 23:29:30 echo - ContentType:application/json, Verb:POST, QueryString:, hello
```
</details>

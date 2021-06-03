# Dapr examples

In order to follow these examples, you need to [install Dapr](https://docs.dapr.io/getting-started/install-dapr-selfhost/).

## Bindings via gRPC

This input source will be executed every 10s (Refer to [cron.yaml](../config/cron.yaml)).

Prepare a context as follows, name it `input.json`. (You can refer to [types.go](../openfunction-context/types.go) to learn more about the OpenFunction Context)

>This indicates that the input of the function is a Dapr  Component with parameters are:
>
>`spec.type` is "bindings.*" derived from `input.in_type`
>
>`metadata.name` is "cron_input" derived from `input.name`
>
>`app-protocol` is "gRPC" derived from `input.kind`
>
>`app-port` is "50001" derived from `input.port`

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "input": {
    "name": "cron_input",
    "enabled": true,
    "kind": "gRPC",
    "port": "50001",
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
export FUNC_CONTEXT='{"name":"MyFunc","version":"v1","request_id":"a0f2ad8d-5062-4812-91e9-95416489fb01","input":{"enabled":true,"name":"cron_input","kind":"gRPC","port":"50001","in_type":"bindings"},"outputs":{"enabled":false},"runtime":"Dapr"}'
```

Start the service and watch the logs.

```shell
dapr run --app-id serving_function \
    --app-protocol grpc \
    --app-port 50001 \
    --dapr-grpc-port 3500 \
    --components-path ../config \
    go run ../serving/grpc/main.go
```

<details>
<summary>View detailed logs.</summary>


```shell
ℹ️  Starting Dapr with id serving_function. HTTP Port: 33787. gRPC Port: 3500

ℹ️  Updating metadata for app command: go run ../serving/grpc/main.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == 2021/06/01 16:17:46 Function serving grpc: listening on port 50001

== APP == 2021/06/01 16:17:56 binding - Data:, Meta:map[readTimeUTC:2021-06-01 08:17:56.000560838 +0000 UTC timeZone:Local]

== APP == 2021/06/01 16:18:06 binding - Data:, Meta:map[readTimeUTC:2021-06-01 08:18:06.000266153 +0000 UTC timeZone:Local]
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
>`app-id` is "output_demo" derived from the key of `item`
>
>target is "http://localhost:7490/v1.0/invoke/output_demo/method/echo" derived from `item.pattern`
>
>verb is "POST" derived from `item.request_method`

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "input": {
    "name": "cron_input",
    "enabled": true,
    "kind": "gRPC",
    "port": "50001",
    "in_type": "bindings"
  },
  "outputs": {
    "enabled": true,
    "output_objects": {
      "output_demo": {
        "pattern": "http://localhost:7490/v1.0/invoke/output_demo/method/echo",
        "request_method": "POST"
      }
    }
  },
  "runtime": "Dapr"
}
```

Create an environment variable `FUNC_CONTEXT` and assign the above context to it.

```shell
export FUNC_CONTEXT='{"name":"MyFunc","version":"v1","request_id":"a0f2ad8d-5062-4812-91e9-95416489fb01","input":{"name":"cron_input","enabled":true,"kind":"gRPC","port":"50001","in_type":"bindings"},"outputs":{"enabled":true,"output_objects":{"output_demo":{"pattern":"http://localhost:7490/v1.0/invoke/output_demo/method/echo","request_method":"POST"}}},"runtime":"Dapr"}'
```

Start the service and watch the logs.

```shell
dapr run --app-id serving_function \
    --app-protocol grpc \
    --app-port 50001 \
    --dapr-grpc-port 3500 \
    --components-path ../config \
    go run ../serving/grpc/main.go
```

The logs of user function is ...

<details>
<summary>View detailed logs.</summary>

```shell
ℹ️  Starting Dapr with id serving_function. HTTP Port: 45509. gRPC Port: 3500

== APP == 2021/06/02 16:08:45 binding - Data:, Meta:map[readTimeUTC:2021-06-02 08:08:45.000811446 +0000 UTC timeZone:Local]

== APP == 2021/06/02 16:08:55 binding - Data:, Meta:map[readTimeUTC:2021-06-02 08:08:55.000742456 +0000 UTC timeZone:Local]
```

</details>

And the logs of output target app is ...

<details>
<summary>View detailed logs.</summary>

```shell
ℹ️  Starting Dapr with id output_demo. HTTP Port: 7490. gRPC Port: 43973

ℹ️  Updating metadata for app command: go run ../outputs/main.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == 2021/06/02 16:08:45 Receive a message:

== APP == 2021/06/02 16:08:45 {"data":null,"metadata":{"readTimeUTC":"2021-06-02 08:08:45.000811446 +0000 UTC","timeZone":"Local"}}

== APP == 2021/06/02 16:08:55 Receive a message:

== APP == 2021/06/02 16:08:55 {"data":null,"metadata":{"readTimeUTC":"2021-06-02 08:08:55.000742456 +0000 UTC","timeZone":"Local"}}
```

</details>

## Bindings via HTTP

This input source will be executed every 10s (Refer to [cron.yaml](../config/cron.yaml)).

Prepare a context as follows, name it `input.json`. (You can refer to [types.go](../openfunction-context/types.go) to learn more about the OpenFunction Context)

>This indicates that the input of the function is a Dapr  Component with parameters are:
>
>`spec.type` is "bindings.*" derived from `input.in_type`
>
>`metadata.name` is "cron_input" derived from `input.name`
>
>`app-protocol` is "gRPC" derived from `input.kind`
>
>`app-port` is "50001" derived from `input.port`

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "input": {
    "name": "cron_input",
    "enabled": true,
    "kind": "HTTP",
    "port": "8080",
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
export FUNC_CONTEXT='{"name":"MyFunc","version":"v1","request_id":"a0f2ad8d-5062-4812-91e9-95416489fb01","input":{"name":"cron_input","enabled":true,"kind":"HTTP","port":"8080","in_type":"bindings"},"outputs":{"enabled":false},"runtime":"Dapr"}'
```

Start the service and watch the logs.

```shell
dapr run --app-id serving_function \
    --app-protocol http \
    --app-port 8080 \
    --components-path ../config \
    go run ../serving/http/main.go
```

<details>
<summary>View detailed logs.</summary>

```shell
ℹ️  Starting Dapr with id serving_function. HTTP Port: 44947. gRPC Port: 44865

ℹ️  Updating metadata for app command: go run ../serving/http/main.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == 2021/06/02 16:53:06 Function serving http: listening on port 8080

== APP == 2021/06/02 16:53:16 binding - Data:, Header:map[Content-Length:[0] Content-Type:[application/json] Readtimeutc:[2021-06-02 08:53:16.000607515 +0000 UTC] Timezone:[Local] Traceparent:[00-2f615e493cae54a8f8bf35a40411c230-8ffb1208b9c865c9-01] User-Agent:[fasthttp]]

== APP == 2021/06/02 16:53:26 binding - Data:, Header:map[Content-Length:[0] Content-Type:[application/json] Readtimeutc:[2021-06-02 08:53:26.001180281 +0000 UTC] Timezone:[Local] Traceparent:[00-fc13285fa1ae6fb1d03f8c138be006aa-245d99c40c12b3bf-01] User-Agent:[fasthttp]]
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
>`app-id` is "output_demo" derived from the key of `item`
>
>target is "http://localhost:7490/v1.0/invoke/output_demo/method/echo" derived from `item.pattern`
>
>verb is "POST" derived from `item.request_method`

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "input": {
    "name": "cron_input",
    "enabled": true,
    "kind": "HTTP",
    "port": "8080",
    "in_type": "bindings"
  },
  "outputs": {
    "enabled": true,
    "output_objects": {
      "output_demo": {
        "pattern": "http://localhost:7490/v1.0/invoke/output_demo/method/echo",
        "request_method": "POST"
      }
    }
  },
  "runtime": "Dapr"
}
```

Create an environment variable `FUNC_CONTEXT` and assign the above context to it.

```shell
export FUNC_CONTEXT='{"name":"MyFunc","version":"v1","request_id":"a0f2ad8d-5062-4812-91e9-95416489fb01","input":{"name":"cron_input","enabled":true,"kind":"HTTP","port":"8080","in_type":"bindings"},"outputs":{"enabled":true,"output_objects":{"output_demo":{"pattern":"http://localhost:7490/v1.0/invoke/output_demo/method/echo","request_method":"POST"}}},"runtime":"Dapr"}'
```

Start the service and watch the logs.

```shell
dapr run --app-id serving_function \
    --app-protocol http \
    --app-port 8080 \
    --components-path ../config \
    go run ../serving/http/main.go
```

The logs of user function is ...

<details>
<summary>View detailed logs.</summary>

```shell
ℹ️  Starting Dapr with id serving_function. HTTP Port: 44075. gRPC Port: 36069

ℹ️  Updating metadata for app command: go run ../serving/http/main.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == 2021/06/02 17:24:28 Function serving http: listening on port 8080

== APP == 2021/06/02 17:24:28 binding - Data:, Header:map[Content-Length:[0] Content-Type:[application/json] Traceparent:[00-00000000000000000000000000000000-0000000000000000-00] User-Agent:[fasthttp]]

== APP == 2021/06/02 17:24:28 Send hello world to output_demo

== APP == 2021/06/02 17:24:38 binding - Data:, Header:map[Content-Length:[0] Content-Type:[application/json] Readtimeutc:[2021-06-02 09:24:38.000536907 +0000 UTC] Timezone:[Local] Traceparent:[00-3e0bb9b7a98215e5123bacbfe54798f2-2eb96a13a3f0c846-01] User-Agent:[fasthttp]]

== APP == 2021/06/02 17:24:38 Send hello world to output_demo
```

</details>

And the logs of output target app is ...

<details>
<summary>View detailed logs.</summary>

```shell
ℹ️  Starting Dapr with id output_demo. HTTP Port: 7490. gRPC Port: 38851

ℹ️  Updating metadata for app command: go run ../outputs/main.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == 2021/06/02 17:24:28 Receive a message:

== APP == 2021/06/02 17:24:28 {"data":"hello world","operation":"create"}

== APP == 2021/06/02 17:24:38 Receive a message:

== APP == 2021/06/02 17:24:38 {"data":"hello world","operation":"create"}
```

</details>

## Pub/Sub via gRPC

Prepare a context as follows, name it `input.json`. (You can refer to [types.go](../openfunction-context/types.go) to learn about the OpenFunction Context)

>This indicates that the input of the function is a Dapr  Component with parameters are:
>
>`spec.type` is "pubsub.*" derived from `input.in_type`
>
>`metadata.name` is "msg" derived from `input.name`
>
>pubsub topic is "my_topic" derived from `input.pattern`
>
>`app-protocol` is "gRPC" derived from `input.kind`
>
>`app-port` is "50001" derived from `input.port`

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "input": {
    "name": "msg",
    "enabled": true,
    "pattern": "my_topic",
    "kind": "gRPC",
    "port": "50001",
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
export FUNC_CONTEXT='{"name": "MyFunc","version": "v1","request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01","input": {"name": "msg","enabled": true,"pattern": "my_topic","kind": "gRPC","port": "50001","in_type": "pubsub"},"outputs": {"enabled": false},"runtime": "Dapr"}'
```

Start the service and watch the logs.

>You need to modify the `../serving/grpc/main.go`, uncomment the following content, and comment the others
>
>```go
>	//if err := register(user_functions.PubsubGRPCFunction); err != nil {
>	//	log.Fatalf("Failed to register: %v\n", err)
>	//}
>```

```shell
dapr run --app-id serving_function \
    --app-protocol grpc \
    --app-port 50001 \
    --dapr-grpc-port 3500 \
    --components-path ../config \
    go run ../serving/grpc/main.go
```

Start the client to publish message.

```shell
dapr run --app-id caller \
    --components-path ../config \
    go run ../client/grpc/pubsub_client.go
```

<details>
<summary>View detailed logs.</summary>


```shell
ℹ️  Starting Dapr with id serving_function. HTTP Port: 44693. gRPC Port: 3500

ℹ️  Updating metadata for app command: go run ../serving/grpc/main.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == 2021/06/02 17:39:43 Function serving grpc: listening on port 50001

== APP == 2021/06/02 17:40:14 event - PubsubName:msg, Topic:my_topic, ID:22edfcf5-e0a5-40cf-84a8-f33a4eb53eb7, Data: { "message": "hello" }
```
</details>

## Service invocation via gRPC

Prepare  a context as follows, name it `input.json`. (You can refer to [types.go](../openfunction-context/types.go) to learn about the OpenFunction Context)

>This indicates that the input of the function is a Dapr  Component with parameters are:
>
>type is "invoke.*" derived from `input.in_type`
>
>name is "echo" derived from `input.name`
>
>`app-protocol` is "gRPC" derived from `input.kind`
>
>`app-port` is "50001" derived from `input.port`

```json
{
  "name": "MyFunc",
  "version": "v1",
  "request_id": "a0f2ad8d-5062-4812-91e9-95416489fb01",
  "input": {
    "name": "echo",
    "enabled": true,
    "kind": "gRPC",
    "port": "50001",
    "in_type": "invoke"
  },
  "outputs": {
    "enabled": false
  },
  "runtime": "Dapr"
}
```

Create an environment variable `FUNC_CONTEXT` and assign the above context to it.

```shell
export FUNC_CONTEXT='{"name":"MyFunc","version":"v1","request_id":"a0f2ad8d-5062-4812-91e9-95416489fb01","input":{"name":"echo","enabled":true,"kind":"gRPC","port":"50001","in_type":"invoke"},"outputs":{"enabled":false},"runtime":"Dapr"}'
```

Start the service and watch the logs.
>You need to modify the `../serving/grpc/main.go`, uncomment the following content, and comment the others
>
>```go
>	//if err := register(user_functions.ServiceGRPCFunction); err != nil {
>	//	log.Fatalf("Failed to register: %v\n", err)
>	//}
>```
```shell
dapr run --app-id serving_function \
    --app-protocol grpc \
    --app-port 50001 \
    --dapr-grpc-port 3500 \
    --components-path ../config \
    go run ../serving/grpc/main.go
```

Start the client to post request.

```shell
dapr run --app-id caller \
    --components-path ../config \
    go run ../client/grpc/service_client.go
```

<details>
<summary>View detailed logs.</summary>

```shell
ℹ️  Starting Dapr with id serving_function. HTTP Port: 40441. gRPC Port: 3500

ℹ️  Updating metadata for app command: go run ../serving/grpc/main.go
✅  You're up and running! Both Dapr and your app logs will appear here.

== APP == 2021/06/02 17:43:25 Function serving grpc: listening on port 50001

== APP == 2021/06/02 17:43:42 echo - ContentType:text/plain, Verb:POST, QueryString:, hellow
```
</details>

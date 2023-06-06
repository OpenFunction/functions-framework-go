# Sync-HTTP-Variables

To run test cases locally

## Run locally

```sh
$ cd test/declarative/sync-http-variables
$ go run main.go http.go
```

## Send request

### HTTP

```sh
$ curl -X POST "http://localhost:8080/hello/openfunction?key1=value1" -d 'hello'

{"hello":"openfunction"}% 
```

### CloudEvent

```sh
# binary
$ curl "http://localhost:8080/foo/openfunction" \
    -H "Ce-Specversion: 1.0" \
    -H "Ce-Type: io.openfunction.samples.helloworld" \
    -H "Ce-Source: io.openfunction.samples/helloworldsource" \
    -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
    -H "Content-Type: application/json" \
    -d '{"data":"hello"}'

I0605 01:38:57.481196   81001 http.go:87] cloudevent - Data: {"hello":"openfunction"}
I0605 01:38:57.481310   81001 plugin-example.go:88] plugin - Result: {"sum":2}

# structured
$ curl "http://localhost:8080/foo/openfunction" \
    -H "Content-Type: application/cloudevents+json" \
    -d '{"specversion":"1.0","type":"io.openfunction.samples.helloworld","source":"io.openfunction.samples/helloworldsource","id":"536808d3-88be-4077-9d7a-a3f162705f79","data":{"data":"hello"}}'

I0605 01:46:52.336317   81001 http.go:87] cloudevent - Data: {"hello":"openfunction"}
I0605 01:46:52.336342   81001 plugin-example.go:88] plugin - Result: {"sum":2}
```

### OpenFunction

```sh
# HTTP
$ curl -X GET "http://localhost:8080/bar/openfunction?key1=value1" -d '{"data":"hello"}'

{"hello":"openfunction"}%

# CloudEvent
## binary
$ curl "http://localhost:8080/bar/openfunction" \
    -H "Ce-Specversion: 1.0" \
    -H "Ce-Type: io.openfunction.samples.helloworld" \
    -H "Ce-Source: io.openfunction.samples/helloworldsource" \
    -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
    -H "Content-Type: application/json" \
    -d '{"data":"hello"}'

{"hello":"openfunction"}

## structured
$ curl "http://localhost:8080/bar/openfunction" \
    -H "Content-Type: application/cloudevents+json" \
    -d '{"specversion":"1.0","type":"io.openfunction.samples.helloworld","source":"io.openfunction.samples/helloworldsource","id":"536808d3-88be-4077-9d7a-a3f162705f79","data":{"data":"hello"}}'

{"hello":"openfunction"}%
```
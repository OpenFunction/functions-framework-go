# OpenFunction functions-framework for Go

**functions-framework-go** is an implementation of functions-framework in Go. It follows the functions-framework criteria for function-to-application conversions.

You can visit the following to learn more about the functions-framework criteria.

- [functions-framework proposal](https://github.com/OpenFunction/OpenFunction/blob/main/docs/proposals/202105_add_function_framework.md#function-context)
- [functions-framework repo](https://github.com/OpenFunction/functions-framework)

## Usage

**functions-framework-go** requires a Go 1.15+ environment. You can import this pkg by configuring the following in the `go.mod` file:

> You can find the correct *version* in the compatibility section below.

```go
require (
	github.com/OpenFunction/functions-framework-go <version>
)
```

## Samples

You can learn how to use **function-framework-go** and how it works from the [functions-framework samples](https://github.com/OpenFunction/samples#functions-framework-samples) in the OpenFunction Samples repository.

## Compatibility

| Version                            | OpenFunction | Context | Builder (Go)                                 |
| ---------------------------------- | ------------ | ------- | -------------------------------------------- |
| v0.0.0-20210628081257-4137e46a99a6 | v0.3.*       | v0.1.0  | v0.2.2 (openfunction/builder-go:v0.2.2-1.15) |
| v0.0.0-20210922063920-81a7b2951b8a | v0.4.*       | v0.2.0  | v0.3.0 (openfunction/builder-go:v0.3.0-1.15) |
| v0.1.1                             | v0.5.*       | v0.2.0  | v0.4.0 (openfunction/builder-go:v0.4.0-1.15) |

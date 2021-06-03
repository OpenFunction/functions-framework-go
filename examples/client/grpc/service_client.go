package main

import (
	"context"
	"fmt"
	dapr "github.com/dapr/go-sdk/client"
)

func main() {
	// just for this demo
	ctx := context.Background()

	// create the client
	client, err := dapr.NewClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// invoke a method called EchoMethod on another dapr enabled service
	content := &dapr.DataContent{
		ContentType: "text/plain",
		Data:        []byte("hellow"),
	}
	resp, err := client.InvokeMethodWithContent(ctx, "serving_function", "echo", "post", content)
	if err != nil {
		panic(err)
	}
	fmt.Printf("service method invoked, response: %s\n", string(resp))

	fmt.Println("DONE (CTRL+C to Exit)")
}

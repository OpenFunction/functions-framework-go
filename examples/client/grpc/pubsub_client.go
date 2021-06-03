package main

import (
	"context"
	"fmt"
	dapr "github.com/dapr/go-sdk/client"
)

func main() {
	// just for this demo
	ctx := context.Background()
	json := `{ "message": "hello" }`
	data := []byte(json)
	pubsub := "msg"
	topic := "my_topic"

	// create the client
	client, err := dapr.NewClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// publish a message to the topic demo
	if err := client.PublishEvent(ctx, pubsub, topic, data); err != nil {
		panic(err)
	}
	fmt.Println("data published")

	fmt.Println("DONE (CTRL+C to Exit)")
}

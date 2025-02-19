package main

import (
	"context"
	"fmt"
	"log"

	"github.com/m-mizutani/opaq"
)

func main() {
	// Create a new client with policy files from a directory
	client, err := opaq.New(opaq.Files("./policy"))
	if err != nil {
		log.Fatal(err)
	}

	// Define input data
	input := map[string]any{
		"user":     "alice",
		"action":   "read",
		"resource": "document-123",
	}

	// Define output structure
	var output struct {
		Allow bool `json:"allow"`
	}

	// Query the policy
	err = client.Query(
		context.Background(),
		"data.authz",
		input,
		&output,
	)
	if err != nil {
		log.Fatal(err)
	}

	if output.Allow {
		fmt.Println("Access granted")
	} else {
		fmt.Println("Access denied")
	}
	// Output:
	// Access granted
}

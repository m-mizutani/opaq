package main

import (
	"context"
	"fmt"
	"log"

	"github.com/m-mizutani/opaq"
	"github.com/open-policy-agent/opa/v1/topdown/print"
)

func main() {
	// Create a new client with logger
	client, err := opaq.New(
		opaq.Files("./policy"),
	)
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

	hook := func(ctx print.Context, msg string) error {
		fmt.Println("📣", msg) // Show print statements from Rego policy
		return nil
	}
	// Query the policy
	err = client.Query(
		context.Background(),
		"data.authz",
		input,
		&output,
		opaq.WithPrintHook(hook),
	)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 📣 input {"action": "read", "resource": "document-123", "user": "alice"}
}

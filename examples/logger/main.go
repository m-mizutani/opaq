package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/m-mizutani/opaq"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create a new client with logger
	client, err := opaq.New(
		opaq.Files("./policy"),
		opaq.WithLogger(logger),
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
}

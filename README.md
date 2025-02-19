# opaq [![Go Reference](https://pkg.go.dev/badge/github.com/m-mizutani/opaq.svg)](https://pkg.go.dev/github.com/m-mizutani/opaq) [![test](https://github.com/m-mizutani/opaq/actions/workflows/test.yml/badge.svg)](https://github.com/m-mizutani/opaq/actions/workflows/test.yml) [![gosec](https://github.com/m-mizutani/opaq/actions/workflows/gosec.yml/badge.svg)](https://github.com/m-mizutani/opaq/actions/workflows/gosec.yml) [![trivy](https://github.com/m-mizutani/opaq/actions/workflows/trivy.yml/badge.svg)](https://github.com/m-mizutani/opaq/actions/workflows/trivy.yml) [![lint](https://github.com/m-mizutani/opaq/actions/workflows/lint.yml/badge.svg)](https://github.com/m-mizutani/opaq/actions/workflows/lint.yml)

Rego query library with local policy file or data based on OPA (Open Policy Agent). This library is a wrapper of [Open Policy Agent](https://www.openpolicyagent.org/) and [Rego](https://www.openpolicyagent.org/docs/policy-language.html) to evaluate local policy files or data.

## Install
```
go get github.com/m-mizutani/opaq
```

## Basic Usage

Here is a basic example of how to use the library. The example code is in [examples/basic](./examples/basic).

```rego:policy/authz.rego
package authz

allow if {
    input.user == "alice"
    input.action == "read"
}
```

And here is the example code.

```go
package main

import (
    "context"
    "log"
    "fmt"

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
        "user": "alice",
        "action": "read",
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
```

## Advanced Features

### Custom Logger

`opaq` supports logger with `slog` package. This is useful when you want to see the debug logs from Rego policy. The example code is in [examples/logger](./examples/logger).

```go
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
```

It outputs the debug logs like below when the query is evaluated.

```
time=2025-02-20T05:07:11.280+09:00 level=DEBUG msg="Evaluating query" query_id=BDSXIVAWYU7YSZFKR2KYGI54HQ query=data.authz input="map[action:read resource:document-123 user:alice]"
time=2025-02-20T05:07:11.280+09:00 level=DEBUG msg="Query evaluated" query_id=BDSXIVAWYU7YSZFKR2KYGI54HQ result="[{Expressions:[map[allow:true]] Bindings:map[]}]"
time=2025-02-20T05:07:11.280+09:00 level=DEBUG msg="Unmarshaled result" query_id=BDSXIVAWYU7YSZFKR2KYGI54HQ output=&{Allow:true}
```

### Print Hook

`opaq` supports print hook to show the print statements from Rego policy. The example code is in [examples/print-hook](./examples/print-hook).

```go
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
```

And the policy should have `print` statement.

```rego:policy/authz.rego
package authz

allow if {
    print("input", input)
    input.user == "alice"
    input.action == "read"
}
```

Then the output is like below.

```
📣 input {"action": "read", "resource": "document-123", "user": "alice"}
```

### Rego Version Selection

```go
import "github.com/open-policy-agent/opa/ast"

client, err := opaq.New(
    opaq.Files("./policies"),
    opaq.WithRegoVersion(ast.RegoV1),
)
```

### Accessing Policy Metadata

`opaq` supports accessing policy metadata. The metadata is the annotations in the policy file. See [official documentation](https://www.openpolicyagent.org/docs/latest/policy-language/#metadata) for more details about Rego metadata.

```go
// Get policy annotations
metadata := client.Metadata()
```

### Accessing Policy Sources

`opaq` supports accessing policy sources. The sources are the policy files. Please note that the sources can not be changed after the client is created.

```go
// Get policy sources
sources := client.Sources()
```

## License

Apache License, Version 2.0

See [LICENSE](./LICENSE) for more details.
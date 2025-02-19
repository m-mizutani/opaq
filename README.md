# opaq

Rego query library with local policy file or data based on OPA (Open Policy Agent)

## Usage

```
go get github.com/m-mizutani/opaq
```

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/m-mizutani/opaq"
)

func main() {
    // Create a new client with policy files from a directory
    client, err := opaq.New(opaq.Files("./policies"))
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
        "data.authz.allow",
        input,
        &output,
    )
    if err != nil {
        log.Fatal(err)
    }

    if output.Allow {
        log.Println("Access granted")
    } else {
        log.Println("Access denied")
    }
}
```

### Advanced Features

#### Custom Logger

```go
import "log/slog"

logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
client, err := opaq.New(
    opaq.Files("./policies"),
    opaq.WithLogger(logger),
)
```

#### Print Hook

```go
client.Query(
    context.Background(),
    "data.authz.allow",
    input,
    &output,
    opaq.WithPrintHook(func(string) {
        // Handle print statements from Rego policy
    }),
)
```

#### Rego Version Selection

```go
import "github.com/open-policy-agent/opa/ast"

client, err := opaq.New(
    opaq.Files("./policies"),
    opaq.WithRegoVersion(ast.RegoV1),
)
```

#### Accessing Policy Metadata

```go
// Get policy annotations
metadata := client.Metadata()

// Get policy sources
sources := client.Sources()
```

### Example Policy

```rego
package authz

default allow = false

allow {
    input.user == "alice"
    input.action == "read"
}
```

## License

[License information here]









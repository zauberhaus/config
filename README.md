# Config

`config` is a Go package designed for flexible and robust application configuration management. It allows applications to load configuration from various sources, including files, environment variables, and command-line flags, with a clear and predictable precedence order.

## Features

-   **Multiple Configuration Sources**: Load settings from YAML and JSON files, environment variables, and command-line flags (primarily via `pflag`).
-   **Structured Configuration**: Map configuration settings directly into Go structs, supporting default values defined via struct tags.
-   **Configuration Precedence**: A well-defined hierarchy ensures that configuration values are applied consistently.
-   **Easy Integration**: Designed for seamless integration into existing Go applications, with explicit support for [cobra](https://github.com/spf13/cobra) and [pflag](https://github.com/spf13/pflag).

## Installation

To install the `config` package, use `go get`:

```sh
go get github.com/zauberhaus/config
```

## Usage

The `config` package provides a simple API to load configurations. Define your configuration structure with `default` tags for initial values:

```go
package main

import (
	"fmt"
	"log"

	"github.com/zauberhaus/config"
)

type MyConfig struct {
	Host string `default:"localhost"`
	Port int    `default:"3000"`
}

func main() {
	cfg, _, err := config.Load[*MyConfig]()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	fmt.Printf("Host: %s\n", cfg.Host)
	fmt.Printf("Port: %d\n", cfg.Port)
}
```

For more advanced usage, including integration with `pflag` and `cobra`, refer to the `examples/` directory:
-   [`examples/basic`](./examples/basic/README.md)
-   [`examples/cobra`](./examples/cobra/README.md)
-   [`examples/pflags`](./examples/pflags/README.md)

## Configuration Precedence

When multiple configuration sources are defined, `config` resolves values based on a strict order of precedence, from lowest to highest:

1.  **Default values in the struct**: Values specified using the `default:"value"` struct tag.
2.  **Environment variables**: Values provided via environment variables (e.g., `APP_HOST`, `APP_PORT`).
3.  **Configuration files**: Settings loaded from YAML files (e.g., `config.yaml`, `app.yaml`).
4.  **Command-line flags**: Values passed as command-line arguments (e.g., `--host`, `-p`).

This order ensures that command-line flags always override environment variables, which in turn override configuration file settings, and finally, struct defaults provide a baseline.

## License

Copyright 2026 Zauberhaus

Licensed to Zauberhaus under one or more agreements.
Zauberhaus licenses this file to you under the Apache 2.0 License.
See the [LICENSE](LICENSE) file in the project root for more information.
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

The `config` package provides a simple API to load configurations. Here's an example using `cobra` to define command-line flags and load configuration:

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/zauberhaus/config"
	"github.com/zauberhaus/config/pkg/flags"
)

type MyConfig struct {
	Host string `default:"localhost"`
	Port int    `default:"3000"`
}

func main() {
	if err := RootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "my-app",
		Short: "My application with config",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bind Cobra flags to the config struct
			flagList := flags.NewFlagList(nil)
			_ = flagList.BindCmdFlag(cmd, "Host", "host")
			_ = flagList.BindCmdFlag(cmd, "Port", "port")

			// Define config options
			o := []config.Option{
				config.WithName("my-app"), // Prefix for environment variables (e.g., MY_APP_HOST)
				config.WithFlags(flagList),
			}

			// Optionally, specify a config file via flag or environment variable
			configFile := os.Getenv("MY_APP_CONFIG_FILE") // Custom env var for config file
			if cmd.Flags().Changed("config") {
				if cfgFile, err := cmd.Flags().GetString("config"); err == nil && cfgFile != "" {
					configFile = cfgFile
				}
			}

			if configFile != "" {
				o = append(o, config.WithFile(configFile))
			}

			// Load the configuration
			cfg, loadedFile, err := config.Load[*MyConfig](o...)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			if loadedFile != "" {
				fmt.Printf("Configuration loaded from: %s\n", loadedFile)
			} else {
				fmt.Println("No configuration file loaded, using defaults and environment variables.")
			}

			fmt.Printf("Host: %s\n", cfg.Host)
			fmt.Printf("Port: %d\n", cfg.Port)

			return nil
		},
	}

	// Define Cobra flags
	cmd.Flags().StringP("host", "H", "", "Specify the host")
	cmd.Flags().IntP("port", "P", 0, "Specify the port")
	cmd.Flags().StringP("config", "c", "", "Path to the configuration file (e.g., config.yaml)")

	return cmd
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
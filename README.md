# Config

`config` is a Go package designed for flexible and robust application configuration management. It allows applications to load configuration from various sources, including files, environment variables, and command-line flags, with a clear and predictable precedence order.

## Features

-   **Multiple Configuration Sources**: Load settings from YAML and JSON files, environment variables, and command-line flags (primarily via `pflag`).
-   **Structured Configuration**: Map configuration settings directly into Go structs, supporting default values defined via struct tags.
-   **Pluggable Storage**: Merge values from any backend (database, key-value store, remote service) by implementing the small `Storage` interface.
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

## Config File Lookup

`Load` reads at most one config file:

1.  The file given with `config.WithFile`, or else
2.  the file named by the `CONFIG` environment variable, or else
3.  the first file named `<name>.json`, `<name>.yaml` or `<name>.yml` found in these directories, in this order:
    1.  the directories given with `config.WithPaths`,
    2.  the current working directory, **only** with `config.WithWorkingDir(true)`,
    3.  the home directory.

`<name>` is the name given with `config.WithName`, or `config` without one. `config.WithExtension` and `config.WithExtensions` change the accepted file extensions.

The current working directory isn't searched by default. Otherwise a config file in any directory a tool is run from, such as a freshly cloned repository, would silently change its settings.

### File checks

-   Paths with a `..` element (e.g. `../app.yaml`) are rejected. Dots inside a name (`app..v2.yaml`) are fine.
-   The config file must be a regular file.
-   A file that is writable by others (e.g. mode `0666`) is rejected, unless `config.WithWorldWritable(true)` is set. This check is skipped on Windows.
-   A file found in a search directory may be a symlink, but only to a file inside that directory. A file given with `WithFile` or `CONFIG` may point anywhere.

### Values in error messages

Values from environment variables, flags and storage can be secrets, so they are replaced with `***` in error messages, e.g. `env MY_APP_PORT: strconv.ParseInt: parsing "***": invalid syntax`. The original error is still available with `errors.As` as `*errors.Error` from `github.com/zauberhaus/config/pkg/errors`.

## Configuration Precedence

When multiple configuration sources are defined, `config` resolves values based on a strict order of precedence, from lowest to highest:

1.  **Default values in the struct**: Values specified using the `default:"value"` struct tag. An invalid default (e.g. `default:"abc"` on an `int`) makes `Load` return an error.
2.  **Configuration files**: Settings loaded from YAML or JSON files (e.g., `config.yaml`, `app.json`).
3.  **Storage**: Values returned by a custom [`Storage`](#custom-storage) passed with `config.WithStorage`.
4.  **Environment variables**: Values provided via environment variables (e.g., `MY_APP_HOST`, `MY_APP_PORT`).
5.  **Command-line flags**: Values passed as command-line arguments (e.g., `--host`, `-P`) and bound one by one with `flags.NewFlagList` and `BindCmdFlag`, or all at once by name with `flags.FromCommand(cmd, idx)`: every flag whose name matches a field (`--host` → `Host`, `--sub-name` → `Sub.Name`) is bound, other flags are skipped. A `flag` tag overrides the name: with `flag:"addr"` only `--addr` sets the field, with `flag:"-"` no flag is bound to it by name.

This order ensures that command-line flags always override environment variables, which in turn override storage and configuration file settings, and finally, struct defaults provide a baseline.

## Custom Storage

Use `config.WithStorage` to load values from a source other than a file, such as a database or a key-value store. A storage implements the `config.Storage` interface:

```go
type Storage interface {
	All() (map[string]any, error)
	Get(key string) (any, error)
	Set(key string, val any) error
}
```

`Load` calls `All()` once and applies every entry to the configuration struct. Keys are lower-case, dot-separated field paths (e.g. `host`, `sub.name`, `sub2.name`) and values must be assignable to the target field. `Load` returns an error if `All()` fails, if a key doesn't match a field, or if a value has the wrong type. `Get` and `Set` are not used by `Load`; they are there so the same storage can be used to read and persist individual settings.

```go
type MyConfig struct {
	Host string `default:"localhost"`
	Port int    `default:"3000"`
	Sub  struct {
		Name string
	}
}

// memStorage is a minimal in-memory Storage.
type memStorage map[string]any

func (m memStorage) All() (map[string]any, error) { return m, nil }

func (m memStorage) Get(key string) (any, error) {
	v, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return v, nil
}

func (m memStorage) Set(key string, val any) error {
	m[key] = val
	return nil
}

func main() {
	storage := memStorage{"host": "db.example.com", "sub.name": "from-storage"}

	cfg, _, err := config.Load[*MyConfig](
		config.WithName("my-app"),
		config.WithStorage(storage),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(cfg.Host) // db.example.com (environment variables and flags still take precedence)
}
```

## License

Copyright 2026 Zauberhaus

Licensed to Zauberhaus under one or more agreements.
Zauberhaus licenses this file to you under the Apache 2.0 License.
See the [LICENSE](LICENSE) file in the project root for more information.
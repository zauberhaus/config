# Cobra Binding Example

This example demonstrates how to use the `github.com/zauberhaus/config` package with a `cobra` command
whose flags are bound to the configuration automatically with `flags.FromCommand`.
Every flag whose name matches a field (`--host` → `Host`, `--sub-name` → `Sub.Name`) is bound;
other flags such as `--config` are skipped. A `flag` tag overrides the name, e.g. ``Host string `flag:"addr"` `` binds `--addr`,
and `flag:"-"` keeps a field from being bound by name.
It shows how to load configuration from files, environment variables, and flags, and how the precedence works.

## Configuration

The application can be configured through a configuration file, environment variables, and flags.

### Parameters

| Name | Type   | Default     | Description             | Flag                | Environment Variable |
|------|--------|-------------|-------------------------|---------------------|----------------------|
| Host | string | "localhost" | The host to connect to. | `--server`, `-s`    | `APP_HOST`           |
| Port | int    | 3000        | The port to connect to. | `--port`, `-p`      | `APP_PORT`           |
| Level | Level | "info"     | The log level (`debug`, `info`, `warn`, `error`). | `--level`, `-l` | `APP_LEVEL` |

### Configuration Files

The application looks for a `config.yaml` file in the current directory by default.
You can specify a different configuration file using the `--config` or `-c` flag, or by setting the `CONFIG_FILE` environment variable.

#### `config.yaml` (default)
```yaml
host: "localhost"
port: 8080
```

#### `alternate_config.yaml`
```yaml
host: "alternate.cobra.host"
port: 5678
level: "warn"
```

## Usage Examples

### Default Configuration
Running the application without any flags or environment variables will use the default `config.yaml`.

```sh
$ go run main.go
Configuration loaded from: config.yaml
Host: localhost
Port: 8080
Level: info
```

### Using Flags
You can override configuration values with flags. Flags have the highest precedence.

```sh
$ go run main.go --server "flag.host" --port 1234
Configuration loaded from: config.yaml
Host: flag.host
Port: 1234
Level: info
```

### Using a Custom Flag Type
`--level` is a custom flag: `Level` implements `pflag.Value` (`String`, `Set`, `Type`) and is registered with `cmd.Flags().VarP`.
It is bound by name like any other flag; because the flag variable has the type of the field, `flags.FromCommand` sets the field with the value directly instead of parsing its text.
Because `Level` also implements `encoding.TextUnmarshaler`, the same names work in the config file, in `APP_LEVEL` and in the `default` tag.
Invalid values are rejected by the flag parser.

```sh
$ go run main.go --level warn
Configuration loaded from: config.yaml
Host: localhost
Port: 8080
Level: warn

$ go run main.go --level verbose
Error: invalid argument "verbose" for "-l, --level" flag: unknown level "verbose" (one of debug, info, warn, error)
```

### Using Environment Variables
You can also use environment variables. They have a lower precedence than flags.

```sh
$ APP_PORT=9999 go run main.go
Configuration loaded from: config.yaml
Host: localhost
Port: 9999
Level: info
```

### Using an Alternate Configuration File via Flag
You can specify a different configuration file with the `--config` or `-c` flag.

```sh
$ go run main.go -c alternate_config.yaml
Configuration loaded from: alternate_config.yaml
Host: alternate.cobra.host
Port: 5678
Level: warn
```

### Using an Alternate Configuration File via Environment Variable
You can also specify the configuration file using the `CONFIG_FILE` environment variable.

```sh
$ CONFIG_FILE=alternate_config.yaml go run main.go
Configuration loaded from: alternate_config.yaml
Host: alternate.cobra.host
Port: 5678
Level: warn
```

### Precedence
The configuration is loaded in the following order of precedence (from lowest to highest):

1.  **Default values in the struct:** `Host: "localhost"`, `Port: 3000`, `Level: info`
2.  **Configuration file:** (e.g., `config.yaml`)
3.  **Environment variables:** (e.g., `APP_PORT`)
4.  **Flags:** (e.g., `--server`)

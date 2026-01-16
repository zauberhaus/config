# pflag Example

This example demonstrates how to use the `github.com/zauberhaus/config` package with `pflag` for advanced configuration management in Go applications. It showcases loading configurations from files, environment variables, and command-line flags, illustrating the precedence rules.

## Configuration

The application's configuration can be provided through a YAML file, environment variables, or `pflag` command-line flags.

### Parameters

| Name | Type   | Default     | Description             | Flag          | Environment Variable |
|------|--------|-------------|-------------------------|---------------|----------------------|
| Host | string | "localhost" | The host to connect to. | `--host`      | `APP_HOST`           |
| Port | int    | 3000        | The port to connect to. | `--port`      | `APP_PORT`           |

### Configuration Files

By default, the application searches for a `config.yaml` file in the current directory. You can specify an alternative configuration file using the `--config` (or `-c`) flag, or by setting the `ADD_CONFIG` environment variable.

#### `config.yaml` (default)
```yaml
host: "localhost"
port: 8080
```

#### `alternate_config.yaml`
```yaml
host: "alternate.host.com"
port: 1234
```

## Usage Examples

### Default Configuration
Running the application without any flags or environment variables will use the values from the default `config.yaml` file.

```sh
$ go run main.go
Configuration loaded from: config.yaml
Host: localhost
Port: 8080
```

### Using Flags
You can override configuration values with flags. Flags have the highest precedence.

```sh
$ go run main.go --host "flag.host" --port 1234
Configuration loaded from: config.yaml
Host: flag.host
Port: 1234
```

### Using Environment Variables
You can also use environment variables. They have a lower precedence than flags but higher than the configuration file.

```sh
$ APP_PORT=9999 go run main.go
Configuration loaded from: config.yaml
Host: localhost
Port: 9999
```

### Using an Alternate Configuration File via Flag
Specify a different configuration file with the `--config` or `-c` flag.

```sh
$ go run main.go -c alternate_config.yaml
Configuration loaded from: alternate_config.yaml
Host: alternate.host.com
Port: 1234
```

### Using an Alternate Configuration File via Environment Variable
You can also specify the configuration file using the `ADD_CONFIG` environment variable.

```sh
$ ADD_CONFIG=alternate_config.yaml go run main.go
Configuration loaded from: alternate_config.yaml
Host: alternate.host.com
Port: 1234
```

## Precedence Order

The configuration values are resolved in the following order of precedence (from lowest to highest):

1.  **Default values in the struct:** `Host: "localhost"`, `Port: 3000`
2.  **Configuration file:** (e.g., `config.yaml`)
3.  **Environment variables:** (e.g., `APP_PORT`)
4.  **Flags:** (e.g., `--host`)

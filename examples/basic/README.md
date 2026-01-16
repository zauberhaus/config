# Basic Example

This example demonstrates how to use the `github.com/zauberhaus/config` package.
It shows how to load configuration from files and environment variables and how the precedence works.

## Configuration

The application can be configured through a configuration file and environment variables.

### Parameters

| Name | Type   | Default     | Description             | Environment Variable |
|------|--------|-------------|-------------------------|----------------------|
| Host | string | "localhost" | The host to connect to. | `APP_HOST`           |
| Port | int    | 3000        | The port to connect to. | `APP_PORT`           |

### Configuration Files

The application does not load any configuration file by default.
You can specify a configuration file using the `-c` flag or the `ADD_CONFIG` environment variable.

#### `app.yaml`
```yaml
host: "default config file"
port: 1234
```

#### `config.yaml`
```yaml
host: "extra config file"
port: 8080
```

## Usage Examples

### Default Configuration
Running the application without any flags or environment variables will use the default values defined in the `MyConfig` struct.

```sh
$ go run main.go
No configuration file loaded, using defaults and environment variables.
Host: localhost
Port: 3000
```

### Using a Configuration File
You can specify a configuration file with the `-c` flag.

```sh
$ go run main.go -c examples/basic/app.yaml
Configuration loaded from: examples/basic/app.yaml
Host: default config file
Port: 1234
```

### Using Environment Variables
You can also use environment variables. They have a higher precedence than the configuration file.

```sh
$ APP_PORT=9999 go run main.go -c examples/basic/app.yaml
Configuration loaded from: examples/basic/app.yaml
Host: default config file
Port: 9999
```

### Using an Alternate Configuration File
You can specify a different configuration file with the `-c` flag.

```sh
$ go run main.go -c examples/basic/config.yaml
Configuration loaded from: examples/basic/config.yaml
Host: extra config file
Port: 8080
```

### Using `ADD_CONFIG` environment variable
You can specify a configuration file with the `ADD_CONFIG` environment variable.

```sh
$ ADD_CONFIG=examples/basic/config.yaml go run main.go
Configuration loaded from: examples/basic/config.yaml
Host: extra config file
Port: 8080
```

### Precedence
The configuration is loaded in the following order of precedence (from lowest to highest):

1.  **Default values in the struct:** `Host: "localhost"`, `Port: 3000`
2.  **Configuration file:** (e.g., `app.yaml`)
3.  **Environment variables:** (e.g., `APP_HOST`, `APP_PORT`)
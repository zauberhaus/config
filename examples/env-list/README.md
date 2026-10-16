# Env List Example

This example demonstrates how to list all environment variables a configuration struct accepts,
using `env.List` and `index.New` from the `github.com/zauberhaus/config` package.

## Configuration

```go
type MyConfig struct {
	LogLevel  string `default:"info"`
	Server    Server
	Database  Database `env:"DB"` // custom name
	Common    Database `env:"-"`  // no prefix, fields are added to the parent
	Endpoints []Endpoint
	Labels    map[string]string
	Internal  string `env:"--"`  // ignored
}
```

The prefix is derived from the name passed to `env.WithName` (`app` → `APP_`).
Field names are converted to SNAKE_CASE unless an `env` tag is set.
Slices and maps contain a `[]` placeholder, which is replaced by an index or key (e.g. `APP_ENDPOINTS[0]_NAME`).

## Usage Examples

### List all environment variables

```sh
$ go run main.go
APP_DB
APP_DB_HOST
APP_DB_PORT
APP_DB_USER_NAME
APP_ENDPOINTS
APP_ENDPOINTS[]
APP_ENDPOINTS[]_NAME
APP_ENDPOINTS[]_URL
APP_HOST
APP_LABELS
APP_LABELS[]
APP_LOG_LEVEL
APP_PORT
APP_SERVER
APP_SERVER_ADDR
APP_SERVER_PORT
APP_USER_NAME
```

### Show the current values

`env.List` also returns the values of the environment variables that are set.

```sh
$ env APP_DB_HOST=db.local 'APP_ENDPOINTS[0]_NAME=x' go run main.go -v
APP_DB=
APP_DB_HOST=db.local
APP_DB_PORT=
APP_DB_USER_NAME=
APP_ENDPOINTS=
APP_ENDPOINTS[0]_NAME=x
APP_ENDPOINTS[]=
...
```

### Show the config path and type

`index.New` builds the index without the prefix and maps every name to its config path and type.

```sh
$ go run main.go -t
- DB:
    database: main.Database
- DB_HOST:
    database.host: string
...
- HOST:
    common.host: string
...
```

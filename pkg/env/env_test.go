// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package env_test

import (
	"net"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zauberhaus/config/pkg/env"
	"github.com/zauberhaus/config/pkg/index"
	"github.com/zauberhaus/lookup"
)

type TestConfig struct {
	Server struct {
		Host      string
		Port      int
		WhiteList []net.IP `env:"WL"`
		Route     map[int]net.IP
		Flags     [2]bool
		Settings  []struct {
			Name  string
			Value int
			Tags  map[string]string
		}

		Settings2 struct {
			Name string
		} `env:"-"`

		Settings3 struct {
			Name string
		} `env:"--"`

		hidden string
		Hidden string `env:"-"`
	}
	Db struct {
		User string
		Tags map[string]string
	}
}

func TestSetEnv_Map(t *testing.T) {
	t.Setenv("APP_SERVER_ROUTE[99]", "192.168.1.99")
	t.Setenv("APP_DB_TAGS[99]", "test")

	var cfg TestConfig
	updated, err := env.Set(&cfg, env.WithName("APP"), env.Strict)
	if assert.NoError(t, err) {

		assert.Equal(t, map[int]net.IP{
			99: {0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff, 0xc0, 0xa8, 0x1, 0x63},
		}, updated.Server.Route)

		assert.Equal(t, map[string]string{
			"99": "test",
		}, updated.Db.Tags)

	}
}

func TestSetEnv_WithPrefix(t *testing.T) {
	t.Setenv("APP_SERVER_HOST", "localhost")
	t.Setenv("APP_SERVER_PORT", "8080")
	t.Setenv("APP_DB_USER", "admin")
	t.Setenv("APP_SERVER_WL", "192.168.1.1,192.168.1.2")
	t.Setenv("OTHER_VAR", "should be ignored")
	t.Setenv("APP_SERVER_WL[]", "192.168.1.3")
	t.Setenv("APP_SERVER_WL[0]", "192.168.1.0")
	t.Setenv("APP_SERVER_ROUTE[99]", "192.168.1.99")
	t.Setenv("APP_SERVER_HIDDEN", "123456")
	t.Setenv("APP_SERVER_NAME", "my server")

	var cfg TestConfig
	updated, err := env.Set(&cfg, env.WithName("APP"))
	if assert.NoError(t, err) {
		assert.Equal(t, "localhost", updated.Server.Host)
		assert.Equal(t, 8080, updated.Server.Port)
		assert.Equal(t, "admin", updated.Db.User)
		assert.Equal(t, "", updated.Server.Hidden)
		assert.Equal(t, []net.IP{
			{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff, 0xc0, 0xa8, 0x1, 0x0},
			{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff, 0xc0, 0xa8, 0x1, 0x2},
			{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff, 0xc0, 0xa8, 0x1, 0x3},
		}, updated.Server.WhiteList)

		assert.Equal(t, &cfg, updated)
		assert.Equal(t, "my server", updated.Server.Settings2.Name)
	}
}

func TestSetEnv_SliceOfStructs(t *testing.T) {
	t.Setenv("APP_SERVER_SETTINGS[0]_NAME", "s0")
	t.Setenv("APP_SERVER_SETTINGS[0]_VALUE", "10")
	t.Setenv("APP_SERVER_SETTINGS[0]_TAGS[A]", "valA")
	t.Setenv("APP_SERVER_SETTINGS[1]_NAME", "s1")
	t.Setenv("APP_SERVER_SETTINGS[1]_VALUE", "20")

	var cfg TestConfig
	updated, err := env.Set(&cfg, env.WithName("APP"))
	require.NoError(t, err)

	// Using require.Len to handle out-of-order slice initialization
	require.Len(t, updated.Server.Settings, 2)

	// Sort to have a predictable order for assertion
	slices.SortFunc(updated.Server.Settings, func(a, b struct {
		Name  string
		Value int
		Tags  map[string]string
	}) int {
		return strings.Compare(a.Name, b.Name)
	})

	assert.Equal(t, "s0", updated.Server.Settings[0].Name)
	assert.Equal(t, 10, updated.Server.Settings[0].Value)
	assert.Equal(t, map[string]string{"a": "valA"}, updated.Server.Settings[0].Tags)

	assert.Equal(t, "s1", updated.Server.Settings[1].Name)
	assert.Equal(t, 20, updated.Server.Settings[1].Value)
	assert.Nil(t, updated.Server.Settings[1].Tags)
}

func TestSetEnv_WithoutPrefix(t *testing.T) {
	t.Setenv("SERVER_HOST", "remote host")
	t.Setenv("SERVER_PORT", "9090")

	var cfg TestConfig
	updated, err := env.Set(&cfg)
	if assert.NoError(t, err) {
		assert.Equal(t, "remote host", updated.Server.Host)
		assert.Equal(t, 9090, updated.Server.Port)
	}
}

func TestSetEnv_NoEnvVarsSet(t *testing.T) {
	var cfg TestConfig
	initialConfig := cfg
	updated, err := env.Set(
		&cfg,
		env.WithName("APP"),
		env.Strict,
	)
	if assert.NoError(t, err) {
		assert.Equal(t, &initialConfig, updated)
	}
}

func TestSetEnv_NoMatchingEnvVars(t *testing.T) {
	t.Setenv("OTHER_VAR", "some_value")
	var cfg TestConfig
	initialConfig := cfg

	updated, err := env.Set(
		&cfg,
		env.WithName("APP"),
		env.Strict,
	)
	if assert.NoError(t, err) {
		assert.Equal(t, &initialConfig, updated)
	}
}

func TestSetEnv_MalformedKey(t *testing.T) {
	t.Setenv("APP_SERVER_INVALID_FIELD", "some_value")
	var cfg TestConfig

	_, err := env.Set(&cfg, env.WithName("APP"), env.Strict)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "field not found: SERVER_INVALID_FIELD")

	got, err := env.Set(&cfg, env.WithName("APP"))
	assert.NoError(t, err)
	assert.Equal(t, &cfg, got)

}

func TestSetEnv_MalformedValueForInt(t *testing.T) {
	t.Setenv("APP_SERVER_PORT", "not-an-int")
	var cfg TestConfig
	_, err := env.Set(&cfg, env.WithName("APP"))
	assert.Error(t, err)
	assert.EqualError(t, err, "strconv.ParseInt: parsing \"not-an-int\": invalid syntax")
}

func TestSetEnv_KeyAndValueWithSpacesAndMixedCase(t *testing.T) {
	t.Setenv("APP_  server_HOST  ", "  spaced-host  ")
	var cfg TestConfig
	updated, err := env.Set(&cfg, env.WithName("APP"))
	if assert.NoError(t, err) {
		assert.Equal(t, "spaced-host", updated.Server.Host)
	}
}

func TestPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "APP", "APP_"},
		{"with dots", "my.app", "MY_APP_"},
		{"with dashes", "my-app", "MY_APP_"},
		{"mixed", "My-App.v1", "MY_APP_V1_"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix := env.Prefix(tt.input)
			assert.Equal(t, tt.expected, prefix)
		})
	}
}

func TestSetEnv_WithIndex(t *testing.T) {
	t.Setenv("MY_HOST", "custom-host")

	idx := index.Index{
		"MY_HOST": {Path: "server.host", Type: reflect.TypeOf("")},
	}

	var cfg TestConfig
	updated, err := env.Set(&cfg, env.WithIndex(idx))
	require.NoError(t, err)
	assert.Equal(t, "custom-host", updated.Server.Host)
}

func TestSetEnv_WithReplacer(t *testing.T) {
	t.Setenv("SRV_HOST", "replaced-host")

	var cfg TestConfig
	updated, err := env.Set(&cfg, env.WithReplacer(map[string]string{"Server": "Srv"}))
	require.NoError(t, err)
	assert.Equal(t, "replaced-host", updated.Server.Host)
}

func TestSetEnv_WithStrict(t *testing.T) {
	t.Setenv("UNKNOWN_VAR", "val")

	var cfg TestConfig
	// Strict = true
	_, err := env.Set(&cfg, env.WithStrict(true))
	assert.Error(t, err)

	// Strict = false
	_, err = env.Set(&cfg, env.WithStrict(false))
	assert.NoError(t, err)
}

func TestSetEnv_LookupNotFoundError(t *testing.T) {
	idx := index.Index{
		"MY_VAR": {Path: "non.existent.field", Type: reflect.TypeOf("")},
	}
	t.Setenv("MY_VAR", "val")

	var cfg TestConfig

	// Strict = false (default)
	_, err := env.Set(&cfg, env.WithIndex(idx))
	assert.NoError(t, err)

	// Strict = true
	_, err = env.Set(&cfg, env.WithIndex(idx), env.Strict)
	assert.Error(t, err)
	assert.IsType(t, &lookup.NotFoundError{}, err)
}

func TestSetEnv_NonStruct(t *testing.T) {
	var i int
	_, err := env.Set(&i)
	assert.NoError(t, err)
}

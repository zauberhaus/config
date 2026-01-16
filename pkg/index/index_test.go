// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package index_test

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zauberhaus/config/pkg/index"
)

type IndexTestConfig struct {
	Server struct {
		Host         string
		Port         int
		WhiteList    []net.IP `env:"WL"`
		Route        map[int]net.IP
		Flags        [2]bool
		VeryLongName bool
		APIUrl       string
		Settings     []struct {
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

func Test_Index(t *testing.T) {
	var cfg IndexTestConfig
	expected := index.Index{
		"DB":                       {Path: "db", Type: reflect.TypeOf(cfg.Db)},
		"DB_TAGS":                  {Path: "db.tags", Type: reflect.TypeOf(cfg.Db.Tags)},
		"DB_TAGS[]":                {Path: "db.tags[]", Type: reflect.TypeOf(cfg.Db.Tags).Elem()},
		"DB_USER":                  {Path: "db.user", Type: reflect.TypeOf(cfg.Db.User)},
		"SERVER":                   {Path: "server", Type: reflect.TypeOf(cfg.Server)},
		"SERVER_API_URL":           {Path: "server.apiurl", Type: reflect.TypeOf(cfg.Server.APIUrl)},
		"SERVER_FLAGS":             {Path: "server.flags", Type: reflect.TypeOf(cfg.Server.Flags)},
		"SERVER_FLAGS[]":           {Path: "server.flags[]", Type: reflect.TypeOf(cfg.Server.Flags).Elem()},
		"SERVER_HOST":              {Path: "server.host", Type: reflect.TypeOf(cfg.Server.Host)},
		"SERVER_NAME":              {Path: "server.settings2.name", Type: reflect.TypeOf(cfg.Server.Settings2.Name)},
		"SERVER_PORT":              {Path: "server.port", Type: reflect.TypeOf(cfg.Server.Port)},
		"SERVER_ROUTE":             {Path: "server.route", Type: reflect.TypeOf(cfg.Server.Route)},
		"SERVER_ROUTE[]":           {Path: "server.route[]", Type: reflect.TypeOf(cfg.Server.Route).Elem()},
		"SERVER_SETTINGS":          {Path: "server.settings", Type: reflect.TypeOf(cfg.Server.Settings)},
		"SERVER_SETTINGS[]":        {Path: "server.settings[]", Type: reflect.TypeOf(cfg.Server.Settings).Elem()},
		"SERVER_SETTINGS[]_NAME":   {Path: "server.settings[].name", Type: reflect.TypeOf("")},
		"SERVER_SETTINGS[]_TAGS":   {Path: "server.settings[].tags", Type: reflect.TypeOf(map[string]string{})},
		"SERVER_SETTINGS[]_TAGS[]": {Path: "server.settings[].tags[]", Type: reflect.TypeOf("")},
		"SERVER_SETTINGS[]_VALUE":  {Path: "server.settings[].value", Type: reflect.TypeOf(0)},
		"SERVER_VERY_LONG_NAME":    {Path: "server.verylongname", Type: reflect.TypeOf(true)},
		"SERVER_WL":                {Path: "server.whitelist", Type: reflect.TypeOf(cfg.Server.WhiteList)},
		"SERVER_WL[]":              {Path: "server.whitelist[]", Type: reflect.TypeOf(cfg.Server.WhiteList).Elem()},
	}

	expected2 := `- DB:
    db: struct { User string; Tags map[string]string }
- DB_TAGS:
    db.tags: map[string]string
- DB_TAGS[]:
    db.tags[]: string
- DB_USER:
    db.user: string
- SERVER:
    server: struct { Host string; Port int; WhiteList []net.IP "env:\"WL\""; Route map[int]net.IP; Flags [2]bool; VeryLongName bool; APIUrl string; Settings []struct { Name string; Value int; Tags map[string]string }; Settings2 struct { Name string } "env:\"-\""; Settings3 struct { Name string } "env:\"--\""; hidden string; Hidden string "env:\"-\"" }
- SERVER_API_URL:
    server.apiurl: string
- SERVER_FLAGS:
    server.flags: '[2]bool'
- SERVER_FLAGS[]:
    server.flags[]: bool
- SERVER_HOST:
    server.host: string
- SERVER_NAME:
    server.settings2.name: string
- SERVER_PORT:
    server.port: int
- SERVER_ROUTE:
    server.route: map[int]net.IP
- SERVER_ROUTE[]:
    server.route[]: net.IP
- SERVER_SETTINGS:
    server.settings: '[]struct { Name string; Value int; Tags map[string]string }'
- SERVER_SETTINGS[]:
    server.settings[]: struct { Name string; Value int; Tags map[string]string }
- SERVER_SETTINGS[]_NAME:
    server.settings[].name: string
- SERVER_SETTINGS[]_TAGS:
    server.settings[].tags: map[string]string
- SERVER_SETTINGS[]_TAGS[]:
    server.settings[].tags[]: string
- SERVER_SETTINGS[]_VALUE:
    server.settings[].value: int
- SERVER_VERY_LONG_NAME:
    server.verylongname: bool
- SERVER_WL:
    server.whitelist: '[]net.IP'
- SERVER_WL[]:
    server.whitelist[]: net.IP
`

	tests := []struct {
		key   string
		value string
	}{
		{"SERVER", "server"},
		{"SERVER_WL", "server.whitelist"},
		{"SERVER_FLAGS", "server.flags"},
		{"SERVER_FLAGS[xyz]", "server.flags[xyz]"},
		{"SERVER_SETTINGS[1]_TAGS[abc]", "server.settings[1].tags[abc]"},
		{"SERVER_API_URL", "server.apiurl"},
		{"SERVER_VERY_LONG_NAME", "server.verylongname"},
	}

	dict, err := index.New[IndexTestConfig](map[string]string{"API": "Api"})
	assert.NoError(t, err)
	assert.Equal(t, expected, dict)

	txt := dict.String()
	if !assert.Equal(t, expected2, txt) {
		fmt.Println(txt)
		fmt.Println(expected2)
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			path, ok := dict.Find(tt.key)
			if assert.True(t, ok) {
				assert.Equal(t, tt.value, path)
			}
		})
	}
}

func TestIndex_Helpers(t *testing.T) {
	dict, err := index.New[IndexTestConfig](map[string]string{"API": "Api"})
	require.NoError(t, err)

	t.Run("Exists", func(t *testing.T) {
		assert.True(t, dict.Exists("SERVER_HOST"))
		assert.True(t, dict.Exists("SERVER_SETTINGS[]_NAME"))
		assert.True(t, dict.Exists("SERVER_SETTINGS[123]_NAME")) // with index
		assert.False(t, dict.Exists("NON_EXISTENT"))
	})

	t.Run("PathExists", func(t *testing.T) {
		assert.True(t, dict.PathExists("server.host"))
		assert.True(t, dict.PathExists("server.settings[].name"))
		assert.True(t, dict.PathExists("server.settings[123].name")) // with index
		assert.False(t, dict.PathExists("non.existent"))
	})

	t.Run("Keys", func(t *testing.T) {
		keys := dict.Keys()
		expectedKeys := []string{
			"DB", "DB_TAGS", "DB_TAGS[]", "DB_USER", "SERVER", "SERVER_API_URL", "SERVER_FLAGS", "SERVER_FLAGS[]",
			"SERVER_HOST", "SERVER_NAME", "SERVER_PORT", "SERVER_ROUTE", "SERVER_ROUTE[]",
			"SERVER_SETTINGS", "SERVER_SETTINGS[]", "SERVER_SETTINGS[]_NAME", "SERVER_SETTINGS[]_TAGS",
			"SERVER_SETTINGS[]_TAGS[]", "SERVER_SETTINGS[]_VALUE", "SERVER_VERY_LONG_NAME",
			"SERVER_WL", "SERVER_WL[]",
		}
		assert.Equal(t, expectedKeys, keys)
	})

	t.Run("Items", func(t *testing.T) {
		items := dict.Items()
		assert.Len(t, items, 22)
		// Check if it's sorted by path
		assert.Equal(t, "db", items[0].Path)
		assert.Equal(t, "server.whitelist[]", items[len(items)-1].Path)
	})
}

func TestNewIndex_Error(t *testing.T) {
	t.Run("error on skipping nested struct in slice", func(t *testing.T) {
		type Top struct {
			Nested []struct {
				Name string
			} `env:"-"`
		}
		type InvalidConfig struct {
			Top
		}

		_, err := index.New[InvalidConfig](map[string]string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't skip slice, array or map: top.nested")
	})
}

func TestSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple", "SIMPLE"},
		{"camelCase", "CAMEL_CASE"},
		{"APIUrl", "APIURL"},
		{"ALLCAPS", "ALLCAPS"},
		{"alllower", "alllower"},
		{"ServerHost", "SERVER_HOST"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, index.SnakeCase(tt.input))
		})
	}
}

func TestNew_NonStruct(t *testing.T) {
	idx, err := index.New[int](nil)
	assert.NoError(t, err)
	assert.Nil(t, idx)

	idx, err = index.New[*int](nil)
	assert.NoError(t, err)
	assert.Nil(t, idx)
}

type TUStruct struct {
	Inner string
}

func (t *TUStruct) UnmarshalText(text []byte) error {
	return nil
}

func TestIndex_TextUnmarshaler_Recursion(t *testing.T) {
	type Config struct {
		List []TUStruct
	}

	idx, err := index.New[Config](nil)
	require.NoError(t, err)

	assert.True(t, idx.Exists("LIST"))
	assert.True(t, idx.Exists("LIST[]"))
	assert.False(t, idx.Exists("LIST[]_INNER"), "Should not recurse into TextUnmarshaler elements in slice")
}

func TestIndex_Pointers(t *testing.T) {
	type Config struct {
		PtrField  *int
		StructPtr *struct {
			Nested int
		}
	}

	idx, err := index.New[Config](nil)
	require.NoError(t, err)

	item, ok := idx["PTR_FIELD"]
	require.True(t, ok)
	assert.True(t, item.Optional)
	assert.Equal(t, "ptrfield", item.Path)

	item, ok = idx["STRUCT_PTR"]
	require.True(t, ok)
	assert.True(t, item.Optional)

	item, ok = idx["STRUCT_PTR_NESTED"]
	require.True(t, ok)
	assert.False(t, item.Optional) // The int inside is not a pointer
}

func TestIndex_Replacer(t *testing.T) {
	type Config struct {
		FooBar string
	}

	// Without replacer
	idx, err := index.New[Config](nil)
	require.NoError(t, err)
	assert.True(t, idx.Exists("FOO_BAR"))

	// With replacer
	idx, err = index.New[Config](map[string]string{"Foo": "Baz"})
	require.NoError(t, err)
	assert.True(t, idx.Exists("BAZ_BAR"))
	assert.False(t, idx.Exists("FOO_BAR"))
}

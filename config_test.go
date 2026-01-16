// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zauberhaus/config"
	"github.com/zauberhaus/config/pkg/flags"
	"github.com/zauberhaus/config/pkg/index"
)

type TestLoadConfig struct {
	Host    string `default:"localhost"`
	Port    int    `default:"8080"`
	Enabled bool   `default:"true"`
	Sub     struct {
		Name string `default:"sub-default"`
	}
	Sub2 *struct {
		Name  string
		Other string `default:"sub2-default"`
	}
	Slice []string
}

func TestLoad(t *testing.T) {
	// --- Setup: Create temporary config files and directories ---
	tempDir := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir) // Mock home index

	// JSON config file
	jsonContent := `{"host": "json.host.com", "port": 9090, "sub": {"name": "json-sub"}}`
	jsonFile := filepath.Join(tempDir, "my-app.json")
	err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	require.NoError(t, err)

	// YAML config file in "home"
	yamlContent := `
host: yaml.host.com
port: 9091
enabled: false
sub:
  name: yaml-sub
`
	yamlFile := filepath.Join(homeDir, "my-app.yaml")
	err = os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Another YAML file for direct path testing
	directYamlContent := `{"host": "direct.yaml.com"}`
	directYamlFile := filepath.Join(tempDir, "direct.yaml")
	err = os.WriteFile(directYamlFile, []byte(directYamlContent), 0644)
	require.NoError(t, err)

	// Another YAML file for direct path testing
	homeYamlConfigFile := filepath.Join(homeDir, "config.yaml")
	err = os.WriteFile(homeYamlConfigFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// --- Test Cases ---

	t.Run("load with defaults only", func(t *testing.T) {
		// Change to a index with no config file
		require.NoError(t, os.Chdir(t.TempDir()))

		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("default-app"))
		require.NoError(t, err)
		assert.Empty(t, f)

		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 8080, cfg.Port)
		assert.True(t, cfg.Enabled)
		assert.Equal(t, "sub-default", cfg.Sub.Name)
	})

	t.Run("load from json file in search path", func(t *testing.T) {
		require.NoError(t, os.Chdir(tempDir))

		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("my-app"), config.WithExtension(".json", config.JSON))
		require.NoError(t, err)
		assert.NotEmpty(t, f)

		assert.Equal(t, "json.host.com", cfg.Host)
		assert.Equal(t, 9090, cfg.Port)
		assert.True(t, cfg.Enabled) // From default
		assert.Equal(t, "json-sub", cfg.Sub.Name)
	})

	t.Run("load from yaml file in home index", func(t *testing.T) {
		// Change to a index with no config file to ensure home dir is used
		require.NoError(t, os.Chdir(t.TempDir()))

		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("my-app"), config.WithExtension(".yaml", config.YAML))
		require.NoError(t, err)
		assert.NotEmpty(t, f)

		assert.Equal(t, "yaml.host.com", cfg.Host)
		assert.Equal(t, 9091, cfg.Port)
		assert.False(t, cfg.Enabled)
		assert.Equal(t, "yaml-sub", cfg.Sub.Name) // From default
	})

	t.Run("load from yaml file in custom search path", func(t *testing.T) {
		customPathDir := t.TempDir()
		yamlContent := `host: custom.path.com`
		yamlFile := filepath.Join(customPathDir, "config.yaml")
		err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
		require.NoError(t, err)

		// Change to a index with no config file to ensure custom path is used
		require.NoError(t, os.Chdir(t.TempDir()))

		cfg, f, err := config.Load[*TestLoadConfig](
			config.WithPaths(customPathDir),
			config.WithExtension(".yaml", config.YAML),
		)
		require.NoError(t, err)
		assert.Equal(t, yamlFile, f)
		assert.Equal(t, "custom.path.com", cfg.Host)
	})

	t.Run("load with CONFIG env var", func(t *testing.T) {
		t.Setenv("CONFIG", directYamlFile)
		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("my-app"))
		require.NoError(t, err)
		assert.NotEmpty(t, f)

		assert.Equal(t, "direct.yaml.com", cfg.Host)
	})

	t.Run("load with environment variables", func(t *testing.T) {
		require.NoError(t, os.Chdir(tempDir))

		t.Setenv("ENV_APP_HOST", "env.host.com")
		t.Setenv("ENV_APP_PORT", "9999")
		t.Setenv("ENV_APP_SUB_NAME", "test name")
		t.Setenv("ENV_APP_SLICE", "a,b,c,d,e,f")

		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("env-app"))
		require.NoError(t, err)
		assert.Empty(t, f)

		// Env vars should override JSON file
		assert.Equal(t, "env.host.com", cfg.Host)
		assert.Equal(t, 9999, cfg.Port)
		assert.Equal(t, "test name", cfg.Sub.Name)
		assert.Equal(t, []string{"a", "b", "c", "d", "e", "f"}, cfg.Slice)
	})

	t.Run("default value for struct pointer", func(t *testing.T) {
		require.NoError(t, os.Chdir(tempDir))

		t.Run("env", func(t *testing.T) {
			t.Setenv("TEST_APP_SUB2_NAME", "test name")

			cfg, f, err := config.Load[*TestLoadConfig](config.WithName("test-app"))
			require.NoError(t, err)
			assert.Empty(t, f)

			if assert.NotNil(t, cfg.Sub2) {
				assert.Equal(t, "test name", cfg.Sub2.Name)
				assert.Equal(t, "sub2-default", cfg.Sub2.Other)
			}
		})

		t.Run("flags", func(t *testing.T) {
			flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flagSet.String("name", "test-name", "sub2 name")

			require.NoError(t, flagSet.Set("name", "new name"))

			flags := flags.NewFlagList(nil)

			require.NoError(t, flags.BindFlag(flagSet, "Sub2.Name", flagSet.Lookup("name")))

			cfg, f, err := config.Load[*TestLoadConfig](config.WithName("not-found"), config.WithFlags(flags))
			require.NoError(t, err)
			assert.Empty(t, f)

			if assert.NotNil(t, cfg.Sub2) {
				assert.Equal(t, "new name", cfg.Sub2.Name)
				assert.Equal(t, "sub2-default", cfg.Sub2.Other)
			}
		})

		t.Run("config file", func(t *testing.T) {
			file := filepath.Join(tempDir, "sub2.yaml")
			content := `{"sub2": {"name": "new name"}}`
			err = os.WriteFile(file, []byte(content), 0644)
			require.NoError(t, err)

			cfg, f, err := config.Load[*TestLoadConfig](config.WithName("sub2"))
			require.NoError(t, err)
			assert.NotEmpty(t, f)

			if assert.NotNil(t, cfg.Sub2) {
				assert.Equal(t, "new name", cfg.Sub2.Name)
				assert.Equal(t, "sub2-default", cfg.Sub2.Other)
			}
		})

		t.Run("not set", func(t *testing.T) {
			cfg, f, err := config.Load[*TestLoadConfig]()
			require.NoError(t, err)
			assert.NotEmpty(t, f)
			assert.Nil(t, cfg.Sub2)
		})

	})

	t.Run("load with flags", func(t *testing.T) {
		require.NoError(t, os.Chdir(tempDir))

		// Setup flags
		flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flagSet.String("host", "default-flag-host", "host name")
		flagSet.Int("port", 1111, "port number")
		flagSet.StringSlice("slice", []string{"a", "b"}, "a slice")
		flagSet.String("name", "test-name", "sub name")

		// Simulate setting flags from command line
		require.NoError(t, flagSet.Set("host", "flag.host.com"))
		require.NoError(t, flagSet.Set("slice", "c,d"))
		require.NoError(t, flagSet.Set("name", "new name"))

		// Setup environment variable to check precedence
		t.Setenv("MY_APP_PORT", "9999")
		t.Setenv("MY_APP_SUB_NAME", "invalid name")

		// Bind flags
		flags := flags.NewFlagList(nil)

		require.NoError(t, flags.BindFlag(flagSet, "Host", flagSet.Lookup("host")))
		require.NoError(t, flags.BindFlag(flagSet, "Port", flagSet.Lookup("port"))) // Port not set via flag, should come from env
		require.NoError(t, flags.BindFlag(flagSet, "Slice", flagSet.Lookup("slice")))
		require.NoError(t, flags.BindFlag(flagSet, "Sub.Name", flagSet.Lookup("name")))

		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("my-app"), config.WithFlags(flags))
		require.NoError(t, err)
		assert.NotEmpty(t, f)

		// Flag value should have highest precedence
		assert.Equal(t, "flag.host.com", cfg.Host)

		// Port flag was not changed, so env var should be used
		assert.Equal(t, 9999, cfg.Port)

		// Slice should be from flag
		assert.Equal(t, []string{"c", "d"}, cfg.Slice)

		// Sub.Name should be from flag
		assert.Equal(t, "new name", cfg.Sub.Name)
	})

	t.Run("precedence: default < file < env < flag", func(t *testing.T) {
		require.NoError(t, os.Chdir(tempDir))

		// 1. Default: Port=8080

		// 2. File overrides default: Port=9090
		// jsonContent := `{"port": 9090}`

		// 3. Env overrides file: Port=9999
		t.Setenv("MY_APP_PORT", "9999")

		// 4. Flag overrides env
		flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flagSet.Int("port", 0, "port number")
		require.NoError(t, flagSet.Set("port", "12345"))

		flags := flags.NewFlagList(nil)
		require.NoError(t, flags.BindFlag(flagSet, "Port", flagSet.Lookup("port")))

		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("my-app"), config.WithFlags(flags))
		require.NoError(t, err)
		assert.NotEmpty(t, f)

		assert.Equal(t, 12345, cfg.Port)
	})

	t.Run("load from file with flags and without prefix", func(t *testing.T) {
		// Ensure we are in the index with the json file
		require.NoError(t, os.Chdir(tempDir))

		// Setup flags
		flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flagSet.String("host", "default-flag-host", "host name")
		flagSet.Int("port", 1111, "port number")
		flagSet.String("name", "test-name", "sub name")

		// Simulate setting flags from command line
		require.NoError(t, flagSet.Set("host", "flag.host.com"))
		// Port is not set by flag, it should come from the file.
		require.NoError(t, flagSet.Set("name", "flag-sub-name"))

		// Bind flags
		flags := flags.NewFlagList(nil)
		require.NoError(t, flags.BindFlag(flagSet, "Host", flagSet.Lookup("host")))
		require.NoError(t, flags.BindFlag(flagSet, "Port", flagSet.Lookup("port")))
		require.NoError(t, flags.BindFlag(flagSet, "Sub.Name", flagSet.Lookup("name")))

		cfg, f, err := config.Load[*TestLoadConfig](config.WithExtension(".yaml", config.YAML), config.WithFlags(flags))
		require.NoError(t, err)
		assert.NotEmpty(t, f)

		assert.Equal(t, "flag.host.com", cfg.Host)     // from flag
		assert.Equal(t, 9091, cfg.Port)                // from my-app.json
		assert.Equal(t, "flag-sub-name", cfg.Sub.Name) // from flag
	})

	t.Run("error on invalid file content", func(t *testing.T) {
		invalidFile := filepath.Join(tempDir, "invalid.json")
		err := os.WriteFile(invalidFile, []byte(`{"host": "bad",`), 0644)
		require.NoError(t, err)

		t.Setenv("CONFIG", invalidFile)
		_, f, err := config.Load[*TestLoadConfig](config.WithName("invalid"))
		assert.Error(t, err)
		assert.NotEmpty(t, f)

		assert.Contains(t, err.Error(), "unexpected end of JSON input")
	})

	t.Run("error on unknown file type from CONFIG", func(t *testing.T) {
		unknownFile := filepath.Join(tempDir, "config.txt")
		err := os.WriteFile(unknownFile, []byte(`host=somehost`), 0644)
		require.NoError(t, err)

		t.Setenv("CONFIG", unknownFile)
		_, f, err := config.Load[*TestLoadConfig](config.WithName("my-app"))
		assert.Error(t, err)
		assert.Empty(t, f)
		assert.Contains(t, err.Error(), "unknown file type")
	})

	t.Run("error on unknown file type from WithFile", func(t *testing.T) {
		unknownFile := filepath.Join(tempDir, "config.txt")
		err := os.WriteFile(unknownFile, []byte(`host=somehost`), 0644)
		require.NoError(t, err)

		_, f, err := config.Load[*TestLoadConfig](config.WithFile(unknownFile))
		assert.Error(t, err)
		assert.Equal(t, unknownFile, f)
		assert.Contains(t, err.Error(), "unknown file type")
	})

	t.Run("error with CONFIG env var as index", func(t *testing.T) {
		t.Setenv("CONFIG", tempDir)
		_, f, err := config.Load[*TestLoadConfig]()
		assert.Error(t, err)
		assert.Empty(t, f)
		assert.Contains(t, err.Error(), "unknown file type")
	})

	t.Run("load with custom prefix", func(t *testing.T) {
		t.Setenv("CUSTOM_PREFIX_HOST", "custom.prefix.com")

		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("CUSTOM_PREFIX"))
		require.NoError(t, err)
		assert.Empty(t, f)

		assert.Equal(t, "custom.prefix.com", cfg.Host)
	})

	t.Run("load with WithFile option", func(t *testing.T) {
		jsonContent := `{"host": "explicit.file.host"}`
		jsonFile := filepath.Join(tempDir, "explicit.json")
		err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
		require.NoError(t, err)

		cfg, f, err := config.Load[*TestLoadConfig](
			config.WithName("my-app2"),
			config.WithFile(jsonFile),
		)

		require.NoError(t, err)
		assert.Equal(t, jsonFile, f)
		assert.Equal(t, "explicit.file.host", cfg.Host)
	})

	t.Run("load with replacer", func(t *testing.T) {
		t.Setenv("MY_APP3_SECTION_NAME", "section-name")

		cfg, f, err := config.Load[*TestLoadConfig](
			config.WithName("my-app3"),
			config.WithReplacer(map[string]string{"Sub": "Section"}),
		)
		require.NoError(t, err)
		assert.Empty(t, f)
		assert.Equal(t, "section-name", cfg.Sub.Name)
	})

	t.Run("load with custom index", func(t *testing.T) {
		t.Setenv("MY_APP2_CUSTOM_HOST", "custom-val")

		idx := index.Index{
			"CUSTOM_HOST": {Path: "host", Type: reflect.TypeOf("")},
		}

		cfg, f, err := config.Load[*TestLoadConfig](
			config.WithName("my-app2"),
			config.WithIndex(idx),
		)
		require.NoError(t, err)
		assert.Empty(t, f)
		assert.Equal(t, "custom-val", cfg.Host)
	})

	t.Run("load with other extensions", func(t *testing.T) {
		confContent := `host: conf.host.com`
		confFile := filepath.Join(tempDir, "my-app99.conf")
		err := os.WriteFile(confFile, []byte(confContent), 0644)
		require.NoError(t, err)

		require.NoError(t, os.Chdir(tempDir))

		cfg, f, err := config.Load[*TestLoadConfig](
			config.WithName("my-app99"),
			config.WithPaths(tempDir),
			config.WithExtensions([]config.Extension{
				{Name: ".conf", FileType: config.YAML},
			}),
		)
		require.NoError(t, err)
		assert.Equal(t, confFile, f)
		assert.Equal(t, "conf.host.com", cfg.Host)
	})

	t.Run("error when CONFIG file does not exist", func(t *testing.T) {
		t.Setenv("CONFIG", filepath.Join(tempDir, "non-existent.json"))
		_, _, err := config.Load[*TestLoadConfig]()
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("strict mode with unknown env var", func(t *testing.T) {
		t.Setenv("MY_APP_UNKNOWN", "val")
		_, _, err := config.Load[*TestLoadConfig](
			config.WithName("MY_APP"),
			config.Strict,
		)
		assert.Error(t, err)
	})

	t.Run("no error on invalid default tag", func(t *testing.T) {
		type InvalidDefaultConfig struct {
			Port int `default:"not-an-int"`
		}
		_, _, err := config.Load[*InvalidDefaultConfig]()
		assert.NoError(t, err)
	})

	t.Run("error on flag type mismatch", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flagSet.String("port", "not-an-int", "port number")
		require.NoError(t, flagSet.Set("port", "not-an-int"))

		flags := flags.NewFlagList(nil)
		require.NoError(t, flags.BindFlag(flagSet, "Port", flagSet.Lookup("port")))

		_, _, err := config.Load[*TestLoadConfig](config.WithFlags(flags))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid syntax")
	})

	t.Run("error on index creation failure", func(t *testing.T) {
		type InvalidConfig struct {
			Nested []struct{ Name string } `env:"-"`
		}
		_, _, err := config.Load[*InvalidConfig]()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "can't skip slice")
	})
}

func TestGetFileType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		exts     []config.Extension
		expected config.FileType
	}{
		{"json", "config.json", nil, config.JSON},
		{"yaml", "config.yaml", nil, config.YAML},
		{"yml", "config.yml", nil, config.YAML},
		{"unknown", "config.txt", nil, config.UnknownFileType},
		{"no extension", "config", nil, config.UnknownFileType},
		{"custom json", "config.jso", []config.Extension{{Name: ".jso", FileType: config.JSON}}, config.JSON},
		{"custom but not matching", "config.json", []config.Extension{{Name: ".jso", FileType: config.JSON}}, config.UnknownFileType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ft := config.GetFileType(tt.filename, tt.exts...)
			assert.Equal(t, tt.expected, ft)
		})
	}
}
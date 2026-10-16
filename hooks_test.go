// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package config_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zauberhaus/config"
	"github.com/zauberhaus/config/pkg/flags"
	"github.com/zauberhaus/config/pkg/index"
	"github.com/zauberhaus/lookup"
)

type hookSecret interface {
	Reveal() string
}

type textSecret struct {
	text string
}

func (s *textSecret) Reveal() string {
	return s.text
}

type hookConfig struct {
	Host   string
	Pin    hookSecret
	Nested hookNested
}

type hookNested struct {
	Token hookSecret
}

var secretHook = lookup.NewParserHookFor[hookSecret](func(txt string) (any, error) {
	return &textSecret{text: txt}, nil
})

func TestLoad_HooksSetInterfaceFieldsFromEnv(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("HOOK_APP_PIN", "1234")
	t.Setenv("HOOK_APP_NESTED_TOKEN", "abc")

	cfg, _, err := config.Load[*hookConfig](config.WithName("hook-app"), config.WithParserHooks(secretHook))
	require.NoError(t, err)

	require.NotNil(t, cfg.Pin)
	assert.Equal(t, "1234", cfg.Pin.Reveal())
	require.NotNil(t, cfg.Nested.Token)
	assert.Equal(t, "abc", cfg.Nested.Token.Reveal())
}

func TestLoad_WithoutHooksAnInterfaceFieldFailsWithoutItsValue(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("HOOK_APP_PIN", "s3cr3t")

	_, _, err := config.Load[*hookConfig](config.WithName("hook-app"))
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "s3cr3t")
}

func TestLoad_HooksSetInterfaceFieldsFromStorage(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage := newMemStorage(map[string]any{"pin": "from-storage"})

	cfg, _, err := config.Load[*hookConfig](config.WithStorage(storage), config.WithParserHooks(secretHook))
	require.NoError(t, err)

	require.NotNil(t, cfg.Pin)
	assert.Equal(t, "from-storage", cfg.Pin.Reveal())
}

func TestLoad_HooksSetInterfaceFieldsFromFlags(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cmd := &cobra.Command{Use: "test", Run: func(cmd *cobra.Command, args []string) {}}
	cmd.Flags().String("pin", "", "")
	require.NoError(t, cmd.Flags().Parse([]string{"--pin", "from-flag"}))

	idx, err := index.New[hookConfig](nil)
	require.NoError(t, err)

	flagList, err := flags.FromCommand(cmd, idx)
	require.NoError(t, err)

	cfg, _, err := config.Load[*hookConfig](config.WithFlags(flagList), config.WithParserHooks(secretHook))
	require.NoError(t, err)

	require.NotNil(t, cfg.Pin)
	assert.Equal(t, "from-flag", cfg.Pin.Reveal())
}

func TestWithParserHooks_Adds(t *testing.T) {
	o := &config.ConfigOptions{}

	config.WithParserHooks(secretHook).Set(o)
	config.WithParserHooks(secretHook).Set(o)

	assert.Len(t, o.Hooks, 2)
}

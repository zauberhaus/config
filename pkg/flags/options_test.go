// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package flags

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zauberhaus/lookup"
)

func TestOptionFunc_Set(t *testing.T) {
	t.Run("implements Option", func(t *testing.T) {
		var _ Option = optionFunc(func(*FlagOptions) {})
	})

	t.Run("calls the function with the given options", func(t *testing.T) {
		var got *FlagOptions

		o := &FlagOptions{}
		optionFunc(func(fo *FlagOptions) { got = fo }).Set(o)

		assert.Same(t, o, got)
	})

	t.Run("calls the function once per Set", func(t *testing.T) {
		calls := 0
		opt := optionFunc(func(*FlagOptions) { calls++ })

		opt.Set(&FlagOptions{})
		opt.Set(&FlagOptions{})

		assert.Equal(t, 2, calls)
	})
}

func TestSetFlags_Options(t *testing.T) {
	type cfg struct {
		Host string
	}

	newFlags := func(t *testing.T) *Flags {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		fs.String("host", "", "host")
		require.NoError(t, fs.Set("host", "flag.host"))

		fl := NewFlagList(nil)
		require.NoError(t, fl.BindFlag(fs, "Host", fs.Lookup("host")))

		return fl
	}

	t.Run("without options", func(t *testing.T) {
		c := &cfg{}
		require.NoError(t, SetFlags(c, newFlags(t)))
		assert.Equal(t, "flag.host", c.Host)
	})

	t.Run("with options", func(t *testing.T) {
		opt := optionFunc(func(*FlagOptions) {})

		c := &cfg{}
		require.NoError(t, SetFlags(c, newFlags(t), opt, opt))
		assert.Equal(t, "flag.host", c.Host)
	})

	t.Run("with nil option list", func(t *testing.T) {
		c := &cfg{}
		require.NoError(t, SetFlags(c, newFlags(t), []Option(nil)...))
		assert.Equal(t, "flag.host", c.Host)
	})
}

type secret interface {
	Reveal() string
}

type textSecret string

func (s textSecret) Reveal() string {
	return string(s)
}

func TestWithParserHooks(t *testing.T) {
	type cfg struct {
		Token secret
	}

	hook := lookup.NewParserHookFor[secret](func(txt string) (any, error) {
		return textSecret(txt), nil
	})

	newFlags := func(t *testing.T) *Flags {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		fs.String("token", "", "token")
		require.NoError(t, fs.Set("token", "abc"))

		fl := NewFlagList(nil)
		require.NoError(t, fl.BindFlag(fs, "Token", fs.Lookup("token")))

		return fl
	}

	t.Run("adds hooks", func(t *testing.T) {
		o := &FlagOptions{}
		WithParserHooks(hook).Set(o)
		WithParserHooks(hook, hook).Set(o)
		assert.Len(t, o.Hooks, 3)
	})

	t.Run("parses into an interface type", func(t *testing.T) {
		c := &cfg{}
		require.NoError(t, SetFlags(c, newFlags(t), WithParserHooks(hook)))
		require.NotNil(t, c.Token)
		assert.Equal(t, "abc", c.Token.Reveal())
	})

	t.Run("fails without hook", func(t *testing.T) {
		err := SetFlags(&cfg{}, newFlags(t))
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "abc")
	})
}

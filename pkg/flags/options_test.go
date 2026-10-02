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

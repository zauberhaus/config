// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package errors_test

import (
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config_errors "github.com/zauberhaus/config/pkg/errors"
)

func TestWrap(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.NoError(t, config_errors.Wrap("env", "KEY", "s3cr3t", nil))
	})

	t.Run("redacts a quoted value", func(t *testing.T) {
		_, inner := strconv.Atoi("s3cr3t")

		err := config_errors.Wrap("env", "APP_PORT", "s3cr3t", inner)
		assert.EqualError(t, err, `env APP_PORT: strconv.Atoi: parsing "***": invalid syntax`)
		assert.ErrorIs(t, err, strconv.ErrSyntax)

		var ve *config_errors.Error
		require.ErrorAs(t, err, &ve)
		assert.Equal(t, "env", ve.Source)
		assert.Equal(t, "APP_PORT", ve.Key)
		assert.Same(t, inner, ve.Err)
	})

	t.Run("redacts a quoted value with escapes", func(t *testing.T) {
		_, inner := strconv.Atoi(`a"b`)

		err := config_errors.Wrap("flag", "port", `a"b`, inner)
		assert.EqualError(t, err, `flag port: strconv.Atoi: parsing "***": invalid syntax`)
	})

	t.Run("redacts an unquoted value", func(t *testing.T) {
		inner := errors.New("p4ssw0rd (string) doesn't implement net.IP")

		err := config_errors.Wrap("storage", "ip", "p4ssw0rd", inner)
		assert.EqualError(t, err, "storage ip: *** (string) doesn't implement net.IP")
	})

	t.Run("short values are only redacted when quoted", func(t *testing.T) {
		inner := errors.New(`parsing "ab": 1 of 2 ab`)

		err := config_errors.Wrap("env", "KEY", "ab", inner)
		assert.EqualError(t, err, `env KEY: parsing "***": 1 of 2 ab`)
	})

	t.Run("non-string values", func(t *testing.T) {
		inner := errors.New("value 123456 out of range")

		err := config_errors.Wrap("storage", "port", 123456, inner)
		assert.EqualError(t, err, "storage port: value *** out of range")
	})

	t.Run("errors without the value are unchanged", func(t *testing.T) {
		inner := errors.New("set supports only struct pointers")

		assert.Same(t, inner, config_errors.Wrap("flag", "host", "s3cr3t", inner))
	})

	t.Run("empty value", func(t *testing.T) {
		inner := errors.New(`parsing "": invalid syntax`)

		assert.Same(t, inner, config_errors.Wrap("env", "KEY", "", inner))
	})
}

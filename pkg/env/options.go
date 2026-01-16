// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package env

import (
	"strings"

	"github.com/zauberhaus/config/pkg/index"
)

type EnvOptions struct {
	Prefix   string
	Strict   bool
	Index    index.Index
	Replacer map[string]string
}

type Option interface {
	Set(*EnvOptions)
}

type optionFunc func(o *EnvOptions)

func (f optionFunc) Set(o *EnvOptions) {
	f(o)
}

func WithName(val string) Option {
	return optionFunc(func(o *EnvOptions) {
		if len(val) != 0 {
			val = strings.ToUpper(val)
			val = strings.ReplaceAll(val, ".", "_")
			val = strings.ReplaceAll(val, "-", "_")
		}

		if len(val) > 0 {
			o.Prefix = val + "_"
		} else {
			o.Prefix = ""
		}
	})
}

var Strict = optionFunc(func(o *EnvOptions) {
	o.Strict = true
})

func WithStrict(val bool) Option {
	return optionFunc(func(o *EnvOptions) {
		o.Strict = val
	})
}

func WithIndex(val index.Index) Option {
	return optionFunc(func(o *EnvOptions) {
		o.Index = val
	})
}

func WithReplacer(val map[string]string) Option {
	return optionFunc(func(o *EnvOptions) {
		o.Replacer = val
	})
}

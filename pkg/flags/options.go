// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package flags

import "github.com/zauberhaus/lookup"

type FlagOptions struct {
	Hooks []lookup.ParserHook
}

// WithParserHooks parses the flag values into types the parser doesn't
// know, such as an interface type.
func WithParserHooks(hooks ...lookup.ParserHook) Option {
	return optionFunc(func(o *FlagOptions) {
		o.Hooks = append(o.Hooks, hooks...)
	})
}

type Option interface {
	Set(*FlagOptions)
}

type optionFunc func(o *FlagOptions)

func (f optionFunc) Set(o *FlagOptions) {
	f(o)
}

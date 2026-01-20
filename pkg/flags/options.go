// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package flags

type FlagOptions struct {
}

type Option interface {
	Set(*FlagOptions)
}

type optionFunc func(o *FlagOptions)

func (f optionFunc) Set(o *FlagOptions) {
	f(o)
}

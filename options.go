// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package config

import (
	"github.com/zauberhaus/config/pkg/flags"
	"github.com/zauberhaus/config/pkg/index"
)

type ConfigOptions struct {
	File       string
	FileType   FileType
	Name       string
	Paths      []string
	Strict     bool
	Index      index.Index
	Flags      *flags.Flags
	Extensions []Extension
	Replacer   map[string]string
}

type Option interface {
	Set(*ConfigOptions)
}

type optionFunc func(o *ConfigOptions)

func (f optionFunc) Set(o *ConfigOptions) {
	f(o)
}

func WithFile(val string) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.File = val
	})
}

func WithPaths(val ...string) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Paths = val
	})
}

func WithName(val string) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Name = val
	})
}

func WithIndex(val index.Index) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Index = val
	})
}

func WithReplacer(m map[string]string) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Replacer = m
	})
}

func WithFlags(val *flags.Flags) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Flags = val
	})
}

func WithExtension(ext string, fileType FileType) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Extensions = []Extension{
			{
				Name:     ext,
				FileType: fileType,
			},
		}
	})
}

func WithExtensions(val []Extension) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Extensions = val
	})
}

var Strict Option = optionFunc(func(o *ConfigOptions) {
	o.Strict = true
})

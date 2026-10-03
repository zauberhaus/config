// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package config

import (
	"context"

	"github.com/zauberhaus/config/pkg/flags"
	"github.com/zauberhaus/config/pkg/index"
)

type ConfigOptions struct {
	File       string
	FileType   FileType
	Storage    Storage
	Name       string
	Paths      []string
	Strict     bool
	Index      index.Index
	Flags      *flags.Flags
	Extensions []Extension
	Replacer   map[string]string

	WorkingDir    bool
	WorldWritable bool

	ctx context.Context
}

type Option interface {
	Set(*ConfigOptions)
}

type optionFunc func(o *ConfigOptions)

func (f optionFunc) Set(o *ConfigOptions) {
	f(o)
}

// WithFile sets the config file to load and disables the search for one,
// including $CONFIG. The type is taken from the extension (.json, .yaml or
// .yml) and a path with a ".." element is rejected.
func WithFile(val string) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.File = val
	})
}

// WithStorage merges all values from val after the config file and before
// environment variables. Load calls only val.All(); keys are lower-case,
// dot-separated field paths such as "sub.name".
func WithStorage(val Storage) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Storage = val
	})
}

// WithPaths sets the directories searched for a config file before the
// working directory (see WithWorkingDir) and the home directory. It replaces
// the paths of an earlier WithPaths.
func WithPaths(val ...string) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Paths = val
	})
}

// WithName sets the base name of the config file (default "config") and the
// prefix of environment variables, e.g. "my-app" reads MY_APP_HOST.
// Environment variables are only read when a name is set.
func WithName(val string) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Name = val
	})
}

// WithIndex sets the index that maps environment variable names to field
// paths. Without it, the index is built from the config struct, so this is
// only needed for custom mappings.
func WithIndex(val index.Index) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Index = val
	})
}

// WithReplacer replaces parts of field names when the environment variable
// names are built, e.g. {"Sub": "Section"} maps Sub.Name to SECTION_NAME.
// It has no effect together with WithIndex.
func WithReplacer(m map[string]string) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Replacer = m
	})
}

// WithFlags applies the bound command-line flags that were set by the user.
// They are applied last, so they override all other sources.
func WithFlags(val *flags.Flags) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Flags = val
	})
}

// WithExtension restricts the config file search, including $CONFIG, to a
// single extension and file type. It replaces any earlier extension options
// but doesn't affect a file given with WithFile.
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

// WithExtensions sets the extensions and file types accepted by the config
// file search, including $CONFIG. It replaces any earlier extension options
// but doesn't affect a file given with WithFile.
func WithExtensions(val []Extension) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.Extensions = val
	})
}

// WithWorkingDir adds the working directory to the config file search,
// between the WithPaths directories and the home directory. It is off by
// default, because a file in an untrusted directory would change the settings.
func WithWorkingDir(val bool) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.WorkingDir = val
	})
}

// WithWorldWritable allows config files that are writable by others. They
// are rejected by default, because anybody could change them. It has no
// effect on Windows.
func WithWorldWritable(val bool) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.WorldWritable = val
	})
}

func WithContext(val context.Context) Option {
	return optionFunc(func(o *ConfigOptions) {
		o.ctx = val
	})
}

// Strict makes Load fail on environment variables with the name prefix (see
// WithName) that don't match a field, instead of ignoring them.
var Strict Option = optionFunc(func(o *ConfigOptions) {
	o.Strict = true
})

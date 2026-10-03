// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/creasty/defaults"
	"github.com/zauberhaus/config/pkg/env"
	"github.com/zauberhaus/config/pkg/errors"
	"github.com/zauberhaus/config/pkg/flags"
	"github.com/zauberhaus/config/pkg/index"
	"github.com/zauberhaus/lookup"
	"go.yaml.in/yaml/v3"
)

var (
	extensions = []Extension{
		{
			Name:     ".json",
			FileType: JSON,
		},
		{
			Name:     ".yaml",
			FileType: YAML,
		},
		{
			Name:     ".yml",
			FileType: YAML,
		},
	}
)

func Load[P ~*T, T any](options ...Option) (P, string, error) {
	o := &ConfigOptions{}
	for _, opt := range options {
		opt.Set(o)
	}

	if o.ctx == nil {
		o.ctx = context.Background()
	}

	// searched is true if the file was found by searching a directory; it is
	// then read without following symlinks out of that directory.
	searched := false

	if o.File == "" {
		f, ft, s, err := findConfigFile(o)
		if err != nil {
			return nil, "", err
		}

		o.File = f
		o.FileType = ft
		searched = s
	} else {
		if hasParentRef(o.File) {
			return nil, "", fmt.Errorf("path traversal attempt: '%s'", o.File)
		}

		o.FileType = GetFileType(o.File, extensions...)
	}

	np := *new(T)
	cfg := &np

	err := defaults.Set(cfg)
	if err != nil {
		return nil, "", err
	}

	if len(o.Index) == 0 {
		d, err := index.New[T](o.Replacer)
		if err != nil {
			return *new(P), "", err
		}

		o.Index = d
	}

	optional := []string{}

	for _, v := range o.Index {
		if v.Optional {
			optional = append(optional, v.Path)
		}
	}

	if len(o.File) > 0 {
		data, err := readConfigFile(o.File, searched, o.WorldWritable)
		if err != nil {
			return nil, o.File, err
		}

		switch o.FileType {
		case JSON:
			err = json.Unmarshal(data, cfg)
		case YAML:
			err = yaml.Unmarshal(data, cfg)
		default:
			return nil, o.File, fmt.Errorf("unknown file type: %s (%v)", o.File, o.FileType)
		}

		if err != nil {
			return nil, o.File, err
		}

		// set default values for struct pointer if set by config file
		if len(optional) > 0 {
			var changed []string

			for _, v := range optional {
				ok, err := lookup.Exists(cfg, v)
				if err != nil {
					return nil, "", err
				}

				if ok {
					changed = append(changed, v)
				}
			}

			// set defaults and reload config file
			if len(changed) > 0 {
				np := *new(T)
				tmp := &np

				err = defaults.Set(tmp)
				if err != nil {
					return nil, "", err
				}

				for _, v := range changed {
					_, err := lookup.Create(tmp, v)
					if err != nil {
						return nil, "", err
					}
				}

				switch o.FileType {
				case JSON:
					err = json.Unmarshal(data, tmp)
				case YAML:
					err = yaml.Unmarshal(data, tmp)
				default:
					return nil, o.File, fmt.Errorf("unknown file type: %s (%v)", o.File, o.FileType)
				}

				if err != nil {
					return nil, o.File, err
				}

				cfg = tmp
			}
		}
	}

	if o.Storage != nil {
		all, err := o.Storage.All(o.ctx)
		if err != nil {
			return nil, o.File, err
		}

		for k, v := range all {
			_, err := lookup.Set(&cfg, k, v)
			if err != nil {
				return nil, o.File, errors.Wrap("storage", k, v, err)
			}
		}
	}

	if len(o.Name) > 0 {
		_, err = env.Set(cfg, env.WithName(o.Name), env.WithStrict(o.Strict), env.WithIndex(o.Index))
		if err != nil {
			return nil, o.File, err
		}
	}

	if o.Flags != nil {
		err = flags.SetFlags(cfg, o.Flags)
		if err != nil {
			return nil, o.File, err
		}
	}

	return cfg, o.File, nil
}

// findConfigFile returns the config file from $CONFIG or the first matching
// file in the search directories, and whether it was found by searching.
func findConfigFile(o *ConfigOptions) (string, FileType, bool, error) {
	if len(o.Extensions) == 0 {
		o.Extensions = extensions
	}

	if o.Name == "" {
		o.Name = "config"
	}

	tmp := os.Getenv("CONFIG")
	if tmp != "" {
		if hasParentRef(tmp) {
			return "", UnknownFileType, false, fmt.Errorf("path traversal attempt: '%s'", tmp)
		}
		fp := filepath.Clean(tmp)

		name, err := filepath.Abs(fp)
		if err != nil {
			return "", UnknownFileType, false, fmt.Errorf("invalid path '%s': %w", fp, err)
		}

		ft := GetFileType(name, o.Extensions...)
		if ft == UnknownFileType {
			return "", UnknownFileType, false, fmt.Errorf("unknown file type: %s", name)
		}

		return name, ft, false, nil
	}

	paths := slices.Clone(o.Paths)

	// The working directory is opt-in: a config file in an untrusted directory
	// would otherwise silently change the settings of every tool run there.
	if o.WorkingDir {
		cwd, err := os.Getwd()
		if err != nil {
			return "", UnknownFileType, false, fmt.Errorf("get current index failed: %v", err)
		}

		paths = append(paths, cwd)
	}

	// Find home index.
	home, err := os.UserHomeDir()
	if err != nil {
		return "", UnknownFileType, false, fmt.Errorf("get homedir failed: %v", err)
	}

	paths = append(paths, home)

	for _, p := range paths {
		fp, err := filepath.Abs(p)
		if err != nil {
			return "", UnknownFileType, false, fmt.Errorf("invalid path '%s': %w", fp, err)
		}

		fp = filepath.Clean(fp)

		entries, err := os.ReadDir(fp)
		if err != nil {
			continue
		}

		for _, e := range entries {
			filename := e.Name()

			if e.IsDir() || len(filename) < 4 || filename[0] == '.' {
				continue
			}

			ext := filepath.Ext(filename)
			if ext == "" {
				continue
			}

			base := strings.TrimSuffix(filename, ext)
			if base != o.Name {
				continue
			}

			ft := GetFileType(filename, o.Extensions...)
			if ft == UnknownFileType {
				continue
			}

			return filepath.Join(fp, filename), ft, true, nil
		}
	}

	return "", UnknownFileType, false, nil
}

// hasParentRef reports whether path contains a ".." element.
func hasParentRef(path string) bool {
	return slices.Contains(strings.Split(filepath.ToSlash(path), "/"), "..")
}

// readConfigFile reads a config file. A file found in a search directory is
// opened in that directory as root, so a symlink can't point outside of it.
// The file must be a regular file and, unless worldWritable is set, must not
// be writable by others.
func readConfigFile(name string, rooted bool, worldWritable bool) ([]byte, error) {
	var (
		stat = os.Stat
		open = os.Open
	)

	if rooted {
		root, err := os.OpenRoot(filepath.Dir(name))
		if err != nil {
			return nil, err
		}

		defer func() { _ = root.Close() }()

		base := filepath.Base(name)
		stat = func(string) (os.FileInfo, error) { return root.Stat(base) }
		open = func(string) (*os.File, error) { return root.Open(base) }
	}

	// Check before opening: opening e.g. a FIFO would block.
	fi, err := stat(name)
	if err != nil {
		return nil, err
	}

	if err := checkConfigFile(name, fi, worldWritable); err != nil {
		return nil, err
	}

	f, err := open(name)
	if err != nil {
		return nil, err
	}

	defer func() { _ = f.Close() }()

	// The file may have been replaced since the check.
	ofi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if !os.SameFile(fi, ofi) {
		return nil, fmt.Errorf("config file %s changed while opening it", name)
	}

	if err := checkConfigFile(name, ofi, worldWritable); err != nil {
		return nil, err
	}

	return io.ReadAll(f)
}

func checkConfigFile(name string, fi os.FileInfo, worldWritable bool) error {
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("config file %s is not a regular file", name)
	}

	if !worldWritable && runtime.GOOS != "windows" && fi.Mode().Perm()&0o002 != 0 {
		return fmt.Errorf("config file %s is writable by others (%v)", name, fi.Mode().Perm())
	}

	return nil
}

func GetFileType(name string, ext ...Extension) FileType {
	if len(ext) == 0 {
		ext = extensions
	}

	for _, v := range ext {
		if strings.HasSuffix(name, v.Name) {
			return v.FileType
		}
	}

	return UnknownFileType
}

// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert/yaml"
	"github.com/zauberhaus/config/pkg/env"
	"github.com/zauberhaus/config/pkg/flags"
	"github.com/zauberhaus/config/pkg/index"
	"github.com/zauberhaus/lookup"
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

	if o.File == "" {
		f, ft, err := findConfigFile(o)
		if err != nil {
			return nil, "", err
		}

		o.File = f
		o.FileType = ft

	} else {
		if strings.Contains(o.File, "..") {
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
		data, err := os.ReadFile(o.File)
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

func findConfigFile(o *ConfigOptions) (string, FileType, error) {
	if len(o.Extensions) == 0 {
		o.Extensions = extensions
	}

	if o.Name == "" {
		o.Name = "config"
	}

	tmp := os.Getenv("CONFIG")
	if tmp != "" {
		if strings.Contains(tmp, "..") {
			return "", UnknownFileType, fmt.Errorf("path traversal attempt: '%s'", tmp)
		}
		fp := filepath.Clean(tmp)

		name, err := filepath.Abs(fp)
		if err != nil {
			return "", UnknownFileType, fmt.Errorf("invalid path '%s': %w", fp, err)
		}

		ft := GetFileType(name, o.Extensions...)
		if ft == UnknownFileType {
			return "", UnknownFileType, fmt.Errorf("unknown file type: %s", name)
		}

		return name, ft, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", UnknownFileType, fmt.Errorf("get current index failed: %v", err)
	}

	paths := append(o.Paths, cwd)

	// Find home index.
	home, err := os.UserHomeDir()
	if err != nil {
		return "", UnknownFileType, fmt.Errorf("get homedir failed: %v", err)
	}

	paths = append(paths, home)

	for _, p := range paths {
		fp, err := filepath.Abs(p)
		if err != nil {
			return "", UnknownFileType, fmt.Errorf("invalid path '%s': %w", fp, err)
		}

		fp = filepath.Clean(fp)

		entries, err := os.ReadDir(fp)
		if err != nil {
			continue
		}

		for _, e := range entries {
			filename := e.Name()

			if strings.Contains(filename, "..") {
				continue
			}

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

			return filepath.Join(fp, filename), ft, nil
		}
	}

	return "", UnknownFileType, nil
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

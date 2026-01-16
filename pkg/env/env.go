// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package env

import (
	"maps"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/zauberhaus/config/pkg/index"
	"github.com/zauberhaus/lookup"
)

func Set[T any](value T, options ...Option) (T, error) {
	o := &EnvOptions{}

	for _, opt := range options {
		opt.Set(o)
	}

	if len(o.Index) == 0 {
		d, err := index.New[T](o.Replacer)
		if err != nil {
			return *new(T), err
		}

		o.Index = d
	}

	m := make(map[string]string)
	for _, envVar := range os.Environ() {
		if i := strings.Index(envVar, "="); i >= 0 {
			key := envVar[:i]
			value := envVar[i+1:]

			if len(o.Prefix) > 0 {
				if !strings.HasPrefix(key, o.Prefix) {
					continue
				}

				key = strings.TrimPrefix(key, o.Prefix)
			}

			key = strings.Trim(key, "_ \n\r\t")
			key = strings.ToUpper(key)

			if k, ok := o.Index.Find(key); ok {
				key = k
			} else {
				if !o.Strict {
					continue
				}

				return *new(T), &lookup.NotFoundError{Name: key}
			}

			value = strings.Trim(value, " \n\r\t")

			m[key] = value
		}
	}

	keys := slices.Collect(maps.Keys(m))
	sort.Strings(keys)

	for _, k := range keys {
		v := m[k]

		_, err := lookup.Set(value, k, v)
		if err != nil {
			if _, ok := err.(*lookup.NotFoundError); ok {
				if !o.Strict {
					continue
				}
			}

			return *new(T), err
		}
	}

	return value, nil
}

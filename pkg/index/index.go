// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package index

import (
	"encoding"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/gobeam/stringy"
	"go.yaml.in/yaml/v3"
)

var (
	braces = regexp.MustCompile(`\[([^\]]*)\]`)
	tm     = reflect.TypeFor[encoding.TextUnmarshaler]()
)

type Item struct {
	Path     string
	Type     reflect.Type
	Optional bool
}

type Index map[string]Item

func New[T any](d map[string]string) (Index, error) {
	v := reflect.TypeFor[T]()

	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, nil
	}

	return collect(v, nil, nil, false, d)
}

func (v Index) String() string {
	var items []any

	keys := slices.Collect(maps.Keys(v))
	sort.Strings(keys)

	for _, k := range keys {
		v := v[k]
		items = append(items, map[string]any{k: map[string]any{v.Path: fmt.Sprintf("%v", v.Type)}})
	}

	date, err := yaml.Marshal(items)
	if err != nil {
		return err.Error()
	} else {
		return string(date)
	}

}

func (v Index) Find(name string) (string, bool) {
	var params []string

	matches := braces.FindAllStringSubmatch(name, -1)
	for _, m := range matches {
		params = append(params, m[1])
	}

	name = braces.ReplaceAllString(name, "[]")

	if r, ok := v[name]; ok {
		for _, p := range params {
			r.Path = strings.Replace(r.Path, "[]", "["+p+"]", 1)
		}

		return r.Path, true
	}

	return "", false
}

func (v Index) Exists(name string) bool {
	name = braces.ReplaceAllString(name, "[]")

	for k := range v {
		if name == k {
			return true
		}
	}

	return false
}

func (v Index) PathExists(name string) bool {
	name = braces.ReplaceAllString(name, "[]")

	for _, v := range v {
		if name == v.Path {
			return true
		}
	}

	return false
}

func (d Index) Keys() []string {
	keys := slices.Collect(maps.Keys(d))
	sort.Strings(keys)
	return keys
}

func (d Index) Items() []Item {
	items := slices.Collect(maps.Values(d))

	sort.Slice(items, func(i, j int) bool {
		v1 := items[i]
		v2 := items[j]

		return strings.Compare(v1.Path, v2.Path) < 0
	})

	return items
}

func collect(v reflect.Type, tag []string, path []string, skip bool, d map[string]string) (map[string]Item, error) {
	m := map[string]Item{}
	isPtr := false

	for v.Kind() == reflect.Pointer {
		isPtr = true
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map:
		e := v.Elem()
		ma := false

		if e.Kind() == reflect.Pointer {
			ma = e.Implements(tm)
			e = e.Elem()
		} else {
			v := reflect.New(e)
			ma = v.Type().Implements(tm)
		}

		if !skip {
			m[strings.Join(tag, "_")] = Item{
				Path:     strings.Join(path, "."),
				Type:     v,
				Optional: isPtr,
			}

			tag[len(tag)-1] += "[]"
			path[len(path)-1] += "[]"

			if ma {
				m[strings.Join(tag, "_")] = Item{
					Path:     strings.Join(path, "."),
					Type:     e,
					Optional: isPtr,
				}
			} else {
				tmp, err := collect(e, tag, path, false, d)
				if err != nil {
					return tmp, err
				}

				maps.Insert(m, maps.All(tmp))
			}
		} else {
			if !ma {
				tmp, err := collect(e, tag, path, skip, d)
				if err != nil {
					return tmp, err
				}

				if len(tmp) > 0 {
					return nil, fmt.Errorf("can't skip slice, array or map: %v", strings.Join(path, "."))
				} else {
					return m, nil
				}
			}
		}

	case reflect.Struct:
		if !skip && len(path) > 0 {

			m[strings.Join(tag, "_")] = Item{
				Path:     strings.Join(path, "."),
				Type:     v,
				Optional: isPtr,
			}
		}

		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)

			if field.IsExported() {
				env := field.Tag.Get("env")

				if env == "--" {
					continue
				} else if env == "-" {

					path := append(path, strings.ToLower(field.Name))
					tmp, err := collect(field.Type, tag, path, true, d)
					if err != nil {
						return tmp, err
					}

					maps.Insert(m, maps.All(tmp))
				} else {
					if len(env) == 0 {
						env = field.Name

						for k, v := range d {
							env = strings.Replace(env, k, v, -1)
						}
					}

					tag := append(tag, SnakeCase(env))
					path := append(path, strings.ToLower(field.Name))

					tmp, err := collect(field.Type, tag, path, false, d)
					if err != nil {
						return tmp, err
					}

					maps.Insert(m, maps.All(tmp))
				}

			}
		}
	default:
		if !skip {
			m[strings.Join(tag, "_")] = Item{
				Path:     strings.Join(path, "."),
				Type:     v,
				Optional: isPtr,
			}
		}
	}

	return m, nil
}

func SnakeCase(s string) string {
	if strings.ToUpper(s) == s || strings.ToLower(s) == s {
		return s
	}

	return stringy.New(s).SnakeCase("?", "").ToUpper()
}

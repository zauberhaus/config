// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package flags

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/zauberhaus/config/pkg/index"
	"github.com/zauberhaus/lookup"
)

type Flag struct {
	fs         *pflag.FlagSet
	flag       *pflag.Flag
	parent     string
	persistent bool
}

func (f *Flag) Name() string {
	return f.flag.Name
}

func (f *Flag) Parent() string {
	return f.parent
}

func (f *Flag) IsPersistent() bool {
	return f.persistent
}

func (f *Flag) Value() any {
	val, err := f.getValue()
	if err != nil {
		panic(err)
	}

	return val
}

func (f *Flag) Changed() bool {
	return f.flag.Changed
}

type Flags struct {
	flags map[string]Flag
	dict  index.Index
}

func NewFlagList(dict index.Index) *Flags {
	return &Flags{
		flags: map[string]Flag{},
		dict:  dict,
	}
}

func (f *Flags) Index() index.Index {
	return f.dict
}

func (f *Flags) Flags() map[string]Flag {
	return f.flags
}

func (f *Flags) BindCmdFlag(cmd *cobra.Command, target string, source string) error {
	if len(target) == 0 {
		return errors.New("empty target name")
	}

	if cmd == nil {
		return errors.New("bind command is nil")
	}

	var fs *pflag.FlagSet
	var flag *pflag.Flag
	persistent := false
	parent := cmd.Use

	if val := cmd.PersistentFlags().Lookup(source); val != nil {
		fs = cmd.PersistentFlags()
		flag = val
		persistent = true
	} else if val := cmd.Flags().Lookup(source); val != nil {
		fs = cmd.Flags()
		flag = val
	} else {
		return fmt.Errorf("source flag not found: %s -> %s", source, target)
	}

	if len(f.dict) > 0 {
		if t, ok := f.dict.Find(target); ok {
			target = t
		} else {
			t := strings.ToLower(target)
			if !f.dict.PathExists(t) {
				return fmt.Errorf("target field not found: %s", target)
			} else {
				target = t
			}
		}
	} else {
		target = strings.ToLower(target)
	}

	f.flags[target] = Flag{
		fs:         fs,
		flag:       flag,
		parent:     parent,
		persistent: persistent,
	}

	return nil
}

func (f *Flags) BindFlag(fs *pflag.FlagSet, target string, flag *pflag.Flag) error {

	if len(target) == 0 {
		return errors.New("empty target name")
	}

	if flag == nil {
		return fmt.Errorf("flag %q not found", target)
	}

	target = strings.ToLower(target)
	f.flags[target] = Flag{
		fs:   fs,
		flag: flag,
	}

	return nil
}

func (f *Flag) getValue() (any, error) {
	switch f.flag.Value.Type() {
	case "string":
		return f.fs.GetString(f.flag.Name)
	case "bool":
		return f.fs.GetBool(f.flag.Name)
	case "int":
		return f.fs.GetInt(f.flag.Name)
	case "stringSlice":
		return f.fs.GetStringSlice(f.flag.Name)
	case "boolSlice":
		return f.fs.GetBoolSlice(f.flag.Name)
	case "intSlice":
		return f.fs.GetIntSlice(f.flag.Name)
	case "durationSlice":
		return f.fs.GetDurationSlice(f.flag.Name)
	case "ipSlice":
		return f.fs.GetIPSlice(f.flag.Name)
	case "int8":
		return f.fs.GetInt8(f.flag.Name)
	case "int16":
		return f.fs.GetInt16(f.flag.Name)
	case "int32":
		return f.fs.GetInt32(f.flag.Name)
	case "int64":
		return f.fs.GetInt64(f.flag.Name)
	case "uint":
		return f.fs.GetUint(f.flag.Name)
	case "uint8":
		return f.fs.GetUint8(f.flag.Name)
	case "uint16":
		return f.fs.GetUint16(f.flag.Name)
	case "uint32":
		return f.fs.GetUint32(f.flag.Name)
	case "uint64":
		return f.fs.GetUint64(f.flag.Name)
	case "float32":
		return f.fs.GetFloat32(f.flag.Name)
	case "float64":
		return f.fs.GetFloat64(f.flag.Name)
	case "duration":
		return f.fs.GetDuration(f.flag.Name)
	case "ip":
		return f.fs.GetIP(f.flag.Name)
	default:
		return f.flag.Value.String(), nil
	}
}

func SetFlags[T any](value T, f *Flags, options ...Option) error {
	for k, v := range f.flags {
		if v.flag.Changed {
			val, err := v.getValue()
			if err != nil {
				return err
			}

			_, err = lookup.Set(value, k, val)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

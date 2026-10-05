// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package flags

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	config_errors "github.com/zauberhaus/config/pkg/errors"
	"github.com/zauberhaus/config/pkg/index"
	"github.com/zauberhaus/lookup"
)

var textUnmarshaler = reflect.TypeFor[encoding.TextUnmarshaler]()

type Secret interface {
	Secret() any
}

type Flag struct {
	fs         *pflag.FlagSet
	flag       *pflag.Flag
	parent     string
	persistent bool
	get        func(val any) (any, error)
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
	return f.BindCmdFlagFunc(cmd, target, source, nil)
}

func (f *Flags) BindCmdFlagFunc(cmd *cobra.Command, target string, source string, get func(val any) (any, error)) error {
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
		get:        get,
	}

	return nil
}

// FromCommand creates a flag list for dict and binds all flags of cmd that
// match a config field (see BindCmdFlags).
func FromCommand(cmd *cobra.Command, dict index.Index) (*Flags, error) {
	f := NewFlagList(dict)
	if err := f.BindCmdFlags(cmd); err != nil {
		return nil, err
	}

	return f, nil
}

// BindCmdFlags binds every flag of cmd, including the persistent flags
// inherited from its parents, whose name matches a field in the index. The
// name is matched like the environment variable of the field, with dashes
// instead of underscores (--sub-name sets Sub.Name), or as the field path
// (--sub.name). A flag tag on the field overrides the name: with
// `flag:"addr"` only --addr sets the field, with `flag:"-"` no flag does.
// Flags without a matching field, such as --config, are skipped, as are
// fields that are already bound and struct fields.
func (f *Flags) BindCmdFlags(cmd *cobra.Command) error {
	if cmd == nil {
		return errors.New("bind command is nil")
	}

	if len(f.dict) == 0 {
		return errors.New("an index is required to bind flags by name")
	}

	sets := []struct {
		fs         *pflag.FlagSet
		persistent bool
	}{
		{cmd.PersistentFlags(), true},
		{cmd.LocalNonPersistentFlags(), false},
		{cmd.InheritedFlags(), true},
	}

	for _, s := range sets {
		s.fs.VisitAll(func(flag *pflag.Flag) {
			target, ok := f.findTarget(flag.Name)
			if !ok {
				return
			}

			if _, ok := f.flags[target]; ok {
				return
			}

			f.flags[target] = Flag{
				fs:         s.fs,
				flag:       flag,
				parent:     cmd.Use,
				persistent: s.persistent,
			}
		})
	}

	return nil
}

// findTarget returns the field path for a flag name, or false if there is no
// field with that name that can be set by a flag. A field with a flag tag is
// only found by the name in the tag, and never if the tag is "-".
func (f *Flags) findTarget(name string) (string, bool) {
	item, ok := f.lookup(name)
	if !ok || item.Flag == "-" {
		return "", false
	}

	if item.Flag != "" && item.Flag != name {
		return "", false
	}

	// a struct can't be set from a single flag, unless it parses text like time.Time
	if item.Type.Kind() == reflect.Struct && !reflect.PointerTo(item.Type).Implements(textUnmarshaler) {
		return "", false
	}

	return item.Path, true
}

// lookup finds the index item for a flag name: by the flag tag first, then
// like the environment variable with dashes instead of underscores, and
// finally by the field path.
func (f *Flags) lookup(name string) (index.Item, bool) {
	for _, v := range f.dict {
		if v.Flag == name {
			return v, true
		}
	}

	if v, ok := f.dict[strings.ToUpper(strings.ReplaceAll(name, "-", "_"))]; ok {
		return v, true
	}

	path := strings.ToLower(name)
	for _, v := range f.dict {
		if v.Path == path {
			return v, true
		}
	}

	return index.Item{}, false
}

func (f *Flags) BindFlag(fs *pflag.FlagSet, target string, flag *pflag.Flag) error {
	return f.BindFlagFunc(fs, target, flag, nil)
}

func (f *Flags) BindFlagFunc(fs *pflag.FlagSet, target string, flag *pflag.Flag, get func(val any) (any, error)) error {

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
		get:  get,
	}

	return nil
}

func (f *Flag) getValue() (any, error) {
	if f.get != nil {
		return f.get(f.flag.Value)
	}

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
	o := &FlagOptions{}
	for _, opt := range options {
		opt.Set(o)
	}

	for k, v := range f.flags {
		if v.flag.Changed {
			val, err := v.getValue()
			if err != nil {
				return config_errors.Wrap("flag", v.flag.Name, v.flag.Value.String(), err)
			}

			_, err = lookup.Set(value, k, val)
			if err != nil {
				return config_errors.Wrap("flag", v.flag.Name, val, err)
			}
		}
	}

	return nil
}

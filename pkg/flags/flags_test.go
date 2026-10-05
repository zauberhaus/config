// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// cspell:words maxconns

package flags_test

import (
	"errors"
	"maps"
	"net"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zauberhaus/config/pkg/flags"
	"github.com/zauberhaus/config/pkg/index"
)

type customValue string

func (c *customValue) String() string     { return string(*c) }
func (c *customValue) Set(s string) error { *c = customValue("custom:" + s); return nil }
func (c *customValue) Type() string       { return "customValue" }

func TestNewFlagList(t *testing.T) {
	t.Run("with nil index", func(t *testing.T) {
		fl := flags.NewFlagList(nil)
		assert.NotNil(t, fl)
	})

	t.Run("with empty index", func(t *testing.T) {
		dict := index.Index{}
		fl := flags.NewFlagList(dict)
		assert.NotNil(t, fl)
	})

	t.Run("with non-empty index", func(t *testing.T) {
		dict := index.Index{"KEY": index.Item{Path: "path.key", Type: reflect.TypeFor[string](), Optional: false}}
		fl := flags.NewFlagList(dict)
		assert.NotNil(t, fl)
	})
}

func TestFlags_Index(t *testing.T) {
	dict := index.Index{"KEY": index.Item{Path: "path.key", Type: reflect.TypeFor[string](), Optional: false}}
	fl := flags.NewFlagList(dict)
	assert.Equal(t, dict, fl.Index())

	fl = flags.NewFlagList(nil)
	assert.Nil(t, fl.Index())
}

func TestFlag_Methods(t *testing.T) {
	cmd := &cobra.Command{Use: "test-cmd"}
	cmd.PersistentFlags().String("host", "localhost", "host")
	cmd.Flags().String("port", "8080", "port")
	require.NoError(t, cmd.PersistentFlags().Set("host", "newhost"))

	fl := flags.NewFlagList(nil)
	require.NoError(t, fl.BindCmdFlag(cmd, "Host", "host"))
	require.NoError(t, fl.BindCmdFlag(cmd, "Port", "port"))

	flagMap := fl.Flags()
	require.Contains(t, flagMap, "host")
	require.Contains(t, flagMap, "port")
	hostFlag := flagMap["host"]
	portFlag := flagMap["port"]

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "host", hostFlag.Name())
		assert.Equal(t, "port", portFlag.Name())
	})

	t.Run("Parent", func(t *testing.T) {
		assert.Equal(t, "test-cmd", hostFlag.Parent())
		assert.Equal(t, "test-cmd", portFlag.Parent())
	})

	t.Run("IsPersistent", func(t *testing.T) {
		assert.True(t, hostFlag.IsPersistent())
		assert.False(t, portFlag.IsPersistent())
	})

	t.Run("Changed", func(t *testing.T) {
		assert.True(t, hostFlag.Changed())
		assert.False(t, portFlag.Changed())
	})

	t.Run("Value", func(t *testing.T) {
		assert.Equal(t, "newhost", hostFlag.Value())
		assert.Equal(t, "8080", portFlag.Value())
	})
}

func TestFlags_BindFlag(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fl := flags.NewFlagList(nil)
	flag := &pflag.Flag{Name: "test-flag"}

	t.Run("successful binding", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		fl := flags.NewFlagList(nil)
		fs.String("test-flag", "default", "usage")
		flag = fs.Lookup("test-flag")

		err := fl.BindFlag(fs, "My.Target", flag)
		require.NoError(t, err)

		flagMap := fl.Flags()
		require.Contains(t, flagMap, "my.target")
		boundFlag := flagMap["my.target"]
		assert.Equal(t, "test-flag", boundFlag.Name())
		assert.False(t, boundFlag.Changed())
	})

	t.Run("error on empty target", func(t *testing.T) {
		err := fl.BindFlag(fs, "", flag)
		assert.Error(t, err)
		assert.EqualError(t, err, "empty target name")
	})

	t.Run("error on nil flag", func(t *testing.T) {
		err := fl.BindFlag(fs, "some.target", nil)
		assert.Error(t, err)
		assert.EqualError(t, err, `flag "some.target" not found`)
	})
}

func TestFlags_BindCmdFlag(t *testing.T) {
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{}
		cmd.PersistentFlags().String("source-flag", "default", "usage")
		cmd.Flags().String("local-flag", "local-default", "usage")
		return cmd
	}

	t.Run("successful binding without index", func(t *testing.T) {
		cmd := newCmd()

		fl := flags.NewFlagList(nil)
		err := fl.BindCmdFlag(cmd, "my.target", "source-flag")
		assert.NoError(t, err)
	})

	t.Run("successful binding with index", func(t *testing.T) {
		cmd := newCmd()

		dict, _ := index.New[struct {
			MyTarget string `env:"MY_TARGET"`
		}](nil)
		fl := flags.NewFlagList(dict)

		err := fl.BindCmdFlag(cmd, "mytarget", "source-flag")
		assert.NoError(t, err)

		err = fl.BindCmdFlag(cmd, "MY_TARGET", "source-flag")
		assert.NoError(t, err)
	})

	t.Run("error when flag not found in flagset", func(t *testing.T) {
		cmd := newCmd()
		fl := flags.NewFlagList(nil)

		err := fl.BindCmdFlag(cmd, "my.target", "non-existent-flag")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source flag not found: non-existent-flag -> my.target")
	})

	t.Run("error when target not found in index", func(t *testing.T) {
		cmd := newCmd()

		type otherStruct struct {
			SomeOtherKey string
		}

		dict, _ := index.New[otherStruct](nil)
		fl := flags.NewFlagList(dict)

		err := fl.BindCmdFlag(cmd, "NON_EXISTENT_TARGET", "source-flag")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target field not found")
		assert.Contains(t, err.Error(), "NON_EXISTENT_TARGET")
	})

	t.Run("error with nil command", func(t *testing.T) {
		fl := flags.NewFlagList(nil)
		err := fl.BindCmdFlag(nil, "my.target", "source-flag")
		require.Error(t, err)
		assert.EqualError(t, err, "bind command is nil")
	})
}

func TestSetFlags(t *testing.T) {
	type SubConfig struct {
		Name string
	}
	type FlagTestConfig struct {
		Host  string
		Port  int
		Sub   SubConfig
		Slice []bool
	}

	t.Run("set simple flag values", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flagSet.String("host", "localhost", "host name")
		flagSet.Int("port", 8080, "port number")
		flagSet.BoolSlice("slice", nil, "bool slice")

		// Simulate setting flags from command line
		require.NoError(t, flagSet.Set("host", "testhost.com"))
		require.NoError(t, flagSet.Set("port", "9090"))
		require.NoError(t, flagSet.Set("slice", "true,false"))

		var cfg FlagTestConfig
		dict, err := index.New[FlagTestConfig](nil)
		require.NoError(t, err)

		fl := flags.NewFlagList(dict)
		require.NoError(t, fl.BindFlag(flagSet, "hOst", flagSet.Lookup("host")))
		require.NoError(t, fl.BindFlag(flagSet, "port", flagSet.Lookup("port")))
		require.NoError(t, fl.BindFlag(flagSet, "Slice", flagSet.Lookup("slice")))

		err = flags.SetFlags(&cfg, fl)

		require.NoError(t, err)
		assert.Equal(t, "testhost.com", cfg.Host)
		assert.Equal(t, 9090, cfg.Port)
		assert.Equal(t, []bool{true, false}, cfg.Slice)
	})

	t.Run("set nested struct field flag value", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flagSet.String("sub-name", "default", "sub name")

		require.NoError(t, flagSet.Set("sub-name", "nested-name"))

		fl := flags.NewFlagList(nil)
		require.NoError(t, fl.BindFlag(flagSet, "sub.name", flagSet.Lookup("sub-name")))

		var cfg FlagTestConfig
		err := flags.SetFlags(&cfg, fl)

		require.NoError(t, err)
		assert.Equal(t, "nested-name", cfg.Sub.Name)
	})

	t.Run("error on setting to non-pointer struct", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flagSet.String("host", "localhost", "host name")
		require.NoError(t, flagSet.Set("host", "somehost"))

		fl := flags.NewFlagList(nil)
		require.NoError(t, fl.BindFlag(flagSet, "Host", flagSet.Lookup("host")))

		var cfg FlagTestConfig // A non-pointer struct
		err := flags.SetFlags(cfg, fl)

		require.Error(t, err)
		assert.EqualError(t, err, "set supports only struct pointers")
	})

	t.Run("no error with no flags bound", func(t *testing.T) {
		fl := flags.NewFlagList(nil)
		var cfg FlagTestConfig
		err := flags.SetFlags(&cfg, fl)
		assert.NoError(t, err)
	})

	t.Run("set with custom flag type", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
		var cv customValue
		flagSet.Var(&cv, "host", "custom host")

		require.NoError(t, flagSet.Set("host", "myhost"))

		var cfg FlagTestConfig
		fl := flags.NewFlagList(nil)
		require.NoError(t, fl.BindFlag(flagSet, "Host", flagSet.Lookup("host")))

		err := flags.SetFlags(&cfg, fl)
		require.NoError(t, err)
		assert.Equal(t, "custom:myhost", cfg.Host)
	})

	t.Run("error on type mismatch between flag and struct field", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
		// Flag is string slice, but target field `Slice` is []bool
		flagSet.StringSlice("slice", nil, "string slice")
		require.NoError(t, flagSet.Set("slice", "true,false"))

		var cfg FlagTestConfig
		dict, err := index.New[FlagTestConfig](nil)
		require.NoError(t, err)

		fl := flags.NewFlagList(dict)
		// Bind string slice flag to the "Slice" field which is []bool
		require.NoError(t, fl.BindFlag(flagSet, "Slice", flagSet.Lookup("slice")))

		err = flags.SetFlags(&cfg, fl)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid data type")
	})
}

func TestSetFlags_MoreTypes(t *testing.T) {
	type MoreTypesConfig struct {
		IntSlice      []int
		DurationSlice []time.Duration
		IPSlice       []net.IP
		Int8          int8
		Int16         int16
		Int32         int32
		Int64         int64
		Uint          uint
		Uint8         uint8
		Uint16        uint16
		Uint32        uint32
		Uint64        uint64
		Float32       float32
		Float64       float64
		Duration      time.Duration
		IP            net.IP
	}

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.IntSlice("int-slice", nil, "")
	flagSet.DurationSlice("duration-slice", nil, "")
	flagSet.IPSlice("ip-slice", nil, "")
	flagSet.Int8("int8", 0, "")
	flagSet.Int16("int16", 0, "")
	flagSet.Int32("int32", 0, "")
	flagSet.Int64("int64", 0, "")
	flagSet.Uint("uint", 0, "")
	flagSet.Uint8("uint8", 0, "")
	flagSet.Uint16("uint16", 0, "")
	flagSet.Uint32("uint32", 0, "")
	flagSet.Uint64("uint64", 0, "")
	flagSet.Float32("float32", 0, "")
	flagSet.Float64("float64", 0, "")
	flagSet.Duration("duration", 0, "")
	flagSet.IP("ip", nil, "")

	require.NoError(t, flagSet.Set("int-slice", "1,2,3"))
	require.NoError(t, flagSet.Set("duration-slice", "1s,2m"))
	require.NoError(t, flagSet.Set("ip-slice", "1.1.1.1,2.2.2.2"))
	require.NoError(t, flagSet.Set("int8", "8"))
	require.NoError(t, flagSet.Set("int16", "16"))
	require.NoError(t, flagSet.Set("int32", "32"))
	require.NoError(t, flagSet.Set("int64", "64"))
	require.NoError(t, flagSet.Set("uint", "1"))
	require.NoError(t, flagSet.Set("uint8", "8"))
	require.NoError(t, flagSet.Set("uint16", "16"))
	require.NoError(t, flagSet.Set("uint32", "32"))
	require.NoError(t, flagSet.Set("uint64", "64"))
	require.NoError(t, flagSet.Set("float32", "32.32"))
	require.NoError(t, flagSet.Set("float64", "64.64"))
	require.NoError(t, flagSet.Set("duration", "5s"))
	require.NoError(t, flagSet.Set("ip", "127.0.0.1"))

	mapping := map[string]string{
		"IntSlice":      "int-slice",
		"DurationSlice": "duration-slice",
		"IPSlice":       "ip-slice",
		"Int8":          "int8",
		"Int16":         "int16",
		"Int32":         "int32",
		"Int64":         "int64",
		"Uint":          "uint",
		"Uint8":         "uint8",
		"Uint16":        "uint16",
		"Uint32":        "uint32",
		"Uint64":        "uint64",
		"Float32":       "float32",
		"Float64":       "float64",
		"Duration":      "duration",
		"IP":            "ip",
	}

	var cfg MoreTypesConfig
	fl := flags.NewFlagList(nil)

	for _, f := range []string{"IntSlice", "DurationSlice", "IPSlice", "Int8", "Int16", "Int32", "Int64", "Uint", "Uint8", "Uint16", "Uint32", "Uint64", "Float32", "Float64", "Duration", "IP"} {
		name := mapping[f]
		flag := flagSet.Lookup(name)
		require.NoError(t, fl.BindFlag(flagSet, f, flag))
	}

	err := flags.SetFlags(&cfg, fl)
	require.NoError(t, err)

	assert.Equal(t, []int{1, 2, 3}, cfg.IntSlice)
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Minute}, cfg.DurationSlice)
	assert.Equal(t, []net.IP{net.ParseIP("1.1.1.1"), net.ParseIP("2.2.2.2")}, cfg.IPSlice)
	assert.Equal(t, int8(8), cfg.Int8)
	assert.Equal(t, uint(1), cfg.Uint)
	assert.InEpsilon(t, float32(32.32), cfg.Float32, 1e-6)
	assert.InEpsilon(t, float64(64.64), cfg.Float64, 1e-6)
	assert.Equal(t, 5*time.Second, cfg.Duration)
	assert.Equal(t, net.ParseIP("127.0.0.1"), cfg.IP)
}

func TestSetFlags_BoolAndStringSlice(t *testing.T) {
	type Config struct {
		Enable bool
		Tags   []string
	}
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.Bool("enable", false, "")
	flagSet.StringSlice("tags", nil, "")

	require.NoError(t, flagSet.Set("enable", "true"))
	require.NoError(t, flagSet.Set("tags", "a,b"))

	var cfg Config
	fl := flags.NewFlagList(nil)
	require.NoError(t, fl.BindFlag(flagSet, "Enable", flagSet.Lookup("enable")))
	require.NoError(t, fl.BindFlag(flagSet, "Tags", flagSet.Lookup("tags")))

	err := flags.SetFlags(&cfg, fl)
	require.NoError(t, err)
	assert.True(t, cfg.Enable)
	assert.Equal(t, []string{"a", "b"}, cfg.Tags)
}

func TestBindCmdFlag_PathTarget(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("src", "val", "")

	type Config struct {
		Nested struct {
			Field string
		}
	}
	// Index will have NESTED_FIELD -> nested.field
	dict, err := index.New[Config](nil)
	require.NoError(t, err)

	fl := flags.NewFlagList(dict)

	// Bind using path "nested.field" instead of key "NESTED_FIELD"
	err = fl.BindCmdFlag(cmd, "nested.field", "src")
	assert.NoError(t, err)

	// Verify it was added
	assert.Contains(t, fl.Flags(), "nested.field")
}

func TestValue_Panic(t *testing.T) {
	fs1 := pflag.NewFlagSet("fs1", pflag.ContinueOnError)
	fs1.String("foo", "val", "")
	flagFoo := fs1.Lookup("foo")

	fs2 := pflag.NewFlagSet("fs2", pflag.ContinueOnError)
	// fs2 does not have "foo"

	fl := flags.NewFlagList(nil)
	// Bind flagFoo but associate it with fs2
	err := fl.BindFlag(fs2, "target", flagFoo)
	require.NoError(t, err)

	// Now calling Value() on the bound flag should try fs2.GetString("foo") which should fail.
	f := fl.Flags()["target"]

	assert.Panics(t, func() {
		_ = f.Value()
	})
}

func TestBindCmdFlag_EmptyTarget(t *testing.T) {
	cmd := &cobra.Command{}
	fl := flags.NewFlagList(nil)
	err := fl.BindCmdFlag(cmd, "", "src")
	assert.Error(t, err)
	assert.EqualError(t, err, "empty target name")
}

func TestFlags_BindFlagFunc(t *testing.T) {
	t.Run("custom get function is used for Value", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		fs.String("host", "default", "host")
		flag := fs.Lookup("host")

		fl := flags.NewFlagList(nil)
		err := fl.BindFlagFunc(fs, "host", flag, func(val any) (any, error) {
			return "custom-value", nil
		})
		require.NoError(t, err)

		f := fl.Flags()["host"]
		assert.Equal(t, "custom-value", f.Value())
	})

	t.Run("custom get function returning error causes panic", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		fs.String("host", "default", "host")
		flag := fs.Lookup("host")

		fl := flags.NewFlagList(nil)
		err := fl.BindFlagFunc(fs, "host", flag, func(val any) (any, error) {
			return nil, errors.New("get error")
		})
		require.NoError(t, err)

		f := fl.Flags()["host"]
		assert.Panics(t, func() { _ = f.Value() })
	})

	t.Run("error on empty target", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		fs.String("host", "default", "host")
		flag := fs.Lookup("host")

		fl := flags.NewFlagList(nil)
		err := fl.BindFlagFunc(fs, "", flag, nil)
		assert.EqualError(t, err, "empty target name")
	})

	t.Run("error on nil flag", func(t *testing.T) {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		fl := flags.NewFlagList(nil)
		err := fl.BindFlagFunc(fs, "host", nil, nil)
		assert.EqualError(t, err, `flag "host" not found`)
	})
}

func TestFlags_BindCmdFlagFunc(t *testing.T) {
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "test-cmd"}
		cmd.PersistentFlags().String("source-flag", "default", "usage")
		cmd.Flags().String("local-flag", "local-default", "usage")
		return cmd
	}

	t.Run("custom get function is used for Value", func(t *testing.T) {
		cmd := newCmd()
		fl := flags.NewFlagList(nil)

		err := fl.BindCmdFlagFunc(cmd, "my.target", "source-flag", func(val any) (any, error) {
			return "custom-result", nil
		})
		require.NoError(t, err)

		f := fl.Flags()["my.target"]
		assert.Equal(t, "custom-result", f.Value())
	})

	t.Run("custom get function returning error causes panic", func(t *testing.T) {
		cmd := newCmd()
		fl := flags.NewFlagList(nil)

		err := fl.BindCmdFlagFunc(cmd, "my.target", "source-flag", func(val any) (any, error) {
			return nil, errors.New("get error")
		})
		require.NoError(t, err)

		f := fl.Flags()["my.target"]
		assert.Panics(t, func() { _ = f.Value() })
	})

	t.Run("nil get function falls back to flag value", func(t *testing.T) {
		cmd := newCmd()
		require.NoError(t, cmd.PersistentFlags().Set("source-flag", "flag-value"))
		fl := flags.NewFlagList(nil)

		err := fl.BindCmdFlagFunc(cmd, "my.target", "source-flag", nil)
		require.NoError(t, err)

		f := fl.Flags()["my.target"]
		assert.Equal(t, "flag-value", f.Value())
	})

	t.Run("error on empty target", func(t *testing.T) {
		cmd := newCmd()
		fl := flags.NewFlagList(nil)
		err := fl.BindCmdFlagFunc(cmd, "", "source-flag", nil)
		assert.EqualError(t, err, "empty target name")
	})

	t.Run("error on nil command", func(t *testing.T) {
		fl := flags.NewFlagList(nil)
		err := fl.BindCmdFlagFunc(nil, "my.target", "source-flag", nil)
		assert.EqualError(t, err, "bind command is nil")
	})

	t.Run("error when source flag not found", func(t *testing.T) {
		cmd := newCmd()
		fl := flags.NewFlagList(nil)
		err := fl.BindCmdFlagFunc(cmd, "my.target", "non-existent", nil)
		assert.Contains(t, err.Error(), "source flag not found: non-existent -> my.target")
	})
}

// badIntValue claims to be an int flag but holds a value that is not a number.
type badIntValue struct{}

func (badIntValue) String() string   { return "not-a-number" }
func (badIntValue) Set(string) error { return nil }
func (badIntValue) Type() string     { return "int" }

func TestSetFlags_ValueError(t *testing.T) {
	type cfg struct {
		Port int
	}

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.Var(badIntValue{}, "port", "port number")
	require.NoError(t, fs.Set("port", "1"))

	fl := flags.NewFlagList(nil)
	require.NoError(t, fl.BindFlag(fs, "Port", fs.Lookup("port")))

	c := &cfg{}
	err := flags.SetFlags(c, fl)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "not-a-number")
	assert.Contains(t, err.Error(), "flag port: ")
	assert.Zero(t, c.Port)
}

func TestFromCommand(t *testing.T) {
	type Sub struct {
		Name string
	}

	type Config struct {
		Host     string
		Port     int
		MaxConns int `flag:"max-conns"`
		Timeout  time.Duration
		Started  time.Time
		Renamed  string `env:"ALIAS"`
		Sub      Sub
		Tags     []string
	}

	dict, err := index.New[Config](nil)
	require.NoError(t, err)

	newCmd := func() (*cobra.Command, *cobra.Command) {
		root := &cobra.Command{Use: "root"}
		root.PersistentFlags().String("host", "", "")

		child := &cobra.Command{Use: "child", Run: func(*cobra.Command, []string) {}}
		child.PersistentFlags().Int("port", 0, "")
		child.Flags().Int("max-conns", 0, "") // named by the flag tag
		child.Flags().Duration("timeout", 0, "")
		child.Flags().String("started", "", "")
		child.Flags().String("renamed", "", "")
		child.Flags().String("sub-name", "", "")
		child.Flags().String("sub", "", "")
		child.Flags().StringSlice("tags", nil, "")
		child.Flags().String("config", "", "")
		root.AddCommand(child)

		return root, child
	}

	t.Run("binds matching flags", func(t *testing.T) {
		_, child := newCmd()

		fl, err := flags.FromCommand(child, dict)
		require.NoError(t, err)

		got := fl.Flags()
		flag := func(name string) *flags.Flag {
			f, ok := got[name]
			require.True(t, ok, "flag for %s not bound", name)
			return &f
		}

		// the keys are field paths: lower-cased field names, as lookup.Set expects them
		assert.ElementsMatch(t,
			[]string{"host", "port", "maxconns", "timeout", "started", "renamed", "sub.name", "tags"},
			slices.Collect(maps.Keys(got)))

		assert.Equal(t, "max-conns", flag("maxconns").Name())
		assert.Equal(t, "sub-name", flag("sub.name").Name())
		assert.True(t, flag("host").IsPersistent()) // inherited from root
		assert.True(t, flag("port").IsPersistent())
		assert.False(t, flag("timeout").IsPersistent())
		assert.Equal(t, "child", flag("host").Parent())
	})

	t.Run("skips flags without a field and struct fields", func(t *testing.T) {
		_, child := newCmd()

		fl, err := flags.FromCommand(child, dict)
		require.NoError(t, err)

		assert.NotContains(t, fl.Flags(), "config")
		assert.NotContains(t, fl.Flags(), "sub")
	})

	t.Run("sets the config from the command line", func(t *testing.T) {
		root, child := newCmd()

		fl, err := flags.FromCommand(child, dict)
		require.NoError(t, err)

		root.SetArgs([]string{"child", "--host", "h.example.com", "--max-conns", "7", "--sub-name", "s", "--tags", "a,b", "--timeout", "3s", "--started", "2026-01-02T03:04:05Z"})
		require.NoError(t, root.Execute())

		var cfg Config
		require.NoError(t, flags.SetFlags(&cfg, fl))

		assert.Equal(t, "h.example.com", cfg.Host)
		assert.Equal(t, 7, cfg.MaxConns)
		assert.Equal(t, "s", cfg.Sub.Name)
		assert.Equal(t, []string{"a", "b"}, cfg.Tags)
		assert.Equal(t, 3*time.Second, cfg.Timeout)
		assert.Equal(t, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), cfg.Started)
		assert.Equal(t, 0, cfg.Port) // not changed
	})

	t.Run("an explicit binding is kept", func(t *testing.T) {
		_, child := newCmd()
		child.Flags().String("address", "", "")

		fl := flags.NewFlagList(dict)
		require.NoError(t, fl.BindCmdFlag(child, "host", "address"))
		require.NoError(t, fl.BindCmdFlags(child))

		host := fl.Flags()["host"]
		assert.Equal(t, "address", host.Name())
	})

	t.Run("errors", func(t *testing.T) {
		_, child := newCmd()

		_, err := flags.FromCommand(nil, dict)
		assert.EqualError(t, err, "bind command is nil")

		_, err = flags.FromCommand(child, nil)
		assert.EqualError(t, err, "an index is required to bind flags by name")
	})
}

func TestFromCommand_FlagTag(t *testing.T) {
	type Config struct {
		Host   string `flag:"addr"`
		Port   int    `flag:"-"`
		Secret string `flag:"-"`
		Sub    struct {
			Name string `flag:"title"`
		}
		Plain string
	}

	dict, err := index.New[Config](nil)
	require.NoError(t, err)

	cmd := &cobra.Command{Use: "app", Run: func(*cobra.Command, []string) {}}
	cmd.Flags().String("addr", "", "")
	cmd.Flags().String("host", "", "") // the field name, overridden by the tag
	cmd.Flags().Int("port", 0, "")
	cmd.Flags().String("sub-name", "", "")
	cmd.Flags().String("title", "", "")
	cmd.Flags().String("plain", "", "")

	fl, err := flags.FromCommand(cmd, dict)
	require.NoError(t, err)

	got := fl.Flags()
	assert.ElementsMatch(t, []string{"host", "sub.name", "plain"}, slices.Collect(maps.Keys(got)))

	host := got["host"]
	assert.Equal(t, "addr", host.Name())

	name := got["sub.name"]
	assert.Equal(t, "title", name.Name())

	cmd.SetArgs([]string{"--addr", "a.example.com", "--host", "ignored", "--port", "1", "--title", "t", "--plain", "p"})
	require.NoError(t, cmd.Execute())

	var cfg Config
	require.NoError(t, flags.SetFlags(&cfg, fl))

	assert.Equal(t, "a.example.com", cfg.Host)
	assert.Equal(t, 0, cfg.Port) // flag:"-"
	assert.Equal(t, "t", cfg.Sub.Name)
	assert.Equal(t, "p", cfg.Plain)
}

// level is an enum flag: its String() is a name, not the number it holds.
type level int

var levelNames = []string{"debug", "info", "warn"}

func (l *level) String() string { return levelNames[*l] }
func (l *level) Type() string   { return "level" }
func (l *level) Set(s string) error {
	i := slices.Index(levelNames, s)
	if i < 0 {
		return errors.New("unknown level")
	}

	*l = level(i)

	return nil
}

// endpoint is a struct flag in the form host:port.
type endpoint struct {
	Host string
	Port string
}

func (e *endpoint) String() string { return e.Host + ":" + e.Port }
func (e *endpoint) Type() string   { return "endpoint" }
func (e *endpoint) Set(s string) error {
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return err
	}

	e.Host, e.Port = host, port

	return nil
}

func TestFromCommand_ValueFlags(t *testing.T) {
	type Config struct {
		Level    level
		MaxLevel *level
		Name     string
		Endpoint endpoint
	}

	dict, err := index.New[Config](nil)
	require.NoError(t, err)

	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "app", Run: func(*cobra.Command, []string) {}}

		var lvl, maxLvl, name level
		var ep endpoint

		cmd.Flags().Var(&lvl, "level", "")
		cmd.Flags().Var(&maxLvl, "max-level", "")
		cmd.Flags().Var(&name, "name", "") // the field is a string: String() is used
		cmd.Flags().Var(&ep, "endpoint", "")

		return cmd
	}

	t.Run("custom values are set with their type", func(t *testing.T) {
		cmd := newCmd()

		fl, err := flags.FromCommand(cmd, dict)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"level", "maxlevel", "name", "endpoint"}, slices.Collect(maps.Keys(fl.Flags())))

		cmd.SetArgs([]string{"--level", "warn", "--max-level", "info", "--name", "debug", "--endpoint", "db:5432"})
		require.NoError(t, cmd.Execute())

		var cfg Config
		require.NoError(t, flags.SetFlags(&cfg, fl))

		assert.Equal(t, level(2), cfg.Level)
		if assert.NotNil(t, cfg.MaxLevel) {
			assert.Equal(t, level(1), *cfg.MaxLevel)
		}
		assert.Equal(t, "debug", cfg.Name)
		assert.Equal(t, endpoint{Host: "db", Port: "5432"}, cfg.Endpoint)
	})

	t.Run("the config doesn't share the flag's variable", func(t *testing.T) {
		cmd := newCmd()

		fl, err := flags.FromCommand(cmd, dict)
		require.NoError(t, err)

		cmd.SetArgs([]string{"--max-level", "info"})
		require.NoError(t, cmd.Execute())

		var cfg Config
		require.NoError(t, flags.SetFlags(&cfg, fl))

		require.NoError(t, cmd.Flags().Set("max-level", "warn"))
		assert.Equal(t, level(1), *cfg.MaxLevel)
	})

	t.Run("BindCmdFlag knows the field type", func(t *testing.T) {
		cmd := newCmd()

		fl := flags.NewFlagList(dict)
		require.NoError(t, fl.BindCmdFlag(cmd, "Level", "level"))

		cmd.SetArgs([]string{"--level", "info"})
		require.NoError(t, cmd.Execute())

		var cfg Config
		require.NoError(t, flags.SetFlags(&cfg, fl))
		assert.Equal(t, level(1), cfg.Level)
	})

	t.Run("without an index String() is used", func(t *testing.T) {
		cmd := newCmd()

		fl := flags.NewFlagList(nil)
		require.NoError(t, fl.BindCmdFlag(cmd, "Level", "level"))

		cmd.SetArgs([]string{"--level", "info"})
		require.NoError(t, cmd.Execute())

		var cfg Config
		err := flags.SetFlags(&cfg, fl)
		assert.ErrorContains(t, err, "flag level")
	})
}

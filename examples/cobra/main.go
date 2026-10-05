// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package main

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zauberhaus/config"
	"github.com/zauberhaus/config/pkg/flags"
)

type MyConfig struct {
	Host  string `default:"localhost"`
	Port  int    `default:"3000"`
	Level Level  `default:"info"`
}

// Level is a log level. As a pflag.Value it can be used as a custom flag type
// (--level warn); as an encoding.TextUnmarshaler it is read by name from the
// config file, environment variables and the default tag.
type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
)

var levelNames = []string{"debug", "info", "warn", "error"}

func (l Level) String() string {
	if l < 0 || int(l) >= len(levelNames) {
		return fmt.Sprintf("level(%d)", int(l))
	}

	return levelNames[l]
}

func (l *Level) Set(s string) error {
	i := slices.Index(levelNames, strings.ToLower(s))
	if i < 0 {
		return fmt.Errorf("unknown level %q (one of %s)", s, strings.Join(levelNames, ", "))
	}

	*l = Level(i)

	return nil
}

func (l *Level) Type() string { return "level" }

func (l *Level) UnmarshalText(text []byte) error { return l.Set(string(text)) }

func (l Level) MarshalText() ([]byte, error) { return []byte(l.String()), nil }

func main() {
	if err := RootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cobra-example",
		Short: "A simple cobra example for the config package",
		RunE: func(cmd *cobra.Command, args []string) error {
			flagList := flags.NewFlagList(nil)
			err := flagList.BindCmdFlag(cmd, "Host", "host")
			if err != nil {
				return err
			}
			err = flagList.BindCmdFlag(cmd, "Port", "port")
			if err != nil {
				return err
			}
			err = flagList.BindCmdFlag(cmd, "Level", "level")
			if err != nil {
				return err
			}

			o := []config.Option{
				config.WithName("app"),
				config.WithFlags(flagList),
			}

			configFile := os.Getenv("CONFIG_FILE")

			if cmd.Flags().Changed("config") || len(configFile) == 0 {
				cfgFile, err := cmd.Flags().GetString("config")
				if err != nil {
					return err
				}

				if cfgFile != "" {
					configFile = cfgFile
				}
			}

			if len(configFile) > 0 {
				o = append(o, config.WithFile(configFile))
			}

			cfg, configFile, err := config.Load[*MyConfig](o...)
			if err != nil {
				return err
			}

			if configFile != "" {
				fmt.Printf("Configuration loaded from: %s\n", configFile)
			} else {
				fmt.Println("No configuration file loaded, using defaults and environment variables.")
			}

			fmt.Printf("Host: %s\n", cfg.Host)
			fmt.Printf("Port: %d\n", cfg.Port)
			fmt.Printf("Level: %s\n", cfg.Level)

			return nil
		},
	}

	cmd.Flags().StringP("host", "d", "", "host name")
	cmd.Flags().IntP("port", "p", 0, "port number")
	cmd.Flags().StringP("config", "c", "config.yaml", "configuration file")

	// a custom flag type; its initial value is only shown in --help,
	// the default of the config is set by the default tag
	level := Info
	cmd.Flags().VarP(&level, "level", "l", "log level (debug, info, warn, error)")

	return cmd
}

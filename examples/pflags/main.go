// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/pflag"
	"github.com/zauberhaus/config"
	"github.com/zauberhaus/config/pkg/flags"
)

type MyConfig struct {
	Host string `default:"localhost"`
	Port int    `default:"3000"`
}

func main() {
	pflag.String("host", "", "host name")
	pflag.Int("port", 0, "port number")
	pflag.StringP("config", "c", "config.yaml", "config file name")

	pflag.Parse()

	flagList := flags.NewFlagList(nil)
	err := flagList.BindFlag(pflag.CommandLine, "Host", pflag.Lookup("host"))
	if err != nil {
		log.Fatal(err)
	}
	err = flagList.BindFlag(pflag.CommandLine, "Port", pflag.Lookup("port"))
	if err != nil {
		log.Fatal(err)
	}

	o := []config.Option{
		config.WithName("app"),
		config.WithFlags(flagList),
	}

	configFile := os.Getenv("ADD_CONFIG")
	p := pflag.Lookup("config")

	if p.Changed || len(configFile) == 0 {
		cfgFile := p.Value.String()

		if cfgFile != "" {
			configFile = cfgFile
		}
	}

	if len(configFile) > 0 {
		o = append(o, config.WithFile(configFile))
	}

	cfg, configFile, err := config.Load[*MyConfig](o...)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	if configFile != "" {
		fmt.Printf("Configuration loaded from: %s\n", configFile)
	} else {
		fmt.Println("No configuration file loaded, using defaults and environment variables.")
	}

	fmt.Printf("Host: %s\n", cfg.Host)
	fmt.Printf("Port: %d\n", cfg.Port)
}

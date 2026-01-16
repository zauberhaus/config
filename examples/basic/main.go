// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/zauberhaus/config"
)

type MyConfig struct {
	Host string `default:"localhost"`
	Port int    `default:"3000"`
}

func main() {
	cfgFile := flag.String("c", "", "config file")
	flag.Parse()

	o := []config.Option{
		config.WithName("app"),
	}

	configFile := os.Getenv("ADD_CONFIG")
	if *cfgFile != "" {
		configFile = *cfgFile
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

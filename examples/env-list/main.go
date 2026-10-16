// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package main

import (
	"flag"
	"fmt"
	"log"
	"maps"
	"slices"

	"github.com/zauberhaus/config/pkg/env"
	"github.com/zauberhaus/config/pkg/index"
)

type Database struct {
	Host     string `default:"localhost"`
	Port     int    `default:"5432"`
	UserName string
}

type Server struct {
	ListenAddress string `env:"ADDR"`
	Port          int    `default:"8080"`
}

type Endpoint struct {
	Name string
	URL  string
}

type MyConfig struct {
	LogLevel  string `default:"info"`
	Server    Server
	Database  Database `env:"DB"`
	Common    Database `env:"-"`
	Endpoints []Endpoint
	Labels    map[string]string
	Internal  string `env:"--"`
}

func main() {
	values := flag.Bool("v", false, "show the current values of the environment variables")
	types := flag.Bool("t", false, "show the config path and type of each environment variable")
	flag.Parse()

	name := "app"

	if *types {
		idx, err := index.New[MyConfig](nil)
		if err != nil {
			log.Fatalf("failed to build index: %v", err)
		}

		fmt.Print(idx.String())
		return
	}

	vars, err := env.List(MyConfig{}, env.WithName(name))
	if err != nil {
		log.Fatalf("failed to list environment variables: %v", err)
	}

	for _, k := range slices.Sorted(maps.Keys(vars)) {
		if *values {
			fmt.Printf("%s=%s\n", k, vars[k])
		} else {
			fmt.Println(k)
		}
	}
}

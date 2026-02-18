// Package main is the entrypoint to the server binary.
package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"

	"github.com/urfave/cli/v2"

	"github.com/lzambarda/hbt/internal/config"
	"github.com/lzambarda/hbt/internal/graph/database"
	"github.com/lzambarda/hbt/internal/graph/naive"
	"github.com/lzambarda/hbt/internal/server"
)

//nolint:gochecknoglobals,err113 // Fine for now.
var (
	graph     server.Graph
	cachePath string
	root      = &cli.App{
		Name:        "hbt",
		Usage:       "a zsh suggestion system",
		Description: `Spawn a TCP server listening on the local port 43111 (can be changed with HBT_PORT).`,
		Version:     config.Version,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Aliases:     []string{"d"},
				DefaultText: "false",
				Destination: &config.Debug,
				EnvVars:     []string{config.DebugName},
			},
			&cli.StringFlag{
				Name:        "cache",
				Aliases:     []string{"c"},
				DefaultText: "location of cache",
				Value:       config.DefaultCachePath,
				Destination: &config.CachePath,
				EnvVars:     []string{config.CachePathName},
			},
			&cli.UintFlag{
				Name:        "port",
				Aliases:     []string{"p"},
				DefaultText: strconv.FormatUint(uint64(config.DefaultPort), 10),
				Value:       config.DefaultPort,
				Destination: &config.Port,
				EnvVars:     []string{config.PortName},
			},
			&cli.DurationFlag{
				Name:        "save-interval",
				Aliases:     []string{"s"},
				DefaultText: config.DefaultSaveInterval.String(),
				Value:       config.DefaultSaveInterval,
				Usage:       "if the driver is postgres, then this is unused",
				Destination: &config.SaveInterval,
				EnvVars:     []string{config.SaveIntervalName},
			},
			&cli.StringFlag{
				Name:        "driver",
				Aliases:     []string{"drv"},
				DefaultText: config.DefaultDriverName,
				Usage:       "what graph implementation to use",
				Value:       config.DefaultDriverName,
				Destination: &config.Driver,
				EnvVars:     []string{config.DriverName},
			},
		},
		Before: func(ctx *cli.Context) error {
			switch config.Driver {
			case config.DriverSQLite:
				var err error

				cachePath = path.Join(config.CachePath, "hbt.sqlite")

				graph, err = database.New(ctx.Context, cachePath)
				if err != nil {
					return fmt.Errorf("unable to create database: %w", err)
				}

				return nil
			case config.DriverNaive:
				cachePath = path.Join(config.CachePath, config.CacheName)

				graph = naive.NewGraph(10, 3) //nolint:mnd // It's arbitrary.

				return graph.Load(ctx.Context, cachePath)
			default:
				return fmt.Errorf("unknown driver %q", config.Driver)
			}
		},
		// By default start a server
		Action: func(ctx *cli.Context) error {
			return server.Start(ctx.Context, graph, cachePath)
		},
		Commands: []*cli.Command{
			{
				Name:    "cli",
				Aliases: []string{"c"},
				Usage:   "run a command without starting a server",
				Action: func(ctx *cli.Context) error {
					result, err := server.ProcessCommand(ctx.Context, ctx.Args().Slice(), graph)
					if err != nil {
						return fmt.Errorf("process command: %w", err)
					}

					log.Println(result)

					return nil
				},
			},
		},
	}
)

func main() {
	if err := root.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

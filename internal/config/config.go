// Package config contains all global configuration variables.
//
//nolint:gochecknoglobals,revive,stylecheck,godoclint // Fine for now.
package config

import (
	"time"
)

const (
	DebugName        = "HBT_DEBUG"
	CachePathName    = "HBT_CACHE_PATH"
	PortName         = "HBT_PORT"
	SaveIntervalName = "HBT_SAVE_INTERVAL"
	DriverName       = "HBT_DRIVER_NAME"
)

const (
	// It was found by looking at unused ports at:
	// https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers
	DefaultPort         uint = 43111
	DefaultCachePath         = "."
	DefaultSaveInterval      = time.Minute * 10
	DefaultDriverName        = DriverSQLite
)

var (
	// Debug is a flag to enable debug logging.
	Debug bool
	// CachePath is the path to the cache or database or other storage medium.
	CachePath string
	// Port is the port server will listen to.
	Port uint
	// SaveInterval is the interval at which to save the data used by the
	// server.
	SaveInterval time.Duration
	// Version must be var, otherwise -X flag can't modify it.
	Version = "unknown"
	// Driver is which implementation of the graph to use.
	Driver string
)

const (
	// CacheName is the name of the cache file.
	CacheName = ".hbtcache"
)

const (
	DriverUnknown string = "unknown"
	DriverNaive   string = "naive"
	DriverSQLite  string = "sqlite"
)

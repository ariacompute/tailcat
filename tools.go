//go:build tools
// +build tools

// Package tools pins modules used by aria-tailcat so that `go mod tidy` does not
// prune the tailcat library (github.com/tailscale/tailcat) and the dependencies only
// its CLI (not the library) requires. `aria-tailcat` imports the `tailcat` package and
// runs the server in-process; no separate tailcat binary is built or bundled.
//
// This file is never compiled into the aria-tailcat binary (build tag "tools").
// The explicit imports below keep `go mod tidy` from pruning dependencies that
// only the tailcat CLI (not the tailcat library) requires.
package tools

import (
	_ "github.com/peterbourgon/ff/v4"
	_ "github.com/tailscale/tailcat"
	_ "github.com/toqueteos/webbrowser"
)

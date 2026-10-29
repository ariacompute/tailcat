//go:build tools
// +build tools

// Package tools pins modules used by aria-tailcat's build pipeline so that
// `go build github.com/tailscale/tailcat/cmd/tailcat` (used by debian/rules and
// scripts to bundle the tailcat binary) resolves from this module's graph.
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

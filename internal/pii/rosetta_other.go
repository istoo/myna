//go:build !darwin

package pii

// translated says myna runs under Rosetta, which only macOS has.
var translated = func() bool { return false }

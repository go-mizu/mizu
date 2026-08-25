package main

import (
	"context"
	"io"
	"testing"
)

// Registering the commands happens on every run of the binary, before anything
// anybody asked for, so it is worth knowing what it costs.
func BenchmarkNewApp(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		newApp()
	}
}

// The whole of a run: build the app, take the global flags out of the command
// line, find the command, parse its flags and print. This is what stands
// between typing mizu version and reading the answer.
func BenchmarkVersion(b *testing.B) {
	ctx := context.Background()
	argv := []string{"version"}
	b.ReportAllocs()
	for b.Loop() {
		newApp().Start(ctx, nil, io.Discard, io.Discard, argv)
	}
}

func BenchmarkHelp(b *testing.B) {
	ctx := context.Background()
	argv := []string{"--help"}
	b.ReportAllocs()
	for b.Loop() {
		newApp().Start(ctx, nil, io.Discard, io.Discard, argv)
	}
}

// The report is a page of prose built for one reading, so what it costs does
// not matter much. It is here because it is the only thing in this package
// that builds a string rather than printing one, and a regression in it would
// otherwise go unnoticed.
func BenchmarkReport(b *testing.B) {
	t := withFloor(onTarget())
	b.ReportAllocs()
	for b.Loop() {
		report(t)
	}
}

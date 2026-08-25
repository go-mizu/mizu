// Package broken is what the generator's error tests replace with an overlay.
// What is written here matters only in that the go command has to see the
// package and everything it imports before a test can put something else in
// its place.
package broken

import (
	"context"
	"net/netip"
	"time"

	"github.com/go-mizu/mizu/console"
)

var (
	_ context.Context
	_ time.Time
	_ netip.Addr
	_ console.Spec
)

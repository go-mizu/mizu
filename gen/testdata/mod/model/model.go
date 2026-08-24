// Package model holds the types the loader tests read.
package model

import "time"

// A User is a person with an account.
//
//mizu:table users
type User struct {
	ID      int64
	Email   string
	Created time.Time
}

// Label is the name to show for a user.
func (u *User) Label() string { return u.Email }

// A Status is where an order has got to.
type Status int

const (
	Pending Status = iota
	Shipped
)

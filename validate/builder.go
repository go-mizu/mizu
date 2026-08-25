package validate

// A V is a list of checks written out in the order somebody wants them run.
//
// It is the mode for rules a struct tag cannot say: a rule that depends on
// another field, on a row in a database, or on which branch of a form was
// filled in. A struct that can be described in tags does not need one, because
// the generator writes a [Validator] method instead and it costs nothing at
// runtime.
//
// The zero V is not ready. Use [New].
type V struct {
	bad Errors
}

// New starts a list of checks.
func New() *V { return &V{} }

// Msgs sets the [Messages] every sentence after it is written with.
//
// It is on the builder rather than on each check because the locale is one
// decision, made at the top, and every line under it is the code it would be in
// a program with one language in it.
func (v *V) Msgs(m Messages) *V {
	v.bad.Msgs = m
	return v
}

// Field starts a chain of checks on one value.
//
// The name is what the request called it, publish_at or items.0.quantity, and
// it is what comes back on the error. The value is whatever the field holds: a
// chain reads the value's type to decide what a size rule counts.
//
//	v.Field("email", in.Email).Required().Email()
//
// A chain stops at its first failure, so Required guards everything after it
// and one field produces one sentence rather than a list of consequences.
func (v *V) Field(name string, value any) *Check {
	return &Check{v: v, name: name, value: value}
}

// When runs add only if cond is true.
//
// It is how a rule that depends on another field is written where the
// condition is, rather than as a tag on the field that names a field somewhere
// else:
//
//	v.When(in.Type == "company", func(v *V) {
//		v.Field("vat", in.VAT).Required()
//	})
//
// The condition is an ordinary Go expression, so anything that can be worked
// out from the value is allowed and nothing has to be spelled in a string.
func (v *V) When(cond bool, add func(*V)) *V {
	if cond {
		add(v)
	}
	return v
}

// Errors is what has failed so far.
//
// A caller reaching for this wants to add a failure the chain has no method
// for, or to ask whether an earlier field passed before doing expensive work
// for a later one.
func (v *V) Errors() *Errors { return &v.bad }

// Err is nil when nothing failed, and otherwise what [Errors.OrNil] returns.
func (v *V) Err() error { return v.bad.OrNil() }

// A Check is one field partway through its rules.
//
// It is returned by [V.Field] and every method on it returns it again, so the
// rules for a field are one statement. Once a rule fails, the rest of the chain
// does nothing.
type Check struct {
	v     *V
	name  string
	value any
	done  bool
}

// Required fails unless the value is filled in.
//
// Filled in means not the zero value, with an empty list counting as missing:
// "", 0, false, a nil pointer, an empty slice or map and the zero time.Time all
// fail. A field that has to tell 0 apart from absent says so in its type, with
// a pointer or an xs.Option, and that is a decision the binder made before the
// value arrived here.
func (c *Check) Required() *Check {
	if c.done {
		return c
	}
	if isEmpty(c.value) {
		return c.fail(Failed("required"))
	}
	return c
}

// Optional stops the chain when the value is missing, and does nothing when it
// is there.
//
// It is what a field that may be left blank puts in front of its format rules,
// since none of those pass on an empty string:
//
//	v.Field("website", in.Website).Optional().URL()
//
// Nothing is recorded either way. Optional is not a rule, it is the absence of
// Required written down so that the reader of the chain can see which one was
// meant.
func (c *Check) Optional() *Check {
	if !c.done && isEmpty(c.value) {
		c.done = true
	}
	return c
}

// Min fails when the value is smaller than n.
//
// What smaller means comes from the value: characters for a string, the number
// itself for a number, elements for a list or a map, and length for a duration.
// The bound keeps the type it was written with, so Min(3) says 3 and
// Min(time.Hour) says 1h0m0s.
func (c *Check) Min(n any) *Check {
	return c.size("min", func(have, want float64) bool { return have >= want }, n)
}

// Max fails when the value is larger than n, counting the same things [Check.Min]
// counts.
func (c *Check) Max(n any) *Check {
	return c.size("max", func(have, want float64) bool { return have <= want }, n)
}

// Size fails unless the value is exactly n, counting the same things
// [Check.Min] counts.
func (c *Check) Size(n any) *Check {
	return c.size("size", func(have, want float64) bool { return have == want }, n)
}

// Between fails unless the value is at least lo and at most hi, counting the
// same things [Check.Min] counts.
func (c *Check) Between(lo, hi any) *Check {
	if c.done {
		return c
	}
	subject, have := sizeOf(c.value)
	if have < number(lo) || have > number(hi) {
		return c.fail(Failed("between", lo, hi).Of(subject))
	}
	return c
}

// size is the shape all four size rules have: read what the value counts, and
// compare it with the bound.
func (c *Check) size(rule string, ok func(have, want float64) bool, n any) *Check {
	if c.done {
		return c
	}
	subject, have := sizeOf(c.value)
	if !ok(have, number(n)) {
		return c.fail(Failed(rule, n).Of(subject))
	}
	return c
}

// That fails with the named rule unless ok is true.
//
// It is the way in for anything this package has no method for, including a
// check that had to ask a database:
//
//	taken, err := users.EmailTaken(ctx, in.Email)
//	if err != nil {
//		return err
//	}
//	v.Field("email", in.Email).Required().That(!taken, "unique")
//
// The parameters are what the sentence fills in and what a client reads off
// the error, so they are the rule's configuration and not the value.
func (c *Check) That(ok bool, rule string, params ...any) *Check {
	if c.done || ok {
		return c
	}
	return c.fail(Failed(rule, params...))
}

// Email fails unless the value is an email address. See [IsEmail].
func (c *Check) Email() *Check { return c.format("email", IsEmail) }

// URL fails unless the value is an http or https URL. See [IsURL].
func (c *Check) URL() *Check { return c.format("url", IsURL) }

// URI fails unless the value is an absolute URI. See [IsURI].
func (c *Check) URI() *Check { return c.format("uri", IsURI) }

// Hostname fails unless the value is a host name. See [IsHostname].
func (c *Check) Hostname() *Check { return c.format("hostname", IsHostname) }

// IP fails unless the value is an IP address of either family. See [IsIP].
func (c *Check) IP() *Check { return c.format("ip", IsIP) }

// IPv4 fails unless the value is an IPv4 address. See [IsIPv4].
func (c *Check) IPv4() *Check { return c.format("ipv4", IsIPv4) }

// IPv6 fails unless the value is an IPv6 address. See [IsIPv6].
func (c *Check) IPv6() *Check { return c.format("ipv6", IsIPv6) }

// CIDR fails unless the value is an address and a prefix length. See [IsCIDR].
func (c *Check) CIDR() *Check { return c.format("cidr", IsCIDR) }

// MAC fails unless the value is a hardware address. See [IsMAC].
func (c *Check) MAC() *Check { return c.format("mac", IsMAC) }

// Port fails unless the value is a port number. See [IsPort].
func (c *Check) Port() *Check { return c.format("port", IsPort) }

// UUID fails unless the value is a UUID. See [IsUUID].
func (c *Check) UUID() *Check { return c.format("uuid", IsUUID) }

// ULID fails unless the value is a ULID. See [IsULID].
func (c *Check) ULID() *Check { return c.format("ulid", IsULID) }

// E164 fails unless the value is a phone number in international format. See
// [IsE164].
func (c *Check) E164() *Check { return c.format("e164", IsE164) }

// format runs one of the checks from format.go.
//
// It panics on a value that is not a string, because a format rule on an int is
// a mistake in the program. A nil pointer reads as the empty string, which
// fails, and pairing the rule with Optional is how a blank field passes.
func (c *Check) format(rule string, ok func(string) bool) *Check {
	if c.done {
		return c
	}
	s, isText := text(c.value)
	if !isText {
		panic("validate: the " + rule + " rule on a value that is not a string")
	}
	if !ok(s) {
		return c.fail(Failed(rule))
	}
	return c
}

// fail records one failure and stops the chain.
func (c *Check) fail(r RuleError) *Check {
	c.v.bad.Add(c.name, r)
	c.done = true
	return c
}

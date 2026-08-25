package validate

import (
	"fmt"
	"strings"
	"unicode"
)

// Messages writes the sentence shown next to a field when a rule fails.
//
// It is one method because that is all a translation needs to be. The field
// name arrives as the request spells it, publish_at or items.0.quantity, and
// what to call it in a sentence is part of the job: a locale that says
// "Erscheinungsdatum" cannot be handed "Publish at" and asked to fix it.
type Messages interface {
	Message(field string, r RuleError) string
}

// English is the built-in [Messages], and what an [Errors] uses when its Msgs
// is nil.
//
// It is a value rather than a function so that a program can hold a Messages
// and not care which one it has.
var English Messages = english{}

type english struct{}

// Message writes the sentence for one failure.
//
// A rule the table has never heard of gets a sentence that names the field and
// says it is not valid, rather than a blank or the rule name shown to somebody
// who does not know what it means. A custom rule with no entry here is a
// missing translation and not a bug in the request.
func (english) Message(field string, r RuleError) string {
	label := Label(field)

	t, ok := templates[r.key()]
	if !ok || len(r.Params) != t.params {
		return label + " is not valid."
	}

	args := make([]any, 0, len(r.Params)+1)
	args = append(args, label)
	args = append(args, r.Params...)
	return fmt.Sprintf(t.text, args...)
}

// Label is what a field is called in a sentence.
//
// Names arrive the way the request spells them, so title is a word already,
// publish_at is two, and items.0.quantity is a path. Underscores and dots
// become spaces and the first letter is capitalised, which is what a form
// label would have said.
//
// It is exported because a [Messages] of somebody's own wants the same
// treatment for the fields its table has no entry for.
func Label(field string) string {
	if field == "" {
		return "This field"
	}

	spaced := strings.Map(func(r rune) rune {
		if r == '_' || r == '.' {
			return ' '
		}
		return r
	}, field)

	first := []rune(spaced)
	first[0] = unicode.ToUpper(first[0])
	return string(first)
}

// A template is one sentence and how many parameters it fills in.
//
// The count is written down rather than counted out of the verbs so that a
// rule handing over the wrong number of parameters produces a plain sentence
// instead of %!v(MISSING) in front of a user. TestTemplateParamsMatchTheVerbs
// keeps the two honest.
type template struct {
	params int
	text   string
}

// templates is the English table, keyed the way [RuleError.key] spells it: the
// rule on its own, and the rule with a subject after it for a rule whose
// sentence depends on what is being counted.
//
// The first verb is always the field's label. The rest are the rule's
// parameters in the order it recorded them.
var templates = map[string]template{
	"required": {0, "%s is required."},

	"min.string":   {1, "%s must be at least %v characters."},
	"min.numeric":  {1, "%s must be at least %v."},
	"min.array":    {1, "%s must have at least %v items."},
	"min.file":     {1, "%s must be at least %v kilobytes."},
	"min.duration": {1, "%s must be at least %v."},

	"max.string":   {1, "%s must not be longer than %v characters."},
	"max.numeric":  {1, "%s must not be greater than %v."},
	"max.array":    {1, "%s must not have more than %v items."},
	"max.file":     {1, "%s must not be larger than %v kilobytes."},
	"max.duration": {1, "%s must not be longer than %v."},

	"between.string":   {2, "%s must be between %v and %v characters."},
	"between.numeric":  {2, "%s must be between %v and %v."},
	"between.array":    {2, "%s must have between %v and %v items."},
	"between.file":     {2, "%s must be between %v and %v kilobytes."},
	"between.duration": {2, "%s must be between %v and %v."},

	"size.string":   {1, "%s must be %v characters."},
	"size.numeric":  {1, "%s must be %v."},
	"size.array":    {1, "%s must have %v items."},
	"size.file":     {1, "%s must be %v kilobytes."},
	"size.duration": {1, "%s must be %v."},

	"cidr":     {0, "%s must be a network in CIDR notation, such as 10.0.0.0/8."},
	"e164":     {0, "%s must be a phone number in international format, starting with a country code."},
	"email":    {0, "%s must be an email address."},
	"hostname": {0, "%s must be a host name."},
	"ip":       {0, "%s must be an IP address."},
	"ipv4":     {0, "%s must be an IPv4 address."},
	"ipv6":     {0, "%s must be an IPv6 address."},
	"mac":      {0, "%s must be a hardware address, such as 00:1b:44:11:3a:b7."},
	"port":     {0, "%s must be a port number between 1 and 65535."},
	"ulid":     {0, "%s must be a ULID."},
	"uri":      {0, "%s must be a URI, including the scheme."},
	"url":      {0, "%s must be a URL, starting with http:// or https://."},
	"uuid":     {0, "%s must be a UUID."},
}

package micro

// signup is the struct the reflective interpreter and the generated validator
// are both measured on: 12 fields and 20 rules, which is a form somebody would
// actually post.
//
// The rules are the ones a public form carries. Four of them are format checks,
// because a form that takes an address and a link is the common case and those
// are the checks that cost the most.
type signup struct {
	Email string   `json:"email" validate:"required,email"`
	Title string   `json:"title" validate:"required,min=3,max=120"`
	Slug  string   `json:"slug" validate:"required"`
	Site  string   `json:"site" validate:"omitempty,url"`
	IP    string   `json:"ip" validate:"omitempty,ip"`
	ID    string   `json:"id" validate:"required,ulid"`
	Ref   string   `json:"ref" validate:"uuid"`
	Body  string   `json:"body" validate:"required,min=10"`
	Tags  []string `json:"tags" validate:"required,max=5"`
	Count int      `json:"count" validate:"between=1 10"`
	Phone string   `json:"phone" validate:"e164"`
	Host  string   `json:"host" validate:"hostname"`
}

// genSignup is the same struct with a generated validator on it.
//
// A defined type has the fields and the tags of the type it is over and none of
// its methods, so validate/gen and validate/reflect are checking the same
// twelve fields against the same twenty rules and the only difference between
// them is which validator ran. Writing the rules out twice would let the two
// drift, and a pair of numbers that are not measuring the same work is worse
// than no numbers.
//
// This file is not a test file, because a generator reads the package rather
// than the tests in it. validate_gen.go next to it is the output, checked in,
// and TestGeneratedValidatorIsCurrent is what keeps it that way.
//
//mizu:validate
type genSignup signup

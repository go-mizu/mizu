package postgrest

import (
	"strconv"
	"strings"
)

// Preferences represents parsed Prefer header values.
type Preferences struct {
	Return      ReturnPreference
	Count       CountPreference
	Resolution  ResolutionPreference
	Missing     MissingPreference
	MaxAffected *int
	Handling    HandlingPreference
	Timezone    string
	Transaction TxPreference
}

// ReturnPreference specifies what to return after INSERT/UPDATE/DELETE.
type ReturnPreference int

const (
	ReturnMinimal ReturnPreference = iota
	ReturnRepresentation
	ReturnHeadersOnly
)

// CountPreference specifies how to count rows.
type CountPreference int

const (
	CountNone CountPreference = iota
	CountExact
	CountPlanned
	CountEstimated
)

// ResolutionPreference specifies upsert conflict resolution.
type ResolutionPreference int

const (
	ResolutionNone ResolutionPreference = iota
	ResolutionMergeDuplicates
	ResolutionIgnoreDuplicates
)

// MissingPreference specifies handling of missing columns.
type MissingPreference int

const (
	MissingDefault MissingPreference = iota
	MissingNull
)

// HandlingPreference specifies strict vs lenient mode.
type HandlingPreference int

const (
	HandlingLenient HandlingPreference = iota
	HandlingStrict
)

// TxPreference specifies transaction handling.
type TxPreference int

const (
	TxCommit TxPreference = iota
	TxRollback
)

// ParsePrefer parses the Prefer header value.
func ParsePrefer(prefer string) *Preferences {
	prefs := &Preferences{}

	if prefer == "" {
		return prefs
	}

	// Split by comma
	parts := strings.Split(prefer, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by = for key=value pairs
		kv := strings.SplitN(part, "=", 2)
		key := strings.TrimSpace(kv[0])
		var value string
		if len(kv) == 2 {
			value = strings.TrimSpace(kv[1])
		}

		switch key {
		case "return":
			switch value {
			case "minimal":
				prefs.Return = ReturnMinimal
			case "representation":
				prefs.Return = ReturnRepresentation
			case "headers-only":
				prefs.Return = ReturnHeadersOnly
			}

		case "count":
			switch value {
			case "exact":
				prefs.Count = CountExact
			case "planned":
				prefs.Count = CountPlanned
			case "estimated":
				prefs.Count = CountEstimated
			}

		case "resolution":
			switch value {
			case "merge-duplicates":
				prefs.Resolution = ResolutionMergeDuplicates
			case "ignore-duplicates":
				prefs.Resolution = ResolutionIgnoreDuplicates
			}

		case "missing":
			switch value {
			case "default":
				prefs.Missing = MissingDefault
			case "null":
				prefs.Missing = MissingNull
			}

		case "max-affected":
			if v, err := strconv.Atoi(value); err == nil {
				prefs.MaxAffected = &v
			}

		case "handling":
			switch value {
			case "strict":
				prefs.Handling = HandlingStrict
			case "lenient":
				prefs.Handling = HandlingLenient
			}

		case "timezone":
			prefs.Timezone = value

		case "tx":
			switch value {
			case "commit":
				prefs.Transaction = TxCommit
			case "rollback":
				prefs.Transaction = TxRollback
			}
		}
	}

	return prefs
}

// PreferenceApplied returns a header value for applied preferences.
func (p *Preferences) PreferenceApplied() string {
	var parts []string

	if p.Return == ReturnRepresentation {
		parts = append(parts, "return=representation")
	} else if p.Return == ReturnHeadersOnly {
		parts = append(parts, "return=headers-only")
	}

	if p.Count == CountExact {
		parts = append(parts, "count=exact")
	} else if p.Count == CountPlanned {
		parts = append(parts, "count=planned")
	} else if p.Count == CountEstimated {
		parts = append(parts, "count=estimated")
	}

	if p.Resolution == ResolutionMergeDuplicates {
		parts = append(parts, "resolution=merge-duplicates")
	} else if p.Resolution == ResolutionIgnoreDuplicates {
		parts = append(parts, "resolution=ignore-duplicates")
	}

	return strings.Join(parts, ", ")
}

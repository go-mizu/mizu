package contract

// Common helper functions shared across transport implementations.

// jsonIsNull checks if a byte slice represents JSON null.
func jsonIsNull(b []byte) bool {
	b = jsonTrimSpace(b)
	return len(b) == 4 &&
		(b[0] == 'n' || b[0] == 'N') &&
		(b[1] == 'u' || b[1] == 'U') &&
		(b[2] == 'l' || b[2] == 'L') &&
		(b[3] == 'l' || b[3] == 'L')
}

// jsonTrimSpace is a small, allocation-free trim for JSON whitespace.
func jsonTrimSpace(b []byte) []byte {
	i := 0
	j := len(b)

	// Trim left
	for i < j {
		c := b[i]
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			i++
		} else {
			break
		}
	}

	// Trim right
	for j > i {
		c := b[j-1]
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			j--
		} else {
			break
		}
	}

	return b[i:j]
}

// jsonSafeErr returns a safe error message for JSON responses.
func jsonSafeErr(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}

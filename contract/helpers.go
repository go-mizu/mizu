package contract

// Common helper functions shared across transport implementations.
// These delegate to the transport package functions for consistency.

// jsonIsNull checks if a byte slice represents JSON null.
// Deprecated: Use IsJSONNull instead.
func jsonIsNull(b []byte) bool {
	return IsJSONNull(b)
}

// jsonTrimSpace is a small, allocation-free trim for JSON whitespace.
// Deprecated: Use TrimJSONSpace instead.
func jsonTrimSpace(b []byte) []byte {
	return TrimJSONSpace(b)
}

// jsonSafeErr returns a safe error message for JSON responses.
// Deprecated: Use SafeErrorString instead.
func jsonSafeErr(err error) any {
	if err == nil {
		return nil
	}
	return SafeErrorString(err)
}

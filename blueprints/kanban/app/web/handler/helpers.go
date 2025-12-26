package handler

// errResponse creates an error response map.
func errResponse(msg string) map[string]string {
	return map[string]string{"error": msg}
}

package validator

// ValidationError represents a single validation issue.
type ValidationError struct {
	Resource string
	Field    string
	Severity string // "error" | "warning" | "info"
	Message  string
}

// Validate checks generated manifests for common issues.
// TODO: implement in Prompt 9
func Validate() []ValidationError {
	return nil
}

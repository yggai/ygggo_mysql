package ygggo_mysql

// ErrorClass is a placeholder for error classification.
type ErrorClass int

const (
	ErrClassUnknown ErrorClass = iota
	ErrClassRetryable
	ErrClassConflict
	ErrClassReadonly
	ErrClassConstraint
)

// Classify classifies error (placeholder).
func Classify(err error) ErrorClass { return ErrClassUnknown }


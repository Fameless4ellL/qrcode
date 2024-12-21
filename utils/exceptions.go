package utils

// DataOverflowError represents an error when data exceeds the allowed limit.
type DataOverflowError struct {
	message string
}

// NewDataOverflowError creates a new DataOverflowError with the given message.
func NewDataOverflowError(message string) *DataOverflowError {
	return &DataOverflowError{message: message}
}
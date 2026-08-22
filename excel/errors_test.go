package excel

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExportError_Error(t *testing.T) {
	err := NewExportError("TestOp", "Sheet1", 10, 5, errors.New("test error"))
	errMsg := err.Error()
	assert.Contains(t, errMsg, "TestOp")
	assert.Contains(t, errMsg, "Sheet1")
	assert.Contains(t, errMsg, "row")
	assert.Contains(t, errMsg, "col")
	assert.Contains(t, errMsg, "test error")
}

func TestExportError_Unwrap(t *testing.T) {
	original := errors.New("original error")
	err := NewExportError("TestOp", "Sheet1", 10, 5, original)
	assert.Equal(t, original, errors.Unwrap(err))
}

func TestImportError_Error(t *testing.T) {
	err := NewImportError("TestOp", "Sheet1", 10, 5, errors.New("test error"))
	errMsg := err.Error()
	assert.Contains(t, errMsg, "TestOp")
	assert.Contains(t, errMsg, "Sheet1")
	assert.Contains(t, errMsg, "row")
	assert.Contains(t, errMsg, "col")
	assert.Contains(t, errMsg, "test error")
}

func TestImportError_Unwrap(t *testing.T) {
	original := errors.New("original error")
	err := NewImportError("TestOp", "Sheet1", 10, 5, original)
	assert.Equal(t, original, errors.Unwrap(err))
}

func TestValidationError_Error(t *testing.T) {
	err := NewValidationError(5, "age", -1, "must be positive", nil)
	assert.Contains(t, err.Error(), "validation")
	assert.Contains(t, err.Error(), "age")
	assert.Contains(t, err.Error(), "must be positive")
	assert.Contains(t, err.Error(), "row 5")
}

func TestValidationError_Unwrap(t *testing.T) {
	err := NewValidationError(5, "age", -1, "must be positive", nil)
	assert.Nil(t, errors.Unwrap(err))
}

func TestValidationError_UnwrapCause(t *testing.T) {
	cause := errors.New("underlying conversion error")
	err := NewValidationError(5, "age", "abc", "must be positive", cause)
	assert.ErrorIs(t, err, cause)
}

func TestConfigError_Error(t *testing.T) {
	err := NewConfigError("timeout", 0, "must be greater than zero")
	assert.Contains(t, err.Error(), "config error")
	assert.Contains(t, err.Error(), "timeout")
	assert.Contains(t, err.Error(), "must be greater than zero")
}

func TestConfigError_Unwrap(t *testing.T) {
	err := NewConfigError("timeout", 0, "must be greater than zero")
	// ConfigError doesn't wrap an error, so Unwrap should return nil
	assert.Nil(t, errors.Unwrap(err))
}

func TestPredefinedErrors(t *testing.T) {
	assert.Equal(t, "empty data provided", ErrEmptyData.Error())
	assert.Equal(t, "invalid file format", ErrInvalidFormat.Error())
	assert.Equal(t, "file not found", ErrFileNotFound.Error())
	assert.Equal(t, "sheet not found", ErrSheetNotFound.Error())
	assert.Equal(t, "invalid sheet name", ErrInvalidSheetName.Error())
	assert.Equal(t, "invalid file", ErrInvalidFile.Error())
	assert.Equal(t, "type mismatch", ErrTypeMismatch.Error())
}

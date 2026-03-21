package domain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdjun99/lazy-dbx/internal/domain"
)

func TestResult_Success(t *testing.T) {
	r := domain.Result[string]{Data: "hello", Error: nil}
	assert.Equal(t, "hello", r.Data)
	assert.NoError(t, r.Error)
}

func TestResult_Error(t *testing.T) {
	err := errors.New("something failed")
	r := domain.Result[string]{Error: err}
	assert.Equal(t, "", r.Data)
	assert.EqualError(t, r.Error, "something failed")
}

func TestResult_Int(t *testing.T) {
	r := domain.Result[int]{Data: 42}
	assert.Equal(t, 42, r.Data)
	assert.NoError(t, r.Error)
}

package errors

import (
	"fmt"

	"github.com/go-errors/errors"
)

// Err intelligently creates/handles errors, while preserving the stack trace.
// It works with errors from github.com/pkg/errors too.
func Err(err interface{}, fmtParams ...interface{}) error {
	if err == nil {
		return nil
	}

	type causer interface {
		Cause() error
	}

	if _, ok := err.(causer); ok {
		err = fmt.Errorf("%+v", err)
	} else if errString, ok := err.(string); ok && len(fmtParams) > 0 {
		err = fmt.Errorf(errString, fmtParams...)
	}

	return errors.Wrap(err, 1)
}

// Is compares two wrapped errors to determine if the underlying errors are the same
func Is(e error, original error) bool {
	return errors.Is(e, original)
}

// Prefix prefixes the message of the error with the given string
func Prefix(prefix string, err interface{}) error {
	if err == nil {
		return nil
	}
	return errors.WrapPrefix(Err(err), prefix, 0)
}

// Trace returns the stack trace
func Trace(err error) string {
	if err == nil {
		return ""
	}
	return string(errors.Wrap(Err(err), 0).Stack())
}


// FullTrace returns the error type, message, and stack trace
func FullTrace(err error) string {
	if err == nil {
		return ""
	}
	return errors.Wrap(Err(err), 0).ErrorStack()
}

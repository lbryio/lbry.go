package errors

import (
	base "errors"
	"fmt"

	"github.com/go-errors/errors"
)

// interop with pkg/errors
type causer interface {
	Cause() error
}

// Err intelligently creates/handles errors, while preserving the stack trace.
// It works with errors from github.com/pkg/errors too.
func Err(err interface{}, fmtParams ...interface{}) error {
	if err == nil {
		return nil
	}

	if _, ok := err.(causer); ok {
		err = fmt.Errorf("%+v", err)
	} else if errString, ok := err.(string); ok && len(fmtParams) > 0 {
		err = fmt.Errorf(errString, fmtParams...)
	}

	return errors.Wrap(err, 1)
}

// Wrap calls errors.Wrap, in case you want to skip a different amount
func Wrap(err interface{}, skip int) *errors.Error {
	if err == nil {
		return nil
	}

	if _, ok := err.(causer); ok {
		err = fmt.Errorf("%+v", err)
	}

	return errors.Wrap(err, skip+1)
}

// Is compares two wrapped errors to determine if the underlying errors are the same
// It also interops with errors from pkg/errors
func Is(e error, original error) bool {
	if c, ok := e.(causer); ok {
		e = c.Cause()
	}
	if c, ok := original.(causer); ok {
		original = c.Cause()
	}
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
	return string(Err(err).(*errors.Error).Stack())
}

// FullTrace returns the error type, message, and stack trace
func FullTrace(err error) string {
	if err == nil {
		return ""
	}
	return Err(err).(*errors.Error).ErrorStack()
}

// Base returns a simple error with no stack trace attached
func Base(text string) error {
	return base.New(text)
}

// HasTrace checks if error has a trace attached
func HasTrace(err error) bool {
	_, ok := err.(*errors.Error)
	return ok
}

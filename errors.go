package goblstripe

import (
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/cbc"
)

// Error contains the standard error definition for this domain.
type Error struct {
	key     cbc.Key
	fields  gobl.FieldErrors
	message string
	code    string
}

var (
	ErrRounding   = NewError("rounding")
	ErrValidation = NewError("validation")
)

// NewError instantiates a new error.
func NewError(key cbc.Key) *Error {
	return &Error{key: key}
}

func (e *Error) copy() *Error {
	ne := new(Error)
	*ne = *e
	return ne
}

// WithCause attaches an error instance to the Error if it is serializable,
// otherwise the message is updated.
func (e *Error) WithCause(cause error) *Error {
	ne := e.copy()
	switch err := cause.(type) {
	case *Error:
		return err
	case *gobl.Error:
		// we only know about fields from GOBL at the moment
		ne.fields = err.Fields()
		ne.message = err.Message()
		if ne.message == "" {
			ne.message = err.Key().String()
		}
	default:
		ne.message = cause.Error()
	}
	return ne
}

// WithMsg adds a message to the Error.
func (e *Error) WithMsg(message string, args ...any) *Error {
	ne := e.copy()
	if len(args) > 0 {
		ne.message = fmt.Sprintf(message, args...)
	} else {
		ne.message = message
	}
	return ne
}

// WithCode adds a code to the Error.
func (e *Error) WithCode(code string) *Error {
	ne := e.copy()
	ne.code = code
	return ne
}

// Error provides the string representation of the error meant for log output
func (e *Error) Error() string {
	msg := e.message
	if e.fields != nil {
		fe := e.fields.Error()
		if msg != "" {
			msg = fmt.Sprintf("%s (%s)", msg, fe)
		} else {
			msg = fe
		}
	}
	return fmt.Sprintf("%s: %s", e.key, msg)
}

// Is checks to see if the target error matches the current error or
// part of the chain.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.key == t.key
}

// In returns true if the error matches one of those provided.
func (e *Error) In(targets ...error) bool {
	for _, te := range targets {
		if e.Is(te) {
			return true
		}
	}
	return false
}

// Key provides the key value for the error.
func (e *Error) Key() cbc.Key {
	return e.key
}

// Fields returns the errors associated with the fields, if present.
func (e *Error) Fields() gobl.FieldErrors {
	return e.fields
}

// Message provides the error's message, if set.
func (e *Error) Message() string {
	return e.message
}

// Code provides the error's code, if set.
func (e *Error) Code() string {
	return e.code
}

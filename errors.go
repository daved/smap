package smap

import (
	"errors"
	"fmt"
)

// Sentinel errors for API consumers to detect via errors.Is.
var (
	ErrDstInvalid             = errors.New("invalid dst: non-nil struct ptr required")
	ErrSrcInvalid             = errors.New("invalid src: struct or non-nil ptr required")
	ErrTagInvalid             = errors.New("invalid path in tag")
	ErrFieldTypesIncompatible = errors.New("source field type is incompatible with destination field type")
	ErrTagEmpty               = errors.New("empty smap tag")
	ErrTagPathUnresolvable    = errors.New("unresolvable tag path")
)

// ErrorMergeField is a complex error type for mergeField failures.
type ErrorMergeField struct {
	child       error  // Unexported underlying error
	TagValue    string // Relevant tag or path portion
	DstTypeName string // Destination type name
	SrcTypeName string // Source type name
}

// NewMergeFieldError constructs an ErrorMergeField with the given details.
func NewMergeFieldError(child error, tagValue, dstTypeName, srcTypeName string) *ErrorMergeField {
	return &ErrorMergeField{
		child:       child,
		TagValue:    tagValue,
		DstTypeName: dstTypeName,
		SrcTypeName: srcTypeName,
	}
}

// Error implements the error interface.
func (e *ErrorMergeField) Error() string {
	return fmt.Sprintf("merge field error: dst type: %s, src type: %s, tag: %q: %v",
		e.DstTypeName, e.SrcTypeName, e.TagValue, e.child)
}

// Unwrap returns the underlying error for errors.Is checks.
func (e *ErrorMergeField) Unwrap() error {
	return e.child
}

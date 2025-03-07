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
	ErrTagPathNotFound        = errors.New("tag path field not found")
	ErrTagPathEmpty           = errors.New("tag path is empty")
	ErrTagPathInvalidKeyType  = errors.New("tag path references map with non-string key type")
	// errKeepLooking is unexported for internal control flow
	errKeepLooking = errors.New("keep looking for next path")
)

// MergeFieldError is a complex error type for mergeField failures.
type MergeFieldError struct {
	child       error  // Unexported underlying error
	TagValue    string // Relevant tag or path portion
	DstTypeName string // Destination type name
	SrcTypeName string // Source type name
}

// NewMergeFieldError constructs a MergeFieldError with the given details.
func NewMergeFieldError(child error, tagValue, dstTypeName, srcTypeName string) *MergeFieldError {
	return &MergeFieldError{
		child:       child,
		TagValue:    tagValue,
		DstTypeName: dstTypeName,
		SrcTypeName: srcTypeName,
	}
}

// Error implements the error interface.
func (e *MergeFieldError) Error() string {
	return fmt.Sprintf("merge field (tag: %q, dst type: %s, src type: %s): %v",
		e.TagValue, e.DstTypeName, e.SrcTypeName, e.child)
}

// Unwrap returns the underlying error for errors.Is checks.
func (e *MergeFieldError) Unwrap() error {
	return e.child
}

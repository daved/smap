// Package smap provides functionality to merge struct fields based on struct tags.
package smap

import (
	"errors"
	"reflect"
	"strings"
)

// TagKey is the struct tag key used to define source paths.
const TagKey = "smap"

// Errors for API consumers to detect via errors.Is.
var (
	ErrDstInvalid             = errors.New("invalid dst: non-nil struct ptr required")
	ErrSrcInvalid             = errors.New("invalid src: struct or non-nil ptr required")
	ErrTagInvalid             = errors.New("invalid path in tag")
	ErrFieldTypesIncompatible = errors.New("source field type is incompatible with destination field type")
	ErrTagEmpty               = errors.New("empty smap tag")
)

// Merge merges values from src into dst based on dst's smap struct tags.
func Merge(dst, src interface{}) error {
	dstVal, err := makeDstValue(dst)
	if err != nil {
		return err
	}

	srcVal, err := makeSrcValue(src)
	if err != nil {
		return err
	}

	return mergeFields(dstVal, srcVal)
}

// makeDstValue ensures dst is a non-nil pointer to a struct and returns its value.
func makeDstValue(dst interface{}) (reflect.Value, error) {
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr || dstVal.IsNil() {
		return reflect.Value{}, ErrDstInvalid
	}
	dstVal = dstVal.Elem()
	if dstVal.Kind() != reflect.Struct {
		return reflect.Value{}, ErrDstInvalid
	}
	return dstVal, nil
}

// makeSrcValue ensures src is a struct or non-nil pointer to a struct and returns its value.
func makeSrcValue(src interface{}) (reflect.Value, error) {
	srcVal := reflect.ValueOf(src)
	if srcVal.Kind() == reflect.Ptr {
		if srcVal.IsNil() {
			return reflect.Value{}, ErrSrcInvalid
		}
		srcVal = srcVal.Elem()
	}
	if srcVal.Kind() != reflect.Struct {
		return reflect.Value{}, ErrSrcInvalid
	}
	return srcVal, nil
}

// mergeFields applies the smap tag mappings from srcVal to dstVal.
func mergeFields(dstVal, srcVal reflect.Value) error {
	dstType := dstVal.Type()
	for i := 0; i < dstType.NumField(); i++ {
		field := dstType.Field(i)
		smapTag, ok := field.Tag.Lookup(TagKey)
		if !ok {
			continue
		}
		tagPathsParts, err := makeTagPathsParts(smapTag)
		if err != nil {
			return err
		}
		if err := mergeField(dstVal.Field(i), srcVal, tagPathsParts); err != nil {
			return err
		}
	}
	return nil
}

// mergeField sets dstField based on the smap tag paths in srcVal.
func mergeField(dstField, srcVal reflect.Value, tagPathsParts [][]string) error {
	if len(tagPathsParts) == 0 {
		return ErrTagEmpty
	}

	var finalValue reflect.Value
	for _, pathParts := range tagPathsParts {
		value := lookupField(srcVal, pathParts)
		if value.IsValid() {
			finalValue = value
		}
	}
	if !finalValue.IsValid() {
		return ErrTagInvalid
	}
	if !finalValue.Type().AssignableTo(dstField.Type()) {
		return ErrFieldTypesIncompatible
	}
	dstField.Set(finalValue)
	return nil
}

// makeTagPathsParts splits a smap tag into a slice of path segments, erroring on malformed tags.
func makeTagPathsParts(tag string) ([][]string, error) {
	paths := strings.Split(tag, "|")
	var tagPathsParts [][]string
	for _, path := range paths {
		if path == "" {
			continue
		}
		parts := strings.Split(path, ".")
		for _, part := range parts {
			if part == "" {
				return nil, ErrTagInvalid // Empty segment (e.g., "Foo..Bar")
			}
		}
		tagPathsParts = append(tagPathsParts, parts)
	}
	if len(tagPathsParts) == 0 {
		return nil, ErrTagEmpty // Tag is empty or only empty segments (e.g., "", "|")
	}
	return tagPathsParts, nil
}

// lookupField navigates srcVal using the path parts and returns the value.
func lookupField(srcVal reflect.Value, parts []string) reflect.Value {
	current := srcVal
	for _, part := range parts {
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return reflect.Value{} // Skipping nil pointer errors for now
			}
			current = current.Elem()
		}
		if current.Kind() != reflect.Struct {
			return reflect.Value{}
		}
		current = current.FieldByName(part)
		if !current.IsValid() {
			return reflect.Value{}
		}
	}
	if current.Kind() == reflect.Ptr && !current.IsNil() {
		return current.Elem()
	}
	return current
}

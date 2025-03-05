// Package smap provides functionality to merge struct fields based on struct tags.
package smap

import (
	"errors"
	"reflect"
	"strings"
)

// Errors for API consumers to detect via errors.Is.
var (
	ErrInvalidDst             = errors.New("invalid dst: non-nil struct ptr required")
	ErrInvalidSrc             = errors.New("invalid src: struct or non-nil ptr required")
	ErrInvalidPath            = errors.New("invalid path in tag")
	ErrFieldTypesIncompatible = errors.New("source field type is incompatible with destination field type")
)

// Merge merges values from src into dst based on dst's smap struct tags.
func Merge(dst, src interface{}) error {
	dstVal, err := validateDst(dst)
	if err != nil {
		return err
	}

	srcVal, err := validateSrc(src)
	if err != nil {
		return err
	}

	return mergeFields(dstVal, srcVal)
}

// validateDst ensures dst is a non-nil pointer to a struct.
func validateDst(dst interface{}) (reflect.Value, error) {
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr || dstVal.IsNil() {
		return reflect.Value{}, ErrInvalidDst
	}
	dstVal = dstVal.Elem()
	if dstVal.Kind() != reflect.Struct {
		return reflect.Value{}, ErrInvalidDst
	}
	return dstVal, nil
}

// validateSrc ensures src is a struct or non-nil pointer to a struct.
func validateSrc(src interface{}) (reflect.Value, error) {
	srcVal := reflect.ValueOf(src)
	if srcVal.Kind() == reflect.Ptr {
		if srcVal.IsNil() {
			return reflect.Value{}, ErrInvalidSrc
		}
		srcVal = srcVal.Elem()
	}
	if srcVal.Kind() != reflect.Struct {
		return reflect.Value{}, ErrInvalidSrc
	}
	return srcVal, nil
}

// mergeFields applies the smap tag mappings from srcVal to dstVal.
func mergeFields(dstVal, srcVal reflect.Value) error {
	dstType := dstVal.Type()
	for i := 0; i < dstType.NumField(); i++ {
		field := dstType.Field(i)
		smapTag := field.Tag.Get("smap")
		if smapTag == "" {
			continue
		}
		if err := mergeField(dstVal.Field(i), srcVal, smapTag); err != nil {
			return err
		}
	}
	return nil
}

// mergeField sets dstField based on the smap tag paths in srcVal.
func mergeField(dstField, srcVal reflect.Value, smapTag string) error {
	srcPaths := strings.Split(smapTag, "|")
	var finalValue reflect.Value
	for _, path := range srcPaths {
		if path == "" {
			continue
		}
		value := lookupField(srcVal, strings.Split(path, "."))
		if value.IsValid() {
			finalValue = value
		}
	}
	if !finalValue.IsValid() {
		return ErrInvalidPath
	}
	if !finalValue.Type().AssignableTo(dstField.Type()) {
		return ErrFieldTypesIncompatible
	}
	dstField.Set(finalValue)
	return nil
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

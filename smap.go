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
	ErrInvalidDst             = errors.New("invalid dst: non-nil struct ptr required")
	ErrInvalidSrc             = errors.New("invalid src: struct or non-nil ptr required")
	ErrInvalidPath            = errors.New("invalid path in tag")
	ErrFieldTypesIncompatible = errors.New("source field type is incompatible with destination field type")
	ErrEmptyTag               = errors.New("empty smap tag")
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
		return reflect.Value{}, ErrInvalidDst
	}
	dstVal = dstVal.Elem()
	if dstVal.Kind() != reflect.Struct {
		return reflect.Value{}, ErrInvalidDst
	}
	return dstVal, nil
}

// makeSrcValue ensures src is a struct or non-nil pointer to a struct and returns its value.
func makeSrcValue(src interface{}) (reflect.Value, error) {
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
		smapTag, ok := field.Tag.Lookup(TagKey)
		if !ok {
			continue
		}
		srcPaths := makeSrcPaths(smapTag)
		if err := mergeField(dstVal.Field(i), srcVal, srcPaths); err != nil {
			return err
		}
	}
	return nil
}

// mergeField sets dstField based on the smap tag paths in srcVal.
func mergeField(dstField, srcVal reflect.Value, srcPaths [][]string) error {
	if len(srcPaths) == 0 {
		return ErrEmptyTag
	}

	var finalValue reflect.Value
	for _, pathParts := range srcPaths {
		value := lookupField(srcVal, pathParts)
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

// makeSrcPaths splits a smap tag into a slice of path segments.
func makeSrcPaths(tag string) [][]string {
	paths := strings.Split(tag, "|")
	var srcPaths [][]string
	for _, path := range paths {
		if path == "" {
			continue
		}
		srcPaths = append(srcPaths, strings.Split(path, "."))
	}
	return srcPaths
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

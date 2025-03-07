// Package smap provides functionality to merge struct fields based on struct tags.
package smap

import (
	"reflect"

	"github.com/daved/vtypes"
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
		rawTag, ok := field.Tag.Lookup(TagKey)
		if !ok {
			continue
		}
		tag, err := newSTag(rawTag)
		if err != nil {
			return err
		}
		if err := mergeField(dstVal.Field(i), srcVal, tag); err != nil {
			return err
		}
	}
	return nil
}

// mergeField sets dstField based on the smap tag paths in srcVal.
func mergeField(dstField, srcVal reflect.Value, tag *sTag) error {
	if len(tag.pathsParts) == 0 {
		return NewMergeFieldError(ErrTagEmpty, "", dstField.Type().String(), "")
	}

	var finalValue reflect.Value
	for _, pathParts := range tag.pathsParts {
		value, err := lookUpField(srcVal, pathParts)
		if err != nil {
			return NewMergeFieldError(err, pathParts.String(), dstField.Type().String(), "")
		}
		if value.IsValid() {
			finalValue = value
		}
	}
	if !finalValue.IsValid() {
		return NewMergeFieldError(ErrTagInvalid, tag.String(), dstField.Type().String(), "")
	}

	// Handle hydration if requested and source is a string
	if tag.HasHydrate() && finalValue.Kind() == reflect.String {
		// Create a new value of the destination type to hydrate into
		hydratedPtr := reflect.New(dstField.Type())
		hydrated := hydratedPtr.Interface()
		if err := vtypes.Hydrate(hydrated, finalValue.String()); err != nil {
			return NewMergeFieldError(err, tag.String(), dstField.Type().String(), finalValue.Type().String())
		}
		finalValue = hydratedPtr.Elem()
	}

	if !finalValue.Type().AssignableTo(dstField.Type()) {
		return NewMergeFieldError(ErrFieldTypesIncompatible, tag.String(), dstField.Type().String(), finalValue.Type().String())
	}
	dstField.Set(finalValue)
	return nil
}

// lookUpField navigates srcVal using the path parts and returns the value.
func lookUpField(srcVal reflect.Value, pathParts tagPathParts) (reflect.Value, error) {
	current := srcVal
	for _, part := range pathParts {
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return reflect.Value{}, ErrTagPathUnresolvable
			}
			current = current.Elem()
		}
		if current.Kind() != reflect.Struct {
			return reflect.Value{}, ErrTagPathUnresolvable
		}
		current = current.FieldByName(part)
		if !current.IsValid() {
			return reflect.Value{}, ErrTagPathUnresolvable
		}
	}
	if current.Kind() == reflect.Ptr && !current.IsNil() {
		return current.Elem(), nil
	}
	return current, nil
}

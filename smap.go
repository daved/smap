// Package smap provides functionality to merge struct fields based on struct tags.
package smap

import (
	"errors"
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
			if errors.Is(err, errKeepLooking) {
				continue // Try next path
			}
			return NewMergeFieldError(err, pathParts.String(), dstField.Type().String(), "")
		}
		if value.IsValid() {
			if tag.HasSkipZero() && value.IsZero() {
				continue // Skip zero values if skipzero is set
			}
			finalValue = value // Keep the last valid non-zero value
		}
	}

	// If no valid value found, leave pointer fields unset
	if !finalValue.IsValid() {
		if dstField.Kind() == reflect.Ptr {
			return nil // Leave nil for unset pointers
		}
		return nil // Non-pointers stay zero
	}

	// Handle hydration if requested and source is a string
	if tag.HasHydrate() && finalValue.Kind() == reflect.String {
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
	if len(pathParts) == 0 {
		return reflect.Value{}, ErrTagPathEmpty
	}

	current := srcVal
	for i, part := range pathParts {
		value := current
		if value.Kind() == reflect.Ptr && value.IsNil() {
			return reflect.Value{}, errKeepLooking // Unset, try next path
		}
		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}

		isLastPart := i == len(pathParts)-1
		switch value.Kind() {
		case reflect.Struct:
			// Try field first
			field := value.FieldByName(part)
			typ := value.Type()
			if f, ok := typ.FieldByName(part); ok && field.IsValid() && f.PkgPath == "" { // Check if exported
				current = field
				if isLastPart {
					for current.Kind() == reflect.Ptr && !current.IsNil() {
						current = current.Elem()
					}
					return current, nil
				}
				continue
			}
			// Try method on original (possibly pointer) value
			method := current.MethodByName(part)
			if method.IsValid() && method.Type().NumIn() == 0 {
				results := method.Call(nil)
				switch len(results) {
				case 1:
					return results[0], nil
				case 2:
					if err, ok := results[1].Interface().(error); ok {
						if err != nil {
							return reflect.Value{}, err // Propagate method error
						}
						return results[0], nil
					}
				}
			}
			// No exported field or method found
			if isLastPart {
				return reflect.Value{}, ErrTagPathNotFound
			}
			return reflect.Value{}, errKeepLooking

		case reflect.Map:
			if value.Type().Key().Kind() != reflect.String {
				return reflect.Value{}, ErrTagPathInvalidKeyType
			}
			key := reflect.ValueOf(part)
			field := value.MapIndex(key)
			if !field.IsValid() {
				return reflect.Value{}, errKeepLooking // Unset, try next path
			}
			current = field
			if isLastPart {
				for current.Kind() == reflect.Ptr && !current.IsNil() {
					current = current.Elem()
				}
				return current, nil
			}

		default:
			return reflect.Value{}, errKeepLooking // Non-struct/map, try next path
		}
	}

	// Should not reach here with valid pathParts
	return reflect.Value{}, ErrTagPathNotFound
}

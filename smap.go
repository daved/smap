// Package smap provides functionality to merge struct fields based on struct tags.
package smap

import (
	"errors"
	"reflect"
	"strconv"

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
	if tag.IsEmpty() {
		return NewMergeFieldError(ErrTagEmpty, "", dstField.Type().String(), "")
	}

	finalValue, err := findLeafValueByPathsParts(srcVal, tag)
	if err != nil {
		return NewMergeFieldError(err, tag.String(), dstField.Type().String(), "")
	}

	if !finalValue.IsValid() {
		if dstField.Kind() == reflect.Ptr {
			return nil // Leave nil for unset pointers
		}
		return nil // Non-pointers stay zero
	}

	if tag.HasHydrate() && finalValue.Kind() == reflect.String {
		hydratedValue, err := hydratedElement(dstField.Type(), finalValue.String())
		if err != nil {
			return NewMergeFieldError(err, tag.String(), dstField.Type().String(), finalValue.Type().String())
		}
		finalValue = hydratedValue
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
							return reflect.Value{}, err
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
			keyType := value.Type().Key()
			var key reflect.Value
			// Try converting part to the map's key type
			switch keyType.Kind() {
			case reflect.String:
				key = reflect.ValueOf(part)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if n, err := strconv.ParseInt(part, 10, 64); err == nil {
					key = reflect.ValueOf(n).Convert(keyType)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				if n, err := strconv.ParseUint(part, 10, 64); err == nil {
					key = reflect.ValueOf(n).Convert(keyType)
				}
			case reflect.Float32, reflect.Float64:
				if f, err := strconv.ParseFloat(part, 64); err == nil {
					key = reflect.ValueOf(f).Convert(keyType)
				}
			default:
				return reflect.Value{}, ErrTagPathInvalidKeyType
			}
			if !key.IsValid() {
				return reflect.Value{}, ErrTagPathInvalidKeyType
			}
			field := value.MapIndex(key)
			if !field.IsValid() {
				return reflect.Value{}, errKeepLooking
			}
			current = field
			if isLastPart {
				for current.Kind() == reflect.Ptr && !current.IsNil() {
					current = current.Elem()
				}
				return current, nil
			}

		case reflect.Slice, reflect.Array:
			if idx, err := strconv.Atoi(part); err == nil && idx >= 0 && idx < value.Len() {
				current = value.Index(idx)
				if isLastPart {
					for current.Kind() == reflect.Ptr && !current.IsNil() {
						current = current.Elem()
					}
					return current, nil
				}
				continue
			}
			return reflect.Value{}, errKeepLooking

		default:
			return reflect.Value{}, errKeepLooking
		}
	}

	return reflect.Value{}, ErrTagPathNotFound
}

// findLeafValueByPathsParts finds the last valid, non-zero leaf value from the given paths.
func findLeafValueByPathsParts(srcVal reflect.Value, tag *sTag) (reflect.Value, error) {
	var finalValue reflect.Value
	for _, pathParts := range tag.pathsParts {
		value, err := lookUpField(srcVal, pathParts)
		if err != nil {
			if errors.Is(err, errKeepLooking) {
				continue
			}
			return reflect.Value{}, err
		}
		if value.IsValid() {
			if tag.HasSkipZero() && value.IsZero() {
				continue
			}
			finalValue = value
		}
	}
	return finalValue, nil
}

// hydratedElement hydrates a string value into the destination type.
func hydratedElement(dstType reflect.Type, srcString string) (reflect.Value, error) {
	hydratedPtr := reflect.New(dstType)
	hydrated := hydratedPtr.Interface()
	if err := vtypes.Hydrate(hydrated, srcString); err != nil {
		return reflect.Value{}, err
	}
	return hydratedPtr.Elem(), nil
}

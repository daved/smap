package smap

import (
	"fmt"
	"reflect"
	"strings"
)

// Merge merges values from src into dst based on dst's smap struct tags.
func Merge(dst, src interface{}) error {
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr || dstVal.IsNil() {
		return fmt.Errorf("dst must be a non-nil pointer")
	}
	dstVal = dstVal.Elem()
	if dstVal.Kind() != reflect.Struct {
		return fmt.Errorf("dst must point to a struct")
	}

	srcVal := reflect.ValueOf(src)
	// Handle src as either a struct or a pointer to a struct
	if srcVal.Kind() == reflect.Ptr {
		if srcVal.IsNil() {
			return fmt.Errorf("src must not be nil")
		}
		srcVal = srcVal.Elem()
	}
	if srcVal.Kind() != reflect.Struct {
		return fmt.Errorf("src must be a struct or pointer to a struct")
	}

	dstType := dstVal.Type()
	for i := 0; i < dstType.NumField(); i++ {
		field := dstType.Field(i)
		smapTag := field.Tag.Get("smap")
		if smapTag == "" {
			continue // Skip fields without smap tag
		}

		// Split the smap tag by "|" for precedence
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

		if finalValue.IsValid() && finalValue.Type().AssignableTo(field.Type) {
			dstVal.Field(i).Set(finalValue)
		}
	}

	return nil
}

// lookupField navigates srcVal using the path parts and returns the value.
func lookupField(srcVal reflect.Value, parts []string) reflect.Value {
	current := srcVal
	for _, part := range parts {
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return reflect.Value{}
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

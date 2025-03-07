package smap

import (
	"errors"
	"reflect"
	"testing"
)

func TestUnitNewSTag(t *testing.T) {
	tests := []struct {
		name    string
		rawTag  string
		want    *sTag
		wantErr error
	}{
		{
			name:   "single path",
			rawTag: "EV.AISvcURL",
			want: &sTag{
				pathsParts: tagPathsParts{{"EV", "AISvcURL"}},
				opts:       nil,
			},
			wantErr: nil,
		},
		{
			name:   "multiple paths",
			rawTag: "EV.AISvcURL|FV.Service.URL",
			want: &sTag{
				pathsParts: tagPathsParts{{"EV", "AISvcURL"}, {"FV", "Service", "URL"}},
				opts:       nil,
			},
			wantErr: nil,
		},
		{
			name:    "empty tag",
			rawTag:  "",
			want:    nil,
			wantErr: ErrTagEmpty,
		},
		{
			name:    "only separators",
			rawTag:  "|",
			want:    nil,
			wantErr: ErrTagEmpty,
		},
		{
			name:    "multiple empty segments",
			rawTag:  "||",
			want:    nil,
			wantErr: ErrTagEmpty,
		},
		{
			name:   "empty segment in middle",
			rawTag: "EV.AISvcURL||FV.Service.URL",
			want: &sTag{
				pathsParts: tagPathsParts{{"EV", "AISvcURL"}, {"FV", "Service", "URL"}},
				opts:       nil,
			},
			wantErr: nil,
		},
		{
			name:    "double dot",
			rawTag:  "Foo..Bar",
			want:    nil,
			wantErr: ErrTagInvalid,
		},
		{
			name:    "trailing dot",
			rawTag:  "Foo.Bar.",
			want:    nil,
			wantErr: ErrTagInvalid,
		},
		{
			name:    "leading dot",
			rawTag:  ".Foo.Bar",
			want:    nil,
			wantErr: ErrTagInvalid,
		},
		{
			name:    "mixed valid and invalid",
			rawTag:  "EV.AISvcURL|Foo..Bar",
			want:    nil,
			wantErr: ErrTagInvalid,
		},
		{
			name:   "path with hydrate option",
			rawTag: "EV.AISvcURL,hydrate",
			want: &sTag{
				pathsParts: tagPathsParts{{"EV", "AISvcURL"}},
				opts:       []string{"hydrate"},
			},
			wantErr: nil,
		},
		{
			name:   "path with skipzero option",
			rawTag: "EV.Value|FV.Value,skipzero",
			want: &sTag{
				pathsParts: tagPathsParts{{"EV", "Value"}, {"FV", "Value"}},
				opts:       []string{"skipzero"},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newSTag(tt.rawTag)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("newSTag() error = %v, want %v", err, tt.wantErr)
				}
				if got != nil {
					t.Errorf("newSTag() got = %v, want nil on error", got)
				}
				return
			}
			if err != nil {
				t.Errorf("newSTag() error = %v, want nil", err)
				return
			}
			if !reflect.DeepEqual(got.pathsParts, tt.want.pathsParts) {
				t.Errorf("newSTag().pathsParts = %v, want %v", got.pathsParts, tt.want.pathsParts)
			}
			if !reflect.DeepEqual(got.opts, tt.want.opts) {
				t.Errorf("newSTag().opts = %v, want %v", got.opts, tt.want.opts)
			}
			expectedStr := tt.want.String()
			if got.String() != expectedStr {
				t.Errorf("newSTag().String() = %q, want %q", got.String(), expectedStr)
			}
		})
	}
}

// Define MethodStruct with methods for testing
type MethodStruct struct {
	Value string
}

func (ms *MethodStruct) GetValue() string {
	return ""
}

func (ms *MethodStruct) GetValueErr() (string, error) {
	return "", errors.New("method error")
}

func TestUnitLookUpField(t *testing.T) {
	type Inner struct {
		url string // unexported
		URL string // exported
	}
	type Outer struct {
		Inner    Inner
		Ptr      *Inner
		InnerPtr *Inner
		IntMap   map[int]string
		FloatMap map[float64]int
		Users    []string
		BoolMap  map[bool]string // Added for unsupported key type
	}
	type MapOuter struct {
		Data map[string]string
	}

	tests := []struct {
		name      string
		src       interface{}
		pathParts tagPathParts
		want      interface{}
		wantErr   error
	}{
		// Existing tests unchanged until "non-string map key"
		{
			name:      "valid struct path",
			src:       Outer{Inner: Inner{URL: "http://example.com"}},
			pathParts: tagPathParts{"Inner", "URL"},
			want:      "http://example.com",
			wantErr:   nil,
		},
		{
			name:      "valid pointer path",
			src:       Outer{Ptr: &Inner{URL: "http://example.com"}},
			pathParts: tagPathParts{"Ptr", "URL"},
			want:      "http://example.com",
			wantErr:   nil,
		},
		{
			name:      "nil pointer mid-path",
			src:       Outer{InnerPtr: nil},
			pathParts: tagPathParts{"InnerPtr", "URL"},
			want:      nil,
			wantErr:   errKeepLooking,
		},
		{
			name:      "non-struct mid-path",
			src:       struct{ Str string }{Str: "not a struct"},
			pathParts: tagPathParts{"Str", "URL"},
			want:      nil,
			wantErr:   errKeepLooking,
		},
		{
			name:      "missing field",
			src:       Outer{Inner: Inner{URL: "http://example.com"}},
			pathParts: tagPathParts{"Inner", "Missing"},
			want:      nil,
			wantErr:   ErrTagPathNotFound,
		},
		{
			name:      "unexported field",
			src:       Outer{Inner: Inner{url: "hidden"}},
			pathParts: tagPathParts{"Inner", "url"},
			want:      nil,
			wantErr:   ErrTagPathNotFound,
		},
		{
			name:      "empty path",
			src:       Outer{Inner: Inner{URL: "http://example.com"}},
			pathParts: tagPathParts{},
			want:      nil,
			wantErr:   ErrTagPathEmpty,
		},
		{
			name:      "valid map path",
			src:       MapOuter{Data: map[string]string{"key": "value"}},
			pathParts: tagPathParts{"Data", "key"},
			want:      "value",
			wantErr:   nil,
		},
		{
			name:      "missing map key",
			src:       MapOuter{Data: map[string]string{"other": "value"}},
			pathParts: tagPathParts{"Data", "key"},
			want:      nil,
			wantErr:   errKeepLooking,
		},
		{
			name:      "int map key", // Renamed from "non-string map key"
			src:       Outer{IntMap: map[int]string{1: "value"}},
			pathParts: tagPathParts{"IntMap", "1"},
			want:      "value",
			wantErr:   nil, // Updated expectation
		},
		{
			name:      "method one value",
			src:       &MethodStruct{Value: "struct value"},
			pathParts: tagPathParts{"GetValue"},
			want:      "",
			wantErr:   nil,
		},
		{
			name:      "method value and error",
			src:       &MethodStruct{Value: "struct value"},
			pathParts: tagPathParts{"GetValueErr"},
			want:      nil,
			wantErr:   errors.New("method error"),
		},
		{
			name:      "float map key",
			src:       Outer{FloatMap: map[float64]int{1.5: 42}},
			pathParts: tagPathParts{"FloatMap", "1.5"},
			want:      42,
			wantErr:   nil,
		},
		{
			name:      "slice index",
			src:       Outer{Users: []string{"zero", "one", "two"}},
			pathParts: tagPathParts{"Users", "1"},
			want:      "one",
			wantErr:   nil,
		},
		{
			name:      "slice out of bounds",
			src:       Outer{Users: []string{"zero", "one"}},
			pathParts: tagPathParts{"Users", "2"},
			want:      nil,
			wantErr:   errKeepLooking,
		},
		{
			name:      "unsupported map key type",
			src:       Outer{BoolMap: map[bool]string{true: "yes"}},
			pathParts: tagPathParts{"BoolMap", "true"},
			want:      nil,
			wantErr:   ErrTagPathInvalidKeyType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcVal := reflect.ValueOf(tt.src)
			got, err := lookUpField(srcVal, tt.pathParts)
			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Errorf("lookUpField() error = %v, want %v", err, tt.wantErr)
				}
				if got.IsValid() && tt.want != nil {
					t.Errorf("lookUpField() got = %v, want invalid value on error", got)
				}
				return
			}
			if err != nil {
				t.Errorf("lookUpField() error = %v, want nil", err)
				return
			}
			if tt.want == nil {
				if got.IsValid() {
					t.Errorf("lookUpField() got = %v, want nil", got)
				}
			} else {
				if !got.IsValid() {
					t.Errorf("lookUpField() got invalid value, want %v", tt.want)
					return
				}
				gotVal := got.Interface()
				if !reflect.DeepEqual(gotVal, tt.want) {
					t.Errorf("lookUpField() = %v, want %v", gotVal, tt.want)
				}
			}
		})
	}
}

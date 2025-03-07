package smap

import (
	"errors"
	"reflect"
	"testing"
)

func TestUnitMakeTagPathsParts(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		want    tagPathsParts
		wantErr error
	}{
		{
			name:    "single path",
			tag:     "EV.AISvcURL",
			want:    tagPathsParts{{"EV", "AISvcURL"}},
			wantErr: nil,
		},
		{
			name:    "multiple paths",
			tag:     "EV.AISvcURL|FV.Service.URL",
			want:    tagPathsParts{{"EV", "AISvcURL"}, {"FV", "Service", "URL"}},
			wantErr: nil,
		},
		{
			name:    "empty tag",
			tag:     "",
			want:    nil,
			wantErr: ErrTagEmpty,
		},
		{
			name:    "only separators",
			tag:     "|",
			want:    nil,
			wantErr: ErrTagEmpty,
		},
		{
			name:    "multiple empty segments",
			tag:     "||",
			want:    nil,
			wantErr: ErrTagEmpty,
		},
		{
			name:    "empty segment in middle",
			tag:     "EV.AISvcURL||FV.Service.URL",
			want:    tagPathsParts{{"EV", "AISvcURL"}, {"FV", "Service", "URL"}},
			wantErr: nil,
		},
		{
			name:    "double dot",
			tag:     "Foo..Bar",
			want:    nil,
			wantErr: ErrTagInvalid,
		},
		{
			name:    "trailing dot",
			tag:     "Foo.Bar.",
			want:    nil,
			wantErr: ErrTagInvalid,
		},
		{
			name:    "leading dot",
			tag:     ".Foo.Bar",
			want:    nil,
			wantErr: ErrTagInvalid,
		},
		{
			name:    "mixed valid and invalid",
			tag:     "EV.AISvcURL|Foo..Bar",
			want:    nil,
			wantErr: ErrTagInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makeTagPathsParts(tt.tag)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("makeTagPathsParts() error = %v, want %v", err, tt.wantErr)
				}
				if got != nil {
					t.Errorf("makeTagPathsParts() got = %v, want nil on error", got)
				}
				return
			}
			if err != nil {
				t.Errorf("makeTagPathsParts() error = %v, want nil", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeTagPathsParts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnitLookUpField(t *testing.T) {
	type Inner struct {
		URL string
	}
	type Outer struct {
		Inner Inner
		Ptr   *Inner
	}

	tests := []struct {
		name      string
		src       interface{}
		pathParts tagPathParts
		want      interface{}
		wantErr   error
	}{
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
			src:       Outer{Ptr: nil},
			pathParts: tagPathParts{"Ptr", "URL"},
			want:      nil,
			wantErr:   ErrTagPathUnresolvable,
		},
		{
			name:      "non-struct mid-path",
			src:       struct{ Str string }{Str: "not a struct"},
			pathParts: tagPathParts{"Str", "URL"},
			want:      nil,
			wantErr:   ErrTagPathUnresolvable,
		},
		{
			name:      "missing field",
			src:       Outer{Inner: Inner{URL: "http://example.com"}},
			pathParts: tagPathParts{"Inner", "Missing"},
			want:      nil,
			wantErr:   ErrTagPathUnresolvable,
		},
		{
			name:      "empty path",
			src:       Outer{Inner: Inner{URL: "http://example.com"}},
			pathParts: tagPathParts{},
			want:      Outer{Inner: Inner{URL: "http://example.com"}},
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcVal := reflect.ValueOf(tt.src)
			got, err := lookUpField(srcVal, tt.pathParts)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("lookUpField() error = %v, want %v", err, tt.wantErr)
				}
				if got.IsValid() {
					t.Errorf("lookUpField() got = %v, want invalid value on error", got)
				}
				return
			}
			if err != nil {
				t.Errorf("lookUpField() error = %v, want nil", err)
				return
			}
			if !got.IsValid() {
				t.Errorf("lookUpField() got invalid value, want %v", tt.want)
				return
			}
			gotVal := got.Interface()
			if !reflect.DeepEqual(gotVal, tt.want) {
				t.Errorf("lookUpField() = %v, want %v", gotVal, tt.want)
			}
		})
	}
}

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

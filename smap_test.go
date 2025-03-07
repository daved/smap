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

package smap_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/daved/smap"
)

type Config struct {
	AISvcURL string `smap:"EV.AISvcURL|FV.Service.URL"`
	AISvcKey string `smap:"EV.AISvcKey"`
	Extra    string `smap:"FV.Extra"`
	NoTag    string
}

type ConfigMismatch struct {
	AISvcURL string `smap:"EV.AISvcURL"`
	AISvcKey string `smap:"EV.AISvcKey"`
	Extra    int    `smap:"FV.Extra"` // int vs. string for type mismatch
}

type ConfigEmptyTag struct {
	Empty string `smap:""` // Empty tag to trigger ErrTagEmpty
}

type Sources struct {
	EV *EnvVars
	FV *FileVals
}

type EnvVars struct {
	AISvcURL string
	AISvcKey string
}

type FileVals struct {
	Service struct{ URL string }
	Extra   string
}

func TestSurfaceMerge(t *testing.T) {
	tests := []struct {
		name    string
		dst     interface{}
		src     interface{}
		want    interface{}
		wantErr error
	}{
		{
			name: "struct instance source",
			dst:  &Config{},
			src: Sources{
				EV: &EnvVars{AISvcURL: "env-url", AISvcKey: "env-key"},
				FV: &FileVals{Service: struct{ URL string }{URL: "file-url"}, Extra: "file-extra"},
			},
			want: Config{
				AISvcURL: "file-url",
				AISvcKey: "env-key",
				Extra:    "file-extra",
				NoTag:    "",
			},
			wantErr: nil,
		},
		{
			name: "pointer to struct source",
			dst:  &Config{},
			src: &Sources{
				EV: &EnvVars{AISvcURL: "env-url", AISvcKey: "env-key"},
				FV: &FileVals{Service: struct{ URL string }{URL: "file-url"}, Extra: "file-extra"},
			},
			want: Config{
				AISvcURL: "file-url",
				AISvcKey: "env-key",
				Extra:    "file-extra",
				NoTag:    "",
			},
			wantErr: nil,
		},
		{
			name:    "nil pointer source",
			dst:     &Config{},
			src:     (*Sources)(nil),
			want:    Config{},
			wantErr: smap.ErrSrcInvalid,
		},
		{
			name: "invalid path",
			dst:  &Config{},
			src: Sources{
				EV: &EnvVars{AISvcKey: "env-key"},
			},
			want:    Config{},
			wantErr: smap.ErrTagInvalid,
		},
		{
			name: "incompatible types",
			dst:  &ConfigMismatch{},
			src: Sources{
				EV: &EnvVars{AISvcURL: "env-url", AISvcKey: "env-key"},
				FV: &FileVals{Extra: "file-extra"},
			},
			want:    ConfigMismatch{},
			wantErr: smap.ErrFieldTypesIncompatible,
		},
		{
			name:    "empty tag",
			dst:     &ConfigEmptyTag{},
			src:     Sources{},
			want:    ConfigEmptyTag{},
			wantErr: smap.ErrTagEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := smap.Merge(tt.dst, tt.src)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Merge() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("Merge() error = %v, want nil", err)
				return
			}
			if !reflect.DeepEqual(reflect.ValueOf(tt.dst).Elem().Interface(), tt.want) {
				t.Errorf("Merge() dst = %+v, want %+v", reflect.ValueOf(tt.dst).Elem().Interface(), tt.want)
			}
		})
	}
}

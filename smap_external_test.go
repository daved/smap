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
	Extra    int    `smap:"FV.Extra"`
}

type ConfigEmptyTag struct {
	Empty string `smap:""`
}

type ConfigNilPath struct {
	NilPath string `smap:"EV.Nil.URL"`
}

type ConfigHydrate struct {
	Count int `smap:"EV.Count,hydrate"`
}

type ConfigPointer struct {
	URL *string `smap:"EV.URL|FV.URL"`
}

type ConfigMap struct {
	Value string `smap:"EV.Data.key"`
}

type ConfigMethod struct {
	Value string `smap:"EV.GetValue"`
}

type ConfigSkipZero struct {
	Count int `smap:"EV.Count|FV.Count,skipzero"`
}

type Sources struct {
	EV *EnvVars
	FV *FileVals
}

type EnvVars struct {
	AISvcURL string
	AISvcKey string
	Nil      *struct{ URL string }
	Count    int
	URL      *string
	Data     map[string]string
	Value    string
	IntMap   map[int]string  // Added
	FloatMap map[float64]int // Added
	Users    []string        // Added
}

type FileVals struct {
	Service struct{ URL string }
	Extra   string
	URL     *string
	Count   int
}

func (e *EnvVars) GetValue() string {
	return "method value"
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
			want:    Config{AISvcKey: "env-key"},
			wantErr: nil,
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
		{
			name: "nil path",
			dst:  &ConfigNilPath{},
			src: Sources{
				EV: &EnvVars{Nil: nil},
			},
			want:    ConfigNilPath{},
			wantErr: nil,
		},
		{
			name: "hydrate string to int",
			dst:  &ConfigHydrate{},
			src: Sources{
				EV: &EnvVars{Count: 42},
			},
			want:    ConfigHydrate{Count: 42},
			wantErr: nil,
		},
		{
			name: "unset pointer field",
			dst:  &ConfigPointer{},
			src: Sources{
				EV: &EnvVars{URL: nil},
				FV: &FileVals{URL: nil},
			},
			want:    ConfigPointer{URL: nil},
			wantErr: nil,
		},
		{
			name: "valid map value",
			dst:  &ConfigMap{},
			src: Sources{
				EV: &EnvVars{Data: map[string]string{"key": "value"}},
			},
			want:    ConfigMap{Value: "value"},
			wantErr: nil,
		},
		{
			name: "missing map key",
			dst:  &ConfigMap{},
			src: Sources{
				EV: &EnvVars{Data: map[string]string{"other": "value"}},
			},
			want:    ConfigMap{},
			wantErr: nil,
		},
		{
			name: "method value",
			dst:  &ConfigMethod{},
			src: Sources{
				EV: &EnvVars{Value: "struct value"},
			},
			want:    ConfigMethod{Value: "method value"},
			wantErr: nil,
		},
		{
			name: "skipzero with zero values",
			dst:  &ConfigSkipZero{},
			src: Sources{
				EV: &EnvVars{Count: 0},
				FV: &FileVals{Count: 0},
			},
			want:    ConfigSkipZero{Count: 0},
			wantErr: nil,
		},
		{
			name: "skipzero with non-zero value",
			dst:  &ConfigSkipZero{},
			src: Sources{
				EV: &EnvVars{Count: 0},
				FV: &FileVals{Count: 42},
			},
			want:    ConfigSkipZero{Count: 42},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := smap.Merge(tt.dst, tt.src)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Merge() error = nil, want %v", tt.wantErr)
					return
				}
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

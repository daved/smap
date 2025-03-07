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
	Extra    string // Remove invalid tag "FV.Extra"
	NoTag    string
}

type ConfigMismatch struct {
	AISvcURL string `smap:"EV.AISvcURL"`
	AISvcKey string `smap:"EV.AISvcKey"`
	Extra    int    // Remove invalid tag "FV.Extra"
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
	URL *string `smap:"EV.URL|FV.Service.URL"` // Fix tag
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

type ConfigDefault struct {
	Field string `smap:"EV.Value|FV.Service.URL"`
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
	IntMap   map[int]string
	FloatMap map[float64]int
	Users    []string
}

type FileVals struct {
	Service FileValsService
	Count   int // Add for skipzero tests
}

type FileValsService struct {
	URL *string `yaml:",omitempty"`
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
			name: "struct_instance_source",
			dst:  &Config{},
			src: Sources{
				EV: &EnvVars{AISvcURL: "env-url", AISvcKey: "env-key"},
				FV: &FileVals{Service: FileValsService{URL: strPtr("file-url")}},
			},
			want: Config{
				AISvcURL: "file-url",
				AISvcKey: "env-key",
				Extra:    "",
				NoTag:    "",
			},
			wantErr: nil,
		},
		{
			name: "pointer_to_struct_source",
			dst:  &Config{},
			src: &Sources{
				EV: &EnvVars{AISvcURL: "env-url", AISvcKey: "env-key"},
				FV: &FileVals{Service: FileValsService{URL: strPtr("file-url")}},
			},
			want: Config{
				AISvcURL: "file-url",
				AISvcKey: "env-key",
				Extra:    "",
				NoTag:    "",
			},
			wantErr: nil,
		},
		{
			name:    "nil_pointer_source",
			dst:     &Config{},
			src:     (*Sources)(nil),
			want:    Config{},
			wantErr: smap.ErrSrcInvalid,
		},
		{
			name: "invalid_path",
			dst:  &Config{},
			src: Sources{
				EV: &EnvVars{AISvcKey: "env-key"},
			},
			want:    Config{AISvcKey: "env-key"},
			wantErr: nil,
		},
		{
			name: "incompatible_types",
			dst:  &ConfigMismatch{},
			src: Sources{
				EV: &EnvVars{AISvcURL: "env-url", AISvcKey: "env-key"},
				FV: &FileVals{Service: FileValsService{URL: strPtr("file-url")}},
			},
			want: ConfigMismatch{
				AISvcURL: "env-url",
				AISvcKey: "env-key",
				Extra:    0,
			},
			wantErr: nil, // No type mismatch now
		},
		{
			name:    "empty_tag",
			dst:     &ConfigEmptyTag{},
			src:     Sources{},
			want:    ConfigEmptyTag{},
			wantErr: smap.ErrTagEmpty,
		},
		{
			name: "nil_path",
			dst:  &ConfigNilPath{},
			src: Sources{
				EV: &EnvVars{Nil: nil},
			},
			want:    ConfigNilPath{},
			wantErr: nil,
		},
		{
			name: "hydrate_string_to_int",
			dst:  &ConfigHydrate{},
			src: Sources{
				EV: &EnvVars{Count: 42},
			},
			want:    ConfigHydrate{Count: 42},
			wantErr: nil,
		},
		{
			name: "unset_pointer_field",
			dst:  &ConfigPointer{},
			src: Sources{
				EV: &EnvVars{URL: nil},
				FV: &FileVals{Service: FileValsService{URL: nil}},
			},
			want:    ConfigPointer{URL: nil},
			wantErr: nil,
		},
		{
			name: "valid_map_value",
			dst:  &ConfigMap{},
			src: Sources{
				EV: &EnvVars{Data: map[string]string{"key": "value"}},
			},
			want:    ConfigMap{Value: "value"},
			wantErr: nil,
		},
		{
			name: "missing_map_key",
			dst:  &ConfigMap{},
			src: Sources{
				EV: &EnvVars{Data: map[string]string{"other": "value"}},
			},
			want:    ConfigMap{},
			wantErr: nil,
		},
		{
			name: "method_value",
			dst:  &ConfigMethod{},
			src: Sources{
				EV: &EnvVars{Value: "struct value"},
			},
			want:    ConfigMethod{Value: "method value"},
			wantErr: nil,
		},
		{
			name: "skipzero_with_zero_values",
			dst:  &ConfigSkipZero{},
			src: Sources{
				EV: &EnvVars{Count: 0},
				FV: &FileVals{Count: 0},
			},
			want:    ConfigSkipZero{Count: 0},
			wantErr: nil,
		},
		{
			name: "skipzero_with_non-zero_value",
			dst:  &ConfigSkipZero{},
			src: Sources{
				EV: &EnvVars{Count: 0},
				FV: &FileVals{Count: 42},
			},
			want:    ConfigSkipZero{Count: 42},
			wantErr: nil,
		},
		{
			name: "string_overwrites_default_with_nil_pointer_in_second_path",
			dst:  &ConfigDefault{Field: "default"},
			src: Sources{
				EV: &EnvVars{Value: "overwritten"},
				FV: &FileVals{Service: FileValsService{URL: nil}},
			},
			want: ConfigDefault{
				Field: "overwritten",
			},
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

// Helper to create *string
func strPtr(s string) *string {
	return &s
}

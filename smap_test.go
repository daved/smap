package smap_test

import (
	"reflect"
	"testing"

	"github.com/daved/smap"
)

// Destination type for testing
type Config struct {
	AISvcURL string `smap:"EV.AISvcURL|FV.Service.URL"`
	AISvcKey string `smap:"EV.AISvcKey"`
	Extra    string `smap:"FV.Extra"`
	NoTag    string // No smap tag, should be skipped
}

// Source type for testing
type Sources struct {
	EV *EnvVars
	FV *FileVals
}

type EnvVars struct {
	AISvcURL string
	AISvcKey string
}

type FileVals struct {
	Service struct {
		URL string
	}
	Extra string
}

func TestMerge_Surface(t *testing.T) {
	tests := []struct {
		name    string
		dst     *Config
		src     interface{}
		want    Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "struct instance source",
			dst:  &Config{},
			src: Sources{
				EV: &EnvVars{AISvcURL: "env-url", AISvcKey: "env-key"},
				FV: &FileVals{Service: struct{ URL string }{URL: "file-url"}, Extra: "file-extra"},
			},
			want: Config{
				AISvcURL: "file-url", // FV overrides EV due to precedence
				AISvcKey: "env-key",
				Extra:    "file-extra",
				NoTag:    "", // Should remain empty
			},
			wantErr: false,
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
				NoTag:    "", // Should remain empty
			},
			wantErr: false,
		},
		{
			name:    "nil pointer source",
			dst:     &Config{},
			src:     (*Sources)(nil),
			want:    Config{}, // No changes expected due to error
			wantErr: true,
			errMsg:  "src must not be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := smap.Merge(tt.dst, tt.src)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Merge() error = nil, want error")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Merge() error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("Merge() error = %v, want nil", err)
				return
			}
			if !reflect.DeepEqual(*tt.dst, tt.want) {
				t.Errorf("Merge() dst = %+v, want %+v", *tt.dst, tt.want)
			}
		})
	}
}

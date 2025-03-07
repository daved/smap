package smap

import (
	"strings"
)

// TagKey is the struct tag key used to define source paths.
const TagKey = "smap"

// tagPathParts represents a single path segment in a smap tag.
type tagPathParts []string

// String implements fmt.Stringer for tagPathParts.
func (p tagPathParts) String() string {
	return strings.Join(p, ".")
}

// tagPathsParts represents multiple path segments in a smap tag.
type tagPathsParts []tagPathParts

// String implements fmt.Stringer for tagPathsParts.
func (p tagPathsParts) String() string {
	paths := make([]string, len(p))
	for i, part := range p {
		paths[i] = part.String()
	}
	return strings.Join(paths, "|")
}

// sTag represents a parsed smap tag with paths and options.
type sTag struct {
	pathsParts tagPathsParts
	opts       []string
}

// String recreates the original smap tag string.
func (t *sTag) String() string {
	if len(t.opts) == 0 {
		return t.pathsParts.String()
	}
	return t.pathsParts.String() + "," + strings.Join(t.opts, ",")
}

// HasHydrate checks if the "hydrate" option is present.
func (t *sTag) HasHydrate() bool {
	for _, opt := range t.opts {
		if opt == "hydrate" {
			return true
		}
	}
	return false
}

// HasSkipZero checks if the "skipzero" option is present.
func (t *sTag) HasSkipZero() bool {
	for _, opt := range t.opts {
		if opt == "skipzero" {
			return true
		}
	}
	return false
}

// newSTag constructs an sTag from a tag string.
func newSTag(tag string) (*sTag, error) {
	// Split into paths and options at the first comma
	parts := strings.SplitN(tag, ",", 2)
	pathsStr := strings.TrimSpace(parts[0])

	// Parse paths (split by "|")
	paths := strings.Split(pathsStr, "|")
	var pathsParts tagPathsParts
	for _, path := range paths {
		if path == "" {
			continue
		}
		segments := strings.Split(path, ".")
		for _, segment := range segments {
			if segment == "" {
				return nil, ErrTagInvalid // Empty segment (e.g., "Foo..Bar")
			}
		}
		pathsParts = append(pathsParts, tagPathParts(segments))
	}
	if len(pathsParts) == 0 {
		return nil, ErrTagEmpty // Tag is empty or only empty segments (e.g., "", "|")
	}

	// Parse options if present
	var opts []string
	if len(parts) > 1 {
		opts = strings.Split(strings.TrimSpace(parts[1]), ",")
		for i, opt := range opts {
			opt = strings.TrimSpace(opt)
			if opt == "" {
				return nil, ErrTagInvalid // Empty option (e.g., "path,,hydrate")
			}
			opts[i] = opt
		}
	}

	return &sTag{
		pathsParts: pathsParts,
		opts:       opts,
	}, nil
}

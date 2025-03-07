package smap

import "strings"

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

// makeTagPathsParts splits a smap tag into a slice of path segments, erroring on malformed tags.
func makeTagPathsParts(tag string) (tagPathsParts, error) {
	paths := strings.Split(tag, "|")
	var tagPathsParts tagPathsParts
	for _, path := range paths {
		if path == "" {
			continue
		}
		parts := strings.Split(path, ".")
		for _, part := range parts {
			if part == "" {
				return nil, ErrTagInvalid // Empty segment (e.g., "Foo..Bar")
			}
		}
		tagPathsParts = append(tagPathsParts, tagPathParts(parts))
	}
	if len(tagPathsParts) == 0 {
		return nil, ErrTagEmpty // Tag is empty or only empty segments (e.g., "", "|")
	}
	return tagPathsParts, nil
}

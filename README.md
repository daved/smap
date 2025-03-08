# smap

`smap` is a Go library for merging struct fields from a source struct into a destination struct based on struct tags. It supports flexible path navigation, including nested structs, maps, slices, and methods, with options like skipping zero values and hydrating string values into complex types.

## Installation

```sh
go get github.com/daved/smap
```

## Usage

Define a destination struct with smap tags specifying source paths, then use Merge to populate it from a source struct:

```go
package main

import (
    "fmt"
    "github.com/daved/smap"
)

type Source struct {
    Env struct {
        URL string
    }
    Data map[int]string
    Users []string
}

type Dest struct {
    URL   string `smap:"Env.URL"`
    Value string `smap:"Data.1"`
    User  string `smap:"Users.0"`
}

func main() {
    src := Source{
        Env:   struct{ URL string }{URL: "http://example.com"},
        Data:  map[int]string{1: "value"},
        Users: []string{"alice"},
    }
    dst := &Dest{}
    err := smap.Merge(dst, src)
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    fmt.Printf("Merged: %+v\n", dst) // Merged: &{URL:http://example.com Value:value User:alice}
}
```

## Features

Path Navigation: Access nested struct fields ("A.B.C"), map keys ("Map.key" or "Map.1"), and slice indexes ("Slice.0").

Methods: Call zero-argument methods on structs (e.g., "GetValue").

Options: 
skipzero: Skip zero values in multi-path tags.
hydrate: Convert strings to destination types using vtypes.Hydrate.

Error Handling: Detailed errors with MergeFieldError for debugging.

## API

```txt
func Merge(dst, src interface{}) error
```

Merges src into dst based on smap tags. dst must be a non-nil pointer to a struct; src must be a struct or non-nil pointer to a struct.

## Tag Syntax

Single path: "EV.URL"
Multiple paths: "EV.URL|FV.URL" (last non-nil/non-error value used)
Options: "EV.URL,skipzero,hydrate"

## Examples

See smap_test.go and smap_external_test.go for unit and surface tests demonstrating various use cases.

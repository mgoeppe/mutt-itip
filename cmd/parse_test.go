package cmd

import (
	"bytes"
	"embed"
	"io"
	"path"
	"testing"

	"github.com/DusanKasan/parsemail"

	ics "github.com/arran4/golang-ical"
)

//go:embed test-data/*
var embeddedFiles embed.FS

type testFile struct {
	name string
	in   io.Reader
}

func testFiles(t *testing.T) []testFile {
	r := []testFile{}
	es, err := embeddedFiles.ReadDir("test-data")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range es {
		if e.IsDir() {
			continue
		}
		b, err := embeddedFiles.ReadFile(path.Join("test-data", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		r = append(r, testFile{name: e.Name(), in: bytes.NewReader(b)})
	}
	return r
}

func Test_parseInput(t *testing.T) {
	tests := []struct {
		name    string
		want    *parsemail.Email
		want1   *ics.Calendar
		wantErr bool
	}{
		{
			name: "happy",
		},
	}
	for _, tt := range tests {
		for _, f := range testFiles(t) {
			t.Run(tt.name+"-"+f.name, func(t *testing.T) {
				_, _, err := parseInput(f.in)
				if (err != nil) != tt.wantErr {
					t.Errorf("parseInput() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			})
		}
	}
}

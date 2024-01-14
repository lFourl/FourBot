package main

import (
	"reflect"
	"testing"
)

func TestValidateURLs(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{"Valid URLs", "http://example.com, http://example.org", []string{"http://example.com", "http://example.org"}, false},
		{"Invalid URL", "http://example.com, not-a-url", nil, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateURLs(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateURLs() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("validateURLs() = %v, want %v", got, tc.want)
			}
		})
	}
}

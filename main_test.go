package main

import (
	"log"
	"strings"
	"testing"
)

func TestCleanChirp(t *testing.T) {
	cases := map[string]struct {
		input    string
		bwords []string
		expected string
	}{
		"simple": {input: "This is some stupid shit!", bwords: []string{"shit", "crazy"}, expected: "This is some stupid ****!"},
		"casing": {input: "CrAzy how this Works", bwords: []string{"crazy"}, expected: "**** how this Works"},
		"spacing": {input: "S hit, you    saw that?", bwords: []string{"shit"}, expected: "S hit, you    saw that?"},
	}

	for name, tc := range cases {
        t.Run(name, func(t *testing.T) {
            actual := cleanChirp(tc.input, tc.bwords)
            if strings.Compare(actual, tc.expected) != 0 {
            	log.Fatalf("expected %q, got %q", tc.expected, actual)
            }
        })
    }
}

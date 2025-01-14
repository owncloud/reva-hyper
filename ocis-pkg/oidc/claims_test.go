package oidc_test

import (
	"testing"

	"github.com/owncloud/ocis/v2/ocis-pkg/oidc"
)

type splitWithEscapingTest struct {
	// Name of the subtest.
	name string

	// string to split
	s string

	// seperator to use
	seperator string

	// escape character to use for escaping
	escape string

	expectedParts []string
}

func (swet splitWithEscapingTest) run(t *testing.T) {
	parts := oidc.SplitWithEscaping(swet.s, swet.seperator, swet.escape)
	if len(swet.expectedParts) != len(parts) {
		t.Errorf("mismatching length")
	}
	for i, v := range swet.expectedParts {
		if parts[i] != v {
			t.Errorf("expected part %d to be '%s', got '%s'", i, v, parts[i])
		}
	}
}

func TestSplitWithEscaping(t *testing.T) {
	tests := []splitWithEscapingTest{
		{
			name:          "plain claim name",
			s:             "roles",
			seperator:     ".",
			escape:        "\\",
			expectedParts: []string{"roles"},
		},
		{
			name:          "claim with .",
			s:             "my.roles",
			seperator:     ".",
			escape:        "\\",
			expectedParts: []string{"my", "roles"},
		},
		{
			name:          "claim with escaped .",
			s:             "my\\.roles",
			seperator:     ".",
			escape:        "\\",
			expectedParts: []string{"my.roles"},
		},
		{
			name:          "claim with escaped . left",
			s:             "my\\.other.roles",
			seperator:     ".",
			escape:        "\\",
			expectedParts: []string{"my.other", "roles"},
		},
		{
			name:          "claim with escaped . right",
			s:             "my.other\\.roles",
			seperator:     ".",
			escape:        "\\",
			expectedParts: []string{"my", "other.roles"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

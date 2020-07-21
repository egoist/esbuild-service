package util

import "testing"

func TestStrToBool(t *testing.T) {
	for _, tc := range []struct {
		name        string
		input       string
		expectedOut bool
	}{
		{
			name:        "Should return false with blank input",
			input:       "",
			expectedOut: false,
		},
		{
			name:        "Should return true with input string=true",
			input:       "true",
			expectedOut: true,
		},
		{
			name:        "Should return false with input string=false",
			input:       "false",
			expectedOut: false,
		},
		{
			name:        "Should return false with input string=unknown",
			input:       "unknown",
			expectedOut: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actualOut := StrToBool(tc.input)
			if tc.expectedOut != actualOut {
				t.Fatalf("expected out: %v, got: %v", tc.expectedOut, actualOut)
			}
		})
	}
}

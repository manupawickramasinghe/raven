package protocol

import (
	"bufio"
	"strings"
	"testing"
)

func TestReadNetstring(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
		errContains string
	}{
		{
			name:        "Valid netstring",
			input:       "5:hello,",
			expected:    "hello",
			expectError: false,
		},
		{
			name:        "Valid empty netstring",
			input:       "0:,",
			expected:    "",
			expectError: false,
		},
		{
			name:        "Valid netstring with colon inside data",
			input:       "12:hello:world!,",
			expected:    "hello:world!",
			expectError: false,
		},
		{
			name:        "Valid netstring with comma inside data",
			input:       "12:hello,world!,",
			expected:    "hello,world!",
			expectError: false,
		},
		{
			name:        "Missing length",
			input:       ":hello,",
			expectError: true,
			errContains: "invalid length",
		},
		{
			name:        "Invalid length (not a number)",
			input:       "abc:hello,",
			expectError: true,
			errContains: "invalid length",
		},
		{
			name:        "Negative length",
			input:       "-5:hello,",
			expectError: true,
			errContains: "invalid length: negative value",
		},
		{
			name:        "Exceeds max length",
			input:       "10485761:hello,", // 10MB + 1
			expectError: true,
			errContains: "exceeds maximum allowed size",
		},
		{
			name:        "Missing colon",
			input:       "5hello,",
			expectError: true,
			errContains: "failed to read length", // readString(':') will fail if EOF is reached before ':'
		},
		{
			name:        "Unexpected EOF (missing data)",
			input:       "5:hel",
			expectError: true,
			errContains: "failed to read data", // io.ReadFull fails
		},
		{
			name:        "Missing trailing comma",
			input:       "5:hello.",
			expectError: true,
			errContains: "expected comma, got .",
		},
		{
			name:        "Unexpected EOF (missing trailing comma)",
			input:       "5:hello",
			expectError: true,
			errContains: "failed to read comma",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tc.input))
			result, err := ReadNetstring(reader)

			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.errContains)
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("expected error containing %q, got: %v", tc.errContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if result != tc.expected {
					t.Errorf("expected %q, got %q", tc.expected, result)
				}
			}
		})
	}
}

package protocol

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
)

func TestReadNetstring(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "valid netstring",
			input:   "5:hello,",
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "empty netstring",
			input:   "0:,",
			want:    "",
			wantErr: false,
		},
		{
			name:    "multiple netstrings (reads first)",
			input:   "5:hello,4:world,",
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "invalid length (not a number)",
			input:   "abc:hello,",
			wantErr: true,
		},
		{
			name:    "invalid length (negative)",
			input:   "-1:hello,",
			wantErr: true,
		},
		{
			name:    "invalid length (too large)",
			input:   "10485761:too big,",
			wantErr: true,
		},
		{
			name:    "truncated length",
			input:   "5",
			wantErr: true,
		},
		{
			name:    "missing colon",
			input:   "5hello,",
			wantErr: true,
		},
		{
			name:    "truncated data",
			input:   "5:hell,",
			wantErr: true,
		},
		{
			name:    "missing comma",
			input:   "5:hello",
			wantErr: true,
		},
		{
			name:    "wrong terminator",
			input:   "5:hello.",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			got, err := ReadNetstring(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadNetstring() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadNetstring() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadNetstring_MaximumLength(t *testing.T) {
	const maxLen = 10 * 1024 * 1024
	data := make([]byte, maxLen)
	for i := range data {
		data[i] = 'a'
	}
	input := fmt.Sprintf("%d:%s,", maxLen, string(data))
	reader := bufio.NewReader(strings.NewReader(input))

	got, err := ReadNetstring(reader)
	if err != nil {
		t.Fatalf("ReadNetstring() unexpected error for max length: %v", err)
	}
	if len(got) != maxLen {
		t.Errorf("ReadNetstring() got length = %d, want %d", len(got), maxLen)
	}
}

func TestWriteNetstring(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal string",
			input: "hello",
			want:  "5:hello,",
		},
		{
			name:  "empty string",
			input: "",
			want:  "0:,",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := net.Pipe()
			defer client.Close()
			defer server.Close()

			errCh := make(chan error, 1)
			go func() {
				errCh <- WriteNetstring(client, tt.input)
			}()

			buf := make([]byte, len(tt.want)+10)
			n, err := server.Read(buf)
			if err != nil {
				t.Fatalf("Failed to read from pipe: %v", err)
			}

			if err := <-errCh; err != nil {
				t.Fatalf("WriteNetstring() error = %v", err)
			}

			got := string(buf[:n])
			if got != tt.want {
				t.Errorf("WriteNetstring() wrote %q, want %q", got, tt.want)
			}
		})
	}
}

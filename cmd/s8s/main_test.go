package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string // substring expected in stdout
		wantErr  string // substring expected in stderr
	}{
		{name: "no args prints usage", args: nil, wantCode: 0, wantOut: "Usage:"},
		{name: "version", args: []string{"version"}, wantCode: 0, wantOut: "s8s "},
		{name: "version flag", args: []string{"--version"}, wantCode: 0, wantOut: "s8s "},
		{name: "help", args: []string{"help"}, wantCode: 0, wantOut: "Usage:"},
		{name: "unknown command", args: []string{"bogus"}, wantCode: 2, wantErr: "unknown command"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			if tt.wantOut != "" && !strings.Contains(stdout.String(), tt.wantOut) {
				t.Errorf("stdout = %q, want substring %q", stdout.String(), tt.wantOut)
			}
			if tt.wantErr != "" && !strings.Contains(stderr.String(), tt.wantErr) {
				t.Errorf("stderr = %q, want substring %q", stderr.String(), tt.wantErr)
			}
		})
	}
}

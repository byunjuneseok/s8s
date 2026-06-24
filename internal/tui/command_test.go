package tui

import "testing"

func TestParseCommandLine(t *testing.T) {
	tests := []struct {
		in       string
		wantName string
		wantArgs []string
	}{
		{":quit", "quit", nil},
		{"quit", "quit", nil},
		{"  :ctx use main  ", "ctx", []string{"use", "main"}},
		{"watch add 005930", "watch", []string{"add", "005930"}},
		{"", "", nil},
		{"   ", "", nil},
		{":", "", nil},
	}
	for _, tt := range tests {
		name, args := parseCommandLine(tt.in)
		if name != tt.wantName {
			t.Errorf("parseCommandLine(%q) name = %q, want %q", tt.in, name, tt.wantName)
		}
		if len(args) != len(tt.wantArgs) {
			t.Errorf("parseCommandLine(%q) args = %v, want %v", tt.in, args, tt.wantArgs)
			continue
		}
		for i := range args {
			if args[i] != tt.wantArgs[i] {
				t.Errorf("parseCommandLine(%q) args[%d] = %q, want %q", tt.in, i, args[i], tt.wantArgs[i])
			}
		}
	}
}

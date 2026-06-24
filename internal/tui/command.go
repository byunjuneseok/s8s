package tui

import "strings"

// parseCommandLine splits a command-bar line into a command name and its
// arguments. A leading ":" is tolerated. Returns an empty name for blank input.
func parseCommandLine(line string) (name string, args []string) {
	line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), ":"))
	if line == "" {
		return "", nil
	}
	fields := strings.Fields(line)
	return fields[0], fields[1:]
}

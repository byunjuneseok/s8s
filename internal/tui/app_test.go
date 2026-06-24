package tui

import "testing"

func TestRunCommandDispatch(t *testing.T) {
	a := NewApp()

	var gotArgs []string
	called := 0
	a.RegisterCommand("greet", func(args []string) {
		called++
		gotArgs = args
	})

	a.runCommand(":greet alice bob")
	if called != 1 {
		t.Fatalf("handler called %d times, want 1", called)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "alice" || gotArgs[1] != "bob" {
		t.Errorf("args = %v, want [alice bob]", gotArgs)
	}

	// Blank input is a no-op.
	a.runCommand("   ")
	if called != 1 {
		t.Errorf("blank input triggered handler; called = %d", called)
	}

	// Unknown command must not panic.
	a.runCommand("does-not-exist")
}

func TestBuiltinsRegistered(t *testing.T) {
	a := NewApp()
	for _, name := range []string{"quit", "q", "help"} {
		if _, ok := a.commands[name]; !ok {
			t.Errorf("builtin command %q not registered", name)
		}
	}
}

func TestContextLabel(t *testing.T) {
	a := NewApp()
	a.SetContextLabel(func() string { return "toss-main" })
	if a.contextLabel() != "toss-main" {
		t.Errorf("contextLabel = %q, want toss-main", a.contextLabel())
	}
}

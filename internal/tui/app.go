package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// App is the s8s terminal UI shell: a header, a body that swaps between named
// screens, a status bar, and a ":" command bar.
type App struct {
	app    *tview.Application
	root   *tview.Pages
	body   *tview.Pages
	bottom *tview.Pages
	header *tview.TextView
	status *tview.TextView
	cmd    *tview.InputField

	screens      []string
	contextLabel func() string
	commands     map[string]func(args []string)
}

// NewApp builds the shell. Screens and commands are registered by the caller
// before Run.
func NewApp() *App {
	a := &App{
		app:          tview.NewApplication(),
		body:         tview.NewPages(),
		commands:     map[string]func([]string){},
		contextLabel: func() string { return "no context" },
	}

	a.header = tview.NewTextView().SetDynamicColors(true)
	a.header.SetText("[::b]s8s[::-]")

	a.status = tview.NewTextView().SetDynamicColors(true)

	a.cmd = tview.NewInputField().SetLabel(":")
	a.cmd.SetDoneFunc(func(key tcell.Key) {
		line := a.cmd.GetText()
		a.hideCommandBar()
		if key == tcell.KeyEnter {
			a.runCommand(line)
		}
	})

	a.bottom = tview.NewPages()
	a.bottom.AddPage("status", a.status, true, true)
	a.bottom.AddPage("cmd", a.cmd, true, false)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.header, 1, 0, false).
		AddItem(a.body, 0, 1, true).
		AddItem(a.bottom, 1, 0, false)

	a.root = tview.NewPages().AddPage("main", layout, true, true)
	a.app.SetInputCapture(a.inputCapture)

	a.registerBuiltins()
	a.refreshStatus()
	return a
}

// SetContextLabel sets the function used to render the status bar's context text.
func (a *App) SetContextLabel(fn func() string) {
	if fn != nil {
		a.contextLabel = fn
	}
	a.refreshStatus()
}

// AddScreen registers a screen primitive under name. The first screen added
// becomes visible.
func (a *App) AddScreen(name string, p tview.Primitive) {
	first := len(a.screens) == 0
	a.body.AddPage(name, p, true, first)
	a.screens = append(a.screens, name)
}

// AddMessageScreen registers a simple centered text screen, useful for
// informational states such as "no context configured".
func (a *App) AddMessageScreen(name, msg string) {
	tv := tview.NewTextView().SetText(msg).SetTextAlign(tview.AlignCenter)
	tv.SetBorder(true)
	a.AddScreen(name, tv)
}

// Show switches the body to the named screen.
func (a *App) Show(name string) {
	if a.body.HasPage(name) {
		a.body.SwitchToPage(name)
		a.refreshStatus()
	} else {
		a.flashStatus(fmt.Sprintf("no such screen: %s", name))
	}
}

// RegisterCommand binds a ":" command name to a handler.
func (a *App) RegisterCommand(name string, fn func(args []string)) {
	a.commands[name] = fn
}

// Run starts the UI event loop and blocks until the app stops.
func (a *App) Run() error {
	a.refreshStatus()
	return a.app.SetRoot(a.root, true).EnableMouse(true).Run()
}

// Stop ends the UI event loop.
func (a *App) Stop() { a.app.Stop() }

// Flash shows a transient message in the status bar. It is replaced the next
// time the status bar refreshes (e.g. on a screen switch).
func (a *App) Flash(msg string) { a.flashStatus(msg) }

// QueueUpdate runs fn on the UI goroutine and redraws. It is safe to call from
// background goroutines; use it to apply results of async work to the UI.
func (a *App) QueueUpdate(fn func()) { a.app.QueueUpdateDraw(fn) }

func (a *App) registerBuiltins() {
	a.commands["quit"] = func([]string) { a.app.Stop() }
	a.commands["q"] = a.commands["quit"]
	a.commands["help"] = func([]string) { a.showHelp() }
}

func (a *App) inputCapture(ev *tcell.EventKey) *tcell.EventKey {
	if a.app.GetFocus() == a.cmd {
		return ev
	}
	switch ev.Rune() {
	case ':':
		a.showCommandBar()
		return nil
	case '?':
		a.showHelp()
		return nil
	}
	return ev
}

func (a *App) showCommandBar() {
	a.cmd.SetText("")
	a.bottom.SwitchToPage("cmd")
	a.app.SetFocus(a.cmd)
}

func (a *App) hideCommandBar() {
	a.bottom.SwitchToPage("status")
	a.app.SetFocus(a.body)
}

func (a *App) runCommand(line string) {
	name, args := parseCommandLine(line)
	if name == "" {
		return
	}
	if fn, ok := a.commands[name]; ok {
		fn(args)
		return
	}
	a.flashStatus(fmt.Sprintf("unknown command: %s", name))
}

func (a *App) refreshStatus() {
	a.status.SetText(fmt.Sprintf("[gray]:cmd  ?help  :quit[-]   ctx: [white]%s", a.contextLabel()))
}

func (a *App) flashStatus(msg string) {
	a.status.SetText(fmt.Sprintf("[yellow]%s[-]", msg))
}

func (a *App) showHelp() {
	const help = `s8s — keys & commands

  :            open the command bar
  ?            this help
  :quit / :q   quit

Press Esc or Enter to close.`
	modal := tview.NewModal().
		SetText(help).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(int, string) {
			a.root.RemovePage("help")
			a.app.SetFocus(a.body)
		})
	a.root.AddPage("help", modal, true, true)
	a.app.SetFocus(modal)
}

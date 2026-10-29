package app

// CmdClass says whether a command may run while the console is locked
// (a job running, a session not unlocked).
type CmdClass int

const (
	Mutating  CmdClass = iota // default: refused while locked
	ReadOnly                  // always allowed
	QueryOnly                 // allowed with no arguments
)

// Command is one console verb.
type Command struct {
	Name    string
	Aliases []string
	Summary string // one line for the generated help table
	Usage   string // shown by `help <name>` and for argument errors
	Class   CmdClass

	// Raw hands the handler the line as typed, case preserved, instead of
	// lower-cased fields: for verbs carrying passwords or G-code.
	Raw bool

	// Run receives the arguments without the verb.
	Run func(ctx *CmdCtx, args []string)
}

// CmdCtx is what a handler gets beyond its arguments. Console output goes
// through Printf, so a test can capture it.
type CmdCtx struct {
	Line   string // the raw line as typed
	Printf func(format string, args ...any)
}

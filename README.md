# bmxos/app

The BMXOS application contract: the types a component declares to the
runtime that hosts it, and `apptest` to run components on a host. MIT
licensed (see LICENSE), standard library only.

```go
package blinky

import "github.com/bmxos/app"

func Component(p app.Platform) app.Component {
	return app.Component{
		Name: "blinky",
		Commands: []app.Command{{Name: "hello", Class: app.ReadOnly,
			Run: func(ctx *app.CmdCtx, args []string) { ctx.Printf("hello from %s\n", p.Board) }}},
	}
}
```

```go
h := apptest.New()
if err := h.Register(blinky.Component(app.Platform{Board: "test"})); err != nil {
	t.Fatal(err)
}
out, err := h.Run("hello")
```

| Package | What |
|---|---|
| `app` | Component, Command, Route, Panel, Persisted, Feature, Platform, Host |
| `app/apptest` | console dispatch with the device's lock and argument rules, route matching and response shapes, store regions |

Writing and testing an app needs nothing else. Running it on a board needs
the runtime (`github.com/bmxos/sdk/app`) and a board support package, which
are licensed: through a licence, or the hosted build. `bmx new app` starts a
repository with both.

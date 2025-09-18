package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"strings"

	"tybalt/cron"
	"tybalt/hooks"
	_ "tybalt/migrations"
	"tybalt/routes"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

// Go’s plain embed ignores files starting with an underscore, something that is
// required by the build system of the frontend in v1.52 and beyond due to
// dependencies. Specifically after this tag, the build emits _commonjsHelpers.*.js; it wasn’t
// embedded, so chunks 404ed and routes crashed.
// https://github.com/golang/go/issues/43854

//go:embed all:pb_public/**

var staticFiles embed.FS

func main() {
	app := pocketbase.New()

	// enable/disable automatic migration creation
	// loosely check if it was executed using "go run"
	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())

	migrationsDir := "./migrations"

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangGo, //migratecmd.TemplateLangJS
		Automigrate:  isGoRun,
		Dir:          migrationsDir,
	})

	// app.OnAfterBootstrap().PreAdd(func(e *core.BootstrapEvent) error {
	// 	app.Dao().ModelQueryTimeout = time.Duration(queryTimeout) * time.Second
	// 	return nil
	// })

	// enable/disable index.html forwarding on missing file (eg. in case of SPA)
	indexFallback := true

	// Prepare an embedded file system from pb_public directory
	directory, err := fs.Sub(staticFiles, "pb_public")
	if err != nil {
		log.Fatal("failed to load static files", err)
	}

	// Now bind the static file handler as a fallback immediately
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/{path...}", apis.Static(directory, indexFallback))
		return e.Next()
	})

	// Add the hooks to the app
	hooks.AddHooks(app)

	// Add the routes to the app
	routes.AddRoutes(app)

	// Add the cron jobs to the app
	cron.AddCronJobs(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

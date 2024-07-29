package main

import (
	"log"
	"os"
	"strings"

	"tybalt/hooks"
	_ "tybalt/migrations"
	"tybalt/routes"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

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

	// serves static files from the provided public dir (if exists)
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), indexFallback))
		return nil
	})

	// Add the hooks to the app
	hooks.AddHooks(app)

	// Add the routes to the app
	routes.AddRoutes(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

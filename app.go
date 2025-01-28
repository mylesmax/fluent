package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Set window transparency
	runtime.WindowSetBackgroundColour(ctx, 0, 0, 0, 0)
	runtime.WindowSetDarkTheme(ctx)
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// beforeClose is called when the app is about to quit
func (a *App) beforeClose(ctx context.Context) bool {
	return false
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
}

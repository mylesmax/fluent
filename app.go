package main

import (
	"context"
	"fluent/backend/db"
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

	//startup db
	_, err := db.InitDB()
	if err != nil {
		runtime.LogError(ctx, "Failed to init db: "+err.Error())
	}

	// Set window transparency
	runtime.WindowSetBackgroundColour(ctx, 0, 0, 0, 0)
	runtime.WindowSetDarkTheme(ctx)
}

func (a *App) GetProfiles() []db.Profile {
	profiles, err := db.GetProfiles()
	if err != nil {
		runtime.LogError(a.ctx, "faild to get profs: "+err.Error())
		return []db.Profile{}
	}
	profiles = append(profiles, db.Profile{
		Name:         "Add New",
		Emoji:        "➕",
		GlowColor:    "rgba(52, 211, 153, 0.5)",
		CurrentDrops: 0,
	})
	return profiles
}

func (a *App) AddProfile(profile db.Profile) error {
	return db.AddProfile(profile)
}

func (a *App) UpdateProfileName(oldName, newName string) error {
	return db.UpdateProfileName(oldName, newName)
}

func (a *App) DeleteProfile(name string) error {
	return db.DeleteProfile(name)
}

func (a *App) UpdateDrops(name string, drops int) error {
	return db.UpdateDrops(name, drops)
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

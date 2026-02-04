package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brianndofor/prq/internal/config"
	"github.com/brianndofor/prq/internal/github"
	"github.com/brianndofor/prq/internal/provider"
	"github.com/brianndofor/prq/internal/store"
)

type appKey struct{}

type App struct {
	Config     config.Config
	RepoConfig config.RepoConfig
	GH         *github.Client
	Provider   provider.Runner
	Store      *store.Store
}

func withApp(ctx context.Context, app *App) context.Context {
	return context.WithValue(ctx, appKey{}, app)
}

func getApp(ctx context.Context) (*App, error) {
	app, ok := ctx.Value(appKey{}).(*App)
	if !ok || app == nil {
		return nil, fmt.Errorf("internal error: app not initialized")
	}
	return app, nil
}

func initApp(configPath string) (*App, error) {
	merged, repoCfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	var ghRunner github.Runner = github.RealRunner{}
	var prov provider.Runner = provider.NewClaudeRunner(merged.Provider)
	if os.Getenv("PRQ_MOCK") == "1" {
		fixtures := os.Getenv("PRQ_MOCK_DIR")
		if fixtures == "" {
			fixtures = filepath.Join("testdata", "gh")
		}
		ghRunner = github.NewFixtureRunner(fixtures)
		fixturePath := os.Getenv("PRQ_PROVIDER_FIXTURE")
		if fixturePath == "" {
			fixturePath = filepath.Join("testdata", "provider", "review.json")
		}
		prov = provider.NewFakeRunner(fixturePath)
	}
	gh := github.NewClient(ghRunner)

	storePath := os.Getenv("PRQ_DB_PATH")
	if storePath == "" {
		storeDir := filepath.Join(os.Getenv("HOME"), ".prq")
		storePath = filepath.Join(storeDir, "prq.db")
	}
	st, err := store.Open(storePath)
	if err != nil {
		return nil, err
	}

	return &App{
		Config:     merged,
		RepoConfig: repoCfg,
		GH:         gh,
		Provider:   prov,
		Store:      st,
	}, nil
}

package cli

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/livebud/cli"
	"github.com/matthewmueller/jig"
	"github.com/matthewmueller/jig/internal/directory"
	"github.com/matthewmueller/jig/internal/typegen"
)

// New CLI that logs to log
func New(log *slog.Logger) *CLI {
	return &CLI{log}
}

// CLI for building views
type CLI struct {
	log *slog.Logger
}

// Parse the command line arguments and run the matching command
func (c *CLI) Parse(ctx context.Context, args ...string) error {
	cli := cli.New("jig", "Build views for production and generate types for their Go files")
	in := &Build{}
	cli.Flag("dir", "The project directory").Short('C').String(&in.Dir).Default(".")
	cli.Flag("minify", "Minify the code").Short('m').Bool(&in.Minify).Default(true)
	cli.Arg("views", "The views directory in the project").String(&in.Views).Default(".")
	cli.Run(func(ctx context.Context) error {
		return c.Build(ctx, in)
	})
	return cli.Parse(ctx, args...)
}

// Build input
type Build struct {
	Dir    string
	Views  string
	Minify bool
}

// Build the views into the project's dist directory, then generate TypeScript
// types for the Go files in the views directory
func (c *CLI) Build(ctx context.Context, in *Build) error {
	dir, err := directory.Resolve(in.Dir)
	if err != nil {
		return err
	} else if !filepath.IsLocal(in.Views) {
		return fmt.Errorf("jig: views directory %q must be within %s", in.Views, dir)
	}
	viewer, err := jig.Load(
		jig.WithDir(dir),
		jig.WithLog(c.log),
		jig.WithMinify(in.Minify),
		jig.WithWatch(false),
		jig.WithLiveRoute(""),
	)
	if err != nil {
		return err
	}
	defer viewer.Close()
	if err := viewer.Build(ctx, in.Views); err != nil {
		return err
	}
	// Generated after the build, since the views can be in a package that
	// embeds the build
	return typegen.Generate(ctx, c.log, filepath.Join(dir, in.Views))
}

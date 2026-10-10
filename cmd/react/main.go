package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/livebud/cli"
	"github.com/matthewmueller/glob"
	"github.com/matthewmueller/jig/internal/directory"
	"github.com/matthewmueller/jig/react"
	"github.com/matthewmueller/logs"
)

func main() {
	log := logs.Default()
	ctx := context.Background()
	if err := run(ctx, log, os.Args[1:]...); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger, args ...string) error {
	cli := cli.New("react", "Build React pages")
	cmd := &command{log: log}
	cli.Flag("dir", "The working directory").Short('C').String(&cmd.Dir).Default(".")
	cli.Flag("minify", "Minify the react code").Short('m').Bool(&cmd.Minify).Default(true)
	cli.Args("paths", "paths to React files").Strings(&cmd.Paths)
	cli.Run(cmd.Run)
	return cli.Parse(ctx, args...)
}

type command struct {
	log    *slog.Logger
	Dir    string
	Minify bool
	Paths  []string
}

func (c *command) Run(ctx context.Context) error {
	dir, err := directory.Resolve(c.Dir)
	if err != nil {
		return err
	}
	paths, err := glob.Glob(dir, c.Paths...)
	if err != nil {
		return err
	} else if len(paths) == 0 {
		return fmt.Errorf("react: no files matched")
	}
	for i, path := range paths {
		paths[i], err = filepath.Rel(dir, path)
		if err != nil {
			return err
		} else if strings.HasPrefix(paths[i], "..") {
			return fmt.Errorf("react: path %q is outside of the root directory %q", path, dir)
		}
	}
	viewer, err := react.Load(
		react.WithDir(dir),
		react.WithMinify(c.Minify),
		react.WithWatch(false),
		react.WithLiveRoute(""),
	)
	if err != nil {
		return err
	}
	defer viewer.Close()
	if err := viewer.Build(paths...); err != nil {
		return err
	}
	return nil
}

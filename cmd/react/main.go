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
	"github.com/matthewmueller/logs"
	"github.com/matthewmueller/views/internal/directory"
	"github.com/matthewmueller/views/react"
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
	cli.Args("paths", "paths to React files").Strings(&cmd.Paths)
	cli.Run(cmd.Run)
	return cli.Parse(ctx, args...)
}

type command struct {
	log   *slog.Logger
	Dir   string
	Paths []string
}

func isOutside(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return strings.HasPrefix(rel, "..")
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
	react := react.New(c.log)
	react.Dir = dir
	if err := react.Build(paths...); err != nil {
		return err
	}
	fmt.Println("React command executed", dir, c.Paths)
	return nil
}

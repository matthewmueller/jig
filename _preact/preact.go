package preact

import (
	"context"

	"github.com/matthewmueller/elmo/internal/asset"
	"github.com/matthewmueller/elmo/internal/goja"
	"github.com/matthewmueller/elmo/internal/log"
	"github.com/matthewmueller/elmo/internal/pagedir"
)

type Options struct {
	Log    log.Interface
	Dir    string
	Env    map[string]string
	JS     goja.Renderer
	Minify bool
}

func New(o *Options) *Compiler {
	s := &Static{
		Log:    o.Log,
		Dir:    o.Dir,
		Env:    o.Env,
		JS:     o.JS,
		Minify: o.Minify,
	}
	c := &Client{
		Dir:    o.Dir,
		Env:    o.Env,
		Minify: o.Minify,
		Bundle: false,
	}
	g := &Generator{s, c}
	g.Client.Bundle = true
	return &Compiler{s, c, g}
}

type Compiler struct {
	s *Static
	c *Client
	g *Generator
}

// TODO: prebuild should prebuild the SSR pages

func (c *Compiler) Render(ctx context.Context, page *pagedir.Page, props map[string]interface{}) (*asset.File, error) {
	return c.s.Render(ctx, page, props)
}

func (c *Compiler) Build(ctx context.Context, pages ...*pagedir.Page) ([]*asset.File, error) {
	return c.g.Build(ctx, pages...)
}

func (c *Compiler) Serve(ctx context.Context, page *pagedir.Page) (*asset.File, error) {
	return c.c.Serve(ctx, page)
}

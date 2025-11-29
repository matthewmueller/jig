package preact

import (
	"context"
	_ "embed"
	"sort"

	"github.com/matthewmueller/elmo/internal/asset"
	"github.com/matthewmueller/elmo/internal/pagedir"
)

type Generator struct {
	Static *Static
	Client *Client
}

func (g *Generator) Build(ctx context.Context, pages ...*pagedir.Page) (files []*asset.File, err error) {
	htmlFiles, err := g.Static.Build(ctx, pages...)
	if err != nil {
		return nil, err
	}
	jsFiles, err := g.Client.Build(ctx, pages...)
	if err != nil {
		return nil, err
	}
	files = append(files, htmlFiles...)
	files = append(files, jsFiles...)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

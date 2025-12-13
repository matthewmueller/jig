package styledjsx

import (
	"github.com/matthewmueller/jsx/ast"
	"github.com/matthewmueller/styledjsx"
)

func Rewrite(path, runtime string, script *ast.Script) error {
	rewriter := styledjsx.New()
	rewriter.Import.Name = "Style"
	rewriter.Import.Path = runtime
	rewriter.Import.IsDefault = false
	rewriter.Attr = "className"
	return rewriter.RewriteAST(path, script)
}

func Remove(path string, script *ast.Script) error {
	rewriter := styledjsx.New()
	rewriter.Import.Name = ""
	rewriter.Import.Path = ""
	rewriter.Attr = "className"
	return rewriter.RewriteAST(path, script)
}

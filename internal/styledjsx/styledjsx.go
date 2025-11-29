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
	return rewriter.RewriteAST(path, script)
}

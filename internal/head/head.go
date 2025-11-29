package head

import (
	"strconv"

	"github.com/matthewmueller/jsx/ast"
)

// Rewrite rewrites the script to replace the <head> element with <Head>.
// The <head> must be a child of the root element.
func Rewrite(path, runtime string, script *ast.Script) error {
	v := &visitor{runtime: runtime}
	script.Visit(v)
	if v.err != nil {
		return v.err
	} else if !v.found {
		return nil
	}
	importDecl := &ast.Text{Value: "import {Head} from " + strconv.Quote(runtime) + ";\n"}
	script.Body = append([]ast.Fragment{importDecl}, script.Body...)
	return nil
}

type visitor struct {
	runtime string
	found   bool
	err     error
}

func (v *visitor) VisitScript(s *ast.Script) {
	for _, n := range s.Body {
		n.Visit(v)
	}
}

func (v *visitor) VisitText(t *ast.Text)               {}
func (v *visitor) VisitField(f *ast.Field)             {}
func (v *visitor) VisitStringValue(s *ast.StringValue) {}
func (v *visitor) VisitExpr(e *ast.Expr)               {}
func (v *visitor) VisitBoolValue(b *ast.BoolValue)     {}
func (v *visitor) VisitComment(n *ast.Comment)         {}

func (v *visitor) VisitElement(e *ast.Element) {
	for _, child := range e.Children {
		el, ok := child.(*ast.Element)
		if !ok {
			continue
		}
		if el.Name == "head" {
			el.Name = "Head"
			v.found = true
		}
	}
}

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/tools/go/exact"
	"os"
	// "golang.org/x/tools/go/ast/astutil"
)

// type virtualsensor struct {
// 	sensor
//
// }

type AnalyzeVisitor struct {
	Err                    error
	ExprCount, SensorCount int
	Sensors                []string
}

func (v *AnalyzeVisitor) Visit(n ast.Node) ast.Visitor {

	if v.Err != nil {
		return nil
	}

	switch e := n.(type) {
	case *ast.UnaryExpr:
		switch e.Op {
		case token.ADD:
		case token.SUB:
		default:
			v.Err = fmt.Errorf("Unsupported unary expression '%v' at postiton %v", e.Op.String(), e.OpPos)
			return nil
		}
	case *ast.BinaryExpr:
		switch e.Op {
		case token.ADD:
			fallthrough
		case token.SUB:
			fallthrough
		case token.MUL:
			fallthrough
		case token.QUO:
			fallthrough
		case token.REM:
			v.ExprCount++
		default:
			v.Err = fmt.Errorf("Unsupported binary expression '%v' at postiton %v", e.Op.String(), e.OpPos)
			return nil
		}
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
		case token.FLOAT:
		case token.STRING:
			v.SensorCount++
			v.Sensors = append(v.Sensors, e.Value)
		default:
			v.Err = fmt.Errorf("Unsupported literal '%v' at postiton %v", e.Value, e.ValuePos)
			return nil
		}
	case *ast.CallExpr:
		switch i := e.Fun.(type) {
		case *ast.Ident:
			switch i.Name {
			case "max":
				fallthrough
			case "min":
				v.ExprCount++
			default:
				v.Err = fmt.Errorf("Unsupported function '%v' at postiton %v", i.Name, i.NamePos)
				return nil
			}
		default:
			v.Err = fmt.Errorf("Call without identifier at postiton %v", e.Fun.Pos())
			return nil
		}
	case *ast.ParenExpr:
	case nil:
		return nil
	default:
		v.Err = fmt.Errorf("Unsupported expression at postiton %v", e.Pos())
		return nil
	}

	return v
}

type DecorateVisitor struct {
	parentStack []*ast.Node
	Decoration  map[ast.Node]exact.Value
	Err         error
}

func (v *DecorateVisitor) Visit(n ast.Node) ast.Visitor {
	if v.Decoration == nil {
		v.Decoration = make(map[ast.Node]exact.Value)
	}

	if n == nil {
		p := *v.parentStack[len(v.parentStack)-1]

		switch e := p.(type) {
		case *ast.BasicLit:
			if e.Kind == token.INT || e.Kind == token.FLOAT {
				v.Decoration[p] = exact.MakeFromLiteral(e.Value, e.Kind)
			} else {
				v.Decoration[p] = exact.MakeUnknown()
			}
		case *ast.UnaryExpr:
			v.Decoration[p] = exact.UnaryOp(e.Op, v.Decoration[e.X.(ast.Node)], 0)
		case *ast.BinaryExpr:
			v.Decoration[p] = exact.BinaryOp(v.Decoration[e.X.(ast.Node)], e.Op, v.Decoration[e.Y.(ast.Node)])
		case *ast.CallExpr:
			v.Decoration[p] = exact.MakeUnknown()
		case *ast.ParenExpr:
			v.Decoration[p] = v.Decoration[e.X.(ast.Node)]
		default:
			v.Err = fmt.Errorf("Unsupported expression at postiton %v", e.Pos())
		}

		v.parentStack = v.parentStack[:len(v.parentStack)-1]
		return nil
	}

	v.parentStack = append(v.parentStack, &n)
	return v
}

type SQLVisitor struct {
	Decoration *map[ast.Node]exact.Value
	Query      string
	Err        error
}

// func (v *SQLVisitor) Visit(n ast.Node) ast.Visitor {
// 	if v.Decoration == nil {
// 		v.Err = fmt.Errorf("Missing decoration\n")
// 	}
//
// 	switch e := n.(type) {
// 		case
// 	}
// }

func main() {
	exprs := "(3 + 243) *  \"foo\" + (-8 * 5)"

	exp, Err := parser.ParseExpr(exprs)

	if Err != nil {
		fmt.Println(Err)
	}

	ast.Fprint(os.Stdout, nil, exp, nil)

	v := AnalyzeVisitor{}
	ast.Walk(&v, exp)

	if v.Err != nil {
		fmt.Println(v.Err)
	}

	fmt.Printf("Found %v expressions, %v sensors: %v\n", v.ExprCount, v.SensorCount, v.Sensors)

	d := DecorateVisitor{}
	ast.Walk(&d, exp)
	if d.Err != nil {
		fmt.Println(d.Err)
	}
	fmt.Printf("Decoration %v\n", d.Decoration)
}

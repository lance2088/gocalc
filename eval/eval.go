package eval

import (
	"fmt"
	"misc/calc/ast"
	"misc/calc/parser"
	"misc/calc/token"
)

var builtins = map[string]func([]interface{}) interface{}{
	"+":      funcAdd,
	"-":      funcSub,
	"*":      funcMul,
	"/":      funcDiv,
	"%":      funcMod,
	"=":      funcEq,
	"<":      funcLess,
	"<=":     funcLessEq,
	">":      funcGreater,
	">=":     funcGreaterEq,
	"<>":     funcNotEq,
	"define": funcDefine,
	"if":     funcIf,
	"print":  funcPrint,
	"set":    funcSet,
}

var variables = map[string]interface{}{}
var functions = map[string]func([]interface{}) interface{}{}

func EvalExpr(expr string) interface{} {
	return EvalFile("", expr)
}

func EvalFile(fname, expr string) interface{} {
	f := token.NewFile(fname, expr)
	n := parser.ParseFile(f, expr)
	if f.NumErrors() > 0 {
		f.PrintErrors()
		return nil
	}
	res := eval(f, n)
	if f.NumErrors() > 0 {
		f.PrintErrors()
		return nil
	}
	return res
}

func eval(f *token.File, n ast.Node) interface{} {
	switch node := n.(type) {
	case *ast.File:
		var x interface{}
		for _, n := range node.Nodes {
			x = eval(f, n) // scoping seems like it should come into play here
			switch t := x.(type) {
			case *ast.Identifier:
				f.AddError(t.Pos(), "Unknown identifier: ", t.Lit)
				return nil
			}
		}
		return x
	case *ast.Identifier:
		if fn, ok := builtins[node.Lit]; ok {
			return fn
		}
		if fn, ok := functions[node.Lit]; ok {
			//fmt.Println("found something for:", node.Lit)
			return fn
		}
		if n, ok := variables[node.Lit]; ok {
			return n
		}
		return node
	case *ast.Number:
		return node.Val
	case *ast.Operator:
		// technically, it should be impossible for this to fail. If it does,
		// it should be a catistrophic error (like the panic that will be
		// produced) because Operators are only ever built-in functions. It
		// should be impossible for the scanner to scan something as an operator
		// if it's not a built-in.
		return builtins[string(node.Val)]
	case *ast.Expression:
		//fmt.Println(node.Nodes)
		if len(node.Nodes) < 1 {
			f.AddError(node.Pos(), "Empty expression not allowed")
			return nil
		}
		fn, ok := eval(f, node.Nodes[0]).(func([]interface{}) interface{})
		if !ok {
			f.AddError(node.Nodes[0].Pos(), "First element of an expression must "+
				"be a function.")
			return nil
		}
		//fmt.Println("building args list")
		args := make([]interface{}, 0) //len(node.Nodes[1:]))
		if len(node.Nodes) > 1 {
			for _, node := range node.Nodes[1:] {
				args = append(args, eval(f, node))
			}
		}
		//fmt.Println("calling fn with", len(args), "args")

		res := fn(args)
		if err, ok := res.(error); ok {
			f.AddError(node.Pos(), err)
		}
		//fmt.Println("res:", res)
		return res
	}
	return nil
}

func funcAdd(args []interface{}) interface{} {
	return genFunc(func(a, b int) int { return a + b }, args)
}

func funcSub(args []interface{}) interface{} {
	return genFunc(func(a, b int) int { return a - b }, args)
}

func funcMul(args []interface{}) interface{} {
	return genFunc(func(a, b int) int { return a * b }, args)
}

func funcDiv(args []interface{}) interface{} {
	return genFunc(func(a, b int) int { return a / b }, args)
}

func funcMod(args []interface{}) interface{} {
	return genFunc(func(a, b int) int { return a % b }, args)
}

func genFunc(fn func(a, b int) int, args []interface{}) interface{} {
	if len(args) < 1 {
		return nil
	}
	if len(args) < 2 {
		if i, ok := args[0].(int); ok {
			return i
		}
		return nil
	}
	var res int
	if i, ok := args[0].(int); ok {
		res = i
	}
	for _, x := range args[1:] {
		switch v := x.(type) {
		case int:
			res = fn(res, v)
		default:
			// maybe return something like:
			// errors.New("Function accepts numerical types only, got:", v)
			return nil
		}
	}
	return res
}

func funcEq(args []interface{}) interface{} {
	return genFunc(func(a, b int) int { return convBool(a == b) }, args)
}

func funcLess(args []interface{}) interface{} {
	return genFunc(func(a, b int) int { return convBool(a < b) }, args)
}

func funcLessEq(args []interface{}) interface{} {
	return genFunc(func(a, b int) int { return convBool(a <= b) }, args)
}

func funcGreater(args []interface{}) interface{} {
	return genFunc(func(a, b int) int { return convBool(a > b) }, args)
}

func funcGreaterEq(args []interface{}) interface{} {
	return genFunc(func(a, b int) int { return convBool(a >= b) }, args)
}

func funcNotEq(args []interface{}) interface{} {
	return genFunc(func(a, b int) int { return convBool(a != b) }, args)
}

func convBool(b bool) int {
	if b {
		return 1
	}
	return 0
}

func funcDefine(args []interface{}) interface{} {
	//fmt.Println("define")
	if len(args) != 2 {
		return nil // really feel like this should be an error...not just nil
	}
	if i, ok := args[0].(*ast.Identifier); ok {
		switch args[1].(type) {
		case *ast.Operator:
			return nil // this REALLY should produce an error...
		default:
			//fmt.Println("adding", i.Lit, "to variables list:", args[1])
			r := args[1]
			functions[i.Lit] = func(args []interface{}) interface{} {
				//fmt.Printf("executing function...%v\n", r)
				return r
			}
		}
	}
	return nil
}

func funcIf(args []interface{}) interface{} {
	if len(args) != 3 {
		return nil //should produce error
	}
	if eq, ok := args[0].(int); ok {
		if eq == 0 {
			return args[2]
		}
		return args[1]
	}
	return nil // also an error
}

func funcPrint(args []interface{}) interface{} {
	// some checks should be done on the args. For example, this current
	// implementation will return the address of a built-in function if
	// given as an argument.
	fmt.Println(args...)
	return nil
}

func funcSet(args []interface{}) interface{} {
	if len(args) != 2 {
		return nil // really feel like this should be an error...not just nil
	}
	if i, ok := args[0].(*ast.Identifier); ok {
		switch args[1].(type) {
		case *ast.Operator:
			return nil // this REALLY should produce an error...
		default:
			variables[i.Lit] = args[1]
		}
	}
	return nil
}

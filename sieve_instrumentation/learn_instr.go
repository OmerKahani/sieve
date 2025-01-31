package main

import (
	"fmt"
	"go/token"

	"github.com/dave/dst"
)

func instrumentSharedInformerGoForLearn(ifilepath, ofilepath string) {
	f := parseSourceFile(ifilepath, "cache")
	_, funcDecl := findFuncDecl(f, "HandleDeltas", 1)
	if funcDecl != nil {
		for _, stmt := range funcDecl.Body.List {
			if rangeStmt, ok := stmt.(*dst.RangeStmt); ok {
				instrNotifyLearnBeforeIndexerWrite := &dst.AssignStmt{
					Lhs: []dst.Expr{&dst.Ident{Name: "sieveEventID"}},
					Rhs: []dst.Expr{&dst.CallExpr{
						Fun:  &dst.Ident{Name: "NotifyLearnBeforeIndexerWrite", Path: "sieve.client"},
						Args: []dst.Expr{&dst.Ident{Name: "string(d.Type)"}, &dst.Ident{Name: "d.Object"}},
					}},
					Tok: token.DEFINE,
				}
				instrNotifyLearnBeforeIndexerWrite.Decs.End.Append("//sieve")
				insertStmt(&rangeStmt.Body.List, 0, instrNotifyLearnBeforeIndexerWrite)

				instrNotifyLearnAfterIndexerWrite := &dst.ExprStmt{
					X: &dst.CallExpr{
						Fun:  &dst.Ident{Name: "NotifyLearnAfterIndexerWrite", Path: "sieve.client"},
						Args: []dst.Expr{&dst.Ident{Name: "sieveEventID"}, &dst.Ident{Name: "d.Object"}},
					},
				}
				instrNotifyLearnAfterIndexerWrite.Decs.End.Append("//sieve")
				rangeStmt.Body.List = append(rangeStmt.Body.List, instrNotifyLearnAfterIndexerWrite)
				break
			}
		}
	} else {
		panic(fmt.Errorf("Cannot find function HandleDeltas"))
	}

	writeInstrumentedFile(ofilepath, "cache", f)
}

func instrumentControllerGoForLearn(ifilepath, ofilepath string) {
	f := parseSourceFile(ifilepath, "controller")
	_, funcDecl := findFuncDecl(f, "reconcileHandler", 1)
	if funcDecl != nil {
		index := 0
		beforeReconcileInstrumentation := &dst.ExprStmt{
			X: &dst.CallExpr{
				Fun:  &dst.Ident{Name: "NotifyLearnBeforeReconcile", Path: "sieve.client"},
				Args: []dst.Expr{&dst.Ident{Name: "c.Name"}, &dst.Ident{Name: "c"}},
			},
		}
		beforeReconcileInstrumentation.Decs.End.Append("//sieve")
		insertStmt(&funcDecl.Body.List, index, beforeReconcileInstrumentation)

		index += 1
		afterReconcileInstrumentation := &dst.DeferStmt{
			Call: &dst.CallExpr{
				Fun:  &dst.Ident{Name: "NotifyLearnAfterReconcile", Path: "sieve.client"},
				Args: []dst.Expr{&dst.Ident{Name: "c.Name"}, &dst.Ident{Name: "c"}},
			},
		}
		afterReconcileInstrumentation.Decs.End.Append("//sieve")
		insertStmt(&funcDecl.Body.List, index, afterReconcileInstrumentation)
	} else {
		panic(fmt.Errorf("Cannot find function reconcileHandler"))
	}

	writeInstrumentedFile(ofilepath, "controller", f)
}

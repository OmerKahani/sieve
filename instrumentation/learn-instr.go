package main

import (
	"go/token"
	"fmt"
	"github.com/dave/dst"
)

func instrumentSharedInformerGoForLearn(ifilepath, ofilepath string) {
	f := parseSourceFile(ifilepath, "cache")
	_, funcDecl := findFuncDecl(f, "HandleDeltas")
	if funcDecl != nil {
		for _, stmt := range funcDecl.Body.List {
			if rangeStmt, ok := stmt.(*dst.RangeStmt); ok {
				instrNotifyLearnBeforeIndexerWrite := &dst.AssignStmt{
					Lhs: []dst.Expr{&dst.Ident{Name: "sonarEventID"}},
					Rhs: []dst.Expr{&dst.CallExpr{
						Fun:  &dst.Ident{Name: "NotifyLearnBeforeIndexerWrite", Path: "sonar.client"},
						Args: []dst.Expr{&dst.Ident{Name: "string(d.Type)"}, &dst.Ident{Name: "d.Object"}},
					}},
					Tok: token.DEFINE,
				}
				instrNotifyLearnBeforeIndexerWrite.Decs.End.Append("//sonar")
				insertStmt(&rangeStmt.Body.List, 0, instrNotifyLearnBeforeIndexerWrite)

				instrNotifyLearnAfterIndexerWrite := &dst.ExprStmt{
					X: &dst.CallExpr{
						Fun:  &dst.Ident{Name: "NotifyLearnAfterIndexerWrite", Path: "sonar.client"},
						Args: []dst.Expr{&dst.Ident{Name: "sonarEventID"}, &dst.Ident{Name: "d.Object"}},
					},
				}
				instrNotifyLearnAfterIndexerWrite.Decs.End.Append("//sonar")
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
	_, funcDecl := findFuncDecl(f, "reconcileHandler")
	if funcDecl != nil {
		index := 0
		beforeReconcileInstrumentation := &dst.ExprStmt{
			X: &dst.CallExpr{
				Fun:  &dst.Ident{Name: "NotifyLearnBeforeReconcile", Path: "sonar.client"},
				Args: []dst.Expr{&dst.Ident{Name: "c.Name"}},
			},
		}
		beforeReconcileInstrumentation.Decs.End.Append("//sonar")
		insertStmt(&funcDecl.Body.List, index, beforeReconcileInstrumentation)

		index += 1
		afterReconcileInstrumentation := &dst.DeferStmt{
			Call: &dst.CallExpr{
				Fun:  &dst.Ident{Name: "NotifyLearnAfterReconcile", Path: "sonar.client"},
				Args: []dst.Expr{&dst.Ident{Name: "c.Name"}},
			},
		}
		afterReconcileInstrumentation.Decs.End.Append("//sonar")
		insertStmt(&funcDecl.Body.List, index, afterReconcileInstrumentation)
	} else {
		panic(fmt.Errorf("Cannot find function reconcileHandler"))
	}

	writeInstrumentedFile(ofilepath, "controller", f)
}

func instrumentSplitGoForLearn(ifilepath, ofilepath string) {
	f := parseSourceFile(ifilepath, "client")

	instrumentCacheRead(f, "Get")
	instrumentCacheRead(f, "List")

	writeInstrumentedFile(ofilepath, "client", f)
}

func instrumentCacheRead(f *dst.File, etype string) {
	_, funcDecl := findFuncDecl(f, etype)
	if funcDecl != nil {
		if returnStmt, ok := funcDecl.Body.List[len(funcDecl.Body.List) - 1].(*dst.ReturnStmt); ok {
			modifiedInstruction := &dst.AssignStmt{
				Lhs: []dst.Expr{&dst.Ident{Name: "err"}},
				Tok: token.DEFINE,
				Rhs: returnStmt.Results,
			}
			modifiedInstruction.Decs.End.Append("//sonar")
			funcDecl.Body.List[len(funcDecl.Body.List) - 1] = modifiedInstruction

			if etype == "Get" {
				instrumentationExpr := &dst.ExprStmt{
					X: &dst.CallExpr{
						Fun:  &dst.Ident{Name: "NotifyLearnCacheGet", Path: "sonar.client"},
						Args: []dst.Expr{&dst.Ident{Name: "\"Get\""}, &dst.Ident{Name: "key"}, &dst.Ident{Name: "obj"}, &dst.Ident{Name: "err"}},
					},
				}
				instrumentationExpr.Decs.End.Append("//sonar")
				funcDecl.Body.List = append(funcDecl.Body.List, instrumentationExpr)
			} else if etype == "List" {
				instrumentationExpr := &dst.ExprStmt{
					X: &dst.CallExpr{
						Fun:  &dst.Ident{Name: "NotifyLearnCacheList", Path: "sonar.client"},
						Args: []dst.Expr{&dst.Ident{Name: "\"List\""}, &dst.Ident{Name: "list"}, &dst.Ident{Name: "err"}},
					},
				}
				instrumentationExpr.Decs.End.Append("//sonar")
				funcDecl.Body.List = append(funcDecl.Body.List, instrumentationExpr)
			} else {
				panic(fmt.Errorf("Wrong type %s for CacheRead", etype))
			}

			instrumentationReturn := &dst.ReturnStmt{
				Results: []dst.Expr{&dst.Ident{Name: "err"}},
			}
			instrumentationReturn.Decs.End.Append("//sonar")
			funcDecl.Body.List = append(funcDecl.Body.List, instrumentationReturn)
		} else {
			panic(fmt.Errorf("Last stmt of %s is not return", etype))
		}
	} else {
		panic(fmt.Errorf("Cannot find function %s", etype))
	}
}

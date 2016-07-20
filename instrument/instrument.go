package instrument

import (
	"go/ast"
)

import (
	"github.com/timtadh/data-structures/errors"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

import ()

type instrumenter struct {
	program *loader.Program
	ssa *ssa.Program
	entry string
}

func buildSSA(program *loader.Program) *ssa.Program {
	sp := ssautil.CreateProgram(program, ssa.GlobalDebug)
	sp.Build()
	return sp
}

func Instrument(entryPkgName string, program *loader.Program) (err error) {
	entry := program.Package(entryPkgName)
	if entry == nil {
		return errors.Errorf("The entry package was not found in the loaded program")
	}
	if entry.Pkg.Name() != "main" {
		return errors.Errorf("The entry package was not main")
	}
	i := &instrumenter{
		program: program,
		ssa: buildSSA(program),
		entry: entryPkgName,
	}
	return i.instrument()
}

func (i *instrumenter) instrument() (err error) {
	type bbEntry struct {
		bb *ssa.BasicBlock
		n *ast.Expr
	}
	for pkgType, _ := range i.program.AllPackages {
		ssaPkg := i.ssa.Package(pkgType)
		if ssaPkg == nil {
			return errors.Errorf("Could not find pkg %v", pkgType)
		}
		err := i.functions(ssaPkg, func(fn *ssa.Function) error {
			entries := make([]bbEntry, 0, len(fn.Blocks))
			errors.Logf("INFO", "fn %v", fn)
			for _, blk := range fn.Blocks {
				errors.Logf("INFO", "blk %v", blk)
				found := false
				for _, inst := range blk.Instrs {
					if debug, is := inst.(*ssa.DebugRef); is {
						entries = append(entries, bbEntry{
							bb: blk,
							n: &debug.Expr,
						})
						errors.Logf("INFO", "entry %v %v", debug.Expr, i.program.Fset.Position(debug.Expr.Pos()))
						found = true
						break
					}
				}
				if !found {
					// Some blocks do not have a clear syntactic location
					// for _, inst := range blk.Instrs {
					// 	errors.Logf("INFO", "inst %T %v %v", inst, inst, i.program.Fset.Position(inst.Pos()))
					// }
					// return errors.Errorf("Not entry for %v", blk)
					entries = append(entries, bbEntry{
						bb: blk,
						n: nil,
					})
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		// for _, f := range pkgInfo.Files {
		// 	errors.Logf("INFO", "f %v", f)
		// }
	}
	return nil
}

func (i instrumenter) functions(pkg *ssa.Package, do func(*ssa.Function) error) error {
	var values [10]*ssa.Value
	seen := make(map[*ssa.Function]bool)
	for _, member := range pkg.Members {
		if fn, is := member.(*ssa.Function); is {
			if seen[fn] {
				continue
			}
			seen[fn] = true
			if err := do(fn); err != nil {
				return err
			}
			for _, blk := range fn.Blocks {
				for _, inst := range blk.Instrs {
					for _, op := range inst.Operands(values[:0]) {
						if innerFn, is := (*op).(*ssa.Function); is {
							if seen[innerFn] {
								continue
							}
							seen[innerFn] = true
							if err := do(innerFn); err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}
	return nil
}

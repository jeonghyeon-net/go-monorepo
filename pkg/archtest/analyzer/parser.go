// Package analyzer는 Go 소스 코드를 AST로 파싱하여
// import문, 타입 선언, 함수 선언 등의 정보를 추출하는 패키지다.
package analyzer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ──────────────────────────────────────────────
// 데이터 구조체 정의
// ──────────────────────────────────────────────

// FileInfo는 하나의 Go 파일을 분석한 결과를 담는 구조체다.
type FileInfo struct {
	Path      string       // 파일의 절대 경로
	Package   string       // package 선언 이름
	Imports   []ImportInfo // import 목록
	Types     []TypeInfo   // 타입 선언 목록
	Functions []FuncInfo   // 함수/메서드 목록
}

// ImportInfo는 하나의 import 문 정보를 담는 구조체다.
type ImportInfo struct {
	Alias string // import 별칭 (없으면 빈 문자열)
	Path  string // import 경로
	Line  int    // 소스코드 줄 번호
}

// TypeInfo는 타입 선언 하나의 정보를 담는 구조체다.
type TypeInfo struct {
	Name        string // 타입 이름
	IsExported  bool   // 대문자 시작 여부
	IsInterface bool   // interface 타입 여부
	Line        int    // 소스코드 줄 번호
}

// FuncInfo는 함수 또는 메서드 하나의 정보를 담는 구조체다.
type FuncInfo struct {
	Name        string   // 함수/메서드 이름
	Receiver    string   // 리시버 타입 (함수면 빈 문자열)
	ReturnTypes []string // 반환 타입 목록
	Line        int      // 소스코드 줄 번호
	IsExported  bool     // 대문자 시작 여부
}

// ──────────────────────────────────────────────
// 파일 파싱 함수
// ──────────────────────────────────────────────

// ParseFile은 하나의 Go 파일을 AST로 파싱하고 FileInfo로 변환한다.
func ParseFile(path string) (*FileInfo, error) {
	fset := token.NewFileSet()

	astFile, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("파일 파싱 실패 %s: %w", path, err)
	}

	info := &FileInfo{
		Path:    path,
		Package: astFile.Name.Name,
	}

	for _, imp := range astFile.Imports {
		ii := ImportInfo{
			Path: strings.Trim(imp.Path.Value, `"`),
			Line: fset.Position(imp.Pos()).Line,
		}
		if imp.Name != nil {
			ii.Alias = imp.Name.Name
		}
		info.Imports = append(info.Imports, ii)
	}

	for _, decl := range astFile.Decls {
		switch declNode := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range declNode.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				ti := TypeInfo{
					Name:       ts.Name.Name,
					IsExported: ts.Name.IsExported(),
					Line:       fset.Position(ts.Pos()).Line,
				}

				if _, ok := ts.Type.(*ast.InterfaceType); ok {
					ti.IsInterface = true
				}

				info.Types = append(info.Types, ti)
			}

		case *ast.FuncDecl:
			fi := FuncInfo{
				Name:       declNode.Name.Name,
				IsExported: declNode.Name.IsExported(),
				Line:       fset.Position(declNode.Pos()).Line,
			}

			if declNode.Recv != nil && len(declNode.Recv.List) > 0 {
				fi.Receiver = exprToString(declNode.Recv.List[0].Type)
			}

			if declNode.Type.Results != nil {
				for _, result := range declNode.Type.Results.List {
					fi.ReturnTypes = append(fi.ReturnTypes, exprToString(result.Type))
				}
			}

			info.Functions = append(info.Functions, fi)
		}
	}

	return info, nil
}

// exprToString은 AST 타입 표현(ast.Expr)을 문자열로 변환한다.
func exprToString(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.StarExpr:
		return "*" + exprToString(node.X)
	case *ast.SelectorExpr:
		return exprToString(node.X) + "." + node.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprToString(node.Elt)
	case *ast.MapType:
		return "map[" + exprToString(node.Key) + "]" + exprToString(node.Value)
	default:
		return ""
	}
}

// ──────────────────────────────────────────────
// 디렉토리 파싱 함수
// ──────────────────────────────────────────────

// ParseDirectory는 주어진 디렉토리 아래의 모든 Go 파일을 재귀적으로 파싱한다.
// 숨김 디렉토리, vendor, tmp, _test.go 파일은 제외한다.
func ParseDirectory(root string) ([]*FileInfo, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, nil
	}

	var files []*FileInfo

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fi, parseErr := ParseFile(path)
		if parseErr != nil {
			return parseErr
		}
		files = append(files, fi)
		return nil
	})
	if err != nil {
		return files, fmt.Errorf("디렉토리 순회 실패 %s: %w", root, err)
	}

	return files, nil
}

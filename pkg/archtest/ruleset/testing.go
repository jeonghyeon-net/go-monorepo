// testing.go — 테스트 품질 규칙을 정의한다.
//
// 4가지 규칙:
//   - testing/missing-goleak: 테스트 패키지에 goleak 적용 여부
//   - testing/missing-testify: 테스트 파일에 testify 사용 여부
//   - testing/missing-build-tag: //go:build unit 또는 e2e 태그 존재 여부
//   - testing/raw-assertion: testify와 t.Fatal/t.Error 혼용 감지
package ruleset

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go-monorepo/pkg/archtest/report"
)

const goleakImportPath = "go.uber.org/goleak"

//nolint:gochecknoglobals // 테스트 품질 규칙에서 사용하는 불변 설정값이다.
var testifyImportPrefixes = []string{
	"github.com/stretchr/testify/require",
	"github.com/stretchr/testify/assert",
	"github.com/stretchr/testify/suite",
}

// rawAssertionMethods는 testify로 대체해야 하는 testing.T 메서드 목록이다.
//
//nolint:gochecknoglobals // 테스트 품질 규칙에서 사용하는 불변 설정값이다.
var rawAssertionMethods = map[string]bool{
	"Fatal":  true,
	"Fatalf": true,
	"Error":  true,
	"Errorf": true,
}

type testPackageInfo struct {
	dir          string
	hasTestFiles bool
	hasTestMain  bool
	hasGoleak    bool
}

type testFileInfo struct {
	path             string
	hasTestFn        bool
	hasTestify       bool
	hasRawAssertions bool
	hasBuildTag      bool
}

// CheckTestingPatterns는 internal/ 아래의 모든 테스트 파일에 대해 테스트 품질 규칙을 검사한다.
func CheckTestingPatterns(cfg *Config) []report.Violation {
	var violations []report.Violation

	internalDir := filepath.Join(cfg.ProjectRoot, "internal")
	if _, err := os.Stat(internalDir); os.IsNotExist(err) {
		return nil
	}

	packages, files := collectTestInfo(internalDir)

	// 규칙 1: goleak (패키지 단위)
	for _, pkg := range packages {
		if !pkg.hasTestFiles {
			continue
		}

		relDir, err := filepath.Rel(cfg.ProjectRoot, pkg.dir)
		if err != nil {
			continue
		}

		if !pkg.hasTestMain {
			violations = append(violations, report.Violation{
				Rule:     "testing/missing-goleak",
				Severity: report.Error,
				Message:  "테스트 파일이 있는 패키지에 TestMain + goleak.VerifyTestMain이 없다",
				File:     relDir,
				Fix:      "해당 패키지에 TestMain 함수를 추가하고 goleak.VerifyTestMain(m)을 호출하라",
			})
		} else if !pkg.hasGoleak {
			violations = append(violations, report.Violation{
				Rule:     "testing/missing-goleak",
				Severity: report.Error,
				Message:  "TestMain이 있지만 goleak.VerifyTestMain을 호출하지 않는다",
				File:     relDir,
				Fix:      "TestMain에서 goleak.VerifyTestMain(m)을 호출하라 (go.uber.org/goleak import 필요)",
			})
		}
	}

	// 규칙 2: testify (파일 단위)
	for _, file := range files {
		if !file.hasTestFn || file.hasTestify {
			continue
		}

		relPath, err := filepath.Rel(cfg.ProjectRoot, file.path)
		if err != nil {
			continue
		}
		violations = append(violations, report.Violation{
			Rule:     "testing/missing-testify",
			Severity: report.Error,
			Message:  "Test* 함수가 있지만 testify를 사용하지 않는다",
			File:     relPath,
			Fix:      "testify/require 또는 testify/assert를 import하여 단언문을 작성하라",
		})
	}

	// 규칙 3: 빌드 태그 (파일 단위)
	for _, file := range files {
		if file.hasBuildTag {
			continue
		}

		relPath, err := filepath.Rel(cfg.ProjectRoot, file.path)
		if err != nil {
			continue
		}
		violations = append(violations, report.Violation{
			Rule:     "testing/missing-build-tag",
			Severity: report.Error,
			Message:  "테스트 파일에 //go:build unit 또는 //go:build e2e 태그가 없다",
			File:     relPath,
			Fix:      "파일 첫 줄에 //go:build unit 또는 //go:build e2e를 추가하라",
		})
	}

	// 규칙 4: raw assertion (파일 단위)
	for _, file := range files {
		if !file.hasTestFn || !file.hasTestify || !file.hasRawAssertions {
			continue
		}

		relPath, err := filepath.Rel(cfg.ProjectRoot, file.path)
		if err != nil {
			continue
		}
		violations = append(violations, report.Violation{
			Rule:     "testing/raw-assertion",
			Severity: report.Error,
			Message:  "testify를 import했지만 t.Fatal/t.Error 등 표준 단언도 함께 사용한다",
			File:     relPath,
			Fix:      "t.Fatal → require.FailNow, t.Error → assert.Fail 등 testify 단언으로 교체하라",
		})
	}

	return violations
}

// collectTestInfo는 rootDir 아래의 모든 테스트 패키지/파일 정보를 수집한다.
func collectTestInfo(rootDir string) ([]testPackageInfo, []testFileInfo) {
	pkgMap := make(map[string]*testPackageInfo)
	var files []testFileInfo

	//nolint:errcheck,gosec // WalkDir 콜백 내에서 에러를 개별 처리한다.
	filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // 접근 에러는 건너뛴다.
		}

		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		dir := filepath.Dir(path)

		pkg, exists := pkgMap[dir]
		if !exists {
			pkg = &testPackageInfo{dir: dir}
			pkgMap[dir] = pkg
		}
		pkg.hasTestFiles = true

		fileInfo := analyzeTestFile(path, pkg)
		files = append(files, fileInfo)

		return nil
	})

	packages := make([]testPackageInfo, 0, len(pkgMap))
	for _, pkg := range pkgMap {
		packages = append(packages, *pkg)
	}
	return packages, files
}

// analyzeTestFile은 하나의 _test.go 파일을 AST로 파싱하여 정보를 수집한다.
func analyzeTestFile(path string, pkg *testPackageInfo) testFileInfo {
	info := testFileInfo{path: path}

	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return info
	}

	info.hasBuildTag = detectBuildTag(astFile)

	for _, imp := range astFile.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		if importPath == goleakImportPath {
			pkg.hasGoleak = true
		}

		if slices.Contains(testifyImportPrefixes, importPath) {
			info.hasTestify = true
		}
	}

	for _, decl := range astFile.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv != nil {
			continue
		}

		name := funcDecl.Name.Name

		if name == "TestMain" {
			pkg.hasTestMain = true
		} else if strings.HasPrefix(name, "Test") {
			info.hasTestFn = true
		}
	}

	info.hasRawAssertions = detectRawAssertions(astFile)

	return info
}

// detectRawAssertions는 파일에서 testing.T의 단언 메서드 직접 호출을 감지한다.
// FuncLit(익명 함수)도 검사하므로 t.Run() 콜백 내부의 호출도 잡아낸다.
func detectRawAssertions(astFile *ast.File) bool {
	// *testing.T 파라미터 이름을 수집
	tNames := make(map[string]bool)
	ast.Inspect(astFile, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if name := findTestingTParam(fn.Type.Params); name != "" {
				tNames[name] = true
			}
		case *ast.FuncLit:
			if name := findTestingTParam(fn.Type.Params); name != "" {
				tNames[name] = true
			}
		}
		return true
	})

	if len(tNames) == 0 {
		return false
	}

	// 금지된 메서드 호출 검색 (t.Fatal, t.Error 등)
	found := false
	ast.Inspect(astFile, func(n ast.Node) bool {
		if found {
			return false
		}

		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := selExpr.X.(*ast.Ident)
		if !ok {
			return true
		}

		if tNames[ident.Name] && rawAssertionMethods[selExpr.Sel.Name] {
			found = true
			return false
		}

		return true
	})

	return found
}

// findTestingTParam는 함수 파라미터에서 *testing.T 타입의 이름을 반환한다.
func findTestingTParam(params *ast.FieldList) string {
	if params == nil {
		return ""
	}

	for _, param := range params.List {
		starExpr, ok := param.Type.(*ast.StarExpr)
		if !ok {
			continue
		}

		selExpr, ok := starExpr.X.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		ident, ok := selExpr.X.(*ast.Ident)
		if !ok {
			continue
		}

		if ident.Name == "testing" && selExpr.Sel.Name == "T" {
			if len(param.Names) > 0 {
				return param.Names[0].Name
			}
		}
	}

	return ""
}

//nolint:gochecknoglobals // 테스트 품질 규칙에서 사용하는 불변 설정값이다.
var allowedBuildTags = []string{"unit", "e2e"}

// detectBuildTag는 파일에서 //go:build 태그에 unit 또는 e2e가 포함되어 있는지 확인한다.
func detectBuildTag(f *ast.File) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			text := c.Text
			if !strings.HasPrefix(text, "//go:build ") {
				continue
			}

			constraint := strings.TrimPrefix(text, "//go:build ")

			for _, tag := range allowedBuildTags {
				if strings.Contains(constraint, tag) {
					return true
				}
			}
		}
	}

	return false
}

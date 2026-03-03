package archtest_test

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go-monorepo/pkg/archtest"
)

func TestWorkspaceArchitecture(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	goWorkPath := filepath.Join(repoRoot, "go.work")

	goWorkFile, err := os.Open(goWorkPath)
	if err != nil {
		t.Fatalf("failed to open go.work: %v", err)
	}
	defer func() { _ = goWorkFile.Close() }()

	foundServiceModule := false
	scanner := bufio.NewScanner(goWorkFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "./svc/") {
			continue
		}

		modulePath := strings.TrimPrefix(line, "./")
		serviceRoot := filepath.Join(repoRoot, modulePath)
		if _, err := os.Stat(filepath.Join(serviceRoot, "go.mod")); err != nil {
			t.Fatalf("go.mod not found for workspace service module %q: %v", modulePath, err)
		}

		foundServiceModule = true
		serviceName := filepath.Base(modulePath)
		t.Run(serviceName, func(t *testing.T) {
			archtest.RunAll(t, serviceRoot)
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("failed to scan go.work: %v", err)
	}

	if !foundServiceModule {
		t.Skip("no workspace service modules found in go.work")
	}
}

package archtest_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"go-monorepo/pkg/archtest"
)

func TestWorkspaceArchitecture(t *testing.T) {
	t.Parallel()

	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	svcDir := filepath.Join(repoRoot, "svc")

	entries, err := os.ReadDir(svcDir)
	if err != nil {
		t.Fatalf("failed to read svc directory: %v", err)
	}

	foundServiceModule := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		serviceRoot := filepath.Join(svcDir, entry.Name())
		if _, err := os.Stat(filepath.Join(serviceRoot, "go.mod")); err != nil {
			continue
		}

		foundServiceModule = true
		t.Run(entry.Name(), func(t *testing.T) {
			t.Parallel()
			archtest.RunAll(t, serviceRoot)
		})
	}

	if !foundServiceModule {
		t.Skip("no service modules found in svc/")
	}
}

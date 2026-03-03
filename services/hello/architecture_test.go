package main_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"go-monorepo/pkg/archtest"
)

func TestArchitecture(t *testing.T) {
	_, f, _, _ := runtime.Caller(0)
	archtest.RunAll(t, filepath.Dir(f))
}

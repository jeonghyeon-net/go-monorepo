//go:build unit

package hello_test

import (
	"testing"

	"go-monorepo/pkg/archtest"
)

func TestArchitecture(t *testing.T) {
	t.Parallel()
	archtest.RunAll(t, ".")
}

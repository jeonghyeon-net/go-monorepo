//go:build unit

package greeter_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"go-monorepo/svc/hello/internal/greeter"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestGreet(t *testing.T) {
	t.Parallel()

	got := greeter.Greet("world")
	require.Equal(t, "hello, world!", got)
}

package greeter_test

import (
	"testing"

	"go-monorepo/svc/hello/internal/greeter"
)

func TestGreet(t *testing.T) {
	t.Parallel()

	got := greeter.Greet("world")
	want := "hello, world!"

	if got != want {
		t.Errorf("Greet(\"world\") = %q, want %q", got, want)
	}
}

package main

import (
	"fmt"

	"go-monorepo/svc/hello/internal/greeter"
)

func main() {
	fmt.Println(greeter.Greet("world"))
}

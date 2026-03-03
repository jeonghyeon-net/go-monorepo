package greeter

import "fmt"

// Greet은 이름을 받아 인사 메시지를 반환한다.
func Greet(name string) string {
	return fmt.Sprintf("hello, %s!", name)
}

package util

import (
	"fmt"
	"runtime/debug"
)

func WithRecovery(exec func(), recoverFn func(r any)) {
	defer func() {
		r := recover()
		if recoverFn != nil {
			recoverFn(r)
		}
		if r != nil {
			fmt.Println("panic in the recoverable goroutine")
			fmt.Printf("Recovered panic: %v\n", r)
			fmt.Printf("Stack trace:\n%s\n", debug.Stack())
		}
	}()
	exec()
}

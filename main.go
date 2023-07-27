package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/AlchemillaHQ/owtt/cmd"
)

func handlePanic() {
	if a := recover(); a != nil {
		// Open a file for writing
		f, err := os.OpenFile("panic.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		// Write the panic value and stack trace to the file
		fmt.Fprintln(f, "An error occurred:", a)
		stackTrace := debug.Stack()
		fmt.Fprintln(f, string(stackTrace))
	}
}

func main() {
	defer handlePanic()

	cmd.Execute()
}

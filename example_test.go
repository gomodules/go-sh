package sh_test

import (
	"fmt"

	"gomodules.xyz/go-sh"
)

func ExampleCommand() {
	out, err := sh.Command("echo", "hello").Output()
	fmt.Println(string(out), err)
}

func Example_command_Pipe() {
	out, err := sh.Command("echo", "-n", "hi").Command("wc", "-c").Output()
	fmt.Println(string(out), err)
}

func Example_command_SetDir() {
	out, err := sh.Command("pwd", sh.Dir("/")).Output()
	fmt.Println(string(out), err)
}

func ExampleTest() {
	if sh.Test("dir", "mydir") {
		fmt.Println("mydir exists")
	}
}

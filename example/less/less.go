package main

import "gomodules.xyz/go-sh"

func main() {
	sh.Command("less", "less.go").Run()
}

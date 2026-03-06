package main

import (
	"fmt"
	"time"

	"gomodules.xyz/go-sh"
)

func main() {
	s := sh.NewSession()
	s.ShowCMD = true
	//s.Command("seq", "1", "2000").LeafCommand("xargs").LeafCommand("xargs")
	s.Command("seq", "1", "2000000000")
	var out []byte
	var err error

	ch := make(chan struct{})
	go func() {
		out, err = s.Output()
		//fmt.Println("out:", string(out))
		ch <- struct{}{}
	}()
	_ = out
	_ = err

	tick := time.NewTicker(1 * time.Second)

loop:
	for {
		select {
		case <-tick.C:
			fmt.Println("tick")
			out0, _ := s.CurrentLeafOutput(0)
			size := len(out0)
			fmt.Println("out0 size:", size)
			fmt.Println("out0:", string(out0[size-100:]))
		case <-ch:
			break loop
		}
	}

	fmt.Println("done")

}

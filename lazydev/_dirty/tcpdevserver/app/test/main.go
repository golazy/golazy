package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func tick(i int) {
	if i%2 == 0 {
		fmt.Fprintf(os.Stderr, "Tock %d\n", i)
	} else {
		fmt.Printf("Tick %d\n", i)
	}

}

func main() {

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT)

	ticker := time.NewTicker(time.Second * 2 / 3)
	count := 1
	tick(count)

	for {
		select {
		case <-ticker.C:
			count++
			tick(count)
		case s := <-sigs:
			fmt.Println("Got a signal", s)
			os.Exit(0)

		}
	}

}

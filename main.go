package main

import (
	"./nest"
	"fmt"
	"os"
	"time"
)

func contGetStatus(n *nest.Nest, sleepTime int) {
	for {
		res, _ := n.GetStatus()
		fmt.Print(res)
		time.Sleep(time.Duration(sleepTime) * time.Second)
	}
}

func main() {
	args := os.Args
	if len(args) != 3 {
		fmt.Println("Usage: go run main.go <username> <password>")
	}

	n := nest.NewNest(args[1], args[2])

	fmt.Println("Logging in...")

	n.Login()

	fmt.Println("Done.")

	fmt.Println("Getting status...")

	contGetStatus(n, 60)
}

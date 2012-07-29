package main

import (
	"./gongo"
	"fmt"
	"os"
	"strconv"
)

func UsageError() {
	fmt.Fprintf(os.Stderr, "Usage: %v [moveCount]\n\n", os.Args[0])
	os.Exit(1)
}

func main() {
	moveCount := 10
	gameCount := 1
	if len(os.Args) >= 2 {
		val, err := strconv.Atoi(os.Args[1])
		if err != nil {
			UsageError()
		}
		moveCount = val
	}
	if len(os.Args) >= 3 {
		val, err := strconv.Atoi(os.Args[2])
		if err != nil {
			UsageError()
		}
		gameCount = val
	}
	if len(os.Args) > 3 {
		UsageError()
	}

	var conf gongo.Config
	conf.BoardSize = 9
	for game := 0; game < gameCount; game++ {
		r := gongo.NewConfiguredRobot(conf)
		color := gongo.Black
		for i := 0; i < moveCount; i++ {
			r.GenMove(color)
			color = color.GetOpponent()
		}
		fmt.Println(gongo.BoardToString(r))
	}
}

package main

import (
	"fmt";
	"gongo";
	"os";
	"strconv";
)

func UsageError() {
	fmt.Fprintf(os.Stderr, "Usage: %v [moveCount]\n\n", os.Args[0]);
	os.Exit(1);
}

func main() {
	moveCount := 10;
	if len(os.Args) == 2 {
		val, err := strconv.Atoi(os.Args[1]);
		if err != nil {
			UsageError()
		}
		moveCount = val;
	} else if len(os.Args) > 2 {
		UsageError()
	}

	var conf gongo.Config;
	conf.BoardSize = 9;
	r := gongo.NewConfiguredRobot(conf);
	color := gongo.Black;
	for i := 0; i < moveCount; i++ {
		r.GenMove(color);
		color = color.GetOpponent();
	}
	fmt.Println(gongo.BoardToString(r));
}

package main

import (
	"gongo";
	"os";
)

func main() {
	bot := gongo.NewRobot(9);
	err := gongo.Run(bot, os.Stdin, os.Stdout);
	if err != nil {
		panic(err);
	}
}
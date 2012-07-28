package main

import (
	"fmt"
	"./gongo"
	"io"
	"os"
	"strconv"
)

func UsageError() {
	fmt.Fprintf(os.Stderr, "Usage: %v [sampleCount]\n\n", os.Args[0])
	os.Exit(1)
}

func main() {
	var conf gongo.Config
	if len(os.Args) == 1 {
		conf.SampleCount = 1000
	} else if len(os.Args) == 2 {
		val, err := strconv.Atoi(os.Args[1])
		if err != nil {
			UsageError()
		}
		conf.SampleCount = val
	} else {
		UsageError()
	}
	bot := gongo.NewConfiguredRobot(conf)
	err := gongo.Run(bot, os.Stdin, os.Stdout)
	if err == io.EOF {
		fmt.Fprintln(os.Stderr, "got EOF")
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "Unexpected error: %v", err)
	}
}

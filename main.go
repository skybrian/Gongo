package main

import (
	"gongo";
	"os";
)

func main() {
	gongo.Run(os.Stdin, os.Stdout);
}
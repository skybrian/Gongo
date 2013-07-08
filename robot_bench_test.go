package gongo

import (
	"testing"
	"math/rand"
)

func Benchmark9x9RandomGame(bench *testing.B) {
	b := new(board)
	b.clearBoard(9)
	eboard := new(board)
	eboard.clearBoard(9)
	rng := rand.New(rand.NewSource(int64(2131)))
	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		b.playRandomGame(rng)
		b.copyFrom(eboard)
	}
}

func Benchmark9x9GenMove(b *testing.B) {
	robot := NewRobot(9)
	b.ResetTimer()
	color := Black
	for i := 0; i < b.N; i++ {
		robot.GenMove(color)
		color=color.GetOpponent()
	}
}

func Benchmark19x19RandomGame(bench *testing.B) {
	b := new(board)
	b.clearBoard(19)
	eboard := new(board)
	eboard.clearBoard(19)
	rng := rand.New(rand.NewSource(int64(2131)))
	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		b.playRandomGame(rng)
		b.copyFrom(eboard)
	}
}

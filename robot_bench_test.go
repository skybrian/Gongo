package gongo

import (
	"log"
	"math/rand"
	"testing"
)

// discard logs for benchmarks
type DevNull struct{}

func (DevNull) Write(p []byte) (int, error) {
	return len(p), nil
}

func Benchmark9x9RandomGame(bench *testing.B) {
	b, _ := newBoard(9)
	eboard, _ := newBoard(9)
	rng := rand.New(rand.NewSource(int64(2131)))
	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		b.playRandomGame(rng)
		b.copyFrom(eboard)
	}
}

func Benchmark9x9GenMove(b *testing.B) {
	robot := NewConfiguredRobot(
		Config{
			BoardSize: 9,
			Log:       log.New(new(DevNull), "", 0),
		})
	b.ResetTimer()
	color := Black
	for i := 0; i < b.N; i++ {
		robot.GenMove(color)
		b.StopTimer()
		robot.ClearBoard()
		b.StartTimer()
	}
}

func Benchmark9x9FindWins(b *testing.B) {
	robot := new(robot)
	robot.board = new(board)
	robot.scratchBoard = new(board)
	robot.SetBoardSize(9)
	robot.sampleCount = 1000
	robot.randomness = defaultRandomness
	robot.log = log.New(new(DevNull), "", 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		robot.findWins(1000)
	}
}

func Benchmark19x19RandomGame(bench *testing.B) {
	b, _ := newBoard(19)
	eboard, _ := newBoard(19)
	rng := rand.New(rand.NewSource(int64(2131)))
	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		b.playRandomGame(rng)
		b.copyFrom(eboard)
	}
}

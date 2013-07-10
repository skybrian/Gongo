package gongo

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

// === Multithreaded implementation of GoRobot interface ===

type multirobot struct {
	mr     *robot   // main robot
	slaves []*robot // subrobots
}

func (r *robot) copyFrom(other *robot) {
	for i := 0; i < other.board.moveCount; i++ {
		r.boardHashes[i] = other.boardHashes[i]
	}
	r.board.copyFrom(other.board)
	r.scratchBoard.copyFrom(other.board)
	r.komi = other.komi
	r.sampleCount = other.sampleCount
}

func NewMultiRobot(boardSize int) (m GoRobot, err error) {
	return newMultiRobot(boardSize)
}

func newMultiRobot(boardSize int) (m *multirobot, err error) {
	return newConfiguredMultiRobot(Config{BoardSize: boardSize})
}

func newConfiguredMultiRobot(config Config) (m *multirobot, err error) {
	myconfig := Config{
		BoardSize:   9,
		SampleCount: 1000,
		Randomness:  defaultRandomness,
		Log:         log.New(os.Stderr, "[gongo master]", log.Ltime)}

	usedefaultlog := true
	if config.BoardSize > 0 {
		myconfig.BoardSize = config.BoardSize
	}
	if config.SampleCount > 0 {
		myconfig.SampleCount = config.SampleCount
	}
	if config.Randomness != nil {
		myconfig.Randomness = config.Randomness
	}
	if config.Log != nil {
		myconfig.Log = config.Log
		usedefaultlog = false
	}

	m = new(multirobot)
	m.mr = newRobot(myconfig)
	// create one slave robot per cpu
	for i := 0; i < runtime.NumCPU(); i++ {
		if usedefaultlog {
			myconfig.Log = log.New(os.Stderr, fmt.Sprintf("[gongo %v]", i), log.Ltime)
		}
		m.slaves = append(m.slaves, newRobot(myconfig))
	}
	return m, nil
}

func (m *multirobot) Debug() string {
	return m.mr.Debug()
}

func (m *multirobot) GetBoardSize() int {
	return m.mr.board.GetBoardSize()
}

func (m *multirobot) GetCell(x, y int) Color {
	return m.mr.board.GetCell(x, y)
}

func (m *multirobot) Play(c Color, x, y int) (ok bool, message string) {
	for _, r := range m.slaves {
		r.makeMove(r.board.makePt(x, y))
	}
	result, captures := m.mr.makeMove(m.mr.board.makePt(x, y))
	return result.toPlayResult(captures)
}

func (m *multirobot) SetBoardSize(size int) (ok bool) {
	ok = m.mr.SetBoardSize(size)
	if !ok {
		return false
	}
	for _, r := range m.slaves {
		ok = r.SetBoardSize(size)
		if !ok {
			return false
		}
	}
	return true
}

func (m *multirobot) ClearBoard() {
	m.mr.ClearBoard()
	for _, r := range m.slaves {
		r.ClearBoard()
	}
}

func (m *multirobot) SetKomi(komi float64) {
	m.mr.SetKomi(komi)
	for _, r := range m.slaves {
		r.SetKomi(komi)
	}
}

func (m *multirobot) GenMove(color Color) (x, y int, result MoveResult) {
	return m.genMove(color)
}

func (m *multirobot) genMove(color Color) (x, y int, result MoveResult) {
	x, y, result = m.mr.GenMove(color)
	if result == Passed {
		return 0, 0, Passed
	}
	if result == Played {
		for _, r := range m.slaves {
			r.makeMove(r.board.makePt(x, y))
		}
		return x, y, result
	}
	panic(fmt.Sprintf("could not make move %v, %v (%v)", x, y, result))
}

func (m *multirobot) genMoveMulti(color Color) (x, y int, result MoveResult) {
	if !m.mr.board.isMyTurn(color) {
		// GTP protocol allows generating a move by either side;
		// treat as if the other player passed.
		if ok, message := m.Play(color.GetOpponent(), 0, 0); !ok {
			panic(fmt.Sprintf("other side cannot pass? %s", message))
		}
	}
	m.mr.findWins(m.mr.sampleCount)
	return
}

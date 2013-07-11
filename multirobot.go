package gongo

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime"
	"time"
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
	r.board.commonMoveCount = other.board.commonMoveCount // reset this too
	r.board.copyFrom(other.board)
	r.scratchBoard.copyFrom(other.board)
	r.komi = other.komi
	r.sampleCount = other.sampleCount
	r.candCount = other.candCount
}

func (m *multirobot) syncSlaves() {
	for _, r := range m.slaves {
		r.copyFrom(m.mr)
	}
}

func NewMultiRobot(boardSize int) (m GoRobot, err error) {
	return newMultiRobot(boardSize)
}

func NewConfiguredMultiRobot(config Config) (m GoRobot, err error) {
	return newConfiguredMultiRobot(config)
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
	maxprocs := runtime.GOMAXPROCS(runtime.NumCPU())
	for i := 0; i < maxprocs; i++ {
		m.mr.log.Printf("binding slave bot to cpu[%d]", i)
		if usedefaultlog {
			myconfig.Log = log.New(os.Stderr, fmt.Sprintf("[gongo %v]", i), log.Ltime)
		}
		myconfig.Randomness = &randomness{src: rand.NewSource(time.Now().Unix())}
		slave := newRobot(myconfig)
		slave.log.Printf("slave reporting for duty!")
		m.slaves = append(m.slaves, slave)
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

func (m *multirobot) GenMove(color Color) (x, y int, moveResult MoveResult) {
	m.genMovesMulti(color) // generates candidate moves
	bestMove := m.mr.candidates[0]
	result, _ := m.mr.makeMove(bestMove)
	if result == played {
		x, y = m.mr.board.getCoords(bestMove)
		moveResult = Played
	} else if result == passed {
		x, y, moveResult = 0, 0, Passed
	}
	for _, r := range m.slaves {
		r.makeMove(r.board.makePt(x, y))
	}
	//m.mr.log.Println(m.mr.Debug())
	return x, y, moveResult
}

// splits the work on all slaves
func (m *multirobot) findWinsMulti(numSamples int) (ratio float64) {
	// sync slaves
	m.syncSlaves()
	for i := range m.mr.wins {
		m.mr.wins[i] = 0
		m.mr.hits[i] = 0
	}
	// release the hounds!
	done := make(chan float64)
	for _, slave := range m.slaves {
		go func(r *robot) {
			done <- r.findWins((numSamples / len(m.slaves)) + 1) // at least 1 time
		}(slave)
	}
	// wait
	for i := 0; i < len(m.slaves); i++ {
		ratio += <-done
	}
	ratio /= float64(len(m.slaves))

	// collect results
	for _, slave := range m.slaves {
		for j := range m.mr.hits {
			m.mr.hits[j] += slave.hits[j]
			m.mr.wins[j] += slave.wins[j]
		}
	}
	return ratio
}

func (m *multirobot) genMovesMulti(color Color) (x, y int, result MoveResult) {
	if !m.mr.board.isMyTurn(color) {
		// GTP protocol allows generating a move by either side;
		// treat as if the other player passed.
		if ok, message := m.mr.Play(color.GetOpponent(), 0, 0); !ok {
			panic(fmt.Sprintf("other side cannot pass? %s", message))
		}
	}
	startTime := time.Now()
	m.findWinsMulti(m.mr.sampleCount) // this also syncs slaves
	stopTime := time.Now()
	elapsedTimeSecs := float64(stopTime.Sub(startTime)) / math.Pow10(9)
	m.mr.log.Printf("playouts/second: %.0f", float64(m.mr.sampleCount)/elapsedTimeSecs)

	// find candidate moves
	candidateCount := 0
	for _, pt := range m.mr.board.allPoints {
		if m.mr.hits[pt] > 0 && !m.mr.board.wouldFillEye(pt) && m.mr.checkLegalMove(pt) == played {
			m.mr.candidates[candidateCount] = pt
			candidateCount++
		}
	}

	// sort candidates by win ratio, sample size breaks ties
	// sort in reverse order (greatest value first)
	sortfunc := func(p1, p2 pt) bool {
		p1score := float64(m.mr.wins[p1]) / float64(m.mr.hits[p1])
		p2score := float64(m.mr.wins[p2]) / float64(m.mr.hits[p2])
		if p1score == p2score {
			return m.mr.hits[p1] > m.mr.hits[p2]
		}
		return p1score > p2score
	}
	ptsortfunc(sortfunc).Sort(m.mr.candidates[:candidateCount])
	m.mr.candCount = candidateCount
	return
}

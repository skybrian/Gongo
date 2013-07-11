package gongo

import (
	"testing"
)

func TestMultiRobot(t *testing.T) {
	m, err := newMultiRobot(5)
	if err != nil {
		t.Log("multirobot creation failed")
		t.Error(err)
	}

	assertEqualsInt(t, 5, m.GetBoardSize(), "wrong boardsize")

	c := m.GetCell(3, 3)
	if c != Empty {
		t.Error("board is not empty")
	}

	ok, message := m.Play(Black, 3, 3)
	if !ok {
		t.Error("could not play 3,3")
	}
	if ok && message != "captures: 0" {
		t.Error("expected: captures: 0, got", message)
	}

	ok, message = m.Play(White, 3, 3)
	if ok {
		t.Error("play on occupied point succesful")
	}
	if !ok && message != "occupied" {
		t.Error("expected: occupied, got", message)
	}
	checkBoard(t, m.mr, `
		.....
		.....
		..@..
		.....
		.....`)
	m.SetBoardSize(3)
	checkBoard(t, m.mr, `
		...
		...
		...`)
	m.GenMove(Black)
	checkBoard(t, m.mr, `
		...
		.@.
		...`)
}

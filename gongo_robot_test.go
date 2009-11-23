package gongo

import (
	"testing";
)

func TestBasics(t *testing.T) {
	r := NewRobot(3);
	if r.GetBoardSize() != 3 { t.Error("board is wrong size"); }
}

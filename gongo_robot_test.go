package gongo

import (
	"bytes";
	"strings";
	"testing";
)

func TestNewRobot(t *testing.T) {
	r := NewRobot(3);
	CheckBoard(t, r,
`...
 ...
 ...`);
}


// == end of tests ===

func CheckBoard(t *testing.T, r GoRobot, expectedBoard string) {
	expected := TrimBoard(expectedBoard);
	actual := LoadBoard(r);
	if expected != actual {
		t.Errorf("board is different. Expected:\n%v\nActual:\n%v\n", expected, actual);
	}
}

func TrimBoard(s string) string {
	lines := strings.Split(s, "\n", 0);
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i]);
	}
	return strings.Join(lines, "\n");
}

func LoadBoard(r GoRobot) string {
	var out bytes.Buffer;
	size := r.GetBoardSize();
	for y := size; y >= 1; y-- {
		for x := 1; x <= size; x++ {
			switch r.GetCell(x,y) {
			case Empty: out.WriteString(".");
			case White: out.WriteString("O");
			case Black: out.WriteString("@");
			}
		}
		if y > 1 {
			out.WriteString("\n");
		}
	}
	return out.String();
}
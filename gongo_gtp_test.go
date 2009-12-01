package gongo

import (
	"bytes";
	"regexp";
	"strings";
	"testing";
)

// === GTP driver tests ===

func TestListCommands(t *testing.T) {
	checkCommand(t, nil, "list_commands",
		`boardsize
clear_board
genmove
known_command
komi
list_commands
name
play
protocol_version
quit
showboard
version`)
}

func TestKnownCommand(t *testing.T) {
	checkCommand(t, nil, "known_command version", "true");
	checkCommand(t, nil, "known_command asdf", "false");
	checkCommand(t, nil, "known_command quit", "true");
	checkCommand(t, nil, "known_command known_command", "true");
}

func TestSimpleCommands(t *testing.T) {
	checkCommand(t, nil, "protocol_version", "2");
	checkCommand(t, nil, "name", "gongo");
	checkCommand(t, nil, "version", "");
}

func TestUnknownCommandError(t *testing.T) {
	checkRun(t, nil, "asdf\nquit\n", "? unknown command\n\n= \n\n")
}

func TestQuit(t *testing.T) {
	checkRun(t, nil, "quit\n", "= \n\n");
	checkRun(t, nil, "# comment\n\nquit\n", "= \n\n");
}

func TestBoardSize(t *testing.T) {
	g := NewFakeRobot();
	checkCommand(t, g, "boardsize 9", "");
	if g.board_size != 9 {
		t.Errorf("expected board size %v but got %v", 9, g.board_size)
	}
}

func TestClearBoard(t *testing.T) {
	g := NewFakeRobot();
	checkCommand(t, g, "clear_board", "");
	if !g.board_cleared {
		t.Errorf("board not cleared")
	}
}

func TestKomi(t *testing.T) {
	g := NewFakeRobot();
	checkCommand(t, g, "komi 6.5", "");
	if g.komi != 6.5 {
		t.Errorf("expected komi %v but got %v", 6.5, g.komi)
	}
}

func TestPlay(t *testing.T) {
	g := NewFakeRobot();
	checkCommand(t, g, "play white c10", "");
	if White != g.color {
		t.Error("color mismatch")
	}
	if 3 != g.x {
		t.Error("x mismatch")
	}
	if 10 != g.y {
		t.Error("y mismatch")
	}
}

func TestGenmove(t *testing.T) {
	checkGenmove(t, 3, 10, "C10");
	checkGenmove(t, 8, 4, "H4");
	checkGenmove(t, 9, 4, "J4");
	checkGenmove(t, 10, 4, "K4");
}

func TestGenmove_Passed(t *testing.T) {
	g := NewFakeRobot();
	g.send_moveResult = Passed;
	checkCommand(t, g, "genmove white", "pass");
	if White != g.color {
		t.Errorf("expected %v but got %v", White, g.color)
	}
}

func TestGenmove_Resign(t *testing.T) {
	g := NewFakeRobot();
	g.send_moveResult = Resigned;
	checkCommand(t, g, "genmove white", "resign");
	if White != g.color {
		t.Errorf("expected %v but got %v", White, g.color)
	}
}

func TestShowBoard(t *testing.T) {
	r := NewFakeRobot();
	r.send_boardSize = 5;
	r.send_cell[1][5] = White;
	r.send_cell[5][5] = Black;
	r.send_cell[4][4] = White;
	r.send_cell[5][2] = Black;
	checkCommand(t, r, "showboard",
		`O...@
...O.
.....
....@
.....`);
}

func TestParseColor(t *testing.T) {
	checkColor(t, "b", Black);
	checkColor(t, "w", White);
	checkColor(t, "B", Black);
	checkColor(t, "black", Black);
	checkColor(t, "Black", Black);
	checkColor(t, "WHITE", White);
}

func TestParseVertex(t *testing.T) {
	checkVertex(t, "pass", 0, 0);
	checkVertex(t, "Pass", 0, 0);
	checkVertex(t, "a1", 1, 1);
	checkVertex(t, "H8", 8, 8);
	checkVertex(t, "j9", 9, 9);
	checkVertex(t, "T19", 19, 19);
}

// === end of tests ===

type fake_robot struct {
	board_size	int;
	board_cleared	bool;
	komi		float;
	color		Color;
	x, y		int;
	send_x		int;
	send_y		int;
	send_moveResult	MoveResult;
	send_ok		bool;
	send_boardSize	int;
	send_cell	[MaxBoardSize][MaxBoardSize]Color;
}

func NewFakeRobot() *fake_robot	{ return &fake_robot{send_ok: true} }

func (r *fake_robot) SetBoardSize(value int) bool {
	r.board_size = value;
	return r.send_ok;
}

func (r *fake_robot) ClearBoard()	{ r.board_cleared = true }

func (r *fake_robot) SetKomi(value float)	{ r.komi = value }

func (r *fake_robot) Play(color Color, x, y int) (ok bool, message string) {
	r.color = color;
	r.x = x;
	r.y = y;
	return r.send_ok, "";
}

func (r *fake_robot) GenMove(color Color) (x, y int, result MoveResult) {
	r.color = color;
	return r.send_x, r.send_y, r.send_moveResult;
}

func (r *fake_robot) GetBoardSize() int	{ return r.send_boardSize }

func (r *fake_robot) GetCell(x, y int) Color	{ return r.send_cell[x][y] }

func checkGenmove(t *testing.T, x, y int, expected string) {
	g := NewFakeRobot();
	g.send_x = x;
	g.send_y = y;
	g.send_moveResult = Played;
	checkCommand(t, g, "genmove black", expected);
	if Black != g.color {
		t.Errorf("expected %v but got %v", Black, g.color)
	}
}

func checkColor(t *testing.T, input string, expected Color) {
	actual, ok := ParseColor(input);
	if !ok {
		t.Error("Can't parse color:", input);
		return;
	}
	if expected != actual {
		t.Error("unexpected color for", input)
	}
}

func checkVertex(t *testing.T, input string, expectedX int, expectedY int) {
	x, y, ok := stringToVertex(input);
	if !ok {
		t.Error("Can't parse vertex:", input);
		return;
	}
	if expectedX != x {
		t.Error("unexpected X for", input, "Got:", x)
	}
	if expectedY != y {
		t.Error("unexpected Y for", input, "Got:", y)
	}
}

func checkCommand(t *testing.T, g GoRobot, input, expected string) {
	checkRun(t, g, input+"\nquit\n", "= "+expected+"\n\n= \n\n")
}

func checkRun(t *testing.T, g GoRobot, input, expected string) {
	actual := new(bytes.Buffer);
	var result = Run(g, bytes.NewBufferString(input), actual);
	if expected != actual.String() {
		t.Error("Unexpected response to GTF commands:");
		t.Errorf("input:\n%s\nexpected:\n%s\nactual:\n%s",
			format(input), format(expected), format(actual.String()));
	}
	if result != nil {
		t.Errorf("unexpected error: %s", result.String())
	}
}

var (
	newlines = regexp.MustCompile("\n");
)

func format(in string) string {
	result := newlines.ReplaceAllString(in, "^\n");
	if !strings.HasSuffix(in, "\n") {
		return result + "<-"
	}
	return result;
}

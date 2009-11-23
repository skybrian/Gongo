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
version`);
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
	checkRun(t, nil, "asdf\nquit\n", "? unknown command\n\n= \n\n");
}

func TestQuit(t *testing.T) {
	checkRun(t, nil, "quit\n", "= \n\n");
	checkRun(t, nil, "# comment\n\nquit\n",  "= \n\n");
}

func TestBoardSize(t *testing.T) {
	g := NewFakeRobot();
	checkCommand(t, g, "boardsize 9", "");
	if g.board_size != 9 {
		t.Errorf("expected board size %v but got %v", 9, g.board_size);
	}
}

func TestClearBoard(t *testing.T) {
	g := NewFakeRobot();
	checkCommand(t, g, "clear_board", "");
	if !g.board_cleared {
		t.Errorf("board not cleared");
	}	
}

func TestKomi(t *testing.T) {
	g := NewFakeRobot();
	checkCommand(t, g, "komi 6.5", "");
	if g.komi != 6.5 {
		t.Errorf("expected komi %v but got %v", 6.5, g.komi);
	}	
}

func TestPlay(t *testing.T) {
	g := NewFakeRobot();
	checkCommand(t, g, "play white c10", "");
	expected := Move{White, Vertex{3,10}};
	if !expected.Equals(g.move) {
		t.Errorf("expected %v but got %v", expected, g.move);
	}
}

func TestGenmove(t *testing.T) {
	g := NewFakeRobot();
	g.send_vertex = Vertex{3,10};
	checkCommand(t, g, "genmove black", "C10");
	if Black != g.color {
		t.Errorf("expected %v but got %v", Black, g.color);
	}
}

func TestGenmove_Resign(t *testing.T) {
	g := NewFakeRobot();
	g.send_ok = false;
	checkCommand(t, g, "genmove white", "resign");
	if White != g.color {
		t.Errorf("expected %v but got %v", White, g.color);
	}
}

func TestShowBoard(t *testing.T) {
	b := NewBoard(5);
	b.Set(1, 5, White);
	b.Set(5, 5, Black);
	b.Set(4, 4, White);
	b.Set(5, 2, Black);
	
	g := NewFakeRobot();
	g.send_board = *b;
	checkCommand(t, g, "showboard",
`O...@
...O.
.....
....@
.....`
);
}

func TestBoardToString(t *testing.T) {
	b := NewBoard(3);
	s := b.String();
	if s != "...\n...\n..." {
		t.Error("board printed wrong: " + s);
	}
}

func TestParseMove(t *testing.T) {
	checkMove(t, "b pass", Black, 0, 0);
	checkMove(t, "w Pass", White, 0, 0);
	checkMove(t, "B a1", Black, 1, 1);	
	checkMove(t, "black H8", Black, 8, 8);
	checkMove(t, "b j9", Black, 9, 9);
	checkMove(t, "WHITE T19", White, 19, 19);
}

// === end of tests ===

type fake_game struct {
	board_size int;
	board_cleared bool;
	komi float;
	move Move;
	color Color;
	send_vertex Vertex;
	send_ok bool;
	send_board Board;
}

func NewFakeRobot() *fake_game {
	return &fake_game{send_ok: true};
}

func (g *fake_game) SetBoardSize(value int) bool {
	g.board_size = value;
	return g.send_ok;
}

func (g *fake_game) ClearBoard() {
	g.board_cleared = true;
}

func (g *fake_game) SetKomi(value float) {
	g.komi = value;
}

func (g *fake_game) Play(value Move) bool {
	g.move = value;
	return g.send_ok;
}

func (g *fake_game) GenMove(color Color) (vertex Vertex, ok bool) {
	g.color = color;
	return g.send_vertex, g.send_ok;
}

func (g *fake_game) ShowBoard() Board {
	return g.send_board;
}

func checkMove(t *testing.T, input string, expectedColor Color, expectedX int, expectedY int) {
	actual, ok := ParseMove(input);
	if !ok {
		t.Error("Can't parse move:", input);
		return;
	}
	if expectedColor != actual.Color {
		t.Error("unexpected color for", input);
	}
	if expectedX != actual.Vertex.X {
		t.Error("unexpected X for", input, "Got:", actual.Vertex.X);	
	}
	if expectedY != actual.Vertex.Y {
		t.Error("unexpected Y for", input, "Got:", actual.Vertex.Y);	
	}
}

func checkCommand(t *testing.T, g GoRobot, input, expected string) {
	checkRun(t, g, input + "\nquit\n", "= " + expected + "\n\n= \n\n");
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
		t.Errorf("unexpected error: %s", result.String());
	}
}

var (
	newlines = regexp.MustCompile("\n");
)

func format(in string) string {
	result := newlines.ReplaceAllString(in, "^\n");
	if !strings.HasSuffix(in, "\n") {
		return result + "<-";
	}
	return result;
}

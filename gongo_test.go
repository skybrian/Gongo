package gongo

import (
	"bytes";
	"regexp";
	"strings";
	"testing";
)

func TestSimpleCommands(t *testing.T) {
	checkCommand(t, nil, "protocol_version", "2");	
	checkCommand(t, nil, "name", "gongo");	
	checkCommand(t, nil, "version", "");	
}

func TestKnownCommand(t *testing.T) {
	checkCommand(t, nil, "known_command version", "true");	
	checkCommand(t, nil, "known_command asdf", "false");	
	checkCommand(t, nil, "known_command quit", "true");	
	checkCommand(t, nil, "known_command known_command", "true");	
}

func TestListCommands(t *testing.T) {
	checkCommand(t, nil, "list_commands",
`boardsize
clear_board
known_command
komi
list_commands
name
play
protocol_version
quit
version`);
}

func TestBoardSize(t *testing.T) {
	g := &fake_game{};
	checkCommand(t, g, "boardsize 9", "");
	if g.board_size != 9 {
		t.Errorf("expected board size %v but got %v", 9, g.board_size);
	}
}

func TestClearBoard(t *testing.T) {
	g := &fake_game{};
	checkCommand(t, g, "clear_board", "");
	if !g.board_cleared {
		t.Errorf("board not cleared");
	}	
}

func TestKomi(t *testing.T) {
	g := &fake_game{};
	checkCommand(t, g, "komi 6.5", "");
	if g.komi != 6.5 {
		t.Errorf("expected komi %v but got %v", 6.5, g.komi);
	}	
}

func TestPlay(t *testing.T) {
	g := &fake_game{};
	checkCommand(t, g, "play white c10", "");
	expected := Move{White, Vertex{3,10}};
	if !expected.Equals(g.move) {
		t.Errorf("expected %v but got %v", expected, g.move);
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

func TestUnknownCommandError(t *testing.T) {
	checkRun(t, nil, "asdf\nquit\n", "? unknown command\n\n= \n\n");
}

func TestQuit(t *testing.T) {
	checkRun(t, nil, "quit\n", "= \n\n");
	checkRun(t, nil, "# comment\n\nquit\n",  "= \n\n");
}

// === end of tests ===

type fake_game struct {
	board_size int;
	board_cleared bool;
	komi float;
	move Move;
}

func (g *fake_game) SetBoardSize(value int) bool {
	g.board_size = value;
	return true;
}

func (g *fake_game) ClearBoard() {
	g.board_cleared = true;
}

func (g *fake_game) SetKomi(value float) {
	g.komi = value;
}

func (g *fake_game) Play(value Move) bool {
	g.move = value;
	return true;
}

func checkMove(t *testing.T, input string, expectedColor Color, expectedX int, expectedY int) {
	actual, ok := ParseMove(input);
	if !ok {
		t.Error("Can't parse move:", input);
		return;
	}
	if expectedColor != actual.Color {
		t.Error("unexpected color for %v", input);
	}
	if expectedX != actual.Vertex.X {
		t.Error("unexpected X for %v", input);	
	}
	if expectedY != actual.Vertex.Y {
		t.Error("unexpected Y for %v", input);	
	}
}

func checkCommand(t *testing.T, g Game, input, expected string) {
	checkRun(t, g, input + "\nquit\n", "= " + expected + "\n\n= \n\n");
}

func checkRun(t *testing.T, g Game, input, expected string) {
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

package gongo

import (
	"bytes";
	"bufio";
	"fmt";
	"io";
	"os";
	"regexp";
	"sort";
	"strconv";
	"strings";
)

// The gongo package handles I/O for Go-playing robots written in Go.

// A Go robot is normally implemented as a command-line tool that
// takes commands from a controller on stdin and writes responses to
// stdout. The Go Text Protocol [1] defines how this should be done. 
// A robot that implements GTP can be plugged into various useful tools,
// such as GoGui [2], which provides a user interface.
//
// (The latest version of the GTP spec is a draft; apparently this was
// never finalized, but hasn't changed in a while.)
//
// [1] http://www.lysator.liu.se/~gunnar/gtp/gtp2-spec-draft2/gtp2-spec.html
// [2] http://gogui.sourceforge.net/

// === public API ===

// Executes GTP commands using the specified robot.
// Returns nil after the "quit" command is handled,
// or non nil for an I/O error. 
func Run(robot GoRobot, input io.Reader, out io.Writer) os.Error {
	in := bufio.NewReader(input);
	for {
		command, args, err := parseCommand(in);
		if err != nil { return err; }

		next_handler, ok := handlers[command];
		if !ok {
			fmt.Fprint(out, error("unknown command"));
			continue;
		}

		fmt.Fprint(out, next_handler(request{robot, args}));

		if command == "quit" { break; }
	}
	return nil;
}

// GTP protocol doesn't support larger than 25x25
const MaxBoardSize = 25;


type GoBoard interface {
	// debug support (for showboard)
	GetBoardSize() int;
	GetCell(x, y int) Color;
	
	// Adds a move to the board. Moves can be added for either side in any
	// order, for example to set up a position. If the same player plays twice,
	// it's assumed that the other player passed. The board automatically
	// handle captures.
	// The x and y coordinates start at 1, where x goes from left to right
	// and y from bottom to top. Playing at (0,0) means pass.
	// Returns:
	//   ok - true if the move was accepted or false for an illegal move
	//   message - status or error message, for debugging. May be empty.
	Play(c Color, x, y int) (ok bool, message string);
}

type GoRobot interface {
	// Attempts to change the board size. If the robot doesn't support the
	// new size, return false. (In any case, board sizes above 25 aren't
	// supported by GTP.)
	// The controller should call ClearBoard next, or the results are undefined. 
	SetBoardSize(size int) (ok bool);

	ClearBoard();
	SetKomi(komi float);

	// Asks the robot to generate a move at the current position for the given
	// color. The robot may be asked to play a move for either side.
	// The result is one of Played, Passed, or Resigned.
	GenMove(color Color) (x, y int, result MoveResult);

	GoBoard;
}

// === types used by the GoRobot interface ===

type Color int;
const (
	Empty = Color(0);
	Black = Color(1);
	White = Color(2);
)

func ParseColor(input string) (c Color, ok bool) {
	switch strings.ToLower(input) {
	case "w","white": return White, true;
	case "b","black": return Black, true;
	}
	return Empty, false;
}

func (c Color) GetOpponent() Color {
	switch c {
	case Black: return White;
	case White: return Black;
	}
	panic("can't get opponent for %v", c);
}

func (c Color) String() string {
	switch c {
	case White: return "White";
	case Black: return "Black";
	case Empty: return "Empty";
	}
	panic("invalid color");
}

type MoveResult int;
const (
	Played MoveResult = 0;
	Passed MoveResult = 1;
	Resigned MoveResult = 2;
)

func (m MoveResult) String() string {
	switch m {
	case Played: return "Played";
	case Passed: return "Passed";
	case Resigned: return "Resigned";
	}
	panic("invalid move result");
}

// === driver implementation ===

var word_regexp = regexp.MustCompile("[^  ]+")

func parseCommand(in *bufio.Reader) (cmd string, args []string, err os.Error) {
	for {
		line, err := in.ReadString('\n');
		if err != nil { return "", nil, err; }
		line = strings.TrimSpace(line);
		if line != "" && line[0] != '#' {
			words := word_regexp.AllMatchesString(line, 0);
			return words[0], words[1:len(words)], nil;
		}
	}
	return "", nil, os.NewError("shouldn't get here");
}

type handler func (request) response;

type request struct {
	robot GoRobot;
	args []string;
}

type response struct {
	message string;
	success bool
}

func success(message string) response {
	return response{message, true}
}

func error(message string) response {
	return response{message, false}
}

func (r response) String() string {
	prefix := "=";
	if !r.success { prefix = "?" }
	return prefix + " " + r.message + "\n\n";
}

var (
	// workaround for issue 292
	_known = func(req request) response { return handle_known_command(req) };
	_list = func(req request) response { return handle_list_commands(req) };

	handlers = map[string] handler {
		"boardsize": handle_boardsize,
		"clear_board": func (req request) response { req.robot.ClearBoard(); return success(""); },
		"genmove": handle_genmove,
		"known_command" : _known,
		"komi": handle_komi,
		"list_commands": _list,
		"name" : func(req request) response { return success("gongo") },
		"play": handle_play,
		"protocol_version" : func(req request) response { return success("2") },
		"quit" : func (req request) response { return success("") },
		"showboard" : handle_showboard,
		"version" : func(req request) response { return success("") },

	};
)

func handle_known_command(req request) response {
	if len(req.args) != 1 { return error("wrong number of arguments"); }

	_, ok := handlers[req.args[0]];
	return success(fmt.Sprint(ok));
}

func handle_list_commands(req request) response {
	if len(req.args) != 0 { return error("wrong number of arguments"); }

	names := make([]string, len(handlers));
	i := 0;
	for name := range handlers {
		names[i] = name;
		i++;
	}

	sort.SortStrings(names);
	return success(strings.Join(names, "\n"));
}

func handle_boardsize(req request) response {
	if len(req.args) != 1 { return error("wrong number of arguments"); }

	size, err := strconv.Atoi(req.args[0]);
	if err != nil { return error("unacceptable size"); }

	if !req.robot.SetBoardSize(size) {
		return error("unacceptable size");
	}

	return success("");
}

func handle_komi(req request) response {
	if len(req.args) != 1 { return error("wrong number of arguments"); }
	
	komi, err := strconv.Atof(req.args[0]);
	if err != nil { return error("syntax error"); }
	
	req.robot.SetKomi(komi);
	return success("");
}

func handle_play(req request) response {
	if len(req.args) != 2 { return error("wrong number of arguments"); }

	color, ok := ParseColor(req.args[0]);
	if !ok { return error("syntax error"); }
	
	x, y, ok := stringToVertex(req.args[1]);
	if !ok { return error("syntax error"); }

	ok, _ = req.robot.Play(color, x, y);
	if !ok { return error("illegal move"); }

	return success("");
}

func handle_genmove(req request) (response response) {
	if len(req.args) != 1 { return error("wrong number of arguments"); }

	color, ok := ParseColor(req.args[0]);
	if !ok { return error("syntax error"); }		

	x, y, status := req.robot.GenMove(color);
	switch status {
	case Played:
		message, ok := vertexToString(x, y);
		if ok {	
			response = success(message);
		} else {
			response = error(message);
		}
	case Passed: response = success("pass");
	case Resigned: response = success("resign");
	}

	return;
}

func handle_showboard(req request) response {
	if len(req.args) != 0 { return error("wrong number of arguments"); }
	
	size := req.robot.GetBoardSize();
	buf := &bytes.Buffer{};
	for y := size; y >= 1 ; y-- {
		for x := 1; x <= size; x++ {
			color := req.robot.GetCell(x, y);
			switch color {
			case Empty: buf.WriteString(".");
			case White: buf.WriteString("O");
			case Black: buf.WriteString("@");
			default: panic("shouldn't happen");
			}
		}
		if y > 1 {
			buf.WriteString("\n");
		}
	}
	return success(buf.String());
}

func stringToVertex(input string) (x, y int, ok bool) {
	input = strings.ToUpper(input);
	if len(input) < 2 { return 0, 0, false; }

	if input == "PASS" { return 0, 0, true; }

	x = 1 + int(input[0]) - int('A');
	if (input[0] > 'I') { x--; }
	if x < 1 || x > MaxBoardSize { return 0, 0, false; }

	y, err := strconv.Atoi(input[1:len(input)]); 
	if err != nil || y < 1 || y > MaxBoardSize {
		return 0, 0, false;
	}

	return x, y, true;
}

func vertexToString(x, y int) (result string, ok bool) {
	if x<1 || x>MaxBoardSize || y<1 || y>MaxBoardSize {
		return fmt.Sprintf("invalid: (%v,%v)", x, y), false;
	}
	x_letter := byte(x) - 1 + 'A';
	if x_letter >= 'I' { x_letter--; }
	return fmt.Sprintf("%c%v", x_letter, y ), true;
}
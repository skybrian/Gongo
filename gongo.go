package gongo

import (
	"bufio";
	"fmt";
	"io";
	"os";
	"regexp";
	"sort";
	"strconv";
	"strings";
)

// === public definitions ===

type Game interface {
	SetBoardSize(size int) (ok bool);
	ClearBoard();
	SetKomi(komi float);
	Play(move Move) (ok bool);
}

type Color bool;
const (
	Black = false;
	White = true;
)

func ParseColor(input string) (c Color, ok bool) {
	switch strings.ToLower(input) {
	case "w","white": return White, true;
	case "b","black": return Black, true;
	}
	return false, false;
}

func (c Color) String() string {
	if c {
		return "White";
	}
	return "Black";
}

type Vertex struct {
	X int; // left to right, from 1 (printed as letters starting from 'A', skipping 'I'.)
	Y int; // bottom to top, from 1
}

const (
	x_letters = "_ABCDEFGHJKLMNOPQRSTUVWXYZ"; // no I
)

func ParseVertex(input string) (v Vertex, ok bool) {
	input = strings.ToUpper(input);
	if input == "PASS" {
		return Vertex{}, true;
	}
	if len(input) < 2 { 
		return Vertex{}, false;
	}
	x := strings.Index(x_letters, input[0:1]); 
	if x < 1 {
		return Vertex{}, false;
	}
	y, err := strconv.Atoi(input[1:len(input)]); 
	if err != nil {
		return Vertex{}, false;
	}
	return Vertex{X: x, Y: y}, true;
}

func (v Vertex) IsPass() bool {
	return v.X == 0 && v.Y == 0;
}

func (v Vertex) IsValid(boardSize int) bool {
	return v.IsPass() || (v.X >= 1 && v.X <= boardSize && v.Y >= 1 && v.Y <= boardSize);
}

func (this Vertex) Equals(that Vertex) bool {
	return this.X == that.X && this.Y == that.Y;
}

func (v Vertex) String() string {
	if v.IsPass() {
		return "pass";
	} else if !v.IsValid(25) {
		return fmt.Sprintf("invalid: (%v,%v)", v.X, v.Y);
	}
	return fmt.Sprintf("%c%v", x_letters[v.X], v.Y );
}

type Move struct {
	Color Color;
	Vertex Vertex;
}

var (
	move_regexp = regexp.MustCompile("(w|white|b|black) ([a-z][0-9]+)");
)

func ParseMove(input string) (m Move, ok bool) {
	words := strings.Split(input, " ", 0);
	if len(words) != 2 { return Move{}, false; }
	color, ok := ParseColor(words[0]);
	if !ok { return  Move{}, false; }
	vertex, ok := ParseVertex(words[1]);
	if !ok { return Move{}, false; }
	return Move{color, vertex}, true;
}

func (this Move) Equals(that Move) bool {
	return this.Color == that.Color && this.Vertex.Equals(that.Vertex);
}

func (m Move) String() string {
	return fmt.Sprintf("%v %v", m.Color, m.Vertex);
}

// Reads GTP commands and writes their results.
// Returns nil when a quit command is read, or  non nil on error. 
// For more on the Go Text Protocol, see: 
// http://www.lysator.liu.se/~gunnar/gtp/
func Run(game Game, input io.Reader, out io.Writer) os.Error {
	in := bufio.NewReader(input);
	for {
		command, args, err := parseCommand(in);
		if err != nil {
			return err;
		} else if command == "quit" {
			fmt.Fprint(out, success(""));
			break;						
		}
		next_handler, ok := handlers[command];
		if ok {
			fmt.Fprint(out, next_handler(request{game, args}))
		} else {
			fmt.Fprint(out, error("unknown command"));			
		}
	}
	return nil;
}

// === GTP protocol ===

type handler func (request) response;

type request struct {
	game Game;
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
	if !r.success {
		return "? " + r.message + "\n\n"
	}
	return "= " + r.message + "\n\n"
}

var (
	// workaround for issue 292
	_known = func(req request) response { return handle_known_command(req) };
	_list = func(req request) response { return handle_list_commands(req) };

	handlers = map[string] handler {
		"protocol_version" : func(req request) response { return success("2") },
		"name" : func(req request) response { return success("gongo") },
		"version" : func(req request) response { return success("") },
		"known_command" : _known,
		"list_commands": _list,
		"boardsize": handle_boardsize,
		"clear_board": func (req request) response { req.game.ClearBoard(); return success(""); },
		"komi": handle_komi,
		"play": handle_play,
		"quit" : nil
	};
	word_regexp = regexp.MustCompile("[^  ]+")
)

func handle_known_command(req request) response {
	if len(req.args) != 1 {
		return error("wrong number of arguments");
	}
	_, ok := handlers[req.args[0]];
	if ok {
		return success("true");
	}
	return success("false");
}

func handle_list_commands(req request) response {
	if len(req.args) != 0 {
		return error("wrong number of arguments");
	}
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
	if len(req.args) != 1 {
		return error("wrong number of arguments");
	}
	size, err := strconv.Atoi(req.args[0]);
	if err != nil {
		return error("unacceptable size");
	}
	if !req.game.SetBoardSize(size) {
		return error("unacceptable size");
	}
	return success("");
}

func handle_komi(req request) response {
	if len(req.args) != 1 {
		return error("wrong number of arguments");
	}
	komi, err := strconv.Atof(req.args[0]);
	if err != nil {
		return error("syntax error");
	}
	req.game.SetKomi(komi);
	return success("");
}

func handle_play(req request) response {
	if len(req.args) != 2 {
		return error("wrong number of arguments");
	}
	move, ok := ParseMove(req.args[0] + " " + req.args[1]);
	if !ok {
		return error("syntax error");
	}
	if !req.game.Play(move) {
		return error("illegal move");
	}
	return success("");
}

func parseCommand(in *bufio.Reader) (string, []string, os.Error) {
	for {
		line, err := in.ReadString('\n');
		if err != nil {
			return "", nil, err;
		}
		line = strings.TrimSpace(line);
		if line != "" && line[0] != '#' {
			words := word_regexp.AllMatchesString(line, 0);
			return words[0], words[1:len(words)], nil;
		}
	}
	return "", nil, os.NewError("shouldn't get here");
}

// === The game ===

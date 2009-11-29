package gongo

import (
	"bytes";
	"strings";
	"testing";
)

func TestCaptureAndSuicideRules(t *testing.T) {
	r := NewRobot(3);
	checkBoard(t, r,
`...
 ...
 ...`);
	playLegal(t, r, Black, 1, 1,
`...
 ...
 @..`);
	playLegal(t, r, White, 2, 3,
`.O.
 ...
 @..`);	
	playLegal(t, r, Black, 3, 3,
`.O@
 ...
 @..`);
	playIllegal(t, r, White, 3, 3,
`.O@
 ...
 @..`);
	// check that capturing a single stone works
	playLegal(t, r, White, 3, 2,
`.O.
 ..O
 @..`);
	// verify that suicide is illegal
	playIllegal(t, r, Black, 3, 3,
`.O.
 ..O
 @..`);
	playLegal(t, r, Black, 2, 2,
`.O.
 .@O
 @..`);
	playLegal(t, r, White, 3, 1,
`.O.
 .@O
 @.O`);
	playLegal(t, r, Black, 1, 3,
`@O.
 .@O
 @.O`);
	playLegal(t, r, White, 3, 3,
`@OO
 .@O
 @.O`);
	// check that capturing multiple stones works
	playLegal(t, r, Black, 2, 1,
`@..
 .@.
 @@.`);
}

func TestAllowFillInKo(t *testing.T) {
	r := NewRobot(4);
	setUpBoard(r,
`.@OO
 @.@O
 .@OO
 ....`);
	playLegal(t, r, Black, 2, 3, 
`.@OO
 @@@O
 .@OO
 ....`
	);
}

func TestDisallowSimpleKo(t *testing.T) {
	r := NewRobot(4);
	setUpBoard(r, 
`....
 ....
 .@O.
 @..O`);
	playLegal(t, r, Black, 3, 1,
`....
 ....
 .@O.
 @.@O`);
	playLegal(t, r, White, 2, 1,
`....
 ....
 .@O.
 @O.O`);
	playIllegal(t, r, Black, 3, 1,
`....
 ....
 .@O.
 @O.O`);
}

func TestPlaySameColorTwice(t *testing.T) {
	r := NewRobot(3);
	playLegal(t, r, Black, 1, 1,
`...
 ...
 @..`);
	playLegal(t, r, Black, 2, 1,
`...
 ...
 @@.`);
}

func TestPlayByPassing(t *testing.T) {
	r := NewRobot(3);
	playLegal(t, r, Black, 0, 0,
`...
 ...
 ...`);	
}

// example from: http://senseis.xmp.net/?SendingTwoReturningOne
func TestDisallowPositionalSuperKo(t *testing.T) {
	r := NewRobot(6);
	setUpBoard(r,
`.O.@O.
 @O@@O.
 .@@OO.
 @@O...
 OOO.O.
 ......`);
	playLegal(t, r, Black, 1, 6,
`@O.@O.
 @O@@O.
 .@@OO.
 @@O...
 OOO.O.
 ......`);
	playLegal(t, r, White, 1, 4,
`.O.@O.
 .O@@O.
 O@@OO.
 @@O...
 OOO.O.
 ......`);
	playIllegal(t, r, Black, 1, 5,
`.O.@O.
 .O@@O.
 O@@OO.
 @@O...
 OOO.O.
 ......`);
}

// === move generation tests ===

func TestPassWhenNoMovesLeft(t *testing.T) {
	r := NewRobot(1);
	checkGenPass(t, r, Black, `.`);
}

func TestMakeMoveWhenBoardIsEmpty(t *testing.T) {
	r := NewRobot(2);
	checkGenAnyMove(t, r, Black);
}

func TestMakeMoveWhenSameSidePlayedLast(t *testing.T) {
	r := NewRobot(2);
	playLegal(t, r, Black, 1, 1,
`..
 @.`);
	checkGenAnyMove(t, r, Black);	
}

func TestPassInsteadOfFillingOnePointEyes(t *testing.T) {
	r := NewRobot(3);
	setUpBoard(r,
`.@.
 @.@
 .@.`);
	checkGenPass(t, r, Black, 
`.@.
 @.@
 .@.`);
}

func TestPreferCenter(t *testing.T) {
	r := NewRobot(3);
	checkGenMove(t, r, Black, 
`.,.
 .@.
 ...`);
}

// === test internals ===

func TestGenerateAllSize1Games(t *testing.T) {
	faker := new(fakeRandomness);
	var b board;

	b.clearBoard(1);
	b.playRandomGame(faker);
	checkBoard(t, b, `.`); 
	if faker.next() {
		t.Error("expected only one game");
	}
}

func TestGenerateAllSize2Games(t *testing.T) {
	games, total := generateAllGames(2);
	checkGameCount(t, games, 144,
`@.
 .@`);
	checkGameCount(t, games, 144,
`.@
 @.`);
	checkGameCount(t, games, 64,
`OO
 .O`);
	checkGameCount(t, games, 64,
`OO
 O.`);
	checkGameCount(t, games, 64,
`O.
 OO`);
	checkGameCount(t, games, 64,
`.O
 OO`);
	if total != 544 {
		t.Errorf("number of games changed from 544 to %v", total);
	}
}

// === end of tests ===

func playLegal(t *testing.T, r GoRobot, c Color, x, y int, expectedBoard string) {
	ok := r.Play(c,x,y);
	if !ok {
		t.Errorf("legal move rejected: %v (%v,%v)", c, x, y);
	}
	checkBoard(t, r, expectedBoard);
}

func playIllegal(t *testing.T, r GoRobot, c Color, x, y int, expectedBoard string) {
	ok := r.Play(c,x,y);
	if ok {
		t.Errorf("illegal move not rejected: %v (%v,%v)", c, x, y);
	}
	checkBoard(t, r, expectedBoard);
}

func checkGenPass(t *testing.T, r GoRobot, c Color, expectedBoard string) {
	x, y, result := r.GenMove(c);
	if result != Passed {
		t.Errorf("didn't generate a pass for %v; got %v (%v,%v)", c, result, x, y);
	}
	checkBoard(t, r, expectedBoard);
}

func checkGenMove(t *testing.T, r GoRobot, c Color, expectedBoard string) {
	_, _, result := r.GenMove(c);
	if result != Played {
		t.Errorf("didn't generate a move for %v; got %v", c, result);
	}
	checkBoard(t, r, expectedBoard);
}

func checkGenAnyMove(t *testing.T, r GoRobot, colorToPlay Color) {
	x, y, result := r.GenMove(colorToPlay);
	if result != Played {
		t.Errorf("didn't generate a move for %v; got %v", colorToPlay, result);
		return;
	}
	size := r.GetBoardSize();
	if x<1 || x>size || y<1 || y>size {
		t.Errorf("didn't generate a move on the board; got: (%v,%v)", x, y);
	} else {
		cellColor := r.GetCell(x, y);
		if cellColor != colorToPlay {
			t.Errorf("played cell doesn't contain stone; got: %v", cellColor);
		} 
	}
}

func checkBoard(t *testing.T, b GoBoard, expectedBoard string) {
	expected := trimBoard(expectedBoard);
	actual := loadBoard(b);
	if expected != actual {
		t.Errorf("board is different. Expected:\n%v\nActual:\n%v\n", expected, actual);
	}
}

func trimBoard(s string) string {
	lines := strings.Split(s, "\n", 0);
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i]);
	}
	return strings.Join(lines, "\n");
}

func loadBoard(b GoBoard) string {
	var out bytes.Buffer;
	size := b.GetBoardSize();
	for y := size; y >= 1; y-- {
		for x := 1; x <= size; x++ {
			switch b.GetCell(x,y) {
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

func setUpBoard(r GoRobot, boardString string) {
	r.ClearBoard();
	size := r.GetBoardSize();
	lines := strings.Split(boardString, "\n", 0);
	if len(lines) != size { panic("wrong number of lines"); }
	for rowNum := range lines {
		line := strings.TrimSpace(lines[rowNum]);
		if len(line) != size { panic("line is wrong length"); }
		y := size - rowNum;
		for i,c := range line {
			var ok bool;
			switch c {
			case '@': ok = r.Play(Black, i+1, y);
			case 'O': ok = r.Play(White, i+1, y);
			case '.': ok = true;
			default: panic("invalid character in board");
			}
			if !ok {
				panic("couldn't place stone");
			}
		}
	}	
}

func generateAllGames(size int) (games map[string]int, total int) {

	games = make(map[string]int);

	r := new(fakeRandomness);
	b := new(board);
	
	total = 0;
	for {
		b.clearBoard(size);
		b.playRandomGame(r);
		boardString := loadBoard(b);
		if _,ok := games[boardString]; ok {
			games[boardString]++;
		} else {
			games[boardString] = 1;
		}
		total++;
		if !r.next() {
			break;
		}
	}
	return;
}

func checkGameCount(t *testing.T, games map[string]int, expectedCount int, board string) {
	board = trimBoard(board);
	actual, ok := games[board];
	if !ok {
		t.Errorf("no games found for:\n%v\n", board);
	} else if actual != expectedCount {
		t.Errorf("expected %v but was %v for:\n%v\n", expectedCount, actual, board);
	}
}

// A fake random number generator that can be used to generate all possible
// choices (a depth-first search, like Prolog). The first time through it
// will generate all zeros. When restarted, the sequence will be all zeros
// followed by a 1 at the last output, and so on until all possibilities for
// the last output are chosen. Then the previous output will be incremented,
// and so on, something like an odometer except that the last item can be at
// different depths.

// Invariant: for each index i in outputs, all possible sequences of
// random numbers have been that begin with the prefixes:
//   outputs[0], outputs[1], ... outputs[i-1] + any of [0 .. outputs[i] - 1] 

// This allows for at least 2^64 possible values which is far more than reasonable
const maxOutputs = 64;

type fakeRandomness struct {
	inputs [maxOutputs]int;
	outputs [maxOutputs]int;
	callCount int;
	allCallsCount int;
}

func (r *fakeRandomness) Intn(n int) (result int) {
	if n < 1 {
		panic("illegal argument to Intn");
	}
	if n == 1 {
		r.allCallsCount++;
		return 0; // don't count it when there's only one choice
	}
	r.inputs[r.callCount] = n;
	if r.outputs[r.callCount] >= n {
		panic("can't use fakeRandomness with nondeterministic function")
	}
	result = r.outputs[r.callCount];
	r.callCount++;
	r.allCallsCount++;
	return result;
}

// Resets the fake random number generator in preparation for another
// run. Returns false if all possibilites have been tried. 
func (r *fakeRandomness) next() (hasNext bool) {
	for i := r.callCount - 1; i >= 0; i-- {
		if r.outputs[i] < r.inputs[i] - 1 {
			r.outputs[i]++;
			r.callCount = 0;
			r.allCallsCount = 0;
			return true; // have another possibility to try
		}
		r.outputs[i] = 0;
	}

	// we're done; tried all possibilites
	return false;
}

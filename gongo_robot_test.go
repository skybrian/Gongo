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

// == end of tests ===

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


func checkBoard(t *testing.T, r GoRobot, expectedBoard string) {
	expected := trimBoard(expectedBoard);
	actual := loadBoard(r);
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

func loadBoard(r GoRobot) string {
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
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

func TestDisallowSimpleKo(t *testing.T) {
	r := NewRobot(4);
	playLegal(t, r, Black, 1, 1,
`....
 ....
 ....
 @...`);
	playLegal(t, r, White, 4, 1,
`....
 ....
 ....
 @..O`);
	playLegal(t, r, Black, 2, 2,
`....
 ....
 .@..
 @..O`);
	playLegal(t, r, White, 3, 2,
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
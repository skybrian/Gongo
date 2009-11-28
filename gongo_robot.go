package gongo

// A reimplementation in Go of the Java reference robot by Don Dailey
//
// http://cgos.boardspace.net/public/javabot.zip
// http://groups.google.com/group/computer-go-archive/browse_thread/thread/bda08b9c37f0803e/8cc424b0fb1b6fe0

import (
	"log"
)

// === public API ===

func NewRobot(boardSize int) GoRobot {
	result := new(robot);
	result.board = new(board);
	result.SetBoardSize(boardSize);
	return result;
}

// === implementation ===

const DEBUG = true

// The board format is the same as the Java reference bot,
// a one-dimensional array of integers:
//
//  X axis (from 1) ->
//
//	4 4 4 4 4
//	4 0 0 0
//  4 0 0 0     ^
//  4 0 0 0     |
//  4 4 4 4   Y axis (from 1)
//
//  array index = Y * (board size + 1) + X
//   
//  A1 = (X=1, Y=1) = board size + 2
//
// Neighboring cells can be found by adding a fixed offset to an array index.
// To make board edges easy to detect, the zero row and column aren't used,
// and all cells not in use are filled with EDGE. (The array is made big enough
// to ensure that a diagonal move from the top-right corner won't result in
// going off the end of the array.)

// A cell specified the contents of one position on the board.
type cell int;

const (
	EMPTY cell = 0;
	WHITE cell = 1;
	BLACK cell = 2;
	EDGE cell = 4;

	// A temporary flag that's set on a cell to indicate that it's part of a chain.
	CELL_IN_CHAIN = 64;
)

func colorToCell(c Color) cell {
	switch c {
	case White: return WHITE;
	case Black: return BLACK;
	}
	panic("illegal color: %v", c);
}

func (c cell) toColor() Color {
	switch c {
	case EMPTY: return Empty;
	case WHITE: return White;
	case BLACK: return Black;
	}

    // happens if we pick up an edge or forget to clear the CELL_IN_CHAIN flag
	panic("can't convert cell to color: %s", c);
}

// A pt represents either a point on the Go board or a move to be made on the
// board. When interpreted as a point, it's the index into r.moves[] to get the
// cell at that point. When interpreted as a move, it implies that a stone
// of the current player's color was placed at that cell.
type pt int; 

const (
	// An invalid point that's interpreted as passing as a move.
	PASS pt = 0;

	// A flag used in r.moves to indicate that a move captured exactly one stone.
	// (Used to find simple Ko.)
    ONE_CAPTURE = 1024;

	// converts a move stored in r.moves to an index into r.board
	MOVE_TO_PT_MASK = 1023;
)

const (
	MAX_BOARD_SIZE = 25;
	// includes barrier column, 2 barrier rows, and 1 more for top right diagonal neighbor.
	MAX_CELLS = (MAX_BOARD_SIZE + 1) * (MAX_BOARD_SIZE + 2) + 1; 
)

type board struct {
	boardSize int;
	stride int; // boardSize + 1 to account for barrier column
	dirOffset [4]int; // amount to add to go in each cardinal direction

	cells [MAX_CELLS]cell;

	// A temporary list of pts that are part of a chain being followed. 
	// (Stored here to avoid allocation in loops; it would normally be a local variable.)
	chainPoints []pt; 
}

func (b *board) clearBoard(newSize int) {
	b.boardSize = newSize;
	b.stride = newSize + 1;
	b.dirOffset[0] = 1; // right
	b.dirOffset[1] = -1; // left
	b.dirOffset[2] = b.stride; // up
	b.dirOffset[3] = -b.stride; // down
	
	// fill entire array with board edge
	for i := 0; i < len(b.cells); i++ {
		b.cells[i] = EDGE;
	}
	
	// set all playable points to empty
	for y := 1; y <= b.boardSize; y++ {
		for x := 1; x <= b.boardSize; x++ {
			b.cells[b.makePt(x,y)] = EMPTY;
		}
	}
}

func (b *board) makePt(x,y int) pt {
	return pt(y * b.stride + x);	
}

// Direct translation of capture() function:
    /* ---------------------------------------------------
       capture() - For a given target this method removes
       all stones belonging to target color in same chain
       and returns the number of stones removed.
       --------------------------------------------------- */
// If the group turns out not to be captured, cleans up
// and returns 0.
func (b *board) capture(target pt) int {
	chainColor := b.cells[target];
	chainCount := 0;
	
	b.chainPoints[chainCount] = target; chainCount++;
	b.cells[target] |= CELL_IN_CHAIN;
	
	// Invariants:
	// - Points between 0 and visitIndex-1 have had all their neighbors visited.
	// - Points between visitIndex and chainCount are points that still need to be visited.
	for visitIndex := 0; visitIndex < chainCount; visitIndex++ {
		currentIndex := b.chainPoints[visitIndex];

		for dirIndex := 0; dirIndex < 4; dirIndex++ {
			neighborIndex := currentIndex + pt(b.dirOffset[dirIndex]);
		
			if b.cells[neighborIndex] == 0 {
				// Found a liberty, so this isn't a capture. Revert marks and return.
				for i := 0; i < chainCount; i++ {
					b.cells[ b.chainPoints[i] ] ^= CELL_IN_CHAIN;
				}
				return 0;
			}
		
			if b.cells[neighborIndex] == chainColor {
				// same color and not marked, so add to chain
				b.chainPoints[chainCount] = neighborIndex; chainCount++;
				b.cells[neighborIndex] |= CELL_IN_CHAIN;
			}
	    }
	}
	
	// Postcondition: visited all points in the chain without finding a liberty.
	// Remove the stones from the board.
	for i := 0; i < chainCount; i++ {
		b.cells[ b.chainPoints[i] ] = 0;
	}
	return chainCount;
}


type robot struct {
	*board;

	// list of moves in this game
	moves []pt; 

}

// === implementation of GoRobot interface ===

func (r *robot) SetBoardSize(newSize int) bool {
	r.clearBoard(newSize);

	// assumes no game lasts longer than it would take to fill the board at four times (plus some extra)
	r.moves = make([]pt, 0, len(r.cells) * 4); 
	r.chainPoints = make([]pt, r.boardSize * r.boardSize + 2);

	return true;
}

func (r *robot) ClearBoard() {
	r.SetBoardSize(r.boardSize);
}

func (r *robot) SetKomi(value float) {
}

func (r *robot) Play(color Color, x, y int) bool {
	if x<1 || x>r.boardSize || y<1 || y>r.boardSize || !(color == White || color == Black) {
		return false;
	}
	moveIndex := r.makePt(x, y);

	friendlyStone := cell(2 - (len(r.moves) & 1));
	if (friendlyStone != colorToCell(color)) {
		// GTP protocol allows two moves by same side, but this engine doesn't.
		return false; 
	}

	result := r.makeMove(moveIndex);
	if DEBUG && result > 0 {
		log.Stderrf("captured: %v", result)
	}
	return result >= 0;
}

func (r *robot) GenMove(color Color) (vertex Vertex, ok bool) {
	return Vertex{}, false;
}

func (r *robot) GetBoardSize() int {
	return r.boardSize;
}

func (r *robot) GetCell(x, y int) Color {
	return r.cells[r.makePt(x, y)].toColor();
}

// === internal methods ===

// Direct translation of move function from Java reference bot:
    /* --------------------------------------------------------
       make() - tries to make a move and returns a status code.  

       Does not check for positional superko.
       Does not destroy the position if the move is not legal.

       returns:
         negative value if move is illegal:
           -1 if move is suicide
           -2 if move is simple ko violation
	   -3 if point is occupied

        if legal returns:
            0 - non-capture move
	    n > 0  - number of stones captured
       -------------------------------------------------------- */
func (r *robot) makeMove(move pt) int {
	friendlyStone  := cell(2 - (len(r.moves) & 1));
	enemyStone := friendlyStone ^ 3;
	
	if move == 0 {
		// this is a pass
		r.moves = r.moves[0:len(r.moves)+1];
		r.moves[len(r.moves) - 1] = 0;
		return 0;
	}

	if r.cells[move] != 0 {
		// illegal move: occupied
		if DEBUG { log.Stderrf("disallow occupied"); }
		return -3;
	}
	
	// place stone, but don't commit to it yet
	r.cells[move] = friendlyStone;

	// find any captures and remove them from the board
	captures := 0;
	for direction := 0; direction < 4; direction++ {
	    neighborIndex := move + pt(r.dirOffset[direction]);
	    if (r.cells[neighborIndex] == enemyStone) {
			// see if we can capture the chain with this neighbor
			captures += r.capture(neighborIndex);
	    }
	}
	
	if captures == 0 {
		// check for suicide
		if !r.haveLiberties(move) {
			if DEBUG { log.Stderrf("disallow suicide"); }
			// illegal move; undo and return
			r.cells[move] = EMPTY;
			return -1;
		}
	} else if captures == 1 {
		// check for simple Ko.
		lastMove := r.moves[len(r.moves) - 1];
		if (lastMove & ONE_CAPTURE) != 0 && // previous move captured one stone
			r.cells[lastMove & MOVE_TO_PT_MASK] == EMPTY { // this move captured previous move
			// found a Ko; revert the capture
			r.cells[lastMove & MOVE_TO_PT_MASK] = enemyStone;
			r.cells[move] = EMPTY;
			return -2;
		}
		move = ONE_CAPTURE | move;
	}

	r.moves = r.moves[0:len(r.moves) + 1];
	r.moves[len(r.moves) - 1] = move;
	return captures;
}

// Direct translation of gotLibs function:
    /* -------------------------------------------------
       gotlibs - returns true or false if target stone
       group has at least 1 liberty.    Used for testing
       suicide. 
       ------------------------------------------------- */
func (r *robot) haveLiberties(targetIndex pt) bool {
	color := r.cells[targetIndex];
	chainCount := 0;
	
	r.chainPoints[chainCount] = targetIndex; chainCount++;
	r.cells[targetIndex] |= CELL_IN_CHAIN;
	
	// Invariants:
	// - Points between 0 and visitIndex-1 have had all their neighbors visited.
	// - Points between visitIndex and chainCount are points that still need to be visited.
	for visitIndex := 0; visitIndex < chainCount; visitIndex++ {
		currentIndex := r.chainPoints[visitIndex];

		for dirIndex := 0; dirIndex < 4; dirIndex++ {
			neighborIndex := currentIndex + pt(r.dirOffset[dirIndex]);
		
			if r.cells[neighborIndex] == 0 {
				// unoccupied, so this isn't a suicide. Revert marks and return.
				for i := 0; i < chainCount; i++ {
					r.cells[ r.chainPoints[i] ] ^= CELL_IN_CHAIN;
				}
				return true;
			}
		
			if r.cells[neighborIndex] == color {
				// same color and not marked, so add to chain
				r.chainPoints[chainCount] = neighborIndex; chainCount++;
				r.cells[neighborIndex] |= CELL_IN_CHAIN;
			}
	    }
	}
	
	// Postcondition: visited all points in the chain and didn't find an opening

	// Revert marked positions
	for i := 0; i < chainCount; i++ {
		r.cells[ r.chainPoints[i] ] ^= CELL_IN_CHAIN;
	}
	return false;	
}

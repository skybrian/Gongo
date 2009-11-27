package gongo

import (
	"log"
)

const DEBUG = true

// A reimplementation in Go of the Java reference robot by Don Dailey
//
// http://cgos.boardspace.net/public/javabot.zip
// http://groups.google.com/group/computer-go-archive/browse_thread/thread/bda08b9c37f0803e/8cc424b0fb1b6fe0
//
// The board format is the same as the Java reference bot:
    /*
      note:  The board is a 1 dimension array of integers which
             can be viewd like this (example uses 3x3 board):

	     4 4 4 4 
	     4 0 0 0
	     4 0 0 0
	     4 0 0 0
	     4 4 4 4 4

	     where 0 -> empty legal point
	           1 -> white stone
		   2 -> black stone
		   4 -> board edge

             One extra element appended so that diagonal eye
	     test for lower right corner does not cause array
	     overflow error.
     */
// (Above description copied from Position.java.)

// Note that x=y=1 in the GoRobot interface corresponds to A1 in GTP, which is in the
// lower left, but in the above diagram it's in the upper left.
// (But this doesn't actually matter.)

func NewRobot(boardSize int) GoRobot {
	result := new(robot);
	result.SetBoardSize(boardSize);
	return result;
}

type robot struct {

	boardSize int;
	rowWidth int; // boardSize + 1; accounts for barrier column
	board []int;

	// position of each move in this game; 0=pass, otherwise index into board.
	move []int; 
	moveCount int; // number of moves made so far

	dirOffset []int; // how much to add to an index to move right, left, up, down

	// A temporary list of indexes into board[] that are part of a chain being followed. 
	// (Stored here to avoid allocation in loops; it would normally be a local variable.)
	chainIndex []int; 
}

const (
	// The bit in the board[] array that's temporarily flipped to keep track of visited positions
	// while finding all the positions in a chain
	IN_CHAIN_FLAG = 64;

	// The bit in the move[] array that's flipped to indicate that exactly one stone
	// was captured in that move. (Used to find simple Ko.)
    CAPTURED_ONE_FLAG = 1024;
	MOVE_INDEX_MASK = 1023; // used to get just the index from a cell in the move array.
)

func (r *robot) SetBoardSize(newSize int) bool {
	r.boardSize = newSize;
	r.rowWidth = newSize + 1;
	rowCount := newSize + 2;
	r.board = make([]int, r.rowWidth * rowCount + 1);
	
	// fill entire array with board edge
	for i := 0; i < len(r.board); i++ {
		r.board[i] = 4;
	}
	
	// set all playable points to unoccupied
	for y := 1; y <= r.boardSize; y++ {
		for x := 1; x <= r.boardSize; x++ {
			r.board[y * r.rowWidth + x] = 0;
		}
	}

	// assumes no game lasts longer than it would take to fill the board at four times (plus some extra)
	r.move = make([]int, len(r.board) * 4); 
	r.moveCount = 0;

	r.dirOffset = make([]int, 4);
	r.dirOffset[0] = 1; // right
	r.dirOffset[1] = -1; // left
	r.dirOffset[2] = r.rowWidth; // up
	r.dirOffset[3] = -r.rowWidth; // down

	r.chainIndex = make([]int, r.boardSize * r.boardSize + 2);

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
	moveIndex := r.coordsToIndex(x, y);

	friendlyStone := 2 - (r.moveCount & 1);
	if (friendlyStone != colorToStone(color)) {
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
	return stoneToColor(r.board[r.coordsToIndex(x, y)]);
}

// === end of public methods ===

// input is 1-based indexes (as passed to GoRobot)
func (r *robot) coordsToIndex(x,y int) int {
	return y * r.rowWidth + x;	
}

func colorToStone(c Color) int {
	switch c {
	case White: return 1;
	case Black: return 2;
	}
	panic("illegal color: %v", c);
}

func stoneToColor(c int) Color {
	switch c {
	case 0: return Empty;
	case 1: return White;
	case 2: return Black;
	}
	panic("illegal board value: %s", c);
}

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
func (r *robot) makeMove(moveIndex int) int {
	friendlyStone := 2 - (r.moveCount & 1);
	enemyStone := friendlyStone ^ 3;
	
	if moveIndex == 0 {
		// this is a pass
		r.move[r.moveCount] = 0;
		r.moveCount++;
		return 0;
	}

	if r.board[moveIndex] != 0 {
		// illegal move: occupied
		if DEBUG { log.Stderrf("disallow occupied"); }
		return -3;
	}
	
	// place stone, but don't commit to it yet
	r.board[moveIndex] = friendlyStone;

	// handle any captures
	captures := 0;
	for direction := 0; direction < 4; direction++ {
	    neighborIndex := moveIndex + r.dirOffset[direction];
	    if (r.board[neighborIndex] == enemyStone) {
			// see if we can capture the chain with this neighbor
			captures += r.capture(neighborIndex);
	    }
	}

	if captures == 0 {
		// check for suicide
		if !r.haveLiberties(moveIndex) {
			if DEBUG { log.Stderrf("disallow suicide"); }
			// illegal move; undo and return
			r.board[moveIndex] = 0;
			return -1;
		}
		r.move[r.moveCount] = moveIndex; r.moveCount++;
	} else if captures == 1 {
		// check for simple Ko.
		lastMove := r.move[r.moveCount - 1];
		if (lastMove & CAPTURED_ONE_FLAG) != 0 && // previous move captured one stone
			r.board[lastMove & MOVE_INDEX_MASK] == 0 { // this move captured previous move
			// found; revert this move
			r.board[lastMove & MOVE_INDEX_MASK] = enemyStone;
			r.board[moveIndex] = 0;
			return -2;
		}
		r.move[r.moveCount] = CAPTURED_ONE_FLAG | moveIndex; r.moveCount++;
	} else {
		r.move[r.moveCount] = moveIndex; r.moveCount++;
	}

	return captures;
}

// Direct translation of capture() function:
    /* ---------------------------------------------------
       capture() - For a given target this method removes
       all stones belonging to target color in same chain
       and returns the number of stones removed.
       --------------------------------------------------- */
// If the group turns out not to be captured, cleans up
// and returns 0.
func (r *robot) capture(targetIndex int) int {
	color := r.board[targetIndex];
	chainCount := 0;
	
	r.chainIndex[chainCount] = targetIndex; chainCount++;
	r.board[targetIndex] |= IN_CHAIN_FLAG;
	
	// Invariants:
	// - Points between 0 and visitIndex-1 have had all their neighbors visited.
	// - Points between visitIndex and chainCount are points that still need to be visited.
	for visitIndex := 0; visitIndex < chainCount; visitIndex++ {
		currentIndex := r.chainIndex[visitIndex];

		for dirIndex := 0; dirIndex < 4; dirIndex++ {
			neighborIndex := currentIndex + r.dirOffset[dirIndex];
		
			if r.board[neighborIndex] == 0 {
				// Found a liberty, so this isn't a capture. Revert marks and return.
				for i := 0; i < chainCount; i++ {
					r.board[ r.chainIndex[i] ] ^= IN_CHAIN_FLAG;
				}
				return 0;
			}
		
			if r.board[neighborIndex] == color {
				// same color and not marked, so add to chain
				r.chainIndex[chainCount] = neighborIndex; chainCount++;
				r.board[neighborIndex] |= IN_CHAIN_FLAG;
			}
	    }
	}
	
	// Postcondition: visited all points in the chain without finding a liberty.
	// Remove the stones from the board.
	for i := 0; i < chainCount; i++ {
		r.board[ r.chainIndex[i] ] = 0;
	}
	return chainCount;
}

// Direct translation of gotLibs function:
    /* -------------------------------------------------
       gotlibs - returns true or false if target stone
       group has at least 1 liberty.    Used for testing
       suicide. 
       ------------------------------------------------- */
func (r *robot) haveLiberties(targetIndex int) bool {
	color := r.board[targetIndex];
	chainCount := 0;
	
	r.chainIndex[chainCount] = targetIndex; chainCount++;
	r.board[targetIndex] |= IN_CHAIN_FLAG;
	
	// Invariants:
	// - Points between 0 and visitIndex-1 have had all their neighbors visited.
	// - Points between visitIndex and chainCount are points that still need to be visited.
	for visitIndex := 0; visitIndex < chainCount; visitIndex++ {
		currentIndex := r.chainIndex[visitIndex];

		for dirIndex := 0; dirIndex < 4; dirIndex++ {
			neighborIndex := currentIndex + r.dirOffset[dirIndex];
		
			if r.board[neighborIndex] == 0 {
				// unoccupied, so this isn't a suicide. Revert marks and return.
				for i := 0; i < chainCount; i++ {
					r.board[ r.chainIndex[i] ] ^= IN_CHAIN_FLAG;
				}
				return true;
			}
		
			if r.board[neighborIndex] == color {
				// same color and not marked, so add to chain
				r.chainIndex[chainCount] = neighborIndex; chainCount++;
				r.board[neighborIndex] |= IN_CHAIN_FLAG;
			}
	    }
	}
	
	// Postcondition: visited all points in the chain and didn't find an opening

	// Revert marked positions
	for i := 0; i < chainCount; i++ {
		r.board[ r.chainIndex[i] ] ^= IN_CHAIN_FLAG;
	}
	return false;	
}

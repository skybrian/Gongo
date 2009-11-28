package gongo

// A reimplementation in Go of the Java reference robot by Don Dailey
//
// http://cgos.boardspace.net/public/javabot.zip
// http://groups.google.com/group/computer-go-archive/browse_thread/thread/bda08b9c37f0803e/8cc424b0fb1b6fe0

import (
	"log";
	"rand";
)

const DEBUG = true;

// === public API ===

type Randomness interface {
	Intn(n int) int;
}

var defaultRandomness = rand.New(rand.NewSource(1));

type Config struct {
	BoardSize int;
	Randomness Randomness;
}

func NewRobot(boardSize int) GoRobot {
	config := Config{BoardSize: boardSize, Randomness: defaultRandomness};
	return NewConfiguredRobot(config);
}

func NewConfiguredRobot(config Config) GoRobot {
	result := new(robot);
	result.board = new(board);
	result.randomness = config.Randomness;
	result.SetBoardSize(config.BoardSize);
	return result;	
}

// === implementation of a Go board ===

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

	// A flag on a cell indicating that it's part of the current chain.
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

    // might happens if we pick up an edge or forget to clear CELL_IN_CHAIN
	panic("can't convert cell to color: %s", c);
}

// A pt represents either a point on the Go board or a player's move. When
// interpreted as a point, it's the index into b.cells[] that returns a cell
// representing the stone at that point, if any. When interpreted as a move,
// it represents a move by the current player by placing a stone at that point.
type pt int; 

const (
	// An invalid point. Interpreted as a move, this means the player passes.
	PASS pt = 0;

	// A flag on a recorded move indicating that the move captured exactly one stone.
	// (Used in r.moves to find simple Kos.)
    ONE_CAPTURE = 1024;

	// A mask to remove the ONE_CAPTURE flag from a move, resulting in a point.
	MOVE_TO_PT_MASK = 1023;
)

type board struct {
	boardSize int;
	stride int; // boardSize + 1 to account for barrier column
	dirOffset [4]pt; // amount to add to a pt to move in each cardinal direction

	cells []cell;
	boardPoints []pt; // List of all points on the board.
	
	// Temporary variables to avoid GC:

	chainPoints []pt; // return value of markSurroundedChain
}

func (b *board) clearBoard(newSize int) {
	b.boardSize = newSize;
	b.stride = newSize + 1;
	b.dirOffset[0] = pt(1); // right
	b.dirOffset[1] = pt(-1); // left
	b.dirOffset[2] = pt(b.stride); // up
	b.dirOffset[3] = pt(-b.stride); // down
	
	rowCount := newSize + 2;
	b.cells = make([]cell, rowCount * b.stride + 1); // 1 extra for diagonal move to edge
	b.boardPoints = make([]pt, b.boardSize * b.boardSize);

	// fill entire array with board edge
	for i := 0; i < len(b.cells); i++ {
		b.cells[i] = EDGE;
	}
	
	// set all playable points to empty
	boardPointsAdded := 0;
	for y := 1; y <= b.boardSize; y++ {
		for x := 1; x <= b.boardSize; x++ {
			thisPt := b.makePt(x,y);
			b.cells[thisPt] = EMPTY;
			b.boardPoints[boardPointsAdded] = thisPt;
			boardPointsAdded++; 
		}
	}

	b.chainPoints = make([]pt, len(b.boardPoints));
}

func (b *board) makePt(x,y int) pt {
	return pt(y * b.stride + x);	
}

func (b *board) getCoords(p pt) (x,y int) {
	y = int(p) / b.stride;
	x = int(p) % b.stride;
	return;
}

// Given any point in a chain with no liberties, marks all the cells in
// the chain with CELL_IN_CHAIN and adds those points to chainPoints.
// Returns the number of points found. If the chain is not surrounded,
// does nothing and returns 0.
// Precondition: the target point is occupied and all cells have the
// CELL_IN_CHAIN flag cleared.
func (b *board) markSurroundedChain(target pt) (chainCount int) {
	chainCount = 0;
	chainColor := b.cells[target];
	
	b.chainPoints[chainCount] = target; chainCount++;
	b.cells[target] |= CELL_IN_CHAIN;
	
	// Visit each point, verify that has no liberties, and add its neighbors to the
	// end of chainPoints.
	// Loop invariants:
	// - Points between 0 and visitedCount-1 are surroundded and their same-color
	// neighbors are in chainPoints.
	// - Points between visitedCount and chainCount are known to be in the chain
	// but still need to be visited.
	for visitedCount := 0; visitedCount < chainCount; visitedCount++ {
		thisPt := b.chainPoints[visitedCount];
		for direction := 0; direction < 4; direction++ {
			neighborPt := thisPt + b.dirOffset[direction];
		
			if b.cells[neighborPt] == EMPTY {
				// Found a liberty. Revert marks and return.
				for i := 0; i < chainCount; i++ {
					b.cells[ b.chainPoints[i] ] ^= CELL_IN_CHAIN;
				}
				return 0;
			}
		
			if b.cells[neighborPt] == chainColor {
				// add unvisited same-color neighbor to chain
				// (if it were visited, the comparison would fail)
				b.chainPoints[chainCount] = neighborPt; 
				b.cells[neighborPt] |= CELL_IN_CHAIN;
				chainCount++;
			}
	    }
	}
	
	return chainCount;
}

// Given any point in a chain with no liberties, removes all stones in the
// chain from the board and returns the number of stones removed. Given a
// point in a chain that has liberties, does nothing and returns 0.
// Precondition: same as b.markSurroundedChain
func (b *board) capture(target pt) (chainCount int) {
	chainCount = b.markSurroundedChain(target);

	// Remove the stones from the board
	for i := 0; i < chainCount; i++ {
		b.cells[ b.chainPoints[i] ] = EMPTY;
	}

	return chainCount;
}

// Given any occupied point, returns true if it has any liberties.
// (Used for testing suicide.)
// Precondition: same as b.markSurroundedChain
func (b *board) hasLiberties(target pt) bool {
	chainCount := b.markSurroundedChain(target);
	if chainCount == 0 {
		return true;
	}

	// Revert marked positions
	for i := 0; i < chainCount; i++ {
		b.cells[ b.chainPoints[i] ] ^= CELL_IN_CHAIN;
	}
	return false;	
}


type robot struct {
	*board;

	// list of moves in this game
	moves []pt; 
	moveCount int;

	candidates []pt; // moves to choose from; used in GenMove.
	randomness Randomness;
}

// === implementation of GoRobot interface ===

func (r *robot) SetBoardSize(newSize int) bool {
	r.clearBoard(newSize);

	// assumes no game lasts longer than it would take to fill the board at four times (plus some extra)
	r.moves = make([]pt, len(r.cells) * 4); 

	r.candidates = make([]pt, len(r.boardPoints));

	return true;
}

func (r *robot) ClearBoard() {
	r.SetBoardSize(r.boardSize);
}

func (r *robot) SetKomi(value float) {
}

func (r *robot) Play(color Color, x, y int) bool {
	if (x<1 || y <1) && !(x==0 && y==0) {
		return false;
	}
	if x>r.boardSize || y>r.boardSize || !(color == White || color == Black) {
		return false;
	}

	friendlyStone := cell(2 - (r.moveCount & 1));
	if (friendlyStone != colorToCell(color)) {
		// GTP protocol allows two moves by same side; treat as if the
		// other player passed.
		ok  := r.Play(color.GetOpponent(), 0, 0);
		if !ok {
			return false;
		}
	}
	
	result := r.makeMove(r.makePt(x, y));
	if DEBUG && result > 0 {
		log.Stderrf("captured: %v", result)
	}
	return result >= 0;
}

func (r *robot) GenMove(color Color) (x, y int, result MoveResult) {
	// find unoccupied points
	candidateCount := 0;
	for _, thisPt := range r.boardPoints {
		if r.cells[thisPt] == EMPTY {
			r.candidates[candidateCount] = thisPt;
			candidateCount++;
		}
	}
	
	// try each move at random
	for triedCount := 0; triedCount < candidateCount; triedCount++ {
		// choose random move
		swapIndex := triedCount + r.randomness.Intn(candidateCount - triedCount);
		thisPt := r.candidates[triedCount];
		if swapIndex > triedCount {
			r.candidates[triedCount] = r.candidates[swapIndex];
			r.candidates[swapIndex] = thisPt;
			thisPt = r.candidates[triedCount];
		}
		if r.isLegalMove(thisPt) {
			r.makeMove(thisPt);
			x, y = r.getCoords(thisPt);
			result = Played;
			return;
		}
	}

	// no move found
	return 0, 0, Passed;
}

func (r *robot) GetBoardSize() int {
	return r.boardSize;
}

func (r *robot) GetCell(x, y int) Color {
	return r.cells[r.makePt(x, y)].toColor();
}

// === internal methods ===

func (r *robot) isLegalMove(move pt) (result bool) {
	if r.cells[move] != EMPTY {
		return false;
	}
	friendlyStone := cell(2 - (r.moveCount & 1));
	r.cells[move] = friendlyStone;
	result = r.hasLiberties(move);
	r.cells[move] = EMPTY;
	return result;
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
func (r *robot) makeMove(move pt) int {
	friendlyStone  := cell(2 - (r.moveCount & 1));
	enemyStone := friendlyStone ^ 3;
	
	if move == PASS {
		r.moves[r.moveCount] = PASS;
		r.moveCount++;
		return 0;
	}

	if r.cells[move] != EMPTY {
		// illegal move: occupied
		if DEBUG { log.Stderrf("disallow occupied"); }
		return -3;
	}
	
	// place stone
	r.cells[move] = friendlyStone;
	
	// find any captures and remove them from the board
	captures := 0;
	for direction := 0; direction < 4; direction++ {
	    neighborPt := move + r.dirOffset[direction];
	    if (r.cells[neighborPt] == enemyStone) {
			captures += r.capture(neighborPt);
	    }
	}
	
	if captures == 0 {
		// check for suicide
		if !r.hasLiberties(move) {
			if DEBUG { log.Stderrf("disallow suicide"); }
			// illegal move; undo and return
			r.cells[move] = EMPTY;
			return -1;
		}
	} else if captures == 1 {
		// check for simple Ko.
		lastMove := r.moves[r.moveCount - 1];
		if (lastMove & ONE_CAPTURE) != 0 && // previous move captured one stone
			r.cells[lastMove & MOVE_TO_PT_MASK] == EMPTY { // this move captured previous move
			// found a Ko; revert the capture
			r.cells[lastMove & MOVE_TO_PT_MASK] = enemyStone;
			r.cells[move] = EMPTY;
			return -2;
		}
		move = ONE_CAPTURE | move;
	}

	r.moves[r.moveCount] = move;
	r.moveCount++;
	return captures;
}

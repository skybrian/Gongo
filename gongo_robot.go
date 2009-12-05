package gongo

// A reimplementation in Go of the Java reference robot by Don Dailey
//
// http://cgos.boardspace.net/public/javabot.zip
// http://groups.google.com/group/computer-go-archive/browse_thread/thread/bda08b9c37f0803e/8cc424b0fb1b6fe0

import (
	"fmt";
	"log";
	"math";
	"os";
	"rand";
	"time";
)

// === Public API ===

type Randomness interface {
	Intn(n int) int;
}

var defaultRandomness = rand.New(rand.NewSource(time.Nanoseconds()))

type Config struct {
	BoardSize	int;
	SampleCount	int;	// number of random samples to take to estimate each move
	Randomness	Randomness;
	Log		*log.Logger;
}

func NewRobot(boardSize int) GoRobot {
	return NewConfiguredRobot(Config{BoardSize: boardSize})
}

func NewConfiguredRobot(config Config) GoRobot {
	result := new(robot);
	result.board = new(board);
	result.scratchBoard = new(board);

	if config.BoardSize > 0 {
		result.SetBoardSize(config.BoardSize)
	} else {
		result.SetBoardSize(9)
	}
	if config.SampleCount < 0 {
		result.sampleCount = config.SampleCount
	} else {
		result.sampleCount = 1000
	}
	if config.Randomness != nil {
		result.randomness = config.Randomness
	} else {
		result.randomness = defaultRandomness
	}
	if config.Log != nil {
		result.log = config.Log
	} else {
		result.log = log.New(os.Stderr, nil, "[gongo]", log.Ltime)
	}
	return result;
}

// === Implementation of a Go board ===

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
type cell int

const (
	EMPTY	cell	= 0;
	WHITE	cell	= 1;
	BLACK	cell	= 2;
	EDGE	cell	= 4;

	// A flag on a cell indicating that it's part of the current chain.
	CELL_IN_CHAIN	= 64;
)

func colorToCell(c Color) cell {
	switch c {
	case White:
		return WHITE
	case Black:
		return BLACK
	}
	panic("illegal color: %v", c);
}

func (c cell) toColor() Color {
	switch c {
	case EMPTY:
		return Empty
	case WHITE:
		return White
	case BLACK:
		return Black
	}

	// might happens if we pick up an edge or forget to clear CELL_IN_CHAIN
	panic("can't convert cell to color: %s", c);
}

// A pt represents either a point on the Go board or a player's move. When
// interpreted as a point, it's the index into b.cells[] that returns a cell
// representing the stone at that point, if any. When interpreted as a move,
// it represents a move by the current player by placing a stone at that point.
type pt int

const (
	// An invalid point. Interpreted as a move, this means the player passes.
	PASS	pt	= 0;

	// A flag on a recorded move indicating that the move captured exactly one stone.
	// (Used in r.moves to find simple Kos.)
	ONE_CAPTURE	= 1024;

	// A mask to remove the ONE_CAPTURE flag from a move, resulting in a point.
	MOVE_TO_PT_MASK	= 1023;
)

type moveResult int

const (
	played	moveResult	= iota;
	passed;
	occupied;
	suicide;
	ko;
	superko;
)

func (m moveResult) ok() bool	{ return m == played || m == passed }

func (m moveResult) String() string {
	switch m {
	case played:
		return "played"
	case passed:
		return "passed"
	case occupied:
		return "occupied"
	case suicide:
		return "suicide"
	case ko:
		return "ko"
	case superko:
		return "superko"
	}
	panic("invalid moveResult");
}

func (m moveResult) toPlayResult(captures int) (bool, string) {
	if m == played {
		return true, fmt.Sprintf("captures: %v", captures)
	}
	return m.ok(), m.String();
}

type board struct {
	size		int;
	stride		int;	// boardSize + 1 to account for barrier column
	dirOffset	[4]pt;	// amount to add to a pt to move in each cardinal direction
	diagOffset	[4]pt;	// amount to add to a pt to move in each diagonal direction

	cells		[]cell;
	allPoints	[]pt;	// List of all points on the board. (Skips barrier cells.)

	// List of moves in this game
	moves		[]pt;
	moveCount	int;
	commonMoveCount	int;	// used to avoid recopying moves between boards

	// Scratch variables, reused to avoid GC:
	chainPoints	[]pt;	// return value of markSurroundedChain
	candidates	[]pt;	// moves to choose from; used in playRandomGame.
}

func (b *board) clearBoard(newSize int) {
	b.size = newSize;
	b.stride = newSize + 1;
	b.dirOffset[0] = pt(1);			// right
	b.dirOffset[1] = pt(-1);		// left
	b.dirOffset[2] = pt(b.stride);		// up
	b.dirOffset[3] = pt(-b.stride);		// down
	b.diagOffset[0] = pt(b.stride - 1);	// nw
	b.diagOffset[1] = pt(b.stride + 1);	// ne
	b.diagOffset[2] = pt(-b.stride - 1);	// sw
	b.diagOffset[3] = pt(-b.stride + 1);	// se

	rowCount := newSize + 2;
	b.cells = make([]cell, rowCount*b.stride+1);	// 1 extra for diagonal move to edge
	b.allPoints = make([]pt, b.size*b.size);

	// fill entire array with board edge
	for i := 0; i < len(b.cells); i++ {
		b.cells[i] = EDGE
	}

	// set all playable points to empty and fill allPoints list
	pointsAdded := 0;
	for y := 1; y <= b.size; y++ {
		for x := 1; x <= b.size; x++ {
			pt := b.makePt(x, y);
			b.cells[pt] = EMPTY;
			b.allPoints[pointsAdded] = pt;
			pointsAdded++;
		}
	}

	// assumes no game lasts longer than it would take to fill the board at four times (plus some extra)
	b.moves = make([]pt, len(b.cells)*4);
	b.moveCount = 0;
	b.commonMoveCount = 0;

	b.chainPoints = make([]pt, len(b.allPoints));
	b.candidates = make([]pt, len(b.allPoints));
}

func (b board) GetBoardSize() int	{ return b.size }

func (b board) GetCell(x, y int) Color	{ return b.cells[b.makePt(x, y)].toColor() }

// Simple version of Play() for working with a board directly in tests.
// Doesn't check superko or update r.boardHashes
func (b *board) Play(color Color, x, y int) (ok bool, message string) {
	if !b.checkPlayArgs(color, x, y) {
		return false, "invalid args"
	}

	if !b.isMyTurn(color) {
		// assume the other player passed
		if ok, message := b.Play(color.GetOpponent(), 0, 0); !ok {
			return false, "other side cannot pass? (" + message + ")"
		}
	}

	result, captures := b.makeMove(b.makePt(x, y));
	return result.toPlayResult(captures);
}

func (b *board) checkPlayArgs(color Color, x, y int) bool {
	if color != White && color != Black {
		return false
	}
	if x == 0 && y == 0 {
		return true
	}
	return x > 0 && y > 0 && x <= b.size && y <= b.size;
}

func (b *board) isMyTurn(c Color) bool	{ return b.getFriendlyStone() == colorToCell(c) }

func (b *board) makePt(x, y int) pt	{ return pt(y*b.stride + x) }

func (b *board) getCoords(p pt) (x, y int) {
	y = int(p) / b.stride;
	x = int(p) % b.stride;
	return;
}

// Returns a cell with the correct color stone for the current player's next move
func (b *board) getFriendlyStone() cell	{ return cell(2 - (b.moveCount & 1)) }

// Returns a hash of the current board position, useful for determining whether
// we repeated a board position.
// Based on the hash() function from the Java reference bot:
/* ------------------------------------------------------------
   get a hash of current position - calculating from scratch

   Note: this is DJB hash which was designed for 32 bits even
   though we are using it as a 64 bit hash

   Should be using the superior zobrist hash but I'm lazy,
   this is easier, and performance is not an issue the way it's
   used here.
   ------------------------------------------------------------ */
func (b *board) getHash() int64 {
	var k int64 = 5381;
	for _, pt := range b.allPoints {
		k = ((k << 5) + k) + int64(b.cells[pt])
	}
	return k;
}

// Copies the board and move list from another board of the same size.
// Restriction: the same board must be passed to copyFrom() each time,
// and the other board's move list can only be appended to between copies.
func (b *board) copyFrom(other *board) {
	if b.size != other.size {
		panic("boards must be same size")
	}
	for _, pt := range b.allPoints {
		b.cells[pt] = other.cells[pt]
	}

	// top off move list; assumes other board may have appended some moves
	for i := b.commonMoveCount; i < other.moveCount; i++ {
		b.moves[i] = other.moves[i]
	}
	b.moveCount = other.moveCount;
	b.commonMoveCount = other.moveCount;
}

// Fill the board with a randomly-generated game
func (b *board) playRandomGame(rand Randomness) {
	maxMoves := len(b.allPoints) * 3;

captured:
	for {
		// fill candidates list with unoccupied points
		candCount := 0;
		for _, pt := range b.allPoints {
			if b.cells[pt] == EMPTY {
				b.candidates[candCount] = pt;
				candCount++;
			}
		}

		// Loop invariants:
		// candidates between 0 up to playedCount are non-empty
		// candidates from playedCount to candCount are empty

		// Each time through the played loop:
		// - One move is made (possibly a pass)
		// - moveCount increases by 1
		// - Either playedCount or passedCount increases by 1.

		playedCount := 0;
		passedCount := 0;
	played:
		for b.moveCount < maxMoves {

			// try to play each candidate, in random order
			for i := playedCount; i < candCount; i++ {

				// choose random move from remaining candidates
				randomIndex := i + rand.Intn(candCount-i);
				randomPt := b.candidates[randomIndex];
				// swap next candidate with randomly chosen candidate
				b.candidates[randomIndex], b.candidates[i] = b.candidates[i], randomPt;

				// make the move if we can
				if !b.wouldFillEye(randomPt) {
					result, captures := b.makeMove(randomPt);
					if result == played {
						if captures > 0 {
							// capturing invalidates the candidate list, so restart
							continue captured
						} else {
							playedCount++;
							passedCount = 0;
							continue played;
						}
					}
				}
			}
			// pass because none of the candidates are suitable
			b.makeMove(PASS);
			passedCount++;
			if passedCount == 2 {
				return	// game over
			}
		}
		// Prevent infinite loop by forcing the game to stop.
		// (Possible because we're not checking for superko.)
		return;
	}
}

// Returns the number of black points minus the number of white points,
// assuming the game has been played to the end where all empty points
// are surrounded. (Doesn't include komi.)
func (b *board) getEasyScore() int {
	// 0=unused, 1=white, 2=black
	// 3=surrounded by both (no score; missed point under area scoring)
	var cellCounts [4]int;

	for _, pt := range b.allPoints {
		switch cell := b.cells[pt]; cell {
		case BLACK, WHITE:
			cellCounts[cell]++
		case EMPTY:
			// Find which neighbors are present by OR-ing the cells together.
			// (This works because WHITE and BLACK are single bits and 3 is
			// not used on the board.)
			neighborBits := 0;
			for direction := 0; direction < 4; direction++ {
				neighborCell := b.cells[pt+b.dirOffset[direction]];
				neighborBits = neighborBits | int(neighborCell);
			}
			cellCounts[neighborBits&3]++;
		}
	}
	return cellCounts[BLACK] - cellCounts[WHITE];
}

// A fast version of makeMove() that's good enough for playouts.
// If the given move is legal, update the board, and return true along
// with the number of captures. Otherwise, do nothing and return false.
// Doesn't check superko or update boardHashes.
func (b *board) makeMove(move pt) (result moveResult, captures int) {
	friendlyStone := cell(2 - (b.moveCount & 1));
	enemyStone := friendlyStone ^ 3;

	if move == PASS {
		b.moves[b.moveCount] = PASS;
		b.moveCount++;
		return passed, 0;
	}

	if b.cells[move] != EMPTY {
		return occupied, 0
	}

	// place stone
	b.cells[move] = friendlyStone;

	// find any captures and remove them from the board
	captures = 0;
	for direction := 0; direction < 4; direction++ {
		neighborPt := move + b.dirOffset[direction];
		if b.cells[neighborPt] == enemyStone {
			captures += b.capture(neighborPt)
		}
	}

	if captures == 0 {
		// check for suicide
		if !b.hasLiberties(move) {
			// illegal move; undo and return
			b.cells[move] = EMPTY;
			return suicide, 0;
		}
	} else if captures == 1 {
		// check for simple Ko.
		lastMove := b.moves[b.moveCount-1];
		if (lastMove&ONE_CAPTURE) != 0 &&	// previous move captured one stone
			b.cells[lastMove&MOVE_TO_PT_MASK] == EMPTY {	// this move captured previous move
			// found a Ko; revert the capture
			b.cells[lastMove&MOVE_TO_PT_MASK] = enemyStone;
			b.cells[move] = EMPTY;
			return ko, 0;
		}
		move = ONE_CAPTURE | move;
	}

	b.moves[b.moveCount] = move;
	b.moveCount++;
	return played, captures;
}

// Given any point in a chain with no liberties, removes all stones in the
// chain from the board and returns the number of stones removed. Given a
// point in a chain that has liberties, does nothing and returns 0.
// Preconditions: same as b.markSurroundedChain
func (b *board) capture(target pt) (chainCount int) {
	chainCount = b.markSurroundedChain(target);

	// Remove the stones from the board
	for i := 0; i < chainCount; i++ {
		b.cells[b.chainPoints[i]] = EMPTY
	}
	return chainCount;
}

// Given any occupied point, returns true if it has any liberties.
// (Used for testing suicide.)
// Preconditions: same as b.markSurroundedChain
func (b *board) hasLiberties(target pt) bool {
	chainCount := b.markSurroundedChain(target);
	if chainCount == 0 {
		return true
	}

	// Revert marked positions
	for i := 0; i < chainCount; i++ {
		b.cells[b.chainPoints[i]] ^= CELL_IN_CHAIN
	}
	return false;
}

// Given any point in a chain with no liberties, marks all the cells in
// the chain with CELL_IN_CHAIN and adds those points to chainPoints.
// Returns the number of points found. If the chain is not surrounded,
// does nothing and returns 0.
// Preconditions: the target point is occupied and all cells have the
// CELL_IN_CHAIN flag cleared.
func (b *board) markSurroundedChain(target pt) (chainCount int) {
	chainCount = 0;
	chainColor := b.cells[target];

	b.chainPoints[chainCount] = target;
	chainCount++;
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

		// unrolled loop on the neighboring point in each direction:
		//   - if the neighbor is empty, revert.
		//   - if the neighbor is the same color, mark and add to chainPoints

		rightPt := thisPt + pt(1);
		switch b.cells[rightPt] {
		case EMPTY:
			goto revert
		case chainColor:
			b.chainPoints[chainCount] = rightPt;
			b.cells[rightPt] |= CELL_IN_CHAIN;
			chainCount++;
		}

		leftPt := thisPt + pt(-1);
		switch b.cells[leftPt] {
		case EMPTY:
			goto revert
		case chainColor:
			b.chainPoints[chainCount] = leftPt;
			b.cells[leftPt] |= CELL_IN_CHAIN;
			chainCount++;
		}

		upPt := thisPt + pt(b.stride);
		switch b.cells[upPt] {
		case EMPTY:
			goto revert
		case chainColor:
			b.chainPoints[chainCount] = upPt;
			b.cells[upPt] |= CELL_IN_CHAIN;
			chainCount++;
		}

		downPt := thisPt + pt(-b.stride);
		switch b.cells[downPt] {
		case EMPTY:
			goto revert
		case chainColor:
			b.chainPoints[chainCount] = downPt;
			b.cells[downPt] |= CELL_IN_CHAIN;
			chainCount++;
		}
	}

	return chainCount;

revert:
	// Found a liberty. Revert marks and return.
	for i := 0; i < chainCount; i++ {
		b.cells[b.chainPoints[i]] ^= CELL_IN_CHAIN
	}
	return 0;
}

// Returns true if this move would fill in an eye.
// Based on eyeMove() in Java ref bot:
/*     definition of eye:

       an empty point whose orthogonal neighbors are all of the
       same color AND whose diagonal neighbors contain no more
       than 1 stone of the opposite color unless it's a border
       in which case no diagonal enemies are allowed. */
func (b *board) wouldFillEye(move pt) bool {
	if move == PASS {
		return false
	}
	friendlyStone := cell(2 - (b.moveCount & 1));
	enemyStone := friendlyStone ^ 3;

	// not an eye unless cardinal directions have friendly stones or edge.
	for direction := 0; direction < 4; direction++ {
		neighborPt := move + b.dirOffset[direction];
		neighborCell := b.cells[neighborPt];
		if neighborCell != EDGE && neighborCell != friendlyStone {
			return false
		}
	}

	// count diagonal enemies and edges
	haveEdge := 0;
	enemies := 0;
	for direction := 0; direction < 4; direction++ {
		neighborPt := move + b.diagOffset[direction];
		switch b.cells[neighborPt] {
		case enemyStone:
			enemies++
		case EDGE:
			haveEdge = 1
		}
	}
	return enemies+haveEdge < 2;
}

// === Implementation of GoRobot interface ===

type robot struct {
	board		*board;
	randomness	Randomness;
	log		*log.Logger;
	komi		float;
	sampleCount	int;

	// Contains a hash of each previous board in the current game,
	// for determining whether a move would violate positional superko
	boardHashes	[]int64;

	// Scratch variables, reused to avoid GC
	scratchBoard	*board;
	candidates	[]pt;	// moves to choose from; used in GenMove.
	wins, hits	[]int;	// results of findWins()
}

func (r *robot) SetBoardSize(newSize int) bool {
	r.board.clearBoard(newSize);
	r.scratchBoard.clearBoard(newSize);
	r.boardHashes = make([]int64, len(r.board.moves));
	r.candidates = make([]pt, len(r.board.allPoints));
	r.wins = make([]int, len(r.board.cells));
	r.hits = make([]int, len(r.board.cells));
	return true;
}

func (r *robot) ClearBoard()	{ r.SetBoardSize(r.board.size) }

func (r *robot) SetKomi(value float)	{ r.komi = value }

func (r *robot) Play(color Color, x, y int) (ok bool, message string) {
	if !r.board.checkPlayArgs(color, x, y) {
		return false, "invalid args"
	}

	if !r.board.isMyTurn(color) {
		// GTP protocol allows two moves by the same color, to allow a game
		// to be set up more easily; treat as if the other player passed.
		if ok, message := r.Play(color.GetOpponent(), 0, 0); !ok {
			return false, fmt.Sprintf("other side cannot pass? (%v)", message)
		}
	}

	// use full version of makeMove so we update r.boardHashes
	result, captures := r.makeMove(r.board.makePt(x, y));
	return result.toPlayResult(captures);
}

func (r *robot) GenMove(color Color) (x, y int, moveResult MoveResult) {
	if !r.board.isMyTurn(color) {
		// GTP protocol allows generating a move by either side;
		// treat as if the other player passed.
		if ok, message := r.Play(color.GetOpponent(), 0, 0); !ok {
			panic("other side cannot pass? ", message)
		}
	}

	startTime := time.Nanoseconds();
	r.findWins(r.sampleCount);
	stopTime := time.Nanoseconds();
	elapsedTimeSecs := float64(stopTime-startTime) / math.Pow10(9);
	r.log.Logf("playouts/second: %.0f", float64(r.sampleCount)/elapsedTimeSecs);

	// create a list of possible moves
	candidates := r.candidates;	// reuse array to avoid allocation
	candidateCount := 0;
	for _, pt := range r.board.allPoints {
		if r.hits[pt] > 0 && !r.board.wouldFillEye(pt) && r.checkLegalMove(pt) == played {
			candidates[candidateCount] = pt;
			candidateCount++;
		}
	}

	// choose best move by iterating through candidates
	// (randomly permuted to break ties randomly)
	bestMove := PASS;
	bestScore := float(-99.0);
	for i := 0; i < candidateCount; i++ {

		// permute
		randomIndex := i + rand.Intn(candidateCount-i);
		pt := r.candidates[randomIndex];
		r.candidates[randomIndex], r.candidates[i] = r.candidates[i], pt;

		score := float(r.wins[pt]) / float(r.hits[pt]);
		if score > bestScore {
			bestMove = pt;
			bestScore = score;
		}
	}

	result, _ := r.makeMove(bestMove);

	if result == played {
		x, y := r.board.getCoords(bestMove);
		return x, y, Played;
	} else if result == passed {
		return 0, 0, Passed
	}
	panic("can't make generated move? ", result);
}

func (r *robot) GetBoardSize() int	{ return r.board.GetBoardSize() }

func (r *robot) GetCell(x, y int) Color	{ return r.board.GetCell(x, y) }

// The strict version of makeMove for actually making a move.
// (Checks for superko and updates boardHashes.)
func (r *robot) makeMove(move pt) (result moveResult, captures int) {
	if result := r.checkLegalMove(move); !result.ok() {
		return result, 0
	}
	result, captures = r.board.makeMove(move);
	if !result.ok() {
		panic("isLegalMove ok but makeMove returned: ", result)
	}
	r.boardHashes[r.board.moveCount-1] = r.board.getHash();
	return result, captures;
}

func (r *robot) checkLegalMove(move pt) (result moveResult) {
	// try this move on the scratch board
	sb := r.scratchBoard;
	sb.copyFrom(r.board);
	result, _ = sb.makeMove(move);

	if result == played {
		// check for superko
		newHash := sb.getHash();
		for i := 0; i < r.board.moveCount; i++ {
			if newHash == r.boardHashes[i] {
				// found superko
				return superko
			}
		}
	}

	return result;
}

// Use Monte-Carlo simulation to find a win rate for each point on the board.
// On return, r.wins[pt] will have the number of wins minus losses associated
// with a point and r.hits[pt] will have the number of samples for that point.
func (r *robot) findWins(numSamples int) {
	// clear statistics
	for i := range r.wins {
		r.wins[i] = 0;
		r.hits[i] = 0;
	}

	sb := r.scratchBoard;
	for i := 0; i < numSamples; i++ {
		sb.copyFrom(r.board);
		sb.playRandomGame(r.randomness);
		score := sb.getEasyScore();

		// choose amount to add to points used in this game
		var winAmount int;
		if float(score) > r.komi {
			winAmount = 1
		} else if float(score) < r.komi {
			winAmount = -1
		} else {
			winAmount = 0	// a draw
		}
		if r.board.getFriendlyStone() == WHITE {
			winAmount = -winAmount
		}

		// For each point where the first player to play was the current
		// player, add winAmount. (All Moves As First heuristic)
	scoring:
		for i := r.board.moveCount; i < sb.moveCount; i += 2 {
			pt := sb.moves[i] & MOVE_TO_PT_MASK;

			// check that it hasn't been played yet
			for j := r.board.moveCount; j < i; j++ {
				if sb.moves[j] == pt {
					continue scoring
				}
			}

			r.wins[pt] += winAmount;
			r.hits[pt]++;
		}
	}
}

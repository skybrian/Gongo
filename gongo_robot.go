package gongo

func NewRobot(boardSize int) GoRobot {
	return &robot{boardSize: boardSize};
}

type robot struct {
	boardSize int;
}

func (r *robot) SetBoardSize(value int) bool {
	return false;
}

func (r *robot) ClearBoard() {
}

func (r *robot) SetKomi(value float) {
}

func (r *robot) Play(value Move) bool {
	return false;
}

func (r *robot) GenMove(color Color) (vertex Vertex, ok bool) {
	return Vertex{}, false;
}

func (r *robot) GetBoardSize() int {
	return r.boardSize;
}

func (r *robot) GetCell(x, y int) Color {
	return Empty;
}


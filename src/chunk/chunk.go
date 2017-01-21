package chunk

// chunk status consts
const (
	ChunkStatusReady      = 0
	ChunkStatusInProgress = 1
	ChunkStatusDone       = 2
)

type Chunk struct {
	index       int64
	piece_index int64
	length 		int64
	status      int
	data        []byte
}

func NewChunk(index int64, piece_index int64, length int64) *Chunk {
	if length == 0 {
		panic("00")
	}
	c := Chunk{}
	c.index = index
	c.piece_index = piece_index
	c.data = make([]byte, length)
	c.length = length

	c.status = ChunkStatusReady

	return &c
}

func (ch *Chunk) SetStatus(status int) {
	ch.status = status
}

func (ch *Chunk) SetData(data []byte) {
	ch.data = data[:]
}

func (ch *Chunk) GetIndex() int64 {
	return ch.index
}

func (ch *Chunk) GetPieceIndex() int64 {
	return ch.piece_index
}

func (ch *Chunk) GetLength() int64 {
	return ch.length
}

func (ch *Chunk) GetStatus() int {
	return ch.status
}

func (ch *Chunk) GetData() []byte {
	return ch.data
}

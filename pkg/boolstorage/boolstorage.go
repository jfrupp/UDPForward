package boolstorage

type BoolStorage struct {
	a bool
}

type BoolStorage256v256 struct {
	a [256][4]uint64
}

func NewBoolStorage() *BoolStorage {
	return &BoolStorage{a: false}
}

func (s *BoolStorage) Check() bool {
	return s.a
}

func (s *BoolStorage) Set() {
	s.a = true
}

func (s *BoolStorage) Reset() {
	s.a = false
}

func (s *BoolStorage) Clear() {
	s.a = false
}

func NewBoolStorage256v256() *BoolStorage256v256 {
	return &BoolStorage256v256{}
}

func (s *BoolStorage256v256) Check(id uint8, flow uint8) bool {
	bit := flow % 64
	flow = flow / 64
	return s.a[id][flow]&(1<<bit) != 0
}

func (s *BoolStorage256v256) Set(id uint8, flow uint8) {
	bit := flow % 64
	flow = flow / 64
	s.a[id][flow] |= 1 << bit
}

func (s *BoolStorage256v256) Reset(id uint8, flow uint8) {
	bit := flow % 64
	flow = flow / 64
	s.a[id][flow] &= ^(1 << bit)
}

func (s *BoolStorage256v256) Clear() {
	for i := 0; i < 256; i++ {
		for j := 0; j < 4; j++ {
			s.a[i][j] = 0
		}
	}
}

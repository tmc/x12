package x12

type X12Document struct {
	Segments []Segment
}

type Segment interface {
	X12() string
}

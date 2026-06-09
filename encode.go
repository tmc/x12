package x12

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// An Encoder writes X12 documents to an output stream.
type Encoder struct {
	w io.Writer

	segmentTerminator  string
	elementSeparator   string
	componentSeparator string
	newlines           bool
}

// An EncodeOption configures an Encoder.
type EncodeOption func(*Encoder)

// WithSegmentTerminator sets the segment terminator used when encoding.
// The default is DefaultSegmentTerminator.
func WithSegmentTerminator(s string) EncodeOption {
	return func(enc *Encoder) { enc.segmentTerminator = s }
}

// WithElementSeparator sets the element separator used when encoding.
// The default is DefaultElementSeparator.
func WithElementSeparator(s string) EncodeOption {
	return func(enc *Encoder) { enc.elementSeparator = s }
}

// WithComponentSeparator sets the component element separator used when
// encoding composite elements. The default is DefaultComponentSeparator.
func WithComponentSeparator(s string) EncodeOption {
	return func(enc *Encoder) { enc.componentSeparator = s }
}

// WithNewlines writes a newline after each segment terminator.
func WithNewlines() EncodeOption {
	return func(enc *Encoder) { enc.newlines = true }
}

// NewEncoder returns a new Encoder that writes to w.
func NewEncoder(w io.Writer, opts ...EncodeOption) *Encoder {
	enc := &Encoder{
		w:                  w,
		segmentTerminator:  DefaultSegmentTerminator,
		elementSeparator:   DefaultElementSeparator,
		componentSeparator: DefaultComponentSeparator,
	}
	for _, opt := range opts {
		opt(enc)
	}
	return enc
}

// Marshal returns the X12 encoding of doc.
func Marshal(doc *Document, opts ...EncodeOption) ([]byte, error) {
	var buf bytes.Buffer
	if err := NewEncoder(&buf, opts...).Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Encode writes the X12 encoding of doc to the encoder's writer.
//
// A document with EnvelopeAutomaticallyAdded set is written without its
// synthesized envelope: only the ST segment, the transaction's segments,
// and the SE segment are emitted.
func (enc *Encoder) Encode(doc *Document) error {
	if doc == nil {
		return fmt.Errorf("%w: doc nil", ErrInvalidArgument)
	}
	if doc.Interchange == nil {
		return fmt.Errorf("%w: missing interchange", ErrInvalidArgument)
	}
	if doc.EnvelopeAutomaticallyAdded {
		if len(doc.Interchange.FunctionGroups) != 1 || len(doc.Interchange.FunctionGroups[0].Transactions) != 1 {
			return fmt.Errorf("%w: automatically enveloped document must contain exactly one function group with one transaction", ErrInvalidArgument)
		}
		return enc.encodeTransaction(doc.Interchange.FunctionGroups[0].Transactions[0])
	}
	if doc.Interchange.Header == nil {
		return fmt.Errorf("%w: ISA segment missing", ErrInvalidFormat)
	}
	if doc.Interchange.Trailer == nil {
		return fmt.Errorf("%w: IEA segment missing", ErrInvalidFormat)
	}
	if err := enc.encodeISA(doc.Interchange.Header); err != nil {
		return err
	}
	for _, group := range doc.Interchange.FunctionGroups {
		if err := enc.encodeFunctionGroup(group); err != nil {
			return err
		}
	}
	return enc.encodeIEA(doc.Interchange.Trailer)
}

func (enc *Encoder) encodeFunctionGroup(group *FunctionGroup) error {
	if group.Header == nil {
		return fmt.Errorf("%w: GS segment missing", ErrInvalidFormat)
	}
	if group.Trailer == nil {
		return fmt.Errorf("%w: GE segment missing", ErrInvalidFormat)
	}
	if err := enc.encodeGS(group.Header); err != nil {
		return err
	}
	for _, transaction := range group.Transactions {
		if err := enc.encodeTransaction(transaction); err != nil {
			return err
		}
	}
	return enc.encodeGE(group.Trailer)
}

func (enc *Encoder) encodeTransaction(transaction *Transaction) error {
	if transaction.Header == nil {
		return fmt.Errorf("%w: ST segment missing", ErrInvalidFormat)
	}
	if transaction.Trailer == nil {
		return fmt.Errorf("%w: SE segment missing", ErrInvalidFormat)
	}
	if err := enc.encodeST(transaction.Header); err != nil {
		return err
	}
	for _, segment := range transaction.Segments {
		if err := enc.encodeSegment(segment); err != nil {
			return err
		}
	}
	return enc.encodeSE(transaction.Trailer)
}

func (enc *Encoder) encodeISA(h *ISA) error {
	return enc.writeSegment([]string{
		"ISA",
		h.AuthorizationInfoQualifier,
		h.AuthorizationInformation,
		h.SecurityInfoQualifier,
		h.SecurityInfo,
		h.SenderIDQualifier,
		h.SenderID,
		h.ReceiverIDQualifier,
		h.ReceiverID,
		h.Date,
		h.Time,
		h.RepetitionSeparator,
		h.Version,
		h.ControlNumber,
		h.AcknowledgmentRequested,
		h.UsageIndicator,
		h.ComponentElementSeparator,
	})
}

func (enc *Encoder) encodeIEA(t *IEA) error {
	return enc.writeSegment([]string{
		"IEA",
		t.FunctionalGroupCount,
		t.ControlNumber,
	})
}

func (enc *Encoder) encodeGS(h *GS) error {
	return enc.writeSegment([]string{
		"GS",
		h.FunctionalIDCode,
		h.SenderCode,
		h.ReceiverCode,
		h.Date,
		h.Time,
		h.ControlNumber,
		h.ResponsibleAgencyCode,
		h.Version,
	})
}

func (enc *Encoder) encodeGE(t *GE) error {
	return enc.writeSegment([]string{
		"GE",
		t.TransactionSetCount,
		t.ControlNumber,
	})
}

func (enc *Encoder) encodeST(h *ST) error {
	elements := []string{
		"ST",
		h.IDCode,
		h.ControlNumber,
	}
	if h.ImplementationConventionReference != "" {
		elements = append(elements, h.ImplementationConventionReference)
	}
	return enc.writeSegment(elements)
}

func (enc *Encoder) encodeSE(t *SE) error {
	return enc.writeSegment([]string{
		"SE",
		t.SegmentCount,
		t.ControlNumber,
	})
}

func (enc *Encoder) encodeSegment(s Segment) error {
	elements := []string{s.ID}
	for _, e := range s.Elements {
		elements = append(elements, enc.encodeElement(e))
	}
	return enc.writeSegment(elements)
}

func (enc *Encoder) encodeElement(e Element) string {
	if e.Components == nil {
		return e.Value
	}
	elements := append([]string{e.Value}, e.Components...)
	return strings.Join(elements, enc.componentSeparator)
}

func (enc *Encoder) writeSegment(elements []string) error {
	s := strings.Join(elements, enc.elementSeparator) + enc.segmentTerminator
	if enc.newlines {
		s += "\n"
	}
	_, err := io.WriteString(enc.w, s)
	return err
}

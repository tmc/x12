package x12

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// An Encoder writes X12 documents to an output stream.
//
// Unless overridden by options, the encoder uses the delimiters carried
// by the document being encoded (discovered when it was decoded, or set
// by hand), falling back to the package defaults.
type Encoder struct {
	w io.Writer

	segmentTerminator  string
	elementSeparator   string
	componentSeparator string
	newlines           bool
}

// An EncodeOption configures an Encoder.
type EncodeOption func(*Encoder)

// WithSegmentTerminator sets the segment terminator used when encoding,
// overriding the document's own. The default is
// DefaultSegmentTerminator.
func WithSegmentTerminator(s string) EncodeOption {
	return func(enc *Encoder) { enc.segmentTerminator = s }
}

// WithElementSeparator sets the element separator used when encoding,
// overriding the document's own. The default is DefaultElementSeparator.
func WithElementSeparator(s string) EncodeOption {
	return func(enc *Encoder) { enc.elementSeparator = s }
}

// WithComponentSeparator sets the component element separator used when
// encoding composite elements, overriding the document's ISA16. The
// default is DefaultComponentSeparator.
func WithComponentSeparator(s string) EncodeOption {
	return func(enc *Encoder) { enc.componentSeparator = s }
}

// WithNewlines writes a newline after each segment terminator.
func WithNewlines() EncodeOption {
	return func(enc *Encoder) { enc.newlines = true }
}

// NewEncoder returns a new Encoder that writes to w.
func NewEncoder(w io.Writer, opts ...EncodeOption) *Encoder {
	enc := &Encoder{w: w}
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

// encodeState holds the resolved configuration for a single Encode
// call, so that an Encoder shared between goroutines is never mutated.
type encodeState struct {
	w io.Writer

	segmentTerminator  string
	elementSeparator   string
	componentSeparator string
	newlines           bool
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
	state := &encodeState{
		w:                  enc.w,
		segmentTerminator:  resolve(enc.segmentTerminator, doc.SegmentTerminator, DefaultSegmentTerminator),
		elementSeparator:   resolve(enc.elementSeparator, doc.ElementSeparator, DefaultElementSeparator),
		componentSeparator: resolve(enc.componentSeparator, isa16(doc), DefaultComponentSeparator),
		newlines:           enc.newlines,
	}
	if doc.EnvelopeAutomaticallyAdded {
		groups := doc.Interchange.FunctionGroups
		if len(groups) != 1 || groups[0] == nil || len(groups[0].Transactions) != 1 {
			return fmt.Errorf("%w: automatically enveloped document must contain exactly one function group with one transaction", ErrInvalidArgument)
		}
		return state.encodeTransaction(groups[0].Transactions[0])
	}
	if doc.Interchange.Header == nil {
		return fmt.Errorf("%w: ISA segment missing", ErrInvalidFormat)
	}
	if doc.Interchange.Trailer == nil {
		return fmt.Errorf("%w: IEA segment missing", ErrInvalidFormat)
	}
	if err := state.encodeISA(doc.Interchange.Header); err != nil {
		return err
	}
	for _, group := range doc.Interchange.FunctionGroups {
		if err := state.encodeFunctionGroup(group); err != nil {
			return err
		}
	}
	return state.encodeIEA(doc.Interchange.Trailer)
}

// resolve returns the first non-empty delimiter among the explicitly
// configured value, the document's own value, and the package default.
func resolve(configured, document, fallback string) string {
	if configured != "" {
		return configured
	}
	if document != "" {
		return document
	}
	return fallback
}

// isa16 returns the document's component element separator (ISA16), if
// it has one.
func isa16(doc *Document) string {
	if doc.Interchange.Header == nil {
		return ""
	}
	return doc.Interchange.Header.ComponentElementSeparator
}

func (state *encodeState) encodeFunctionGroup(group *FunctionGroup) error {
	if group == nil {
		return fmt.Errorf("%w: nil function group", ErrInvalidFormat)
	}
	if group.Header == nil {
		return fmt.Errorf("%w: GS segment missing", ErrInvalidFormat)
	}
	if group.Trailer == nil {
		return fmt.Errorf("%w: GE segment missing", ErrInvalidFormat)
	}
	if err := state.encodeGS(group.Header); err != nil {
		return err
	}
	for _, transaction := range group.Transactions {
		if err := state.encodeTransaction(transaction); err != nil {
			return err
		}
	}
	return state.encodeGE(group.Trailer)
}

func (state *encodeState) encodeTransaction(transaction *Transaction) error {
	if transaction == nil {
		return fmt.Errorf("%w: nil transaction", ErrInvalidFormat)
	}
	if transaction.Header == nil {
		return fmt.Errorf("%w: ST segment missing", ErrInvalidFormat)
	}
	if transaction.Trailer == nil {
		return fmt.Errorf("%w: SE segment missing", ErrInvalidFormat)
	}
	if err := state.encodeST(transaction.Header); err != nil {
		return err
	}
	for _, segment := range transaction.Segments {
		if err := state.encodeSegment(segment); err != nil {
			return err
		}
	}
	return state.encodeSE(transaction.Trailer)
}

func (state *encodeState) encodeISA(h *ISA) error {
	return state.writeSegment([]string{
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

func (state *encodeState) encodeIEA(t *IEA) error {
	return state.writeSegment([]string{
		"IEA",
		t.FunctionalGroupCount,
		t.ControlNumber,
	})
}

func (state *encodeState) encodeGS(h *GS) error {
	return state.writeSegment([]string{
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

func (state *encodeState) encodeGE(t *GE) error {
	return state.writeSegment([]string{
		"GE",
		t.TransactionSetCount,
		t.ControlNumber,
	})
}

func (state *encodeState) encodeST(h *ST) error {
	elements := []string{
		"ST",
		h.IDCode,
		h.ControlNumber,
	}
	if h.ImplementationConventionReference != "" {
		elements = append(elements, h.ImplementationConventionReference)
	}
	return state.writeSegment(elements)
}

func (state *encodeState) encodeSE(t *SE) error {
	return state.writeSegment([]string{
		"SE",
		t.SegmentCount,
		t.ControlNumber,
	})
}

func (state *encodeState) encodeSegment(s Segment) error {
	elements := []string{s.ID}
	for _, e := range s.Elements {
		elements = append(elements, state.encodeElement(e))
	}
	return state.writeSegment(elements)
}

func (state *encodeState) encodeElement(e Element) string {
	if e.Components == nil {
		return e.Value
	}
	elements := append([]string{e.Value}, e.Components...)
	return strings.Join(elements, state.componentSeparator)
}

func (state *encodeState) writeSegment(elements []string) error {
	s := strings.Join(elements, state.elementSeparator) + state.segmentTerminator
	if state.newlines {
		s += "\n"
	}
	_, err := io.WriteString(state.w, s)
	return err
}

package x12

import (
	"fmt"
	"strings"
)

type Marshaler struct {
	SegmentSeparator    string
	ElementSeparator    string
	SubElementSeparator string

	NewLines bool
}

func (m *Marshaler) Marshal(x *Document) ([]byte, error) {
	if x == nil {
		return nil, fmt.Errorf("%w: x nil", ErrInvalidArgument)
	}
	if x.Interchange == nil {
		return nil, fmt.Errorf("%w: missing interchange", ErrInvalidArgument)
	}
	builder := strings.Builder{}
	if !x.EnvelopeAutomaticallyAdded {
		m.encodeISA(x.Interchange.Header, &builder)
		m.encodeFunctionGroups(x.Interchange.FunctionGroups, &builder)
		m.encodeIEA(x.Interchange.Trailer, &builder)
	} else {
		if len(x.Interchange.FunctionGroups) != 1 || len(x.Interchange.FunctionGroups[0].Transactions) != 1 {
			return nil, fmt.Errorf("%w: automatically enveloped document must contain exactly one function group with one transaction", ErrInvalidArgument)
		}
		fg := x.Interchange.FunctionGroups[0]
		transaction := fg.Transactions[0]
		m.encodeST(transaction.Header, &builder)
		for _, segment := range transaction.Segments {
			m.encodeSegment(segment, &builder)
		}
		m.encodeSE(transaction.Trailer, &builder)
	}
	return []byte(builder.String()), nil
}

func (m *Marshaler) ss() string {
	ss := m.SegmentSeparator
	if ss == "" {
		ss = SegmentSeparator
	}
	if m.NewLines {
		return ss + "\n"
	}
	return ss
}

func (m *Marshaler) es() string {
	if m.ElementSeparator == "" {
		return ElementSeparator
	}
	return m.ElementSeparator
}

func (m *Marshaler) ses() string {
	if m.SubElementSeparator == "" {
		return SubElementSeparator
	}
	return m.SubElementSeparator
}

func (m *Marshaler) encodeISA(h *ISA, builder *strings.Builder) {
	elements := []string{
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
	}
	builder.WriteString(strings.Join(elements, m.es()))
	builder.WriteString(m.ss())
}

func (m *Marshaler) encodeIEA(t *IEA, builder *strings.Builder) {
	elements := []string{
		"IEA",
		t.FunctionalGroupCount,
		t.ControlNumber,
	}
	builder.WriteString(strings.Join(elements, m.es()))
	builder.WriteString(m.ss())
}

func (m *Marshaler) encodeFunctionGroups(groups []*FunctionGroup, builder *strings.Builder) {
	for _, group := range groups {
		m.encodeGS(group.Header, builder)
		for _, transaction := range group.Transactions {
			m.encodeST(transaction.Header, builder)
			for _, segment := range transaction.Segments {
				m.encodeSegment(segment, builder)
			}
			m.encodeSE(transaction.Trailer, builder)
		}
		m.encodeGE(group.Trailer, builder)
	}
}

func (m *Marshaler) encodeGS(h *GS, builder *strings.Builder) {
	elements := []string{
		"GS",
		h.FunctionalIDCode,
		h.SenderCode,
		h.ReceiverCode,
		h.Date,
		h.Time,
		h.ControlNumber,
		h.ResponsibleAgencyCode,
		h.Version,
	}
	builder.WriteString(strings.Join(elements, m.es()))
	builder.WriteString(m.ss())
}

func (m *Marshaler) encodeGE(t *GE, builder *strings.Builder) {
	elements := []string{
		"GE",
		t.TransactionSetCount,
		t.ControlNumber,
	}
	builder.WriteString(strings.Join(elements, m.es()))
	builder.WriteString(m.ss())
}

func (m *Marshaler) encodeST(h *ST, builder *strings.Builder) {
	elements := []string{
		"ST",
		h.IDCode,
		h.ControlNumber,
	}
	if h.ImplementationConventionReference != "" {
		elements = append(elements, h.ImplementationConventionReference)
	}
	builder.WriteString(strings.Join(elements, m.es()))
	builder.WriteString(m.ss())
}

func (m *Marshaler) encodeSE(t *SE, builder *strings.Builder) {
	elements := []string{
		"SE",
		t.SegmentCount,
		t.ControlNumber,
	}
	builder.WriteString(strings.Join(elements, m.es()))
	builder.WriteString(m.ss())
}

func (m *Marshaler) encodeSegment(s Segment, builder *strings.Builder) {
	elements := []string{s.ID}
	for _, e := range s.Elements {
		elements = append(elements, m.encodeElement(e))
	}
	builder.WriteString(strings.Join(elements, m.es()))
	builder.WriteString(m.ss())
}

func (m *Marshaler) encodeElement(e Element) string {
	if e.Components == nil {
		return e.Value
	}
	elements := append([]string{e.Value}, e.Components...)
	return strings.Join(elements, m.ses())
}

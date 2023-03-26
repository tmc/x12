package x12

import "strings"

type Marshaler struct {
	SegmentSeparator    string
	ElementSeparator    string
	SubElementSeparator string

	NewLines bool
}

func (m *Marshaler) Marshal(x *X12Document) ([]byte, error) {
	builder := strings.Builder{}
	if !x.EnvelopeAutomaticallyAdded {
		m.encodeISA(x.Interchange.Header, &builder)
		m.encodeFunctionGroups(x.Interchange.FunctionGroups, &builder)
		m.encodeIEA(x.Interchange.Trailer, &builder)
	} else {
		// TODO: check that there is exactly one function group and one transaction
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
		ss = "~"
	}
	if m.NewLines {
		return ss + "\n"
	}
	return ss
}

func (m *Marshaler) es() string {
	if m.ElementSeparator == "" {
		return "*"
	}
	return m.es()
}

func (m *Marshaler) ses() string {
	if m.SubElementSeparator == "" {
		return ":"
	}
	return m.ses()
}

func (m *Marshaler) encodeISA(h *ISA, builder *strings.Builder) {
	elements := []string{
		"ISA",
		h.AuthorizationInfoQualifier,
		h.AuthorizationInformation,
		h.SecurityInfoQualifier,
		h.SecurityInfo,
		h.InterchangeSenderIDQualifier,
		h.InterchangeSenderID,
		h.InterchangeReceiverIDQualifier,
		h.InterchangeReceiverID,
		h.InterchangeDate,
		h.InterchangeTime,
		h.InterchangeControlStandardsID,
		h.InterchangeControlVersion,
		h.InterchangeControlNumber,
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
		t.NumberOfIncludedFunctionalGroups,
		t.InterchangeControlNumber,
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
		h.ApplicationSenderCode,
		h.ApplicationReceiverCode,
		h.Date,
		h.Time,
		h.GroupControlNumber,
		h.ResponsibleAgencyCode,
		h.VersionReleaseIndustryID,
	}
	builder.WriteString(strings.Join(elements, m.es()))
	builder.WriteString(m.ss())
}

func (m *Marshaler) encodeGE(t *GE, builder *strings.Builder) {
	elements := []string{
		"GE",
		t.NumberOfIncludedTransactionSets,
		t.GroupControlNumber,
	}
	builder.WriteString(strings.Join(elements, m.es()))
	builder.WriteString(m.ss())
}

func (m *Marshaler) encodeST(h *ST, builder *strings.Builder) {
	elements := []string{
		"ST",
		h.TransactionSetIDCode,
		h.TransactionSetControlNumber,
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
		t.NumberOfIncludedSegments,
		t.TransactionSetControlNumber,
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
	elements := []string{e.Value}
	for _, s := range e.Components {
		elements = append(elements, s)
	}
	return strings.Join(elements, m.ses())
}

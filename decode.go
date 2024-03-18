package x12

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// SegmentSeparator is the character that separates segments.
	SegmentSeparator = "~"
	// ElementSeparator is the character that separates elements.
	ElementSeparator = "*"
	// SubElementSeparator is the character that separates sub-elements.
	SubElementSeparator = ":"
)

var (
	ErrMissingElement  = errors.New("missing element")
	ErrInvalidFormat   = errors.New("invalid format")
	ErrInvalidArgument = errors.New("invalid argument")
)

type decodeState struct {
	doc                  *X12Document
	lineIndex            int
	currentFunctionGroup *FunctionGroup
	currentTransaction   *Transaction

	withRelaxedSegmentIDWhitespace bool
}

// DecodeOption is a function that can be used to configure the decoder.
type DecodeOption func(*decodeState)

// WithRelaxedSegmentIDWhitespace allows the decoder to accept segment IDs with leading and trailing whitespace.
func WithRelaxedSegmentIDWhitespace() DecodeOption {
	return func(state *decodeState) {
		state.withRelaxedSegmentIDWhitespace = true
	}
}

// Decode decodes an X12 document from an io.Reader
func Decode(in io.Reader, opts ...DecodeOption) (*X12Document, error) {

	state := &decodeState{
		doc: &X12Document{
			Interchange: &Interchange{},
		},
		lineIndex:            0,
		currentFunctionGroup: nil,
		currentTransaction:   nil,
	}
	for _, opt := range opts {
		opt(state)
	}

	scanner := bufio.NewScanner(in)
	scanner.Split(scanEDI)

	for scanner.Scan() {
		state.lineIndex++
		text := scanner.Text()
		segment := strings.Trim(text, "\r\n")
		if segment == "" {
			continue
		}
		elements := strings.Split(segment, ElementSeparator)
		segmentID := elements[0]
		if state.withRelaxedSegmentIDWhitespace {
			segmentID = strings.TrimSpace(segmentID)
		}
		switch segmentID {
		case "ISA":
			header, err := parseISA(elements)
			if err != nil {
				return nil, err
			}
			state.doc.Interchange.Header = header
		case "IEA":
			trailer, err := parseIEA(elements)
			if err != nil {
				return nil, err
			}
			state.doc.Interchange.Trailer = trailer
		case "GS":
			header, err := parseGS(elements)
			if err != nil {
				return nil, err
			}
			state.currentFunctionGroup = &FunctionGroup{
				Header: header,
			}
			state.doc.Interchange.FunctionGroups = append(state.doc.Interchange.FunctionGroups, state.currentFunctionGroup)
		case "GE":
			if state.currentFunctionGroup == nil {
				return nil, state.Errorf("%w: GE segment without GS segment", ErrInvalidFormat)
			}
			trailer, err := parseGE(elements)
			if err != nil {
				return nil, state.Errorf("%w: issue parsing GE segment", err)
			}
			state.currentFunctionGroup.Trailer = trailer
		case "ST":
			state.considerAutomaticEnvelope()
			if state.currentFunctionGroup == nil {
				return nil, state.Errorf("%w: ST segment without GS segment", ErrInvalidFormat)
			}
			header, err := parseST(elements)
			if err != nil {
				return nil, err
			}
			state.currentTransaction = &Transaction{
				Header: header,
			}
			state.currentFunctionGroup.Transactions = append(state.currentFunctionGroup.Transactions, state.currentTransaction)
		case "SE":
			if state.currentTransaction == nil {
				return nil, state.Errorf("%w: SE segment without ST segment", ErrInvalidFormat)
			}
			trailer, err := parseSE(elements)
			if err != nil {
				return nil, err
			}
			state.currentTransaction.Trailer = trailer
		default:
			if state.currentTransaction == nil {
				return nil, state.Errorf("%w: '%v' segment without ST segment", ErrInvalidFormat, segmentID)
			}
			segment, err := parseSegment(segmentID, elements[1:])
			if err != nil {
				return nil, state.Errorf("issue parsing segment %v: %w", segmentID, err)
			}
			state.currentTransaction.Segments = append(state.currentTransaction.Segments, segment)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return state.doc, nil
}

// Validate validates the x12 document
func (doc *X12Document) Validate() error {
	if doc == nil {
		return fmt.Errorf("%w: doc nil", ErrInvalidArgument)
	}
	if doc.Interchange == nil {
		return fmt.Errorf("%w: missing interchange", ErrInvalidFormat)
	}
	// check that the ISA and IEA segments are present and match
	if doc.Interchange.Header == nil {
		return fmt.Errorf("%w: ISA segment missing", ErrInvalidFormat)
	}
	if doc.Interchange.Trailer == nil {

		return fmt.Errorf("%w: IEA segment missing", ErrInvalidFormat)
	}
	if doc.Interchange.Header.InterchangeControlNumber != doc.Interchange.Trailer.InterchangeControlNumber {
		return fmt.Errorf("%w: ISA and IEA control numbers do not match (%v != %v)", ErrInvalidFormat, doc.Interchange.Header.InterchangeControlNumber, doc.Interchange.Trailer.InterchangeControlNumber)
	}

	// check that the GS and GE segments are present and match
	for _, functionGroup := range doc.Interchange.FunctionGroups {
		if functionGroup.Header == nil {
			return fmt.Errorf("%w: GS segment missing", ErrInvalidFormat)
		}
		if functionGroup.Trailer == nil {
			return fmt.Errorf("%w: GE segment missing", ErrInvalidFormat)
		}
		if functionGroup.Header.GroupControlNumber != functionGroup.Trailer.GroupControlNumber {
			return fmt.Errorf("%w: GS and GE control numbers do not match (%v != %v)", ErrInvalidFormat, functionGroup.Header.GroupControlNumber, functionGroup.Trailer.GroupControlNumber)
		}
	}

	// check that the ST and SE segments are present and match
	for _, functionGroup := range doc.Interchange.FunctionGroups {
		for _, transaction := range functionGroup.Transactions {
			if transaction.Header == nil {
				return fmt.Errorf("%w: ST segment missing", ErrInvalidFormat)
			}
			if transaction.Trailer == nil {
				return fmt.Errorf("%w: SE segment missing", ErrInvalidFormat)
			}
			if transaction.Header.TransactionSetControlNumber != transaction.Trailer.TransactionSetControlNumber {
				return fmt.Errorf("%w: ST and SE control numbers do not match (%v != %v)", ErrInvalidFormat, transaction.Header.TransactionSetControlNumber, transaction.Trailer.TransactionSetControlNumber)
			}
		}
	}
	return nil
}

// scanEDI is a bufio.SplitFunc that splits an EDI document into segments
func scanEDI(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := strings.Index(string(data), SegmentSeparator); i >= 0 {
		// We have a full segment
		return i + 1, data[0:i], nil
	}

	// If we're at EOF, we have a final, non-empty, non-terminated segment. Return it.
	if atEOF {
		return len(data), data, nil
	}

	// Request more data.
	return 0, nil, nil
}

func parseISA(elements []string) (*ISA, error) {
	if len(elements) < 17 {
		return nil, fmt.Errorf("ISA: %w", ErrMissingElement)
	}
	return &ISA{
		AuthorizationInfoQualifier:     elements[1],
		AuthorizationInformation:       elements[2],
		SecurityInfoQualifier:          elements[3],
		SecurityInfo:                   elements[4],
		InterchangeSenderIDQualifier:   elements[5],
		InterchangeSenderID:            elements[6],
		InterchangeReceiverIDQualifier: elements[7],
		InterchangeReceiverID:          elements[8],
		InterchangeDate:                elements[9],
		InterchangeTime:                elements[10],
		InterchangeControlStandardsID:  elements[11],
		InterchangeControlVersion:      elements[12],
		InterchangeControlNumber:       elements[13],
		AcknowledgmentRequested:        elements[14],
		UsageIndicator:                 elements[15],
		ComponentElementSeparator:      elements[16],
	}, nil
}

func parseIEA(elements []string) (*IEA, error) {
	if len(elements) < 2 {
		return nil, ErrMissingElement
	}
	return &IEA{
		NumberOfIncludedFunctionalGroups: elements[1],
		InterchangeControlNumber:         elements[2],
	}, nil
}

func parseGS(elements []string) (*GS, error) {
	if len(elements) < 8 {
		return nil, ErrMissingElement
	}
	return &GS{
		FunctionalIDCode:         elements[1],
		ApplicationSenderCode:    elements[2],
		ApplicationReceiverCode:  elements[3],
		Date:                     elements[4],
		Time:                     elements[5],
		GroupControlNumber:       elements[6],
		ResponsibleAgencyCode:    elements[7],
		VersionReleaseIndustryID: elements[8],
	}, nil
}

func parseGE(elements []string) (*GE, error) {
	if len(elements) < 2 {
		return nil, ErrMissingElement
	}
	return &GE{
		NumberOfIncludedTransactionSets: elements[1],
		GroupControlNumber:              elements[2],
	}, nil
}

func parseST(elements []string) (*ST, error) {
	if len(elements) < 3 {
		return nil, ErrMissingElement
	}
	r := &ST{
		TransactionSetIDCode:        elements[1],
		TransactionSetControlNumber: elements[2],
	}
	if len(elements) > 3 {
		r.ImplementationConventionReference = elements[3]
	}
	return r, nil
}

func parseSE(elements []string) (*SE, error) {
	if len(elements) < 2 {
		return nil, ErrMissingElement
	}
	return &SE{
		NumberOfIncludedSegments:    elements[1],
		TransactionSetControlNumber: elements[2],
	}, nil
}

func parseSegment(segmentID string, elements []string) (Segment, error) {
	segment := Segment{
		ID: segmentID,
	}
	segment.Elements = parseElements(elements)
	return segment, nil
}

func parseElements(elements []string) []Element {
	parsedElements := make([]Element, len(elements))
	for i, element := range elements {
		parsedElements[i] = Element{
			ID:    fmt.Sprintf("%02d", i+1),
			Value: element,
		}
	}
	return parsedElements

}

// considerAutomaticEnvelope adds an ISA, IEA, GS, and GE envelope to the document if one is not present.
func (s *decodeState) considerAutomaticEnvelope() {
	shouldAdd := s.lineIndex == 1 && s.currentFunctionGroup == nil && s.currentTransaction == nil
	if !shouldAdd {
		return
	}

	s.doc.EnvelopeAutomaticallyAdded = true
	s.doc.Interchange.Header = &ISA{
		InterchangeControlNumber:  "000000001",
		ComponentElementSeparator: ElementSeparator,
	}

	s.doc.Interchange.Trailer = &IEA{
		NumberOfIncludedFunctionalGroups: "1",
		InterchangeControlNumber:         "000000001",
	}

	s.currentFunctionGroup = &FunctionGroup{
		Header: &GS{
			GroupControlNumber: "000000001",
		},
		Trailer: &GE{
			NumberOfIncludedTransactionSets: "1",
			GroupControlNumber:              "000000001",
		},
	}
	s.doc.Interchange.FunctionGroups = append(s.doc.Interchange.FunctionGroups, s.currentFunctionGroup)
}

// Errorf sets the error field of the decodeState to the given error.
func (s *decodeState) Errorf(format string, args ...any) error {
	format = fmt.Sprintf("line %d: %s", s.lineIndex, format)
	return fmt.Errorf(format, args...)
}

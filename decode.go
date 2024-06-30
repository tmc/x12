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
	// When encountering an unknown segment use this parser
	defaultParser = "DEFAULT"
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
	state := initializeDecodeState(opts)
	segmentParsers := state.getSegmentParsers()

	scanner := bufio.NewScanner(in)
	scanner.Split(scanEDI)
	for scanner.Scan() {
		if err := state.processLine(scanner.Text(), segmentParsers); err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return state.doc, nil
}

func initializeDecodeState(opts []DecodeOption) *decodeState {
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
	return state
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

func (s *decodeState) processLine(line string, parsers map[string]segmentParser) error {
	s.lineIndex++
	segment := strings.Trim(line, "\r\n")
	if segment == "" {
		return nil
	}

	elements := strings.Split(segment, ElementSeparator)
	segmentID, _ := s.extractSegmentID(elements)

	parseFunc, exists := parsers[segmentID]
	if !exists {
		parseFunc = parsers[defaultParser]
	}

	return parseFunc(s, elements)
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

type segmentParser func(s *decodeState, elements []string) error

// Not currently using the decodeState, but we may return different parsers on specific conditions in the future.
func (s *decodeState) getSegmentParsers() map[string]segmentParser {
	return map[string]segmentParser{
		"ISA":     (*decodeState).parseISA,
		"IEA":     (*decodeState).parseIEA,
		"GS":      (*decodeState).parseGS,
		"GE":      (*decodeState).parseGE,
		"ST":      (*decodeState).parseST,
		"SE":      (*decodeState).parseSE,
		"DEFAULT": (*decodeState).parseSegment,
	}
}

func (s *decodeState) parseISA(elements []string) error {
	if len(elements) < 17 {
		return s.Errorf("ISA: %w", ErrMissingElement)
	}
	s.doc.Interchange.Header = &ISA{
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
	}
	return nil
}

func (s *decodeState) parseIEA(elements []string) error {
	if len(elements) < 3 {
		return s.Errorf("IEA: %w", ErrMissingElement)
	}
	s.doc.Interchange.Trailer = &IEA{
		NumberOfIncludedFunctionalGroups: elements[1],
		InterchangeControlNumber:         elements[2],
	}
	return nil
}

func (s *decodeState) parseGS(elements []string) error {
	if len(elements) < 9 {
		return s.Errorf("GS: %w", ErrMissingElement)
	}
	s.currentFunctionGroup = &FunctionGroup{
		Header: &GS{
			FunctionalIDCode:         elements[1],
			ApplicationSenderCode:    elements[2],
			ApplicationReceiverCode:  elements[3],
			Date:                     elements[4],
			Time:                     elements[5],
			GroupControlNumber:       elements[6],
			ResponsibleAgencyCode:    elements[7],
			VersionReleaseIndustryID: elements[8],
		},
	}
	s.doc.Interchange.FunctionGroups = append(s.doc.Interchange.FunctionGroups, s.currentFunctionGroup)
	return nil
}

func (s *decodeState) parseGE(elements []string) error {
	if s.currentFunctionGroup == nil {
		return s.Errorf("%w: GE segment without GS segment", ErrInvalidFormat)
	}
	if len(elements) < 3 {
		return s.Errorf("GE: %w", ErrMissingElement)
	}
	s.currentFunctionGroup.Trailer = &GE{
		NumberOfIncludedTransactionSets: elements[1],
		GroupControlNumber:              elements[2],
	}
	return nil
}

func (s *decodeState) parseST(elements []string) error {
	if len(elements) < 3 {
		return s.Errorf("ST: %w", ErrMissingElement)
	}
	s.considerAutomaticEnvelope()
	if s.currentFunctionGroup == nil {
		return s.Errorf("%w: ST segment without GS segment", ErrInvalidFormat)
	}
	s.currentTransaction = &Transaction{
		Header: &ST{
			TransactionSetIDCode:        elements[1],
			TransactionSetControlNumber: elements[2],
		},
	}
	if len(elements) > 3 {
		s.currentTransaction.Header.ImplementationConventionReference = elements[3]
	}
	s.currentFunctionGroup.Transactions = append(s.currentFunctionGroup.Transactions, s.currentTransaction)
	return nil
}

func (s *decodeState) parseSE(elements []string) error {
	if s.currentTransaction == nil {
		return s.Errorf("%w: SE segment without ST segment", ErrInvalidFormat)
	}
	if len(elements) < 3 {
		return s.Errorf("SE: %w", ErrMissingElement)
	}
	s.currentTransaction.Trailer = &SE{
		NumberOfIncludedSegments:    elements[1],
		TransactionSetControlNumber: elements[2],
	}
	return nil
}

func (s *decodeState) parseSegment(elements []string) error {
	segmentID, elements := s.extractSegmentID(elements)
	if s.currentTransaction == nil {
		return s.Errorf("%w: '%v' segment without ST segment", ErrInvalidFormat, segmentID)
	}
	segment := Segment{
		ID:       segmentID,
		Elements: parseElements(elements),
	}
	s.currentTransaction.Segments = append(s.currentTransaction.Segments, segment)
	return nil
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

func (s *decodeState) extractSegmentID(elements []string) (string, []string) {
	segmentID := elements[0]
	if s.withRelaxedSegmentIDWhitespace {
		segmentID = strings.TrimSpace(segmentID)
	}
	return segmentID, elements[1:]
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

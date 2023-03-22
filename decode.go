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
	ErrMissingElement = errors.New("missing element")
	ErrInvalidFormat  = errors.New("invalid format")
)

// Decode decodes an X12 document from an io.Reader
func Decode(in io.Reader) (*X12Document, error) {
	scanner := bufio.NewScanner(in)
	scanner.Split(scanEDI)

	doc := &X12Document{
		Interchange: &Interchange{},
	}
	var currentFunctionGroup *FunctionGroup
	var currentTransaction *Transaction

	for scanner.Scan() {
		segment := strings.TrimSpace(scanner.Text())
		elements := strings.Split(segment, ElementSeparator)
		segmentID := elements[0]
		switch segmentID {
		case "ISA":
			header, err := parseISA(elements)
			if err != nil {
				return nil, err
			}
			doc.Interchange.Header = header
		case "IEA":
			trailer, err := parseIEA(elements)
			if err != nil {
				return nil, err
			}
			doc.Interchange.Trailer = trailer
		case "GS":
			header, err := parseGS(elements)
			if err != nil {
				return nil, err
			}
			currentFunctionGroup = &FunctionGroup{
				Header: header,
			}
			doc.Interchange.FunctionGroups = append(doc.Interchange.FunctionGroups, currentFunctionGroup)
		case "GE":
			if currentFunctionGroup == nil {
				return nil, fmt.Errorf("%w: GE segment without GS segment", ErrInvalidFormat)
			}
			trailer, err := parseGE(elements)
			if err != nil {
				return nil, fmt.Errorf("%w: issue parsing GE segment", err)
			}
			currentFunctionGroup.Trailer = trailer
		case "ST":
			if currentFunctionGroup == nil {
				return nil, fmt.Errorf("%w: ST segment without GS segment", ErrInvalidFormat)
			}
			header, err := parseST(elements)
			if err != nil {
				return nil, err
			}
			currentTransaction = &Transaction{
				Header: header,
			}
			currentFunctionGroup.Transactions = append(currentFunctionGroup.Transactions, currentTransaction)
		case "SE":
			if currentTransaction == nil {
				return nil, fmt.Errorf("%w: SE segment without ST segment", ErrInvalidFormat)
			}
			trailer, err := parseSE(elements)
			if err != nil {
				return nil, err
			}
			currentTransaction.Trailer = trailer
		default:
			if currentTransaction == nil {
				return nil, fmt.Errorf("%w: %v segment without ST segment", ErrInvalidFormat, segmentID)
			}
			segment, err := parseSegment(segmentID, elements[1:])
			if err != nil {
				return nil, fmt.Errorf("issue parsing segment %v: %w", segmentID, err)
			}
			currentTransaction.Segments = append(currentTransaction.Segments, segment)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return doc, nil
}

// Validate validates the x12 document
func (doc *X12Document) Validate() error {
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
	if len(elements) < 16 {
		fmt.Println("elements", elements)
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

package x12

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

const (
	// SegmentSeparator is the character that separates segments.
	SegmentSeparator = "~"
	// ElementSeparator is the character that separates elements.
	ElementSeparator = "*"
)

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

// Decode decodes an X12 document from an io.Reader
func Decode(in io.Reader) (*X12Document, error) {
	scanner := bufio.NewScanner(in)
	scanner.Split(scanEDI)

	doc := &X12Document{}
	var currentFunctionGroup *FunctionGroup
	var currentTransaction *Transaction

	for scanner.Scan() {
		segment := scanner.Text()
		elements := strings.Split(segment, ElementSeparator)
		segmentID := elements[0]

		switch segmentID {
		case "ISA":
			doc.Interchange.Header = parseISA(elements)
		case "IEA":
			doc.Interchange.Trailer = parseIEA(elements)
		case "GS":
			currentFunctionGroup = &FunctionGroup{
				Header: parseGS(elements),
			}
			doc.Interchange.FunctionGroups = append(doc.Interchange.FunctionGroups, *currentFunctionGroup)
		case "GE":
			currentFunctionGroup.Trailer = parseGE(elements)
		case "ST":
			currentTransaction = &Transaction{
				Header: parseST(elements),
			}
			currentFunctionGroup.Transactions = append(currentFunctionGroup.Transactions, *currentTransaction)
		case "SE":
			currentTransaction.Trailer = parseSE(elements)
		default:
			segment := Segment{
				ID:       segmentID,
				Elements: parseElements(elements[1:]),
			}
			currentTransaction.Segments = append(currentTransaction.Segments, segment)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return doc, nil
}

func parseISA(elements []string) ISA {
	return ISA{
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
}

func parseIEA(elements []string) IEA {
	return IEA{
		NumberOfIncludedFunctionalGroups: elements[1],
		InterchangeControlNumber:         elements[2],
	}
}

func parseGS(elements []string) GS {
	return GS{
		FunctionalIDCode:         elements[1],
		ApplicationSenderCode:    elements[2],
		ApplicationReceiverCode:  elements[3],
		Date:                     elements[4],
		Time:                     elements[5],
		GroupControlNumber:       elements[6],
		ResponsibleAgencyCode:    elements[7],
		VersionReleaseIndustryID: elements[8],
	}
}

func parseGE(elements []string) GE {
	return GE{
		NumberOfIncludedTransactionSets: elements[1],
		GroupControlNumber:              elements[2],
	}
}

func parseST(elements []string) ST {
	return ST{
		TransactionSetIDCode:        elements[1],
		TransactionSetControlNumber: elements[2],
	}
}

func parseSE(elements []string) SE {
	return SE{
		NumberOfIncludedSegments:    elements[1],
		TransactionSetControlNumber: elements[2],
	}
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

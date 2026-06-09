package x12

// Document is the root element of an X12 document.
type Document struct {
	Interchange *Interchange

	// EnvelopeAutomaticallyAdded is true if the envelope was automatically added to a decoded document.
	// This may be the case if the document was decoded from a file that did not contain an ISA/IEA envelope.
	EnvelopeAutomaticallyAdded bool
}

// Interchange is the envelope for an X12 interchange.
type Interchange struct {
	Header         *ISA
	FunctionGroups []*FunctionGroup
	Trailer        *IEA
}

// ISA is the Interchange Control Header.
type ISA struct {
	AuthorizationInfoQualifier string // ISA01
	AuthorizationInformation   string // ISA02
	SecurityInfoQualifier      string // ISA03
	SecurityInfo               string // ISA04
	SenderIDQualifier          string // ISA05
	SenderID                   string // ISA06
	ReceiverIDQualifier        string // ISA07
	ReceiverID                 string // ISA08
	Date                       string // ISA09, YYMMDD
	Time                       string // ISA10, HHMM

	// RepetitionSeparator is ISA11. In version 5010 and later it holds
	// the repetition separator (commonly "^"). In earlier versions the
	// same position carried the interchange control standards
	// identifier, conventionally "U"; the value is preserved verbatim
	// either way.
	RepetitionSeparator string

	Version                   string // ISA12, e.g. "00501"
	ControlNumber             string // ISA13
	AcknowledgmentRequested   string // ISA14
	UsageIndicator            string // ISA15, "P" production or "T" test
	ComponentElementSeparator string // ISA16
}

// IEA is the Interchange Control Trailer.
type IEA struct {
	FunctionalGroupCount string // IEA01
	ControlNumber        string // IEA02
}

// FunctionGroup is a group of transactions.
type FunctionGroup struct {
	Header       *GS
	Transactions []*Transaction
	Trailer      *GE
}

// GS is the Functional Group Header.
type GS struct {
	FunctionalIDCode      string // GS01
	SenderCode            string // GS02
	ReceiverCode          string // GS03
	Date                  string // GS04, CCYYMMDD
	Time                  string // GS05, HHMM
	ControlNumber         string // GS06
	ResponsibleAgencyCode string // GS07
	Version               string // GS08, e.g. "005010X222A1"
}

// GE is the Functional Group Trailer.
type GE struct {
	TransactionSetCount string // GE01
	ControlNumber       string // GE02
}

// Transaction is a single transaction.
type Transaction struct {
	Header   *ST
	Segments []Segment
	Trailer  *SE
}

// ST is the Transaction Set Header.
type ST struct {
	IDCode                            string // ST01, e.g. "837"
	ControlNumber                     string // ST02
	ImplementationConventionReference string // ST03
}

// SE is the Transaction Set Trailer.
type SE struct {
	SegmentCount  string // SE01, includes the ST and SE segments
	ControlNumber string // SE02
}

// Segment is a single segment.
// A segment is a single line of an X12 document.
type Segment struct {
	ID       string
	Elements []Element
}

// Element is a single element.
// An element is a single value in a segment. Its position within the
// segment is its index in the segment's Elements slice.
type Element struct {
	Value      string
	Components []string `json:",omitempty"`
}

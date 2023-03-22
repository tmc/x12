package x12

// X12Document is the root element of an X12 document.
type X12Document struct {
	Interchange Interchange
}

// Interchange is the envelope for an X12 interchange.
type Interchange struct {
	Header         ISA
	FunctionGroups []FunctionGroup
	Trailer        IEA
}

// ISA is the Interchange Control Headera.
type ISA struct {
	AuthorizationInfoQualifier     string
	AuthorizationInformation       string
	SecurityInfoQualifier          string
	SecurityInfo                   string
	InterchangeSenderIDQualifier   string
	InterchangeSenderID            string
	InterchangeReceiverIDQualifier string
	InterchangeReceiverID          string
	InterchangeDate                string
	InterchangeTime                string
	InterchangeControlStandardsID  string
	InterchangeControlVersion      string
	InterchangeControlNumber       string
	AcknowledgmentRequested        string
	UsageIndicator                 string
	ComponentElementSeparator      string
}

// IEA is the Interchange Control Trailer.
type IEA struct {
	NumberOfIncludedFunctionalGroups string
	InterchangeControlNumber         string
}

// FunctionGroup is a group of transactions.
type FunctionGroup struct {
	Header       GS
	Transactions []Transaction
	Trailer      GE
}

// GS is the Functional Group Header.
type GS struct {
	FunctionalIDCode         string
	ApplicationSenderCode    string
	ApplicationReceiverCode  string
	Date                     string
	Time                     string
	GroupControlNumber       string
	ResponsibleAgencyCode    string
	VersionReleaseIndustryID string
}

// GE is the Functional Group Trailer.
type GE struct {
	NumberOfIncludedTransactionSets string
	GroupControlNumber              string
}

// Transaction is a single transaction.
type Transaction struct {
	Header   ST
	Segments []Segment
	Trailer  SE
}

// ST is the Transaction Set Header.
type ST struct {
	TransactionSetIDCode        string
	TransactionSetControlNumber string
}

// SE is the Transaction Set Trailer.
type SE struct {
	NumberOfIncludedSegments    string
	TransactionSetControlNumber string
}

// Segment is a single segment.
// A segment is a single line of an X12 document.
type Segment struct {
	ID       string
	Elements []Element
}

// Element is a single element.
// An element is a single value in a segment.
type Element struct {
	ID         string
	Value      string
	Components []string
}

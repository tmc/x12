// Based on http://www.wpc-edi.com/reference/repository/006020CC.PDF

package x12

type ID string

// AN represents an alphanumeric field
type AN string

// SegmentType represents the type of segment in an X12 document.
type SegmentType string

const (
	// ISA Interchange Control Header Segment
	SegmentTypeISA SegmentType = "ISA"
	// GS  Function Group Header Segment
	SegmentTypeGS SegmentType = "GS"
	// ST  Transaction Set Header Segment
	SegmentTypeST SegmentType = "ST"
	// SE  Transaction Set Trailer Segment
	SegmentTypeSE SegmentType = "SE"
	// GE  Function Group Trailer Segment
	SegmentTypeGE SegmentType = "GE"
	// IEA Interchange Control Trailer Segment
	SegmentTypeIEA SegmentType = "IEA"
)

const isaLength = 106

type ISASegment struct {
	AuthorizationInformationQualifier ID   // ISA01 Authorization Information Qualifier - Code identifying the type of information in the Authorization Information.
	AuthorizationInformation          AN   // ISA02 Authorization Information - Information used for additional identification or authorization of the interchange sender or the data in the interchange; the type of information is set by the Authorization Information Qualifier (I01).
	SecurityInformationQualifier      ID   // ISA03 Security Information Qualifier - Code identifying the type of information in the Security Information
	SecurityInformation               AN   // ISA04 Security Information - This is used for identifying the security information about the interchange sender or the data in the interchange; the type of information is set by the Security Information Qualifier (I03).
	InterchangeSenderIDQualifier      ID   // ISA05 Interchange ID Qualifier - Code indicating the system/method of code structure used to designate the sender or receiver ID element being qualified. This ID qualifies the Sender in ISA06.
	InterchangeSenderID               ID   // ISA06 Interchange Sender ID - Identification code published by the sender for other parties to use as the receiver ID to route data to them; the sender always codes this value in the sender ID element.
	InterchangeReceiverIDQualifier    ID   // ISA07 Interchange ID Qualifier - Code indicating the system/method of code structure used to designate the sender or receiver ID element being qualified.
	InterchangeReceiverID             ID   // ISA08 Interchange Receiver ID - Identification code published by the receiver of the data; When sending, it is used by the sender as their sending ID, thus other parties sending to them will use this as a receiving ID to route data to them.
	InterchangeDate                   ID   // ISA09 Interchange Date - Date of the interchange. The date format is YYMMDD.
	InterchangeTime                   ID   // ISA10 Interchange Time - Time of the interchange. The time format is HHMM.
	RepetitionSeparator               byte // ISA11 Repetition Separator - The repetition separator is a delimiter and not a data element; this field provides the delimiter used to separate repeated occurrences of a simple data element or a composite data structure; this value must be different than the data element separator, component element separator, and the segment terminator.
	InterchangeControlVersionNumber   ID   // ISA12 Interchange Control Version Number - Code specifying the version number of the interchange control segments.
	InterchangeControlNumber          int  // ISA13 Interchange Control Number - A control number assigned by the interchange sender.
	AcknowledgementRequested          bool // ISA14 Acknowledgement Requested - Code indicating sender’s request for an interchange acknowledgment.
	TestIndicator                     bool // ISA15 Test Indicator - Code indicating whether data enclosed by this interchange envelope is test, production or information. P - Production, T - Test.
	ComponentElementSeparator         ID   // ISA16 Component Element Separator - The component element separator is a delimiter and not a data element; this field provides the delimiter used to separate component data elements within a composite data structure; this value must be different than the data element separator and the segment terminator.
}

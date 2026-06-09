// Package x12 implements a parser and encoder for the ANSI X12 EDI format.
//
// It focuses on the 5010 version of the format, but is flexible such that
// it can be used for other versions as well.
//
// # Document model
//
// An X12 interchange maps onto the types in this package as follows:
//
//	Document
//	└── Interchange             ISA ... IEA
//	    └── FunctionGroup       GS ... GE
//	        └── Transaction     ST ... SE
//	            └── Segment     e.g. NM1*41*2*ACME~
//	                └── Element
//
// The envelope segments (ISA/IEA, GS/GE, ST/SE) are decoded into
// dedicated header and trailer structs; every other segment is
// represented as a generic Segment holding its Elements.
//
// Element values are kept as strings, exactly as they appear in the
// input. The package does not interpret dates, times, numbers, or code
// values, and it does not validate segments against a transaction-set
// implementation guide. Composite and repeated element values are not
// split: a value containing component (ISA16) or repetition (ISA11)
// separators is preserved verbatim, and the separators themselves are
// available on the decoded document for callers that split values
// further.
//
// # Decoding and encoding
//
// Decode parses a document from an io.Reader:
//
//	doc, err := x12.Decode(r)
//
// If the input begins with an ST segment instead of an ISA envelope, a
// minimal envelope is synthesized and the document's
// EnvelopeAutomaticallyAdded field is set.
//
// Validate checks that the envelope is structurally sound: headers and
// trailers are present, their control numbers match, and the trailer
// counts (IEA01, GE01, SE01) match the document's contents.
package x12

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
//
// # Errors
//
// Syntax errors found while decoding are reported as a *ParseError,
// which records the offending segment's ordinal, ID, and element.
// Both decoding and validation errors wrap the package's sentinel
// errors (ErrMissingElement, ErrInvalidFormat, ErrInvalidArgument) and
// can be matched with errors.Is.
//
// # Design
//
// The package materializes each interchange as an in-memory Document;
// Decoder and Encoder stream bytes, not events. This suits the common
// case of inspecting or transforming whole interchanges, at the cost
// of holding a document in memory while it is processed. An event- or
// segment-level streaming API, and typed transaction-set layers (837,
// 835, ...) validated against implementation guides, are out of scope
// and belong in packages built on top of this one.
package x12

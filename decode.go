package x12

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Default delimiters, used when encoding and decoding unless overridden.
const (
	// DefaultSegmentTerminator terminates each segment.
	DefaultSegmentTerminator = "~"
	// DefaultElementSeparator separates the elements of a segment.
	DefaultElementSeparator = "*"
	// DefaultComponentSeparator separates the components of a composite
	// element.
	DefaultComponentSeparator = ":"

	// When encountering an unknown segment use this parser.
	defaultParser = "DEFAULT"
)

// Sentinel errors wrapped by decoding, encoding, and validation
// failures; match them with errors.Is.
var (
	// ErrMissingElement reports a segment that lacks a required element.
	ErrMissingElement = errors.New("missing element")
	// ErrInvalidFormat reports input or a document that is not
	// well-formed X12, such as a malformed segment or a mismatched
	// trailer.
	ErrInvalidFormat = errors.New("invalid format")
	// ErrInvalidArgument reports an invalid caller-supplied value, such
	// as a nil document passed to Marshal.
	ErrInvalidArgument = errors.New("invalid argument")
)

type decodeState struct {
	doc                  *Document
	lineIndex            int
	currentFunctionGroup *FunctionGroup
	currentTransaction   *Transaction

	// elementSeparator is the element separator in effect, discovered
	// from the ISA segment when present.
	elementSeparator string

	withRelaxedSegmentIDWhitespace bool
	strictSegments                 bool
}

// DecodeOption is a function that can be used to configure the decoder.
type DecodeOption func(*decodeState)

// WithRelaxedSegmentIDWhitespace allows the decoder to accept segment IDs with leading and trailing whitespace.
func WithRelaxedSegmentIDWhitespace() DecodeOption {
	return func(state *decodeState) {
		state.withRelaxedSegmentIDWhitespace = true
	}
}

// WithStrictSegments rejects data segments that the decoder would
// otherwise absorb silently: segments whose ID does not look like an
// X12 segment identifier (two or three characters, an uppercase letter
// followed by uppercase letters or digits), and segments that appear
// after a transaction's SE trailer.
func WithStrictSegments() DecodeOption {
	return func(state *decodeState) {
		state.strictSegments = true
	}
}

// A Decoder reads an X12 document from an input stream.
type Decoder struct {
	r    io.Reader
	opts []DecodeOption
}

// NewDecoder returns a new Decoder that reads from r.
func NewDecoder(r io.Reader, opts ...DecodeOption) *Decoder {
	return &Decoder{r: r, opts: opts}
}

// Decode reads the X12 document from the decoder's input.
//
// If the input begins with an ISA segment, the delimiters are discovered
// from it: the element separator is the byte following the segment ID,
// the component element separator is ISA16, and the segment terminator
// is the byte following ISA16. Otherwise the default delimiters are
// assumed.
//
// Decode returns io.EOF if the input contains no segments.
func (dec *Decoder) Decode() (*Document, error) {
	state := initializeDecodeState(dec.opts)
	segmentParsers := state.getSegmentParsers()

	r := bufio.NewReader(dec.r)
	term := DefaultSegmentTerminator[0]
	if peek, err := r.Peek(4); err == nil && string(peek[:3]) == "ISA" && !isAlnum(peek[3]) {
		elemSep, t, err := state.readISA(r)
		if err != nil {
			return nil, err
		}
		term = t
		state.elementSeparator = string(elemSep)
		state.doc.SegmentTerminator = string(t)
		state.doc.ElementSeparator = string(elemSep)
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(nil, maxSegmentSize)
	scanner.Split(scanSegments(term))
	for scanner.Scan() {
		if err := state.processLine(scanner.Text(), segmentParsers); err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("x12: segment %d: %w", state.lineIndex+1, err)
	}
	if state.lineIndex == 0 {
		return nil, io.EOF
	}
	return state.doc, nil
}

// Decode decodes an X12 document from an io.Reader.
//
// Like Decoder.Decode, it returns io.EOF if the input contains no
// segments.
func Decode(in io.Reader, opts ...DecodeOption) (*Document, error) {
	return NewDecoder(in, opts...).Decode()
}

func initializeDecodeState(opts []DecodeOption) *decodeState {
	state := &decodeState{
		doc: &Document{
			Interchange: &Interchange{},
		},
		elementSeparator: DefaultElementSeparator,
	}
	for _, opt := range opts {
		opt(state)
	}
	return state
}

// isaLen is the length of a canonical fixed-width ISA segment,
// including the segment terminator.
const isaLen = 106

// maxSegmentSize bounds a single segment. A segment this large almost
// always means the segment terminator was wrong, not that the data is
// real; exceeding it surfaces as a wrapped bufio.ErrTooLong.
const maxSegmentSize = 1 << 20

// isaSeparatorOffsets are the byte offsets of the element separators in
// a canonical fixed-width ISA segment. ISA16 occupies the byte before
// the segment terminator at offset 105.
var isaSeparatorOffsets = [...]int{3, 6, 17, 20, 31, 34, 50, 53, 69, 76, 81, 83, 89, 99, 101, 103}

// readISA reads the ISA segment from r and discovers the document's
// delimiters from it. It first tries the canonical fixed-width form and
// falls back to scanning separator-delimited elements, which accepts the
// padded variants that appear in the wild. It returns the element
// separator and the segment terminator.
func (s *decodeState) readISA(r *bufio.Reader) (elemSep, term byte, err error) {
	s.lineIndex = 1
	if buf, perr := r.Peek(isaLen); perr == nil {
		if elements, ok := parseCanonicalISA(buf); ok {
			if err := s.parseISA(elements); err != nil {
				return 0, 0, err
			}
			elemSep, term = buf[3], buf[105]
			if _, err := r.Discard(isaLen); err != nil {
				return 0, 0, err
			}
			return elemSep, term, nil
		}
	}
	return s.readISAVariable(r)
}

// parseCanonicalISA splits a canonical fixed-width ISA segment into its
// elements. It reports ok=false if buf is not such a segment.
func parseCanonicalISA(buf []byte) (elements []string, ok bool) {
	sep := buf[3]
	if isAlnum(sep) {
		return nil, false
	}
	for _, off := range isaSeparatorOffsets {
		if buf[off] != sep {
			return nil, false
		}
	}
	// The same sanity checks readISAVariable applies to ISA16 and the
	// terminator; on failure the caller falls back to it for a precise
	// error.
	if isa16 := buf[104]; isa16 == sep || isa16 == '\r' || isa16 == '\n' {
		return nil, false
	}
	if term := buf[105]; term == sep || isAlnum(term) {
		return nil, false
	}
	elements = make([]string, 0, 17)
	elements = append(elements, "ISA")
	for i, off := range isaSeparatorOffsets[:len(isaSeparatorOffsets)-1] {
		elements = append(elements, string(buf[off+1:isaSeparatorOffsets[i+1]]))
	}
	elements = append(elements, string(buf[104:105])) // ISA16
	return elements, true
}

// readISAVariable reads an ISA segment of non-canonical shape: elements
// may have any width, but there must be 16 of them, ISA16 must be a
// single byte, and the byte after ISA16 is the segment terminator.
func (s *decodeState) readISAVariable(r *bufio.Reader) (elemSep, term byte, err error) {
	if _, err := r.Discard(3); err != nil { // the "ISA" segment ID
		return 0, 0, err
	}
	b, err := r.ReadByte()
	if err != nil {
		return 0, 0, s.parseErrorf("ISA", 1, "%w", ErrMissingElement)
	}
	if b == ' ' || b == '\t' {
		if !s.withRelaxedSegmentIDWhitespace {
			return 0, 0, s.parseErrorf("ISA", 0, "%w: whitespace after segment ID (use WithRelaxedSegmentIDWhitespace)", ErrInvalidFormat)
		}
		for b == ' ' || b == '\t' {
			if b, err = r.ReadByte(); err != nil {
				return 0, 0, s.parseErrorf("ISA", 1, "%w", ErrMissingElement)
			}
		}
	}
	sep := b
	if isAlnum(sep) {
		return 0, 0, s.parseErrorf("ISA", 0, "%w: invalid element separator %q", ErrInvalidFormat, sep)
	}
	elements := []string{"ISA"}
	var field []byte
	for n := 0; len(elements) < 16; n++ {
		if n > 512 {
			return 0, 0, s.parseErrorf("ISA", 0, "%w: unterminated segment", ErrInvalidFormat)
		}
		b, err := r.ReadByte()
		if err != nil {
			return 0, 0, s.parseErrorf("ISA", len(elements), "%w", ErrMissingElement)
		}
		switch b {
		case sep:
			elements = append(elements, string(field))
			field = field[:0]
		case '\r', '\n':
			// A newline inside the ISA means we ran past the end of
			// the segment without finding all of its elements.
			return 0, 0, s.parseErrorf("ISA", len(elements), "%w", ErrMissingElement)
		default:
			field = append(field, b)
		}
	}
	isa16, err := r.ReadByte()
	if err != nil || isa16 == sep || isa16 == '\r' || isa16 == '\n' {
		return 0, 0, s.parseErrorf("ISA", 16, "%w", ErrMissingElement)
	}
	elements = append(elements, string(isa16))
	term, err = r.ReadByte()
	if err != nil {
		return 0, 0, s.parseErrorf("ISA", 0, "%w: unterminated segment", ErrInvalidFormat)
	}
	if term == sep || isAlnum(term) {
		return 0, 0, s.parseErrorf("ISA", 0, "%w: invalid segment terminator %q", ErrInvalidFormat, term)
	}
	if err := s.parseISA(elements); err != nil {
		return 0, 0, err
	}
	return sep, term, nil
}

func isAlnum(b byte) bool {
	return 'A' <= b && b <= 'Z' || 'a' <= b && b <= 'z' || '0' <= b && b <= '9'
}

// scanSegments returns a bufio.SplitFunc that splits an EDI document
// into segments on the given segment terminator.
func scanSegments(term byte) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, term); i >= 0 {
			// We have a full segment.
			return i + 1, data[0:i], nil
		}
		// If we're at EOF, we have a final, non-empty, non-terminated segment. Return it.
		if atEOF {
			return len(data), data, nil
		}
		// Request more data.
		return 0, nil, nil
	}
}

func (s *decodeState) processLine(line string, parsers map[string]segmentParser) error {
	segment := strings.Trim(line, "\r\n")
	if segment == "" {
		// Stray terminators and blank lines are not segments; they do
		// not advance the segment ordinal used by ParseError and the
		// automatic-envelope check.
		return nil
	}
	s.lineIndex++

	elements := strings.Split(segment, s.elementSeparator)
	segmentID, _ := s.extractSegmentID(elements)

	parseFunc, exists := parsers[segmentID]
	if !exists {
		parseFunc = parsers[defaultParser]
	}

	return parseFunc(s, elements)
}

// Validate checks that the document's envelope is structurally sound:
// header and trailer segments are present, their control numbers match,
// and the trailer counts (IEA01, GE01, SE01) match the document's
// contents. It does not validate segments against a transaction-set
// implementation guide.
func (doc *Document) Validate() error {
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
	if !controlNumbersMatch(doc.Interchange.Header.ControlNumber, doc.Interchange.Trailer.ControlNumber) {
		return fmt.Errorf("%w: ISA and IEA control numbers do not match (%v != %v)", ErrInvalidFormat, doc.Interchange.Header.ControlNumber, doc.Interchange.Trailer.ControlNumber)
	}
	if err := checkCount("IEA01 functional group count", doc.Interchange.Trailer.FunctionalGroupCount, len(doc.Interchange.FunctionGroups)); err != nil {
		return err
	}

	// check that the GS and GE segments are present and match
	for _, functionGroup := range doc.Interchange.FunctionGroups {
		if functionGroup == nil {
			return fmt.Errorf("%w: nil function group", ErrInvalidFormat)
		}
		if functionGroup.Header == nil {
			return fmt.Errorf("%w: GS segment missing", ErrInvalidFormat)
		}
		if functionGroup.Trailer == nil {
			return fmt.Errorf("%w: GE segment missing", ErrInvalidFormat)
		}
		if !controlNumbersMatch(functionGroup.Header.ControlNumber, functionGroup.Trailer.ControlNumber) {
			return fmt.Errorf("%w: GS and GE control numbers do not match (%v != %v)", ErrInvalidFormat, functionGroup.Header.ControlNumber, functionGroup.Trailer.ControlNumber)
		}
		if err := checkCount("GE01 transaction set count", functionGroup.Trailer.TransactionSetCount, len(functionGroup.Transactions)); err != nil {
			return err
		}
	}

	// check that the ST and SE segments are present and match
	for _, functionGroup := range doc.Interchange.FunctionGroups {
		for _, transaction := range functionGroup.Transactions {
			if transaction == nil {
				return fmt.Errorf("%w: nil transaction", ErrInvalidFormat)
			}
			if transaction.Header == nil {
				return fmt.Errorf("%w: ST segment missing", ErrInvalidFormat)
			}
			if transaction.Trailer == nil {
				return fmt.Errorf("%w: SE segment missing", ErrInvalidFormat)
			}
			if !controlNumbersMatch(transaction.Header.ControlNumber, transaction.Trailer.ControlNumber) {
				return fmt.Errorf("%w: ST and SE control numbers do not match (%v != %v)", ErrInvalidFormat, transaction.Header.ControlNumber, transaction.Trailer.ControlNumber)
			}
			// SE01 counts every segment in the transaction set,
			// including the ST and SE segments themselves.
			if err := checkCount("SE01 segment count", transaction.Trailer.SegmentCount, len(transaction.Segments)+2); err != nil {
				return err
			}
		}
	}
	return nil
}

// controlNumbersMatch compares control numbers ignoring surrounding
// whitespace: ISA13 is a fixed-width field and may carry padding that
// its IEA02 counterpart lacks.
func controlNumbersMatch(a, b string) bool {
	return strings.TrimSpace(a) == strings.TrimSpace(b)
}

// checkCount verifies that a trailer count element matches the number of
// units actually present in the document.
func checkCount(what, value string, actual int) error {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("%w: %s %q is not a number", ErrInvalidFormat, what, value)
	}
	if n != actual {
		return fmt.Errorf("%w: %s is %d, document contains %d", ErrInvalidFormat, what, n, actual)
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
	if s.doc.Interchange.Header != nil {
		// A Document holds a single interchange; a second ISA would
		// silently overwrite the first.
		return s.parseErrorf("ISA", 0, "%w: multiple ISA segments", ErrInvalidFormat)
	}
	if len(elements) < 17 {
		return s.parseErrorf("ISA", len(elements), "%w", ErrMissingElement)
	}
	s.doc.Interchange.Header = &ISA{
		AuthorizationInfoQualifier: elements[1],
		AuthorizationInformation:   elements[2],
		SecurityInfoQualifier:      elements[3],
		SecurityInfo:               elements[4],
		SenderIDQualifier:          elements[5],
		SenderID:                   elements[6],
		ReceiverIDQualifier:        elements[7],
		ReceiverID:                 elements[8],
		Date:                       elements[9],
		Time:                       elements[10],
		RepetitionSeparator:        elements[11],
		Version:                    elements[12],
		ControlNumber:              elements[13],
		AcknowledgmentRequested:    elements[14],
		UsageIndicator:             elements[15],
		ComponentElementSeparator:  elements[16],
	}
	return nil
}

func (s *decodeState) parseIEA(elements []string) error {
	if s.doc.Interchange.Trailer != nil {
		return s.parseErrorf("IEA", 0, "%w: multiple IEA segments", ErrInvalidFormat)
	}
	if len(elements) < 3 {
		return s.parseErrorf("IEA", len(elements), "%w", ErrMissingElement)
	}
	s.doc.Interchange.Trailer = &IEA{
		FunctionalGroupCount: elements[1],
		ControlNumber:        elements[2],
	}
	return nil
}

func (s *decodeState) parseGS(elements []string) error {
	if len(elements) < 9 {
		return s.parseErrorf("GS", len(elements), "%w", ErrMissingElement)
	}
	s.currentFunctionGroup = &FunctionGroup{
		Header: &GS{
			FunctionalIDCode:      elements[1],
			SenderCode:            elements[2],
			ReceiverCode:          elements[3],
			Date:                  elements[4],
			Time:                  elements[5],
			ControlNumber:         elements[6],
			ResponsibleAgencyCode: elements[7],
			Version:               elements[8],
		},
	}
	s.doc.Interchange.FunctionGroups = append(s.doc.Interchange.FunctionGroups, s.currentFunctionGroup)
	return nil
}

func (s *decodeState) parseGE(elements []string) error {
	if s.currentFunctionGroup == nil {
		return s.parseErrorf("GE", 0, "%w: GE segment without GS segment", ErrInvalidFormat)
	}
	if s.strictSegments && s.currentFunctionGroup.Trailer != nil && !s.doc.EnvelopeAutomaticallyAdded {
		return s.parseErrorf("GE", 0, "%w: duplicate GE segment", ErrInvalidFormat)
	}
	if len(elements) < 3 {
		return s.parseErrorf("GE", len(elements), "%w", ErrMissingElement)
	}
	s.currentFunctionGroup.Trailer = &GE{
		TransactionSetCount: elements[1],
		ControlNumber:       elements[2],
	}
	return nil
}

func (s *decodeState) parseST(elements []string) error {
	if len(elements) < 3 {
		return s.parseErrorf("ST", len(elements), "%w", ErrMissingElement)
	}
	s.considerAutomaticEnvelope()
	if s.currentFunctionGroup == nil {
		return s.parseErrorf("ST", 0, "%w: ST segment without GS segment", ErrInvalidFormat)
	}
	if s.doc.EnvelopeAutomaticallyAdded && len(s.currentFunctionGroup.Transactions) > 0 {
		// The synthesized envelope declares exactly one transaction
		// (and Encode emits only one); accepting more would produce a
		// document that fails its own Validate.
		return s.parseErrorf("ST", 0, "%w: multiple transaction sets without an interchange envelope", ErrInvalidFormat)
	}
	s.currentTransaction = &Transaction{
		Header: &ST{
			IDCode:        elements[1],
			ControlNumber: elements[2],
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
		return s.parseErrorf("SE", 0, "%w: SE segment without ST segment", ErrInvalidFormat)
	}
	if s.strictSegments && s.currentTransaction.Trailer != nil {
		return s.parseErrorf("SE", 0, "%w: duplicate SE segment", ErrInvalidFormat)
	}
	if len(elements) < 3 {
		return s.parseErrorf("SE", len(elements), "%w", ErrMissingElement)
	}
	s.currentTransaction.Trailer = &SE{
		SegmentCount:  elements[1],
		ControlNumber: elements[2],
	}
	return nil
}

func (s *decodeState) parseSegment(elements []string) error {
	segmentID, elements := s.extractSegmentID(elements)
	if s.currentTransaction == nil {
		return s.parseErrorf(segmentID, 0, "%w: segment without ST segment", ErrInvalidFormat)
	}
	if s.strictSegments {
		if !isValidSegmentID(segmentID) {
			return s.parseErrorf(segmentID, 0, "%w: invalid segment ID %q", ErrInvalidFormat, segmentID)
		}
		if s.currentTransaction.Trailer != nil {
			return s.parseErrorf(segmentID, 0, "%w: segment after SE trailer", ErrInvalidFormat)
		}
	}
	segment := Segment{
		ID:       segmentID,
		Elements: parseElements(elements),
	}
	s.currentTransaction.Segments = append(s.currentTransaction.Segments, segment)
	return nil
}

// isValidSegmentID reports whether id looks like an X12 segment
// identifier: two or three characters, an uppercase letter followed by
// uppercase letters or digits.
func isValidSegmentID(id string) bool {
	if len(id) < 2 || len(id) > 3 {
		return false
	}
	if id[0] < 'A' || id[0] > 'Z' {
		return false
	}
	for i := 1; i < len(id); i++ {
		b := id[i]
		if !('A' <= b && b <= 'Z' || '0' <= b && b <= '9') {
			return false
		}
	}
	return true
}

func parseElements(elements []string) []Element {
	parsedElements := make([]Element, len(elements))
	for i, element := range elements {
		parsedElements[i] = Element{Value: element}
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
		ControlNumber:             "000000001",
		ComponentElementSeparator: DefaultComponentSeparator,
	}

	s.doc.Interchange.Trailer = &IEA{
		FunctionalGroupCount: "1",
		ControlNumber:        "000000001",
	}

	s.currentFunctionGroup = &FunctionGroup{
		Header: &GS{
			ControlNumber: "000000001",
		},
		Trailer: &GE{
			TransactionSetCount: "1",
			ControlNumber:       "000000001",
		},
	}
	s.doc.Interchange.FunctionGroups = append(s.doc.Interchange.FunctionGroups, s.currentFunctionGroup)
}

// A ParseError describes a syntax error encountered while decoding an
// X12 document. It wraps one of the package's sentinel errors, so it
// can be matched with errors.Is, and carries the position of the
// offending segment for inputs where line numbers are meaningless
// (an entire interchange is often a single line).
type ParseError struct {
	Segment   int    // 1-based ordinal of the segment within the input
	SegmentID string // segment ID, e.g. "ISA", if known
	Element   int    // 1-based index of the offending element, if known
	Err       error
}

func (e *ParseError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "x12: segment %d", e.Segment)
	if e.SegmentID != "" {
		fmt.Fprintf(&b, " (%s)", e.SegmentID)
	}
	if e.Element > 0 {
		fmt.Fprintf(&b, " element %d", e.Element)
	}
	b.WriteString(": ")
	b.WriteString(e.Err.Error())
	return b.String()
}

func (e *ParseError) Unwrap() error { return e.Err }

// parseErrorf returns a *ParseError for the segment currently being
// decoded. element is the 1-based index of the offending element, or 0
// if not applicable.
func (s *decodeState) parseErrorf(segmentID string, element int, format string, args ...any) error {
	return &ParseError{
		Segment:   s.lineIndex,
		SegmentID: segmentID,
		Element:   element,
		Err:       fmt.Errorf(format, args...),
	}
}

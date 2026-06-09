package x12_test

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tmc/x12"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *x12.Document
		wantErr error

		validateResult string
	}{
		{
			name: "Test1",
			input: `ISA*00*          *00*          *08*9254110060     *ZZ*123456789      *041216*0805*U*00501*000095071*0*P*>~
GS*AG*5137624388*123456789*20041216*0805*95071*X*005010~
ST*824*021390001*005010X186A1~
BGN*11*FFA.ABCDEF.123456*20020709*0932**123456789**WQ~
N1*41*ABC INSURANCE*46*111111111~
PER*IC*JOHN JOHNSON*TE*8005551212*EX*1439~
N1*40*SMITHCO*46*A1234~
OTI*TA*TN*NA***20020709*0902*2*0001*834*005010X220A1~
SE*7*021390001~
GE*1*95071~
IEA*1*000095071~`,
			want: &x12.Document{
				Interchange: &x12.Interchange{
					Header: &x12.ISA{
						AuthorizationInfoQualifier: "00",
						AuthorizationInformation:   "          ",
						SecurityInfoQualifier:      "00",
						SecurityInfo:               "          ",
						SenderIDQualifier:          "08",
						SenderID:                   "9254110060     ",
						ReceiverIDQualifier:        "ZZ",
						ReceiverID:                 "123456789      ",
						Date:                       "041216",
						Time:                       "0805",
						RepetitionSeparator:        "U",
						Version:                    "00501",
						ControlNumber:              "000095071",
						AcknowledgmentRequested:    "0",
						UsageIndicator:             "P",
						ComponentElementSeparator:  ">",
					},
					FunctionGroups: []*x12.FunctionGroup{
						{
							Header: &x12.GS{
								FunctionalIDCode:      "AG",
								SenderCode:            "5137624388",
								ReceiverCode:          "123456789",
								Date:                  "20041216",
								Time:                  "0805",
								ControlNumber:         "95071",
								ResponsibleAgencyCode: "X",
								Version:               "005010",
							},
							Transactions: []*x12.Transaction{
								{
									Header: &x12.ST{
										IDCode:                            "824",
										ControlNumber:                     "021390001",
										ImplementationConventionReference: "005010X186A1",
									},
									Segments: []x12.Segment{
										{
											ID: "BGN",
											Elements: []x12.Element{
												{Value: "11"}, {Value: "FFA.ABCDEF.123456"},
												{Value: "20020709"}, {Value: "0932"}, {},
												{Value: "123456789"}, {}, {Value: "WQ"},
											},
										},
										{
											ID: "N1",
											Elements: []x12.Element{
												{Value: "41"}, {Value: "ABC INSURANCE"},
												{Value: "46"}, {Value: "111111111"},
											},
										},
										{
											ID: "PER",
											Elements: []x12.Element{
												{Value: "IC"}, {Value: "JOHN JOHNSON"},
												{Value: "TE"}, {Value: "8005551212"},
												{Value: "EX"}, {Value: "1439"},
											},
										},
										{
											ID: "N1",
											Elements: []x12.Element{
												{Value: "40"}, {Value: "SMITHCO"}, {Value: "46"},
												{Value: "A1234"},
											},
										},
										{
											ID: "OTI",
											Elements: []x12.Element{
												{Value: "TA"}, {Value: "TN"}, {Value: "NA"},
												{}, {}, {Value: "20020709"},
												{Value: "0902"}, {Value: "2"},

												{Value: "0001"},
												{Value: "834"},
												{Value: "005010X220A1"},
											},
										},
									},
									Trailer: &x12.SE{SegmentCount: "7", ControlNumber: "021390001"},
								},
							},
							Trailer: &x12.GE{
								TransactionSetCount: "1",
								ControlNumber:       "95071",
							},
						},
					},
					Trailer: &x12.IEA{
						FunctionalGroupCount: "1",
						ControlNumber:        "000095071",
					},
				},
				SegmentTerminator: "~",
				ElementSeparator:  "*",
			},
			validateResult: "<nil>",
		},
		{
			name: "ISA Missing Element",
			input: `ISA*00*          *00*          *08*9254110060     *ZZ*123456789      *041216*0805*U*00501*000095071*0*P~
GS*AG*5137624388*123456789*20041216*0805*95071*X*005010~
ST*824*021390001*005010X186A1~
BGN*11*FFA.ABCDEF.123456*20020709*0932**123456789**WQ~
N1*41*ABC INSURANCE*46*111111111~
PER*IC*JOHN JOHNSON*TE*8005551212*EX*1439~
N1*40*SMITHCO*46*A1234~
OTI*TA*TN*NA***20020709*0902*2*0001*834*005010X220A1~
SE*7*021390001~
GE*1*95071~
IEA*1*000095071~`,
			want:           nil,
			wantErr:        x12.ErrMissingElement,
			validateResult: "<nil>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			got, err := x12.Decode(r)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Decode() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				//if diff := cmp.Diff(tt.want.Interchange.FunctionGroups[0].Transactions[0].Segments, got.Interchange.FunctionGroups[0].Transactions[0].Segments); diff != "" {
				t.Errorf("Decode() mismatch (-want +got):\n%s", diff)
			}
			validateErr := fmt.Sprint(got.Validate())
			if validateErr != tt.validateResult {
				t.Errorf("Validate() error = '%v', wantErr '%v'", validateErr, tt.validateResult)
			}
			encoded, err := x12.Marshal(got)
			trimmedInput := strings.ReplaceAll(tt.input, "\n", "")
			if err != nil {
				t.Errorf("Marshal() error = %v", err)
			}
			// test round-tripping
			if diff := cmp.Diff(trimmedInput, string(encoded)); diff != "" {
				t.Errorf("Marshal() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRoundtripping(t *testing.T) {
	// run through all *.edi files in the testdata directory and make sure we can decode and encode them without error.

	// x12.org includes a number of examples that have whitespace after the ISA segment ID, which is technically invalid.
	// we can relax this requirement by passing WithRelaxedSegmentIDWhitespace() to the decoder.
	optMap := map[string]struct {
		RelaxedWhitespace bool
	}{
		"005010x221-example-5a.edi": {RelaxedWhitespace: true},
		"005010x221-example-5b.edi": {RelaxedWhitespace: true},
		"005010x221-example-5c.edi": {RelaxedWhitespace: true},
		"005010x221-example-8a-claim-submitted-incorrect-subscriber-patient-and-incorrect-id.edi": {RelaxedWhitespace: true},
		"005010x221-example-8b-claim-submitted-incorrect-subscriber-name-and-id.edi":              {RelaxedWhitespace: true},
		"005010x221-example-8c-claim-submitted-subscriber-missing-middle-initial.edi":             {RelaxedWhitespace: true},
	}
	files, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".edi") {
			continue
		}
		t.Run(file.Name(), func(t *testing.T) {
			f, err := os.Open(filepath.Join("testdata", file.Name()))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			opts := []x12.DecodeOption{x12.WithStrictSegments()}
			if optMap[file.Name()].RelaxedWhitespace {
				opts = append(opts, x12.WithRelaxedSegmentIDWhitespace())
			}
			edi, err := x12.Decode(f, opts...)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := x12.Marshal(edi, x12.WithNewlines())
			if err != nil {
				t.Fatal(err)
			}
			// compare the original file to the encoded file

			// read the original file
			f.Seek(0, 0)
			original, err := io.ReadAll(f)
			if err != nil {
				t.Fatal(err)
			}

			cmpOpts := []cmp.Option{}
			if optMap[file.Name()].RelaxedWhitespace {
				cmpOpts = append(cmpOpts, cmpopts.AcyclicTransformer("TrimSegmentSpaces", func(in string) string {
					return strings.ReplaceAll(in, "ISA ", "ISA")
				}))
			}
			// compare the original file to the encoded file
			if diff := cmp.Diff(normalizeLineEndings(string(original)), normalizeLineEndings(string(encoded)), cmpOpts...); diff != "" {
				t.Errorf("Marshal() mismatch (-want +got):\n%s", diff)
			}

		})
	}

}

func normalizeLineEndings(input string) string {
	return strings.ReplaceAll(input, "\r\n", "\n")
}

func TestDecodeRelaxedSegmentIDWhitespace(t *testing.T) {
	// Modeled on the x12.org 005010x221 examples, which pad each ISA
	// element (including the segment ID) with trailing whitespace.
	const input = `ISA *00 *          *00 *          *ZZ *SENDER         *ZZ *RECEIVER       *190827 *0212 *^ *00501 *191511902 *0 *P *>~
GS*HP*SENDER*RECEIVER*20190827*0212*1*X*005010X221A1~
ST*835*0001~
BPR*I*11.06*C*CHK~
SE*3*0001~
GE*1*1~
IEA*1*191511902~`

	if _, err := x12.Decode(strings.NewReader(input)); !errors.Is(err, x12.ErrInvalidFormat) {
		t.Fatalf("Decode() without option: error = %v, want ErrInvalidFormat", err)
	}
	doc, err := x12.Decode(strings.NewReader(input), x12.WithRelaxedSegmentIDWhitespace())
	if err != nil {
		t.Fatalf("Decode() with option: %v", err)
	}
	if got, want := doc.Interchange.Header.RepetitionSeparator, "^ "; got != want {
		t.Errorf("ISA11 = %q, want %q", got, want)
	}
	if got, want := doc.Interchange.Header.ControlNumber, "191511902 "; got != want {
		t.Errorf("ISA13 = %q, want %q", got, want)
	}
	// ISA13 is space-padded while IEA02 is not; Validate must treat
	// them as matching.
	if err := doc.Validate(); err != nil {
		t.Errorf("Validate() = %v", err)
	}
}

func TestDecodeAutomaticEnvelope(t *testing.T) {
	const input = `ST*837*0001~NM1*41*2*PREMIER BILLING SERVICE~SE*3*0001~`
	doc, err := x12.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() = %v", err)
	}
	if !doc.EnvelopeAutomaticallyAdded {
		t.Error("EnvelopeAutomaticallyAdded = false, want true")
	}
	if got := len(doc.Interchange.FunctionGroups); got != 1 {
		t.Fatalf("len(FunctionGroups) = %d, want 1", got)
	}
	if got := len(doc.Interchange.FunctionGroups[0].Transactions); got != 1 {
		t.Fatalf("len(Transactions) = %d, want 1", got)
	}
	if err := doc.Validate(); err != nil {
		t.Errorf("Validate() = %v", err)
	}
	// Marshaling an automatically enveloped document emits only ST..SE.
	b, err := x12.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal() = %v", err)
	}
	if diff := cmp.Diff(input, string(b)); diff != "" {
		t.Errorf("Marshal() mismatch (-want +got):\n%s", diff)
	}
}

func TestMarshalHandBuilt(t *testing.T) {
	doc := &x12.Document{
		Interchange: &x12.Interchange{
			Header: &x12.ISA{
				AuthorizationInfoQualifier: "00",
				AuthorizationInformation:   "          ",
				SecurityInfoQualifier:      "00",
				SecurityInfo:               "          ",
				SenderIDQualifier:          "ZZ",
				SenderID:                   "SENDER         ",
				ReceiverIDQualifier:        "ZZ",
				ReceiverID:                 "RECEIVER       ",
				Date:                       "230101",
				Time:                       "1200",
				RepetitionSeparator:        "^",
				Version:                    "00501",
				ControlNumber:              "000000001",
				AcknowledgmentRequested:    "0",
				UsageIndicator:             "T",
				ComponentElementSeparator:  ":",
			},
			FunctionGroups: []*x12.FunctionGroup{{
				Header: &x12.GS{
					FunctionalIDCode:      "HC",
					SenderCode:            "SENDER",
					ReceiverCode:          "RECEIVER",
					Date:                  "20230101",
					Time:                  "1200",
					ControlNumber:         "1",
					ResponsibleAgencyCode: "X",
					Version:               "005010X222A1",
				},
				Transactions: []*x12.Transaction{{
					Header: &x12.ST{IDCode: "837", ControlNumber: "0001"},
					Segments: []x12.Segment{
						{ID: "BHT", Elements: []x12.Element{{Value: "0019"}, {Value: "00"}}},
						{ID: "HI", Elements: []x12.Element{{Value: "BK", Components: []string{"8901"}}}},
					},
					Trailer: &x12.SE{SegmentCount: "4", ControlNumber: "0001"},
				}},
				Trailer: &x12.GE{TransactionSetCount: "1", ControlNumber: "1"},
			}},
			Trailer: &x12.IEA{FunctionalGroupCount: "1", ControlNumber: "000000001"},
		},
	}
	const want = `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *230101*1200*^*00501*000000001*0*T*:~` +
		`GS*HC*SENDER*RECEIVER*20230101*1200*1*X*005010X222A1~` +
		`ST*837*0001~` +
		`BHT*0019*00~` +
		`HI*BK:8901~` +
		`SE*4*0001~` +
		`GE*1*1~` +
		`IEA*1*000000001~`
	b, err := x12.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal() = %v", err)
	}
	if diff := cmp.Diff(want, string(b)); diff != "" {
		t.Errorf("Marshal() mismatch (-want +got):\n%s", diff)
	}
}

func TestMarshalAutomaticEnvelopeGuard(t *testing.T) {
	doc := &x12.Document{EnvelopeAutomaticallyAdded: true, Interchange: &x12.Interchange{}}
	if _, err := x12.Marshal(doc); !errors.Is(err, x12.ErrInvalidArgument) {
		t.Errorf("Marshal() error = %v, want ErrInvalidArgument", err)
	}
}

func TestValidateCounts(t *testing.T) {
	decode := func(t *testing.T) *x12.Document {
		t.Helper()
		doc, err := x12.Decode(strings.NewReader(exampleEDI))
		if err != nil {
			t.Fatal(err)
		}
		return doc
	}
	tests := []struct {
		name   string
		mutate func(*x12.Document)
	}{
		{"IEA01 mismatch", func(d *x12.Document) { d.Interchange.Trailer.FunctionalGroupCount = "2" }},
		{"GE01 mismatch", func(d *x12.Document) { d.Interchange.FunctionGroups[0].Trailer.TransactionSetCount = "0" }},
		{"SE01 mismatch", func(d *x12.Document) { d.Interchange.FunctionGroups[0].Transactions[0].Trailer.SegmentCount = "99" }},
		{"SE01 not a number", func(d *x12.Document) { d.Interchange.FunctionGroups[0].Transactions[0].Trailer.SegmentCount = "x" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := decode(t)
			if err := doc.Validate(); err != nil {
				t.Fatalf("Validate() before mutation = %v", err)
			}
			tt.mutate(doc)
			if err := doc.Validate(); !errors.Is(err, x12.ErrInvalidFormat) {
				t.Errorf("Validate() = %v, want ErrInvalidFormat", err)
			}
		})
	}
}

func TestDecodeDiscoversDelimiters(t *testing.T) {
	// A canonical fixed-width ISA using | as the element separator,
	// > as the component separator (ISA16), and newline as the
	// segment terminator.
	const input = "ISA|00|          |00|          |ZZ|SENDER         |ZZ|RECEIVER       |230101|1200|^|00501|000000001|0|T|>\n" +
		"GS|HC|SENDER|RECEIVER|20230101|1200|1|X|005010\n" +
		"ST|837|0001\n" +
		"NM1|41|2|ACME\n" +
		"SE|3|0001\n" +
		"GE|1|1\n" +
		"IEA|1|000000001\n"

	doc, err := x12.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() = %v", err)
	}
	if got, want := doc.ElementSeparator, "|"; got != want {
		t.Errorf("ElementSeparator = %q, want %q", got, want)
	}
	if got, want := doc.SegmentTerminator, "\n"; got != want {
		t.Errorf("SegmentTerminator = %q, want %q", got, want)
	}
	if got, want := doc.Interchange.Header.RepetitionSeparator, "^"; got != want {
		t.Errorf("ISA11 = %q, want %q", got, want)
	}
	if got, want := doc.Interchange.Header.ComponentElementSeparator, ">"; got != want {
		t.Errorf("ISA16 = %q, want %q", got, want)
	}
	if err := doc.Validate(); err != nil {
		t.Errorf("Validate() = %v", err)
	}

	// Encoding uses the document's own delimiters and round-trips.
	encoded, err := x12.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal() = %v", err)
	}
	if diff := cmp.Diff(input, string(encoded)); diff != "" {
		t.Errorf("Marshal() mismatch (-want +got):\n%s", diff)
	}
}

func TestRepetitionSeparatorRoundTrip(t *testing.T) {
	// In 5010, ISA11 carries the repetition separator. Element values
	// containing repetitions (here HI02) are preserved verbatim rather
	// than split, so they survive a decode/encode round trip.
	const input = `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *230101*1200*^*00501*000000001*0*P*:~` +
		`GS*HC*SENDER*RECEIVER*20230101*1200*1*X*005010X222A1~` +
		`ST*837*0001~` +
		`HI*BK:8901^BF:87200~` +
		`SE*3*0001~` +
		`GE*1*1~` +
		`IEA*1*000000001~`

	doc, err := x12.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() = %v", err)
	}
	if got, want := doc.Interchange.Header.RepetitionSeparator, "^"; got != want {
		t.Errorf("ISA11 = %q, want %q", got, want)
	}
	hi := doc.Interchange.FunctionGroups[0].Transactions[0].Segments[0]
	if got, want := hi.Elements[0].Value, "BK:8901^BF:87200"; got != want {
		t.Errorf("HI01 = %q, want %q", got, want)
	}
	encoded, err := x12.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal() = %v", err)
	}
	if diff := cmp.Diff(input, string(encoded)); diff != "" {
		t.Errorf("Marshal() mismatch (-want +got):\n%s", diff)
	}
}

func TestParseError(t *testing.T) {
	// An SE missing its control number, three segments in.
	const input = `ST*837*0001~NM1*41*2*ACME~SE*3~`
	_, err := x12.Decode(strings.NewReader(input))
	var pe *x12.ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("Decode() error = %v, want *ParseError", err)
	}
	if !errors.Is(err, x12.ErrMissingElement) {
		t.Errorf("errors.Is(err, ErrMissingElement) = false, want true")
	}
	if pe.SegmentID != "SE" {
		t.Errorf("SegmentID = %q, want %q", pe.SegmentID, "SE")
	}
	if pe.Segment != 3 {
		t.Errorf("Segment = %d, want 3", pe.Segment)
	}

	_, err = x12.Decode(strings.NewReader(`ISA*00*bad~`))
	if !errors.As(err, &pe) {
		t.Fatalf("Decode() error = %v, want *ParseError", err)
	}
	if !errors.Is(err, x12.ErrMissingElement) {
		t.Errorf("errors.Is(err, ErrMissingElement) = false, want true")
	}
	if pe.SegmentID != "ISA" {
		t.Errorf("SegmentID = %q, want %q", pe.SegmentID, "ISA")
	}
}

func TestDecodeStrictSegments(t *testing.T) {
	// By default the decoder absorbs any segment into the current
	// transaction; WithStrictSegments rejects suspicious ones.
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid segment ID",
			input: `ST*837*0001~nm1*41*2*ACME~SE*3*0001~`,
		},
		{
			name:  "segment after SE trailer",
			input: `ST*837*0001~NM1*41*2*ACME~SE*3*0001~REF*EV*X~`,
		},
		{
			name:  "duplicate SE",
			input: `ST*837*0001~NM1*41*2*ACME~SE*3*0001~SE*3*0001~`,
		},
		{
			name: "duplicate GE",
			input: `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *230101*1200*^*00501*000000001*0*P*:~` +
				`GS*HC*SENDER*RECEIVER*20230101*1200*1*X*005010~` +
				`ST*837*0001~NM1*41*2*ACME~SE*3*0001~` +
				`GE*1*1~GE*1*1~IEA*1*000000001~`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := x12.Decode(strings.NewReader(tt.input)); err != nil {
				t.Fatalf("Decode() relaxed = %v, want nil", err)
			}
			if _, err := x12.Decode(strings.NewReader(tt.input), x12.WithStrictSegments()); !errors.Is(err, x12.ErrInvalidFormat) {
				t.Errorf("Decode() strict error = %v, want ErrInvalidFormat", err)
			}
		})
	}
}

func TestDecodeRejectsMultipleInterchanges(t *testing.T) {
	// A Document holds one interchange; concatenated interchanges used
	// to be silently merged, with the second ISA/IEA overwriting the
	// first.
	const one = `ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *230101*1200*^*00501*%s*0*P*:~` +
		`GS*HC*SENDER*RECEIVER*20230101*1200*1*X*005010~` +
		`ST*837*0001~NM1*41*2*ACME~SE*3*0001~` +
		`GE*1*1~IEA*1*%s~`
	input := fmt.Sprintf(one, "000000001", "000000001") + fmt.Sprintf(one, "000000002", "000000002")
	_, err := x12.Decode(strings.NewReader(input))
	if !errors.Is(err, x12.ErrInvalidFormat) {
		t.Errorf("Decode() error = %v, want ErrInvalidFormat", err)
	}
	var pe *x12.ParseError
	if !errors.As(err, &pe) || pe.SegmentID != "ISA" {
		t.Errorf("Decode() error = %v, want *ParseError on ISA", err)
	}
}

func TestDecodeAutomaticEnvelopeSingleTransaction(t *testing.T) {
	// The synthesized envelope declares exactly one transaction set, so
	// envelope-less input with a second ST must be rejected rather than
	// decoded into a document that fails its own Validate.
	const input = `ST*837*0001~NM1*41*2*ACME~SE*3*0001~ST*837*0002~SE*2*0002~`
	_, err := x12.Decode(strings.NewReader(input))
	if !errors.Is(err, x12.ErrInvalidFormat) {
		t.Errorf("Decode() error = %v, want ErrInvalidFormat", err)
	}
	var pe *x12.ParseError
	if !errors.As(err, &pe) || pe.SegmentID != "ST" || pe.Segment != 4 {
		t.Errorf("Decode() error = %+v, want *ParseError{SegmentID: ST, Segment: 4}", err)
	}
}

func TestDecodeEmptyInput(t *testing.T) {
	if _, err := x12.Decode(strings.NewReader("")); err != io.EOF {
		t.Errorf("Decode(empty) error = %v, want io.EOF", err)
	}
	if _, err := x12.Decode(strings.NewReader("~~\n")); err != io.EOF {
		t.Errorf("Decode(stray terminators) error = %v, want io.EOF", err)
	}
	dec := x12.NewDecoder(strings.NewReader(exampleEDI))
	if _, err := dec.Decode(); err != nil {
		t.Fatalf("Decode() = %v", err)
	}
	if _, err := dec.Decode(); err != io.EOF {
		t.Errorf("second Decode() error = %v, want io.EOF", err)
	}
}

func TestDecodeLeadingTerminator(t *testing.T) {
	// A stray leading terminator must not count as the first segment,
	// which used to defeat the automatic envelope.
	const input = `~ST*837*0001~NM1*41*2*ACME~SE*3*0001~`
	doc, err := x12.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() = %v", err)
	}
	if !doc.EnvelopeAutomaticallyAdded {
		t.Error("EnvelopeAutomaticallyAdded = false, want true")
	}
}

func TestDecodeLargeSegment(t *testing.T) {
	// Segments over bufio.Scanner's 64KB default must still decode; ones
	// over the package's 1MB bound must fail with a wrapped, identifiable
	// error rather than a bare scanner error.
	build := func(n int) string {
		return `ST*837*0001~NM1*41*2*` + strings.Repeat("A", n) + `~SE*3*0001~`
	}
	if _, err := x12.Decode(strings.NewReader(build(100 << 10))); err != nil {
		t.Errorf("Decode(100KB segment) = %v, want nil", err)
	}
	_, err := x12.Decode(strings.NewReader(build(2 << 20)))
	if !errors.Is(err, bufio.ErrTooLong) {
		t.Errorf("Decode(2MB segment) error = %v, want bufio.ErrTooLong", err)
	}
}

func TestMarshalAndValidateNilEntries(t *testing.T) {
	// Nil pointers inside the slices must produce errors, not panics.
	docs := map[string]*x12.Document{
		"nil function group": {Interchange: &x12.Interchange{
			Header:         &x12.ISA{},
			FunctionGroups: []*x12.FunctionGroup{nil},
			Trailer:        &x12.IEA{FunctionalGroupCount: "1"},
		}},
		"nil transaction": {Interchange: &x12.Interchange{
			Header: &x12.ISA{},
			FunctionGroups: []*x12.FunctionGroup{{
				Header:       &x12.GS{},
				Transactions: []*x12.Transaction{nil},
				Trailer:      &x12.GE{TransactionSetCount: "1"},
			}},
			Trailer: &x12.IEA{FunctionalGroupCount: "1"},
		}},
		"automatic envelope nil function group": {
			EnvelopeAutomaticallyAdded: true,
			Interchange:                &x12.Interchange{FunctionGroups: []*x12.FunctionGroup{nil}},
		},
	}
	for name, doc := range docs {
		t.Run(name, func(t *testing.T) {
			if _, err := x12.Marshal(doc); err == nil {
				t.Error("Marshal() error = nil, want error")
			}
			if err := doc.Validate(); err == nil {
				t.Error("Validate() error = nil, want error")
			}
		})
	}
}

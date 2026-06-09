package x12_test

import (
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
		want    *x12.X12Document
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
			want: &x12.X12Document{
				Interchange: &x12.Interchange{
					Header: &x12.ISA{
						AuthorizationInfoQualifier:     "00",
						AuthorizationInformation:       "          ",
						SecurityInfoQualifier:          "00",
						SecurityInfo:                   "          ",
						InterchangeSenderIDQualifier:   "08",
						InterchangeSenderID:            "9254110060     ",
						InterchangeReceiverIDQualifier: "ZZ",
						InterchangeReceiverID:          "123456789      ",
						InterchangeDate:                "041216",
						InterchangeTime:                "0805",
						InterchangeControlStandardsID:  "U",
						InterchangeControlVersion:      "00501",
						InterchangeControlNumber:       "000095071",
						AcknowledgmentRequested:        "0",
						UsageIndicator:                 "P",
						ComponentElementSeparator:      ">",
					},
					FunctionGroups: []*x12.FunctionGroup{
						{
							Header: &x12.GS{
								FunctionalIDCode:         "AG",
								ApplicationSenderCode:    "5137624388",
								ApplicationReceiverCode:  "123456789",
								Date:                     "20041216",
								Time:                     "0805",
								GroupControlNumber:       "95071",
								ResponsibleAgencyCode:    "X",
								VersionReleaseIndustryID: "005010",
							},
							Transactions: []*x12.Transaction{
								{
									Header: &x12.ST{
										TransactionSetIDCode:              "824",
										TransactionSetControlNumber:       "021390001",
										ImplementationConventionReference: "005010X186A1",
									},
									Segments: []x12.Segment{
										{
											ID: "BGN",
											Elements: []x12.Element{
												{ID: "01", Value: "11"}, {ID: "02", Value: "FFA.ABCDEF.123456"},
												{ID: "03", Value: "20020709"}, {ID: "04", Value: "0932"}, {ID: "05"},
												{ID: "06", Value: "123456789"}, {ID: "07"}, {ID: "08", Value: "WQ"},
											},
										},
										{
											ID: "N1",
											Elements: []x12.Element{
												{ID: "01", Value: "41"}, {ID: "02", Value: "ABC INSURANCE"},
												{ID: "03", Value: "46"}, {ID: "04", Value: "111111111"},
											},
										},
										{
											ID: "PER",
											Elements: []x12.Element{
												{ID: "01", Value: "IC"}, {ID: "02", Value: "JOHN JOHNSON"},
												{ID: "03", Value: "TE"}, {ID: "04", Value: "8005551212"},
												{ID: "05", Value: "EX"}, {ID: "06", Value: "1439"},
											},
										},
										{
											ID: "N1",
											Elements: []x12.Element{
												{ID: "01", Value: "40"}, {ID: "02", Value: "SMITHCO"}, {ID: "03", Value: "46"},
												{ID: "04", Value: "A1234"},
											},
										},
										{
											ID: "OTI",
											Elements: []x12.Element{
												{ID: "01", Value: "TA"}, {ID: "02", Value: "TN"}, {ID: "03", Value: "NA"},
												{ID: "04"}, {ID: "05"}, {ID: "06", Value: "20020709"},
												{ID: "07", Value: "0902"}, {ID: "08", Value: "2"},

												{ID: "09", Value: "0001"},
												{ID: "10", Value: "834"},
												{ID: "11", Value: "005010X220A1"},
											},
										},
									},
									Trailer: &x12.SE{NumberOfIncludedSegments: "7", TransactionSetControlNumber: "021390001"},
								},
							},
							Trailer: &x12.GE{
								NumberOfIncludedTransactionSets: "1",
								GroupControlNumber:              "95071",
							},
						},
					},
					Trailer: &x12.IEA{
						NumberOfIncludedFunctionalGroups: "1",
						InterchangeControlNumber:         "000095071",
					},
				},
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
			encoded, err := (&x12.Marshaler{}).Marshal(got)
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
			var opts []x12.DecodeOption
			if optMap[file.Name()].RelaxedWhitespace {
				opts = append(opts, x12.WithRelaxedSegmentIDWhitespace())
			}
			edi, err := x12.Decode(f, opts...)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := (&x12.Marshaler{
				NewLines: true,
			}).Marshal(edi)
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
	if got, want := doc.Interchange.Header.InterchangeControlStandardsID, "^ "; got != want {
		t.Errorf("ISA11 = %q, want %q", got, want)
	}
	if got, want := doc.Interchange.Header.InterchangeControlNumber, "191511902 "; got != want {
		t.Errorf("ISA13 = %q, want %q", got, want)
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
	b, err := (&x12.Marshaler{}).Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal() = %v", err)
	}
	if diff := cmp.Diff(input, string(b)); diff != "" {
		t.Errorf("Marshal() mismatch (-want +got):\n%s", diff)
	}
}

func TestMarshalHandBuilt(t *testing.T) {
	doc := &x12.X12Document{
		Interchange: &x12.Interchange{
			Header: &x12.ISA{
				AuthorizationInfoQualifier:     "00",
				AuthorizationInformation:       "          ",
				SecurityInfoQualifier:          "00",
				SecurityInfo:                   "          ",
				InterchangeSenderIDQualifier:   "ZZ",
				InterchangeSenderID:            "SENDER         ",
				InterchangeReceiverIDQualifier: "ZZ",
				InterchangeReceiverID:          "RECEIVER       ",
				InterchangeDate:                "230101",
				InterchangeTime:                "1200",
				InterchangeControlStandardsID:  "^",
				InterchangeControlVersion:      "00501",
				InterchangeControlNumber:       "000000001",
				AcknowledgmentRequested:        "0",
				UsageIndicator:                 "T",
				ComponentElementSeparator:      ":",
			},
			FunctionGroups: []*x12.FunctionGroup{{
				Header: &x12.GS{
					FunctionalIDCode:         "HC",
					ApplicationSenderCode:    "SENDER",
					ApplicationReceiverCode:  "RECEIVER",
					Date:                     "20230101",
					Time:                     "1200",
					GroupControlNumber:       "1",
					ResponsibleAgencyCode:    "X",
					VersionReleaseIndustryID: "005010X222A1",
				},
				Transactions: []*x12.Transaction{{
					Header: &x12.ST{TransactionSetIDCode: "837", TransactionSetControlNumber: "0001"},
					Segments: []x12.Segment{
						{ID: "BHT", Elements: []x12.Element{{Value: "0019"}, {Value: "00"}}},
						{ID: "HI", Elements: []x12.Element{{Value: "BK", Components: []string{"8901"}}}},
					},
					Trailer: &x12.SE{NumberOfIncludedSegments: "4", TransactionSetControlNumber: "0001"},
				}},
				Trailer: &x12.GE{NumberOfIncludedTransactionSets: "1", GroupControlNumber: "1"},
			}},
			Trailer: &x12.IEA{NumberOfIncludedFunctionalGroups: "1", InterchangeControlNumber: "000000001"},
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
	b, err := (&x12.Marshaler{}).Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal() = %v", err)
	}
	if diff := cmp.Diff(want, string(b)); diff != "" {
		t.Errorf("Marshal() mismatch (-want +got):\n%s", diff)
	}
}

func TestMarshalAutomaticEnvelopeGuard(t *testing.T) {
	doc := &x12.X12Document{EnvelopeAutomaticallyAdded: true, Interchange: &x12.Interchange{}}
	if _, err := (&x12.Marshaler{}).Marshal(doc); !errors.Is(err, x12.ErrInvalidArgument) {
		t.Errorf("Marshal() error = %v, want ErrInvalidArgument", err)
	}
}

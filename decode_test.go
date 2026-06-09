package x12

import (
	"bufio"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_decodeState_extractSegmentID(t *testing.T) {
	tests := []struct {
		name     string
		state    decodeState
		elements []string
		want     string
		want1    []string
	}{
		{
			name:     "Single Element",
			elements: []string{"ISA"},
			want:     "ISA",
			want1:    []string{},
		},
		{
			name:     "Typical Element",
			elements: []string{"ISA", "00", "          ", "00", "          ", "08", "9254110060     ", "ZZ", "123456789      ", "041216", "0805", "U", "00501", "000095071", "0", "P", ">~"},
			want:     "ISA",
			want1:    []string{"00", "          ", "00", "          ", "08", "9254110060     ", "ZZ", "123456789      ", "041216", "0805", "U", "00501", "000095071", "0", "P", ">~"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.state.extractSegmentID(tt.elements)
			if got != tt.want {
				t.Errorf("extractSegmentID() got = %v, want %v", got, tt.want)
			}
			if diff := cmp.Diff(tt.want1, got1); diff != "" {
				t.Errorf("extractSegmentID() elements mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_decodeState_parseGE(t *testing.T) {
	tests := []struct {
		name     string
		state    decodeState
		elements []string
		want     *GE
		wantErr  error
	}{
		{
			name:     "GE segment without GS segment",
			state:    decodeState{},
			elements: []string{"GE", "1", "95071"},
			want:     nil,
			wantErr:  ErrInvalidFormat,
		},
		{
			name:     "too few elements",
			state:    decodeState{currentFunctionGroup: &FunctionGroup{Header: &GS{}}},
			elements: []string{"GE", "1"},
			want:     nil,
			wantErr:  ErrMissingElement,
		},
		{
			name:     "Typical segment",
			state:    decodeState{currentFunctionGroup: &FunctionGroup{Header: &GS{}}},
			elements: []string{"GE", "1", "95071"},
			want:     &GE{NumberOfIncludedTransactionSets: "1", GroupControlNumber: "95071"},
		},
		{
			name:     "too many elements",
			state:    decodeState{currentFunctionGroup: &FunctionGroup{Header: &GS{}}},
			elements: []string{"GE", "1", "95071", "Hello", "World"},
			want:     &GE{NumberOfIncludedTransactionSets: "1", GroupControlNumber: "95071"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseGE(tt.elements); !errors.Is(err, tt.wantErr) {
				t.Errorf("parseGE() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(tt.want, tt.state.currentFunctionGroup.Trailer); diff != "" {
					t.Errorf("parseGE() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func Test_decodeState_parseGS(t *testing.T) {
	tests := []struct {
		name     string
		state    decodeState
		elements []string
		want     *GS
		wantErr  error
	}{
		{
			name:     "too few elements",
			state:    decodeState{doc: &Document{Interchange: &Interchange{FunctionGroups: []*FunctionGroup{}}}, currentFunctionGroup: &FunctionGroup{}},
			elements: []string{"GS", "AG", "5137624388", "123456789", "20041216", "0805", "95071", "X"},
			want:     nil,
			wantErr:  ErrMissingElement,
		},
		{
			name:     "typical segment",
			state:    decodeState{doc: &Document{Interchange: &Interchange{FunctionGroups: []*FunctionGroup{}}}, currentFunctionGroup: &FunctionGroup{}},
			elements: []string{"GS", "AG", "5137624388", "123456789", "20041216", "0805", "95071", "X", "005010"},
			want: &GS{
				FunctionalIDCode:         "AG",
				ApplicationSenderCode:    "5137624388",
				ApplicationReceiverCode:  "123456789",
				Date:                     "20041216",
				Time:                     "0805",
				GroupControlNumber:       "95071",
				ResponsibleAgencyCode:    "X",
				VersionReleaseIndustryID: "005010",
			},
		},
		{
			name:     "too many elements",
			state:    decodeState{doc: &Document{Interchange: &Interchange{FunctionGroups: []*FunctionGroup{}}}, currentFunctionGroup: &FunctionGroup{}},
			elements: []string{"GS", "AG", "5137624388", "123456789", "20041216", "0805", "95071", "X", "005010", "Hello", "World!"},
			want: &GS{
				FunctionalIDCode:         "AG",
				ApplicationSenderCode:    "5137624388",
				ApplicationReceiverCode:  "123456789",
				Date:                     "20041216",
				Time:                     "0805",
				GroupControlNumber:       "95071",
				ResponsibleAgencyCode:    "X",
				VersionReleaseIndustryID: "005010",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseGS(tt.elements); !errors.Is(err, tt.wantErr) {
				t.Errorf("parseGS() error = %v, want %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, tt.state.currentFunctionGroup.Header); diff != "" {
				t.Errorf("parseGS() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_decodeState_parseIEA(t *testing.T) {
	tests := []struct {
		name     string
		state    decodeState
		elements []string
		want     *IEA
		wantErr  error
	}{
		{
			name:     "too few elements",
			state:    decodeState{doc: &Document{Interchange: &Interchange{}}},
			elements: []string{"IEA", "1"},
			want:     nil,
			wantErr:  ErrMissingElement,
		},
		{
			name:     "typical segment",
			state:    decodeState{doc: &Document{Interchange: &Interchange{}}},
			elements: []string{"IEA", "1", "000095071"},
			want: &IEA{
				NumberOfIncludedFunctionalGroups: "1",
				InterchangeControlNumber:         "000095071",
			},
		},
		{
			name:     "too many elements",
			state:    decodeState{doc: &Document{Interchange: &Interchange{FunctionGroups: []*FunctionGroup{}}}, currentFunctionGroup: &FunctionGroup{}},
			elements: []string{"IEA", "1", "000095071", "Hello", "World!"},
			want: &IEA{
				NumberOfIncludedFunctionalGroups: "1",
				InterchangeControlNumber:         "000095071",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseIEA(tt.elements); !errors.Is(err, tt.wantErr) {
				t.Errorf("parseIEA() error = %v, want %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, tt.state.doc.Interchange.Trailer); diff != "" {
				t.Errorf("parseIEA() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_decodeState_parseISA(t *testing.T) {
	tests := []struct {
		name     string
		state    decodeState
		elements []string
		want     *ISA
		wantErr  error
	}{
		{
			name:     "too few elements",
			state:    decodeState{doc: &Document{Interchange: &Interchange{}}},
			elements: []string{"ISA", "00", "          ", "00", "          ", "08", "9254110060     ", "ZZ", "123456789      ", "041216", "0805", "U", "00501", "000095071", "0", "P"},
			want:     nil,
			wantErr:  ErrMissingElement,
		},
		{
			name:     "typical segment",
			state:    decodeState{doc: &Document{Interchange: &Interchange{}}},
			elements: []string{"ISA", "00", "          ", "00", "          ", "08", "9254110060     ", "ZZ", "123456789      ", "041216", "0805", "U", "00501", "000095071", "0", "P", ">"},
			want: &ISA{
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
		},
		{
			name:     "too many elements",
			state:    decodeState{doc: &Document{Interchange: &Interchange{FunctionGroups: []*FunctionGroup{}}}, currentFunctionGroup: &FunctionGroup{}},
			elements: []string{"ISA", "00", "          ", "00", "          ", "08", "9254110060     ", "ZZ", "123456789      ", "041216", "0805", "U", "00501", "000095071", "0", "P", ">", "Hello", "World!"},
			want: &ISA{
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseISA(tt.elements); !errors.Is(err, tt.wantErr) {
				t.Errorf("parseISA() error = %v, want %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, tt.state.doc.Interchange.Header); diff != "" {
				t.Errorf("parseISA() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_decodeState_parseSE(t *testing.T) {
	tests := []struct {
		name     string
		state    decodeState
		elements []string
		want     *SE
		wantErr  error
	}{
		{
			name:     "SE segment without ST segment",
			state:    decodeState{},
			elements: []string{},
			want:     nil,
			wantErr:  ErrInvalidFormat,
		},
		{
			name:     "too few elements",
			state:    decodeState{currentTransaction: &Transaction{}},
			elements: []string{"SE", "7"},
			want:     nil,
			wantErr:  ErrMissingElement,
		},
		{
			name:     "typical segment",
			state:    decodeState{currentTransaction: &Transaction{}},
			elements: []string{"SE", "7", "021390001"},
			want:     &SE{NumberOfIncludedSegments: "7", TransactionSetControlNumber: "021390001"},
		},
		{
			name:     "too many elements",
			state:    decodeState{currentTransaction: &Transaction{}},
			elements: []string{"SE", "7", "021390001", "Hello", "World!"},
			want:     &SE{NumberOfIncludedSegments: "7", TransactionSetControlNumber: "021390001"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseSE(tt.elements); !errors.Is(err, tt.wantErr) {
				t.Errorf("parseSE() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(tt.want, tt.state.currentTransaction.Trailer); diff != "" {
					t.Errorf("parseSE() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func Test_decodeState_parseST(t *testing.T) {
	tests := []struct {
		name     string
		state    decodeState
		elements []string
		want     *ST
		wantErr  error
	}{
		{
			name:     "too few elements",
			state:    decodeState{currentTransaction: &Transaction{}},
			elements: []string{"ST", "824"},
			want:     nil,
			wantErr:  ErrMissingElement,
		},
		{
			name:     "ST segment without GS segment",
			state:    decodeState{currentTransaction: &Transaction{}},
			elements: []string{"ST", "824", "021390001", "005010X186A1"},
			want:     nil,
			wantErr:  ErrInvalidFormat,
		},
		{
			name:     "typical segment",
			state:    decodeState{doc: &Document{Interchange: &Interchange{}}, lineIndex: 1},
			elements: []string{"ST", "824", "021390001", "005010X186A1"},
			want: &ST{
				TransactionSetIDCode:              "824",
				TransactionSetControlNumber:       "021390001",
				ImplementationConventionReference: "005010X186A1",
			},
		},
		{
			name:     "two element segment",
			state:    decodeState{doc: &Document{Interchange: &Interchange{}}, lineIndex: 1},
			elements: []string{"ST", "824", "021390001"},
			want: &ST{
				TransactionSetIDCode:        "824",
				TransactionSetControlNumber: "021390001",
			},
		},
		{
			name:     "too many elements",
			state:    decodeState{doc: &Document{Interchange: &Interchange{}}, lineIndex: 1},
			elements: []string{"ST", "824", "021390001", "005010X186A1", "Hello", "World!"},
			want: &ST{
				TransactionSetIDCode:              "824",
				TransactionSetControlNumber:       "021390001",
				ImplementationConventionReference: "005010X186A1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseST(tt.elements); !errors.Is(err, tt.wantErr) {
				t.Errorf("parseST() error = %v, want %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, tt.state.currentTransaction.Header); diff != "" {
				t.Errorf("parseST() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_decodeState_parseSegment(t *testing.T) {
	tests := []struct {
		name     string
		state    decodeState
		elements []string
		want     []Segment
		wantErr  error
	}{
		{
			name:     "default segment without ST segment",
			state:    decodeState{},
			elements: []string{"DEF", "1", "2", "3"},
			want:     []Segment{},
			wantErr:  ErrInvalidFormat,
		},
		{
			name:     "typical segment",
			state:    decodeState{currentTransaction: &Transaction{}},
			elements: []string{"DEF", "1", "2", "3"},
			want: []Segment{{
				ID: "DEF",
				Elements: []Element{
					{ID: "01", Value: "1"},
					{ID: "02", Value: "2"},
					{ID: "03", Value: "3"},
				}},
			},
		},
		{
			name:     "id only segment",
			state:    decodeState{currentTransaction: &Transaction{}},
			elements: []string{"DEF"},
			want: []Segment{{
				ID:       "DEF",
				Elements: []Element{}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseSegment(tt.elements); !errors.Is(err, tt.wantErr) {
				t.Errorf("parseSegment() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(tt.want, tt.state.currentTransaction.Segments); diff != "" {
					t.Errorf("parseSegment() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func Test_decodeState_processLine(t *testing.T) {
	tests := []struct {
		name    string
		state   decodeState
		line    string
		wantErr error
	}{
		{
			name:  "Typical Line",
			state: decodeState{currentTransaction: &Transaction{}},
			line:  "DEF 1 2 3",
		},
		{
			name:  "Empty Line",
			state: decodeState{},
			line:  "",
		},
		{
			name:  "Whitespace only",
			state: decodeState{currentTransaction: &Transaction{}},
			line:  " ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsers := tt.state.getSegmentParsers()
			if err := tt.state.processLine(tt.line, parsers); !errors.Is(err, tt.wantErr) {
				t.Errorf("processLine() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func Test_parseElements(t *testing.T) {
	tests := []struct {
		name     string
		elements []string
		want     []Element
	}{
		{
			name:     "No Elements",
			elements: []string{},
			want:     []Element{},
		},
		{
			name:     "Typical Elements",
			elements: []string{"00", "          ", "00", "          ", "08", "9254110060     ", "ZZ", "123456789      ", "041216", "0805", "U", "00501", "000095071", "0", "P", ">~"},
			want: []Element{
				{ID: "01", Value: "00"},
				{ID: "02", Value: "          "},
				{ID: "03", Value: "00"},
				{ID: "04", Value: "          "},
				{ID: "05", Value: "08"},
				{ID: "06", Value: "9254110060     "},
				{ID: "07", Value: "ZZ"},
				{ID: "08", Value: "123456789      "},
				{ID: "09", Value: "041216"},
				{ID: "10", Value: "0805"},
				{ID: "11", Value: "U"},
				{ID: "12", Value: "00501"},
				{ID: "13", Value: "000095071"},
				{ID: "14", Value: "0"},
				{ID: "15", Value: "P"},
				{ID: "16", Value: ">~"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, parseElements(tt.elements)); diff != "" {
				t.Errorf("parseElements() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_scanEDI(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr error
	}{
		{
			name:  "Two Complete Segments",
			input: "SEG1*val1~SEG2*val2~",
			want:  []string{"SEG1*val1", "SEG2*val2"},
		},
		{
			name:  "Incomplete Final Segment",
			input: "SEG1*val1~SEG2*val2",
			want:  []string{"SEG1*val1", "SEG2*val2"},
		},
		{
			name:  "Empty Input",
			input: "",
			want:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := bufio.NewScanner(strings.NewReader(tt.input))
			scanner.Split(scanEDI)

			var segments []string
			for scanner.Scan() {
				segments = append(segments, scanner.Text())
			}

			if err := scanner.Err(); !errors.Is(err, tt.wantErr) {
				t.Errorf("scanEDI() error = %v, want %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, segments); diff != "" {
				t.Errorf("scanEDI() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

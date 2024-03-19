package x12

import (
	"bufio"
	"reflect"
	"strings"
	"testing"
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
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("extractSegmentID() got1 = %v, want %v", got1, tt.want1)
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
		wantErr  bool
	}{
		{
			name:     "GE segment without GS segment",
			state:    decodeState{},
			elements: []string{"GE", "1", "95071"},
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "too few elements",
			state:    decodeState{currentFunctionGroup: &FunctionGroup{Header: &GS{}}},
			elements: []string{"GE", "1"},
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "Typical segment",
			state:    decodeState{currentFunctionGroup: &FunctionGroup{Header: &GS{}}},
			elements: []string{"GE", "1", "95071"},
			want:     &GE{NumberOfIncludedTransactionSets: "1", GroupControlNumber: "95071"},
			wantErr:  false,
		},
		{

			name:     "too many elements",
			state:    decodeState{currentFunctionGroup: &FunctionGroup{Header: &GS{}}},
			elements: []string{"GE", "1", "95071", "Hello", "World"},
			want:     &GE{NumberOfIncludedTransactionSets: "1", GroupControlNumber: "95071"},
			wantErr:  false, // should this case return an error?
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseGE(tt.elements); (err != nil) != tt.wantErr {
				t.Errorf("parseGE() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				got := tt.state.currentFunctionGroup.Trailer
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("parseGE() got = %v, want %v", got, tt.want)
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
		wantErr  bool
	}{
		{
			name:     "too few elements",
			state:    decodeState{doc: &X12Document{Interchange: &Interchange{FunctionGroups: []*FunctionGroup{}}}, currentFunctionGroup: &FunctionGroup{}},
			elements: []string{"GS", "AG", "5137624388", "123456789", "20041216", "0805", "95071", "X"},
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "typical segment",
			state:    decodeState{doc: &X12Document{Interchange: &Interchange{FunctionGroups: []*FunctionGroup{}}}, currentFunctionGroup: &FunctionGroup{}},
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
			wantErr: false,
		},
		{
			name:     "too many elements",
			state:    decodeState{doc: &X12Document{Interchange: &Interchange{FunctionGroups: []*FunctionGroup{}}}, currentFunctionGroup: &FunctionGroup{}},
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
			wantErr: false, // should this case return an error?
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseGS(tt.elements); (err != nil) != tt.wantErr {
				t.Errorf("parseGS() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := tt.state.currentFunctionGroup.Header
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseGS() got = %v, want %v", got, tt.want)
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
		wantErr  bool
	}{
		{
			name:     "too few elements",
			state:    decodeState{doc: &X12Document{Interchange: &Interchange{}}},
			elements: []string{"IEA", "1"},
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "typical segment",
			state:    decodeState{doc: &X12Document{Interchange: &Interchange{}}},
			elements: []string{"IEA", "1", "000095071"},
			want: &IEA{
				NumberOfIncludedFunctionalGroups: "1",
				InterchangeControlNumber:         "000095071",
			},
			wantErr: false,
		},
		{
			name:     "too many elements",
			state:    decodeState{doc: &X12Document{Interchange: &Interchange{FunctionGroups: []*FunctionGroup{}}}, currentFunctionGroup: &FunctionGroup{}},
			elements: []string{"IEA", "1", "000095071", "Hello", "World!"},
			want: &IEA{
				NumberOfIncludedFunctionalGroups: "1",
				InterchangeControlNumber:         "000095071",
			},
			wantErr: false, // should this case return an error?
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseIEA(tt.elements); (err != nil) != tt.wantErr {
				t.Errorf("parseIEA() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := tt.state.doc.Interchange.Trailer
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseIEA() got = %v, want %v", got, tt.want)
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
		wantErr  bool
	}{
		{
			name:     "too few elements",
			state:    decodeState{doc: &X12Document{Interchange: &Interchange{}}},
			elements: []string{"ISA", "00", "          ", "00", "          ", "08", "9254110060     ", "ZZ", "123456789      ", "041216", "0805", "U", "00501", "000095071", "0", "P"},
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "typical segment",
			state:    decodeState{doc: &X12Document{Interchange: &Interchange{}}},
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
			wantErr: false,
		},
		{
			name:     "too many elements",
			state:    decodeState{doc: &X12Document{Interchange: &Interchange{FunctionGroups: []*FunctionGroup{}}}, currentFunctionGroup: &FunctionGroup{}},
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
			wantErr: false, // should this case return an error?
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseISA(tt.elements); (err != nil) != tt.wantErr {
				t.Errorf("parseISA() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := tt.state.doc.Interchange.Header
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseISA() got = %v, want %v", got, tt.want)
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
		wantErr  bool
	}{
		{
			name:     "SE segment without ST segment",
			state:    decodeState{},
			elements: []string{},
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "too few elements",
			state:    decodeState{currentTransaction: &Transaction{}},
			elements: []string{"SE", "7"},
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "typical segment",
			state:    decodeState{currentTransaction: &Transaction{}},
			elements: []string{"SE", "7", "021390001"},
			want:     &SE{NumberOfIncludedSegments: "7", TransactionSetControlNumber: "021390001"},
			wantErr:  false,
		},
		{
			name:     "too many elements",
			state:    decodeState{currentTransaction: &Transaction{}},
			elements: []string{"SE", "7", "021390001", "Hello", "World!"},
			want:     &SE{NumberOfIncludedSegments: "7", TransactionSetControlNumber: "021390001"},
			wantErr:  false, // should this case return an error?
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseSE(tt.elements); (err != nil) != tt.wantErr {
				t.Errorf("parseSE() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				got := tt.state.currentTransaction.Trailer
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("parseSE() got = %v, want %v", got, tt.want)
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
		wantErr  bool
	}{
		{
			name:     "too few elements",
			state:    decodeState{currentTransaction: &Transaction{}},
			elements: []string{"ST", "824"},
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "ST segment without GS segment",
			state:    decodeState{currentTransaction: &Transaction{}},
			elements: []string{"ST", "824", "021390001", "005010X186A1"},
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "typical segment",
			state:    decodeState{doc: &X12Document{Interchange: &Interchange{}}, lineIndex: 1},
			elements: []string{"ST", "824", "021390001", "005010X186A1"},
			want: &ST{
				TransactionSetIDCode:              "824",
				TransactionSetControlNumber:       "021390001",
				ImplementationConventionReference: "005010X186A1",
			},
			wantErr: false,
		},
		{
			name:     "two element segment",
			state:    decodeState{doc: &X12Document{Interchange: &Interchange{}}, lineIndex: 1},
			elements: []string{"ST", "824", "021390001"},
			want: &ST{
				TransactionSetIDCode:        "824",
				TransactionSetControlNumber: "021390001",
			},
			wantErr: false,
		},
		{
			name:     "too many elements",
			state:    decodeState{doc: &X12Document{Interchange: &Interchange{}}, lineIndex: 1},
			elements: []string{"ST", "824", "021390001", "005010X186A1", "Hello", "World!"},
			want: &ST{
				TransactionSetIDCode:              "824",
				TransactionSetControlNumber:       "021390001",
				ImplementationConventionReference: "005010X186A1",
			},
			wantErr: false, // should this case return an error?
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseST(tt.elements); (err != nil) != tt.wantErr {
				t.Errorf("parseST() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := tt.state.currentTransaction.Header
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseST() got = %v, want %v", got, tt.want)
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
		wantErr  bool
	}{
		{
			name:     "default segment without ST segment",
			state:    decodeState{},
			elements: []string{"DEF", "1", "2", "3"},
			want:     []Segment{},
			wantErr:  true,
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
			wantErr: false,
		},
		{
			name:     "id only segment",
			state:    decodeState{currentTransaction: &Transaction{}},
			elements: []string{"DEF"},
			want: []Segment{{
				ID:       "DEF",
				Elements: []Element{}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.parseSegment(tt.elements); (err != nil) != tt.wantErr {
				t.Errorf("parseSegment() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				got := tt.state.currentTransaction.Segments
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("parseSegment() got = %v, want %v", got, tt.want)
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
		wantErr bool
	}{
		{
			name:    "Typical Line",
			state:   decodeState{currentTransaction: &Transaction{}},
			line:    "DEF 1 2 3",
			wantErr: false,
		},
		{
			name:    "Empty Line",
			state:   decodeState{},
			line:    "",
			wantErr: false,
		},
		{
			name:    "Whitespace only",
			state:   decodeState{currentTransaction: &Transaction{}},
			line:    " ",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsers := tt.state.getSegmentParsers()
			if err := tt.state.processLine(tt.line, parsers); (err != nil) != tt.wantErr {
				t.Errorf("processLine() error = %v, wantErr %v", err, tt.wantErr)
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
			if got := parseElements(tt.elements); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseElements() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_scanEDI(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
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

			if err := scanner.Err(); (err != nil) != tt.wantErr {
				t.Errorf("scanEDI() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(segments, tt.want) {
				t.Errorf("scanEDI() = %v, want %v", segments, tt.want)
			}
		})
	}
}

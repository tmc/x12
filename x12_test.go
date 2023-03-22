package x12_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tmc/x12"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *x12.X12Document
		wantErr bool
	}{
		{
			name:  "Test1",
			input: `ISA~00~          ~00~          ~ZZ~TEST           ~ZZ~1234567890     ~220101~1234~|~00501~000000001~0~P~^_GS~AB~TEST~1234567890~20220101~1234~1~X~005010_ST~123~0001_SE~2~0001_GE~1~1_IEA~1~000000001_`,
			want: &x12.X12Document{
				Interchange: x12.Interchange{
					Header: x12.ISA{
						AuthorizationInfoQualifier:     "00",
						AuthorizationInformation:       "          ",
						SecurityInfoQualifier:          "00",
						SecurityInfo:                   "          ",
						InterchangeSenderIDQualifier:   "ZZ",
						InterchangeSenderID:            "TEST           ",
						InterchangeReceiverIDQualifier: "ZZ",
						InterchangeReceiverID:          "1234567890     ",
						InterchangeDate:                "220101",
						InterchangeTime:                "1234",
						InterchangeControlStandardsID:  "|",
						InterchangeControlVersion:      "00501",
						InterchangeControlNumber:       "000000001",
						AcknowledgmentRequested:        "0",
						UsageIndicator:                 "P",
						ComponentElementSeparator:      "^",
					},
					FunctionGroups: []x12.FunctionGroup{
						{
							Header: x12.GS{
								FunctionalIDCode:         "AB",
								ApplicationSenderCode:    "TEST",
								ApplicationReceiverCode:  "1234567890",
								Date:                     "20220101",
								Time:                     "1234",
								GroupControlNumber:       "1",
								ResponsibleAgencyCode:    "X",
								VersionReleaseIndustryID: "005010",
							},
							Transactions: []x12.Transaction{
								{
									Header: x12.ST{
										TransactionSetIDCode:        "123",
										TransactionSetControlNumber: "0001",
									},
									Segments: []x12.Segment{
										{
											ID: "SE",
											Elements: []x12.Element{
												{ID: "01", Value: "2"},
												{ID: "02", Value: "0001"},
											},
										},
									},
									Trailer: x12.SE{
										NumberOfIncludedSegments:    "2",
										TransactionSetControlNumber: "0001",
									},
								},
							},
							Trailer: x12.GE{
								NumberOfIncludedTransactionSets: "1",
								GroupControlNumber:              "1",
							},
						},
					},
					Trailer: x12.IEA{
						NumberOfIncludedFunctionalGroups: "1",
						InterchangeControlNumber:         "000000001",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			got, err := x12.Decode(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Decode() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

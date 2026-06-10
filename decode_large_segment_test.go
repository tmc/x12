package x12_test

import (
	"github.com/tmc/x12"
	"strings"
	"testing"
)

func TestDecodeLargeSegment(t *testing.T) {
	big := strings.Repeat("A", 70000) // > bufio's 64KB default
	in := "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *200101*1200*U*00401*000000001*0*P*:~GS*HC*S*R*20200101*1200*1*X*004010~ST*837*0001~NTE**" + big + "~SE*3*0001~GE*1*1~IEA*1*000000001~"
	doc, err := x12.Decode(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Decode of 70KB segment: %v", err)
	}
	seg := doc.Interchange.FunctionGroups[0].Transactions[0].Segments[0]
	if got := len(seg.Elements[1].Value); got != 70000 {
		t.Errorf("NTE value length = %d, want 70000", got)
	}
}

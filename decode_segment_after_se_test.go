package x12_test

import (
	"errors"
	"github.com/tmc/x12"
	"strings"
	"testing"
)

func TestDecodeSegmentAfterSE(t *testing.T) {
	in := "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *200101*1200*U*00401*000000001*0*P*:~GS*HC*S*R*20200101*1200*1*X*004010~ST*837*0001~NM1*IL*1~SE*2*0001~ZZ*orphan~GE*1*1~IEA*1*000000001~"
	_, err := x12.Decode(strings.NewReader(in))
	if err == nil {
		t.Fatal("Decode with a segment between SE and the next ST: got nil error, want error")
	}
	if !errors.Is(err, x12.ErrInvalidFormat) {
		t.Errorf("got %v, want ErrInvalidFormat", err)
	}
}

package x12_test

import (
	"errors"
	"github.com/tmc/x12"
	"strings"
	"testing"
)

func TestDecodeDuplicateISA(t *testing.T) {
	isa2 := "ISA*00*          *00*          *ZZ*SECOND         *ZZ*RECEIVER       *200101*1200*U*00401*000000002*0*P*:~"
	in := "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *200101*1200*U*00401*000000001*0*P*:~" + isa2 + "IEA*1*000000002~"
	_, err := x12.Decode(strings.NewReader(in))
	if err == nil {
		t.Fatal("Decode of document with two ISA segments: got nil error, want error")
	}
	if !errors.Is(err, x12.ErrInvalidFormat) {
		t.Errorf("got %v, want ErrInvalidFormat", err)
	}
}

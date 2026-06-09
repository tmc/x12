package x12_test

import (
	"errors"
	"github.com/tmc/x12"
	"strings"
	"testing"
)

func TestDecodeGSWithoutISA(t *testing.T) {
	in := "GS*HC*S*R*20200101*1200*1*X*004010~ST*837*0001~SE*1*0001~GE*1*1~"
	_, err := x12.Decode(strings.NewReader(in))
	if err == nil {
		t.Fatal("Decode of GS-without-ISA: got nil error, want error")
	}
	if !errors.Is(err, x12.ErrInvalidFormat) {
		t.Errorf("got %v, want ErrInvalidFormat", err)
	}
}

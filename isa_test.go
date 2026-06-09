package x12_test

import (
	"testing"

	"github.com/tmc/x12"
)

func TestISAElementDescription(t *testing.T) {
	if got, ok := x12.ISAElementDescription("ISA06", "ZZ"); !ok || got != "Mutually Defined" {
		t.Errorf("ISAElementDescription(ISA06, ZZ) = %q, %v; want \"Mutually Defined\", true", got, ok)
	}
	if got, ok := x12.ISAElementDescription("ISA01", "00"); !ok || got == "" {
		t.Errorf("ISAElementDescription(ISA01, 00) = %q, %v; want non-empty, true", got, ok)
	}
	if _, ok := x12.ISAElementDescription("ISA06", "NOPE"); ok {
		t.Error("ISAElementDescription(ISA06, NOPE) = ok=true; want false")
	}
}

func TestInterchangeIDQualifierDescription(t *testing.T) {
	if got, ok := x12.InterchangeIDQualifierDescription("30"); !ok || got != "U.S. Federal Tax Identification Number" {
		t.Errorf("InterchangeIDQualifierDescription(30) = %q, %v", got, ok)
	}
	if _, ok := x12.InterchangeIDQualifierDescription("zz-unknown"); ok {
		t.Error("InterchangeIDQualifierDescription(unknown) = ok=true; want false")
	}
}

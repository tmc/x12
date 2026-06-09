package x12_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/tmc/x12"
)

// benchmarkEDI returns a synthetic interchange containing roughly 3n
// segments, patterned on the 824 example, so the benchmarks run over a
// realistically sized document rather than the small bundled fixtures.
func benchmarkEDI(n int) []byte {
	var b strings.Builder
	b.WriteString("ISA*00*          *00*          *08*9254110060     *ZZ*123456789      *041216*0805*U*00501*000095071*0*P*>~")
	b.WriteString("GS*AG*5137624388*123456789*20041216*0805*95071*X*005010~")
	b.WriteString("ST*824*021390001*005010X186A1~")
	b.WriteString("BGN*11*FFA.ABCDEF.123456*20020709*0932**123456789**WQ~")
	for i := 0; i < n; i++ {
		b.WriteString("N1*41*ABC INSURANCE*46*111111111~")
		b.WriteString("PER*IC*JOHN JOHNSON*TE*8005551212*EX*1439~")
		b.WriteString("OTI*TA*TN*NA***20020709*0902*2*0001*834*005010X220A1~")
	}
	fmt.Fprintf(&b, "SE*%d*021390001~", 3*n+3)
	b.WriteString("GE*1*95071~")
	b.WriteString("IEA*1*000095071~")
	return []byte(b.String())
}

func BenchmarkDecode(b *testing.B) {
	data := benchmarkEDI(1000)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := x12.Decode(bytes.NewReader(data)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshal(b *testing.B) {
	doc, err := x12.Decode(bytes.NewReader(benchmarkEDI(1000)))
	if err != nil {
		b.Fatal(err)
	}
	m := &x12.Marshaler{}
	out, err := m.Marshal(doc)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(out)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := m.Marshal(doc); err != nil {
			b.Fatal(err)
		}
	}
}

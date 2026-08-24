package crypt

import "testing"

func BenchmarkGenerateKey(b *testing.B) {
	for b.Loop() {
		sinkKey = GenerateKey()
	}
}

func BenchmarkParseKey(b *testing.B) {
	text := GenerateKey().Reveal()
	b.ReportAllocs()

	for b.Loop() {
		sinkKey, _ = ParseKey(text)
	}
}

// BenchmarkKeyID matters because the id goes in the header of every ciphertext,
// so it happens once per encryption.
func BenchmarkKeyID(b *testing.B) {
	key := GenerateKey()
	b.ReportAllocs()

	for b.Loop() {
		sinkBytes8 = key.id()
	}
}

func BenchmarkKeyEqual(b *testing.B) {
	a, c := GenerateKey(), GenerateKey()
	b.ReportAllocs()

	for b.Loop() {
		sinkBool = a.Equal(c)
	}
}

func BenchmarkSecretEqual(b *testing.B) {
	s := Secret("t0ps3cr3t-not-a-real-token")
	b.ReportAllocs()

	for b.Loop() {
		sinkBool = s.Equal("t0ps3cr3t-not-a-real-token")
	}
}

func BenchmarkBytes(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		sinkSlice = Bytes(32)
	}
}

func BenchmarkToken(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		sinkString = Token(32)
	}
}

func BenchmarkString(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		sinkString = String(20)
	}
}

func BenchmarkDigits(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		sinkString = Digits(6)
	}
}

func BenchmarkIntn(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		sinkInt = Intn(7)
	}
}

var (
	sinkKey    Key
	sinkBytes8 [8]byte
	sinkBool   bool
	sinkSlice  []byte
	sinkString string
	sinkInt    int
)

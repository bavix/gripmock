package deeply //nolint:testpackage

import "testing"

func BenchmarkDistance_SmallASCII(b *testing.B) {
	s1 := "kitten"
	s2 := "sitting"

	b.ReportAllocs()

	for b.Loop() {
		distance(s1, s2)
	}
}

func BenchmarkDistance_SmallUnicode(b *testing.B) {
	s1 := "kø̃tten" // Norwegian/Danish characters
	s2 := "sø̃tting"

	b.ReportAllocs()

	for b.Loop() {
		distance(s1, s2)
	}
}

func BenchmarkDistance_LargeASCII(b *testing.B) {
	s1 := "Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore et dolore magna aliqua"
	s2 := "Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore et dolore magna aliqua changed"

	b.ReportAllocs()

	for b.Loop() {
		distance(s1, s2)
	}
}

func BenchmarkDistance_LargeUnicode(b *testing.B) {
	s1 := "Lørém ípsüm dölör sít ämét cönsectétür ädïpïscïng élït séd dö èïüsdöd tëmpör ïncïdïdûnt üt läböre ét dölöre mågnä älïqüä"
	s2 := "Lørém ípsüm dölör sít ämét cönsectétür ädïpïscïng élït séd dö èïüsdöd tëmpör ïncïdïdûnt üt läböre ét dölöre mågnä älïqüä chängéd"

	b.ReportAllocs()

	for b.Loop() {
		distance(s1, s2)
	}
}

func BenchmarkDistance_IdenticalStrings(b *testing.B) {
	s := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b.ReportAllocs()

	for b.Loop() {
		distance(s, s)
	}
}

func BenchmarkDistance_EmptyStrings(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		distance("", "")
	}
}

func BenchmarkDistance_OneEmpty(b *testing.B) {
	s := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b.ReportAllocs()

	for b.Loop() {
		distance(s, "")
	}
}

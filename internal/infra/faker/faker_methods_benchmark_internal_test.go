package faker_test

import (
	"testing"
	"time"

	infrafaker "github.com/bavix/gripmock/v3/internal/infra/faker"
)

func benchmarkStringMethod(b *testing.B, fn func() string) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = fn()
	}
}

func benchmarkIntMethod(b *testing.B, fn func() int) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = fn()
	}
}

func benchmarkFloatMethod(b *testing.B, fn func() float64) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = fn()
	}
}

func benchmarkTimeMethod(b *testing.B, fn func() time.Time) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = fn()
	}
}

func BenchmarkPersonMethods(b *testing.B) {
	g := infrafaker.NewWithSeed(1)
	p := g.Person()

	b.Run("FirstName", func(b *testing.B) { benchmarkStringMethod(b, p.FirstName) })
	b.Run("LastName", func(b *testing.B) { benchmarkStringMethod(b, p.LastName) })
	b.Run("Name", func(b *testing.B) { benchmarkStringMethod(b, p.Name) })
	b.Run("Prefix", func(b *testing.B) { benchmarkStringMethod(b, p.Prefix) })
	b.Run("Suffix", func(b *testing.B) { benchmarkStringMethod(b, p.Suffix) })
	b.Run("Gender", func(b *testing.B) { benchmarkStringMethod(b, p.Gender) })
	b.Run("Age", func(b *testing.B) { benchmarkIntMethod(b, p.Age) })
}

func BenchmarkContactMethods(b *testing.B) {
	g := infrafaker.NewWithSeed(1)
	c := g.Contact()

	b.Run("Email", func(b *testing.B) { benchmarkStringMethod(b, c.Email) })
	b.Run("Phone", func(b *testing.B) { benchmarkStringMethod(b, c.Phone) })
	b.Run("Username", func(b *testing.B) { benchmarkStringMethod(b, c.Username) })
	b.Run("URL", func(b *testing.B) { benchmarkStringMethod(b, c.URL) })
}

func BenchmarkGeoMethods(b *testing.B) {
	g := infrafaker.NewWithSeed(1)
	geo := g.Geo()

	b.Run("Country", func(b *testing.B) { benchmarkStringMethod(b, geo.Country) })
	b.Run("CountryCode", func(b *testing.B) { benchmarkStringMethod(b, geo.CountryCode) })
	b.Run("City", func(b *testing.B) { benchmarkStringMethod(b, geo.City) })
	b.Run("State", func(b *testing.B) { benchmarkStringMethod(b, geo.State) })
	b.Run("StateCode", func(b *testing.B) { benchmarkStringMethod(b, geo.StateCode) })
	b.Run("Zip", func(b *testing.B) { benchmarkStringMethod(b, geo.Zip) })
	b.Run("Street", func(b *testing.B) { benchmarkStringMethod(b, geo.Street) })
	b.Run("Latitude", func(b *testing.B) { benchmarkFloatMethod(b, geo.Latitude) })
	b.Run("Longitude", func(b *testing.B) { benchmarkFloatMethod(b, geo.Longitude) })
	b.Run("TimeZone", func(b *testing.B) { benchmarkStringMethod(b, geo.TimeZone) })
}

func BenchmarkNetworkMethods(b *testing.B) {
	g := infrafaker.NewWithSeed(1)
	n := g.Network()

	b.Run("DomainName", func(b *testing.B) { benchmarkStringMethod(b, n.DomainName) })
	b.Run("DomainSuffix", func(b *testing.B) { benchmarkStringMethod(b, n.DomainSuffix) })
	b.Run("IPv4", func(b *testing.B) { benchmarkStringMethod(b, n.IPv4) })
	b.Run("IPv6", func(b *testing.B) { benchmarkStringMethod(b, n.IPv6) })
	b.Run("MAC", func(b *testing.B) { benchmarkStringMethod(b, n.MAC) })
	b.Run("UserAgent", func(b *testing.B) { benchmarkStringMethod(b, n.UserAgent) })
	b.Run("HTTPMethod", func(b *testing.B) { benchmarkStringMethod(b, n.HTTPMethod) })
	b.Run("HTTPStatusCode", func(b *testing.B) { benchmarkIntMethod(b, n.HTTPStatusCode) })
}

func BenchmarkCompanyMethods(b *testing.B) {
	g := infrafaker.NewWithSeed(1)
	c := g.Company()

	b.Run("Company", func(b *testing.B) { benchmarkStringMethod(b, c.Company) })
	b.Run("CompanySuffix", func(b *testing.B) { benchmarkStringMethod(b, c.CompanySuffix) })
	b.Run("JobTitle", func(b *testing.B) { benchmarkStringMethod(b, c.JobTitle) })
	b.Run("JobLevel", func(b *testing.B) { benchmarkStringMethod(b, c.JobLevel) })
	b.Run("JobDescriptor", func(b *testing.B) { benchmarkStringMethod(b, c.JobDescriptor) })
}

func BenchmarkCommerceMethods(b *testing.B) {
	g := infrafaker.NewWithSeed(1)
	c := g.Commerce()

	b.Run("ProductName", func(b *testing.B) { benchmarkStringMethod(b, c.ProductName) })
	b.Run("ProductCategory", func(b *testing.B) { benchmarkStringMethod(b, c.ProductCategory) })
	b.Run("ProductDescription", func(b *testing.B) { benchmarkStringMethod(b, c.ProductDescription) })
	b.Run("CurrencyLong", func(b *testing.B) { benchmarkStringMethod(b, c.CurrencyLong) })
	b.Run("CurrencyShort", func(b *testing.B) { benchmarkStringMethod(b, c.CurrencyShort) })
	b.Run("Price", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			_ = c.Price(10, 20)
		}
	})
}

func BenchmarkTextMethods(b *testing.B) {
	g := infrafaker.NewWithSeed(1)
	txt := g.Text()

	b.Run("Word", func(b *testing.B) { benchmarkStringMethod(b, txt.Word) })
	b.Run("Sentence", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			_ = txt.Sentence(12)
		}
	})
	b.Run("Paragraph", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			_ = txt.Paragraph(2)
		}
	})
	b.Run("Phrase", func(b *testing.B) { benchmarkStringMethod(b, txt.Phrase) })
	b.Run("Quote", func(b *testing.B) { benchmarkStringMethod(b, txt.Quote) })
	b.Run("Question", func(b *testing.B) { benchmarkStringMethod(b, txt.Question) })
}

func BenchmarkDateTimeMethods(b *testing.B) {
	g := infrafaker.New()
	d := g.DateTime()

	b.Run("Date", func(b *testing.B) { benchmarkTimeMethod(b, d.Date) })
	b.Run("PastDate", func(b *testing.B) { benchmarkTimeMethod(b, d.PastDate) })
	b.Run("FutureDate", func(b *testing.B) { benchmarkTimeMethod(b, d.FutureDate) })
	b.Run("Year", func(b *testing.B) { benchmarkIntMethod(b, d.Year) })
	b.Run("Month", func(b *testing.B) { benchmarkIntMethod(b, d.Month) })
	b.Run("Day", func(b *testing.B) { benchmarkIntMethod(b, d.Day) })
	b.Run("Hour", func(b *testing.B) { benchmarkIntMethod(b, d.Hour) })
	b.Run("Minute", func(b *testing.B) { benchmarkIntMethod(b, d.Minute) })
	b.Run("Second", func(b *testing.B) { benchmarkIntMethod(b, d.Second) })
	b.Run("WeekDay", func(b *testing.B) { benchmarkStringMethod(b, d.WeekDay) })
}

func BenchmarkIdentityMethods(b *testing.B) {
	g := infrafaker.NewWithSeed(1)
	i := g.Identity()

	b.Run("UUID", func(b *testing.B) { benchmarkStringMethod(b, i.UUID) })
	b.Run("SSN", func(b *testing.B) { benchmarkStringMethod(b, i.SSN) })
	b.Run("EIN", func(b *testing.B) { benchmarkStringMethod(b, i.EIN) })
}

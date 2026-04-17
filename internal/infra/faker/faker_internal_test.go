package faker_test

import (
	"net/mail"
	"net/url"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	infrafaker "github.com/bavix/gripmock/v3/internal/infra/faker"
)

func TestNewContracts(t *testing.T) {
	t.Parallel()

	g := infrafaker.New()
	require.NotNil(t, g.Person())
	require.NotNil(t, g.Contact())
	require.NotNil(t, g.Geo())
	require.NotNil(t, g.Network())
	require.NotNil(t, g.Company())
	require.NotNil(t, g.Commerce())
	require.NotNil(t, g.Text())
	require.NotNil(t, g.DateTime())
	require.NotNil(t, g.Identity())
}

func TestPersonContract(t *testing.T) {
	t.Parallel()

	p := infrafaker.NewWithSeed(10).Person()
	require.NotEmpty(t, p.FirstName())
	require.NotEmpty(t, p.LastName())
	require.NotEmpty(t, p.Name())
	require.NotEmpty(t, p.Prefix())
	require.NotEmpty(t, p.Suffix())
	require.NotEmpty(t, p.Gender())
	require.Positive(t, p.Age())
}

func TestContactContract(t *testing.T) {
	t.Parallel()

	c := infrafaker.NewWithSeed(11).Contact()

	email := c.Email()
	require.NotEmpty(t, email)
	_, err := mail.ParseAddress(email)
	require.NoError(t, err)

	require.NotEmpty(t, c.Phone())
	require.NotEmpty(t, c.Username())

	u := c.URL()
	require.NotEmpty(t, u)
	parsed, err := url.Parse(u)
	require.NoError(t, err)
	require.NotEmpty(t, parsed.Scheme)
	require.NotEmpty(t, parsed.Host)
}

func TestGeoContract(t *testing.T) {
	t.Parallel()

	geo := infrafaker.NewWithSeed(12).Geo()
	require.NotEmpty(t, geo.Country())
	require.NotEmpty(t, geo.CountryCode())
	require.NotEmpty(t, geo.City())
	require.NotEmpty(t, geo.State())
	require.NotEmpty(t, geo.StateCode())
	require.NotEmpty(t, geo.Zip())
	require.NotEmpty(t, geo.Street())
	require.NotZero(t, geo.Latitude())
	require.NotZero(t, geo.Longitude())
	require.NotEmpty(t, geo.TimeZone())
}

func TestNetworkContract(t *testing.T) {
	t.Parallel()

	n := infrafaker.NewWithSeed(13).Network()
	require.NotEmpty(t, n.DomainName())
	require.NotEmpty(t, n.DomainSuffix())
	require.NotEmpty(t, n.IPv4())
	require.NotEmpty(t, n.IPv6())
	require.NotEmpty(t, n.MAC())
	require.NotEmpty(t, n.UserAgent())
	require.NotEmpty(t, n.HTTPMethod())
	require.GreaterOrEqual(t, n.HTTPStatusCode(), 100)
}

func TestCompanyContract(t *testing.T) {
	t.Parallel()

	c := infrafaker.NewWithSeed(14).Company()
	require.NotEmpty(t, c.Company())
	require.NotEmpty(t, c.CompanySuffix())
	require.NotEmpty(t, c.JobTitle())
	require.NotEmpty(t, c.JobLevel())
	require.NotEmpty(t, c.JobDescriptor())
}

func TestCommerceContract(t *testing.T) {
	t.Parallel()

	c := infrafaker.NewWithSeed(15).Commerce()
	require.NotEmpty(t, c.ProductName())
	require.NotEmpty(t, c.ProductCategory())
	require.NotEmpty(t, c.ProductDescription())
	require.NotEmpty(t, c.CurrencyLong())
	require.NotEmpty(t, c.CurrencyShort())
	require.GreaterOrEqual(t, c.Price(10, 20), 10.0)
	require.LessOrEqual(t, c.Price(10, 20), 20.0)
}

func TestTextContract(t *testing.T) {
	t.Parallel()

	txt := infrafaker.NewWithSeed(16).Text()
	require.NotEmpty(t, txt.Word())
	require.NotEmpty(t, txt.Sentence(5))
	require.NotEmpty(t, txt.Sentence(0))
	require.NotEmpty(t, txt.Sentence(-10))
	require.NotEmpty(t, txt.Sentence(1000))
	require.NotEmpty(t, txt.Paragraph(1))
	require.NotEmpty(t, txt.Paragraph(0))
	require.NotEmpty(t, txt.Paragraph(1000))
	require.NotEmpty(t, txt.Phrase())
	require.NotEmpty(t, txt.Quote())
	require.NotEmpty(t, txt.Question())
}

func TestDateTimeContract(t *testing.T) {
	t.Parallel()

	d := infrafaker.NewWithSeed(17).DateTime()
	require.False(t, d.Date().IsZero())
	require.False(t, d.PastDate().IsZero())
	require.False(t, d.FutureDate().IsZero())
	require.Positive(t, d.Year())
	require.GreaterOrEqual(t, d.Month(), 1)
	require.LessOrEqual(t, d.Month(), 12)
	require.GreaterOrEqual(t, d.Day(), 1)
	require.LessOrEqual(t, d.Day(), 31)
	require.GreaterOrEqual(t, d.Hour(), 0)
	require.LessOrEqual(t, d.Hour(), 23)
	require.GreaterOrEqual(t, d.Minute(), 0)
	require.LessOrEqual(t, d.Minute(), 59)
	require.GreaterOrEqual(t, d.Second(), 0)
	require.LessOrEqual(t, d.Second(), 59)
	require.NotEmpty(t, d.WeekDay())
}

func TestIdentityContract(t *testing.T) {
	t.Parallel()

	i := infrafaker.NewWithSeed(18).Identity()
	require.NotEmpty(t, i.SSN())
	require.NotEmpty(t, i.EIN())

	v := i.UUID()
	require.NotEmpty(t, v)
	_, err := uuid.Parse(v)
	require.NoError(t, err)
}

func TestNewWithSeedDeterministicSequence(t *testing.T) {
	t.Parallel()

	g1 := infrafaker.NewWithSeed(123)
	g2 := infrafaker.NewWithSeed(123)

	seq1 := []string{
		g1.Person().FirstName(),
		g1.Person().LastName(),
		g1.Contact().Email(),
		g1.Contact().Phone(),
		g1.Text().Word(),
		g1.Identity().UUID(),
	}
	seq2 := []string{
		g2.Person().FirstName(),
		g2.Person().LastName(),
		g2.Contact().Email(),
		g2.Contact().Phone(),
		g2.Text().Word(),
		g2.Identity().UUID(),
	}

	require.Equal(t, seq1, seq2)
}

func TestGeneratorConcurrentUsage(t *testing.T) {
	t.Parallel()

	g := infrafaker.NewWithSeed(99)

	const (
		workers = 16
		rounds  = 50
	)

	var wg sync.WaitGroup
	wg.Add(workers)

	results := make(chan string, workers*rounds)

	for i := range workers {
		go func(_ int) {
			defer wg.Done()

			for range rounds {
				results <- g.Identity().UUID()
			}
		}(i)
	}

	wg.Wait()
	close(results)

	count := 0
	for v := range results {
		count++

		require.NotEmpty(t, v)
	}

	require.Equal(t, workers*rounds, count)
}

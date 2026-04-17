package faker

func (g commerceGen) ProductName() string        { return g.faker.ProductName() }
func (g commerceGen) ProductCategory() string    { return g.faker.ProductCategory() }
func (g commerceGen) ProductDescription() string { return g.faker.ProductDescription() }
func (g commerceGen) CurrencyLong() string       { return g.faker.CurrencyLong() }
func (g commerceGen) CurrencyShort() string      { return g.faker.CurrencyShort() }

func (g commerceGen) Price(pMin, pMax float64) float64 {
	return g.faker.Price(pMin, pMax)
}

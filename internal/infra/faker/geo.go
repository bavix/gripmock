package faker

func (g geoGen) Country() string     { return g.faker.Country() }
func (g geoGen) CountryCode() string { return g.faker.CountryAbr() }
func (g geoGen) City() string        { return g.faker.City() }
func (g geoGen) State() string       { return g.faker.State() }
func (g geoGen) StateCode() string   { return g.faker.StateAbr() }
func (g geoGen) Zip() string         { return g.faker.Zip() }
func (g geoGen) Street() string      { return g.faker.Street() }
func (g geoGen) Latitude() float64   { return g.faker.Latitude() }
func (g geoGen) Longitude() float64  { return g.faker.Longitude() }
func (g geoGen) TimeZone() string    { return g.faker.TimeZone() }

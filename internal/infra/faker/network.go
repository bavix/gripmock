package faker

func (g networkGen) DomainName() string   { return g.faker.DomainName() }
func (g networkGen) DomainSuffix() string { return g.faker.DomainSuffix() }
func (g networkGen) IPv4() string         { return g.faker.IPv4Address() }
func (g networkGen) IPv6() string         { return g.faker.IPv6Address() }
func (g networkGen) MAC() string          { return g.faker.MacAddress() }
func (g networkGen) UserAgent() string    { return g.faker.UserAgent() }
func (g networkGen) HTTPMethod() string   { return g.faker.HTTPMethod() }
func (g networkGen) HTTPStatusCode() int  { return g.faker.HTTPStatusCode() }

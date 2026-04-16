package faker

func (g identityGen) UUID() string { return g.faker.UUID() }
func (g identityGen) SSN() string  { return g.faker.SSN() }
func (g identityGen) EIN() string  { return g.faker.EIN() }

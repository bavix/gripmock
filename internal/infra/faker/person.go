package faker

func (g personGen) FirstName() string { return g.faker.FirstName() }
func (g personGen) LastName() string  { return g.faker.LastName() }
func (g personGen) Name() string      { return g.faker.Name() }
func (g personGen) Prefix() string    { return g.faker.NamePrefix() }
func (g personGen) Suffix() string    { return g.faker.NameSuffix() }
func (g personGen) Gender() string    { return g.faker.Gender() }
func (g personGen) Age() int          { return g.faker.Age() }

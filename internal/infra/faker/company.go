package faker

func (g companyGen) Company() string       { return g.faker.Company() }
func (g companyGen) CompanySuffix() string { return g.faker.CompanySuffix() }
func (g companyGen) JobTitle() string      { return g.faker.JobTitle() }
func (g companyGen) JobLevel() string      { return g.faker.JobLevel() }
func (g companyGen) JobDescriptor() string { return g.faker.JobDescriptor() }

package faker

func (g contactGen) Email() string    { return g.faker.Email() }
func (g contactGen) Phone() string    { return g.faker.Phone() }
func (g contactGen) Username() string { return g.faker.Username() }
func (g contactGen) URL() string      { return g.faker.URL() }

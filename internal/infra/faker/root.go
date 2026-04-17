package faker

//nolint:ireturn
func (g *generator) Person() PersonContract { return &g.person }

//nolint:ireturn
func (g *generator) Contact() ContactContract { return &g.contact }

//nolint:ireturn
func (g *generator) Geo() GeoContract { return &g.geo }

//nolint:ireturn
func (g *generator) Network() NetworkContract { return &g.network }

//nolint:ireturn
func (g *generator) Company() CompanyContract { return &g.company }

//nolint:ireturn
func (g *generator) Commerce() CommerceContract { return &g.commerce }

//nolint:ireturn
func (g *generator) Text() TextContract { return &g.text }

//nolint:ireturn
func (g *generator) DateTime() DateTimeContract { return &g.datetime }

//nolint:ireturn
func (g *generator) Identity() IdentityContract { return &g.identity }

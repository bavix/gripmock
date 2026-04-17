package faker

import "time"

func (g dateTimeGen) Date() time.Time       { return g.faker.Date() }
func (g dateTimeGen) PastDate() time.Time   { return g.faker.PastDate() }
func (g dateTimeGen) FutureDate() time.Time { return g.faker.FutureDate() }
func (g dateTimeGen) Year() int             { return g.faker.Year() }
func (g dateTimeGen) Month() int            { return g.faker.Month() }
func (g dateTimeGen) Day() int              { return g.faker.Day() }
func (g dateTimeGen) Hour() int             { return g.faker.Hour() }
func (g dateTimeGen) Minute() int           { return g.faker.Minute() }
func (g dateTimeGen) Second() int           { return g.faker.Second() }
func (g dateTimeGen) WeekDay() string       { return g.faker.WeekDay() }

package faker

import "time"

func (g dateTimeGen) Date() string       { return g.faker.Date().Format(time.RFC3339Nano) }
func (g dateTimeGen) PastDate() string   { return g.faker.PastDate().Format(time.RFC3339Nano) }
func (g dateTimeGen) FutureDate() string { return g.faker.FutureDate().Format(time.RFC3339Nano) }
func (g dateTimeGen) Year() int          { return g.faker.Year() }
func (g dateTimeGen) Month() int         { return g.faker.Month() }
func (g dateTimeGen) Day() int           { return g.faker.Day() }
func (g dateTimeGen) Hour() int          { return g.faker.Hour() }
func (g dateTimeGen) Minute() int        { return g.faker.Minute() }
func (g dateTimeGen) Second() int        { return g.faker.Second() }
func (g dateTimeGen) WeekDay() string    { return g.faker.WeekDay() }

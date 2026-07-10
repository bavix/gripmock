package faker

func (g numberGen) Int() int                                { return g.faker.Int() }
func (g numberGen) IntN(n int) int                          { return g.faker.IntN(n) }
func (g numberGen) IntRange(pMin, pMax int) int             { return g.faker.IntRange(pMin, pMax) }
func (g numberGen) Int32() int32                            { return g.faker.Int32() }
func (g numberGen) Int64() int64                            { return g.faker.Int64() }
func (g numberGen) Float32() float32                        { return g.faker.Float32() }
func (g numberGen) Float32Range(pMin, pMax float32) float32 { return g.faker.Float32Range(pMin, pMax) }
func (g numberGen) Float64() float64                        { return g.faker.Float64() }
func (g numberGen) Float64Range(pMin, pMax float64) float64 { return g.faker.Float64Range(pMin, pMax) }

package faker

func (g textGen) Word() string { return g.faker.Word() }

func (g textGen) Sentence(wordCount int) string {
	return g.faker.Sentence(wordCount)
}

func (g textGen) Paragraph(paragraphCount int) string {
	return g.faker.Paragraph(paragraphCount)
}

func (g textGen) Phrase() string   { return g.faker.Phrase() }
func (g textGen) Quote() string    { return g.faker.Quote() }
func (g textGen) Question() string { return g.faker.Question() }

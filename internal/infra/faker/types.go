package faker

import "github.com/brianvoe/gofakeit/v7"

type generator struct {
	faker    *gofakeit.Faker
	person   personGen
	contact  contactGen
	geo      geoGen
	network  networkGen
	company  companyGen
	commerce commerceGen
	text     textGen
	datetime dateTimeGen
	identity identityGen
}

type (
	personGen   struct{ faker *gofakeit.Faker }
	contactGen  struct{ faker *gofakeit.Faker }
	geoGen      struct{ faker *gofakeit.Faker }
	networkGen  struct{ faker *gofakeit.Faker }
	companyGen  struct{ faker *gofakeit.Faker }
	commerceGen struct{ faker *gofakeit.Faker }
	textGen     struct{ faker *gofakeit.Faker }
	dateTimeGen struct{ faker *gofakeit.Faker }
	identityGen struct{ faker *gofakeit.Faker }
)

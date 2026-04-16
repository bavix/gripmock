package faker

import "time"

type PersonContract interface {
	FirstName() string
	LastName() string
	Name() string
	Prefix() string
	Suffix() string
	Gender() string
	Age() int
}

type ContactContract interface {
	Email() string
	Phone() string
	Username() string
	URL() string
}

type GeoContract interface {
	Country() string
	CountryCode() string
	City() string
	State() string
	StateCode() string
	Zip() string
	Street() string
	Latitude() float64
	Longitude() float64
	TimeZone() string
}

type NetworkContract interface {
	DomainName() string
	DomainSuffix() string
	IPv4() string
	IPv6() string
	MAC() string
	UserAgent() string
	HTTPMethod() string
	HTTPStatusCode() int
}

type CompanyContract interface {
	Company() string
	CompanySuffix() string
	JobTitle() string
	JobLevel() string
	JobDescriptor() string
}

type CommerceContract interface {
	ProductName() string
	ProductCategory() string
	ProductDescription() string
	CurrencyLong() string
	CurrencyShort() string
	Price(pMin, pMax float64) float64
}

type TextContract interface {
	Word() string
	Sentence(wordCount int) string
	Paragraph(paragraphCount int) string
	Phrase() string
	Quote() string
	Question() string
}

type DateTimeContract interface {
	Date() time.Time
	PastDate() time.Time
	FutureDate() time.Time
	Year() int
	Month() int
	Day() int
	Hour() int
	Minute() int
	Second() int
	WeekDay() string
}

type IdentityContract interface {
	UUID() string
	SSN() string
	EIN() string
}

// Generator is high-level faker API grouped by semantic domains.
// This makes template usage predictable and implementation swappable.
type Generator interface {
	Person() PersonContract
	Contact() ContactContract
	Geo() GeoContract
	Network() NetworkContract
	Company() CompanyContract
	Commerce() CommerceContract
	Text() TextContract
	DateTime() DateTimeContract
	Identity() IdentityContract
}

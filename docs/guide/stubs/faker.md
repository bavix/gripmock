---
title: Faker Reference
---

# Faker Reference <VersionTag version="v3.10.0" />

The built-in faker generates values inside stub templates. Keys are written
`faker.DOMAIN.METHOD` and used as <code v-pre>{{faker.DOMAIN.METHOD}}</code>.
Each evaluation produces a new value.

## 1. Person

| Key | Example |
| --- | --- |
| `faker.Person.FirstName` | `Emma` |
| `faker.Person.LastName` | `Johnson` |
| `faker.Person.Name` | `Dr. Emma Johnson` |
| `faker.Person.Prefix` | `Dr.` |
| `faker.Person.Suffix` | `Jr.` |
| `faker.Person.Gender` | `female` |
| `faker.Person.Age` | `34` |

::: v-pre
```yaml
output:
  data:
    first_name: "{{faker.Person.FirstName}}"
    last_name: "{{faker.Person.LastName}}"
    full_name: "{{faker.Person.Name}}"
    age: "{{faker.Person.Age}}"
```
:::

## 2. Contact

| Key | Example |
| --- | --- |
| `faker.Contact.Email` | `john.smith@example.org` |
| `faker.Contact.Phone` | `+1 202-555-0141` |
| `faker.Contact.Username` | `silent-river-42` |
| `faker.Contact.URL` | `https://api.demo-app.io/users/42` |

::: v-pre
```yaml
output:
  data:
    email: "{{faker.Contact.Email}}"
    phone: "{{faker.Contact.Phone}}"
    username: "{{faker.Contact.Username}}"
    website: "{{faker.Contact.URL}}"
```
:::

## 3. Geo

| Key | Example |
| --- | --- |
| `faker.Geo.Country` | `United States` |
| `faker.Geo.CountryCode` | `US` |
| `faker.Geo.City` | `San Francisco` |
| `faker.Geo.State` | `California` |
| `faker.Geo.StateCode` | `CA` |
| `faker.Geo.Zip` | `94107` |
| `faker.Geo.Street` | `127 Market St` |
| `faker.Geo.Latitude` | `37.7749` |
| `faker.Geo.Longitude` | `-122.4194` |
| `faker.Geo.TimeZone` | `America/Los_Angeles` |

::: v-pre
```yaml
output:
  data:
    country: "{{faker.Geo.Country}}"
    city: "{{faker.Geo.City}}"
    lat: "{{faker.Geo.Latitude}}"
    lon: "{{faker.Geo.Longitude}}"
```
:::

## 4. Network

| Key | Example |
| --- | --- |
| `faker.Network.DomainName` | `customer-api.example.net` |
| `faker.Network.DomainSuffix` | `net` |
| `faker.Network.IPv4` | `192.168.14.22` |
| `faker.Network.IPv6` | `2001:db8:85a3::8a2e:370:7334` |
| `faker.Network.MAC` | `3a:8f:52:9d:11:be` |
| `faker.Network.UserAgent` | `Mozilla/5.0 (...)` |
| `faker.Network.HTTPMethod` | `PATCH` |
| `faker.Network.HTTPStatusCode` | `409` |

::: v-pre
```yaml
output:
  data:
    ipv4: "{{faker.Network.IPv4}}"
    ua: "{{faker.Network.UserAgent}}"
    status: "{{faker.Network.HTTPStatusCode}}"
```
:::

## 5. Company

| Key | Example |
| --- | --- |
| `faker.Company.Company` | `Acme Digital Labs` |
| `faker.Company.CompanySuffix` | `LLC` |
| `faker.Company.JobTitle` | `Senior Platform Engineer` |
| `faker.Company.JobLevel` | `Senior` |
| `faker.Company.JobDescriptor` | `Lead` |

::: v-pre
```yaml
output:
  data:
    company: "{{faker.Company.Company}}"
    company_suffix: "{{faker.Company.CompanySuffix}}"
    title: "{{faker.Company.JobTitle}}"
```
:::

## 6. Commerce

| Key | Example |
| --- | --- |
| `faker.Commerce.ProductName` | `Wireless Noise-Canceling Headphones` |
| `faker.Commerce.ProductCategory` | `Electronics` |
| `faker.Commerce.ProductDescription` | `Compact over-ear headphones with active noise cancellation.` |
| `faker.Commerce.CurrencyLong` | `US Dollar` |
| `faker.Commerce.CurrencyShort` | `USD` |
| `faker.Commerce.Price 10 500` | `249.99` |

::: v-pre
```yaml
output:
  data:
    product: "{{faker.Commerce.ProductName}}"
    currency: "{{faker.Commerce.CurrencyShort}}"
    price: "{{faker.Commerce.Price 10 500}}"
```
:::

## 7. Text

| Key | Example |
| --- | --- |
| `faker.Text.Word` | `spectrum` |
| `faker.Text.Sentence 8` | `Service health remains stable under peak request load.` |
| `faker.Text.Paragraph 2` | `Two short random paragraphs for testing long fields.` |
| `faker.Text.Phrase` | `blue horizon` |
| `faker.Text.Quote` | `Small steps every day build strong systems.` |
| `faker.Text.Question` | `Can we safely retry this request?` |

::: v-pre
```yaml
output:
  data:
    title: "{{faker.Text.Phrase}}"
    summary: "{{faker.Text.Sentence 10}}"
    quote: "{{faker.Text.Quote}}"
```
:::

## 8. DateTime

| Key | Example |
| --- | --- |
| `faker.DateTime.Date` | `2026-02-17T10:24:51.123456789Z` |
| `faker.DateTime.PastDate` | `2021-08-03T14:12:11.987654321Z` |
| `faker.DateTime.FutureDate` | `2028-11-29T07:53:02.456789012Z` |
| `faker.DateTime.Year` | `2027` |
| `faker.DateTime.Month` | `9` |
| `faker.DateTime.Day` | `18` |
| `faker.DateTime.Hour` | `16` |
| `faker.DateTime.Minute` | `42` |
| `faker.DateTime.Second` | `5` |
| `faker.DateTime.WeekDay` | `Tuesday` |

::: v-pre
```yaml
output:
  data:
    created_at: "{{faker.DateTime.PastDate}}"
    expires_at: "{{faker.DateTime.FutureDate}}"
    weekday: "{{faker.DateTime.WeekDay}}"
```
:::

## 9. Number <VersionTag version="v3.16.1" />

| Key | Example |
| --- | --- |
| `faker.Number.Int` | `7249581` |
| `faker.Number.IntN 100` | `73` |
| `faker.Number.IntRange 10 50` | `27` |
| `faker.Number.Int32` | `17249581` |
| `faker.Number.Int64` | `38942174958123` |
| `faker.Number.Float32` | `0.7425` |
| `faker.Number.Float32Range 1.5 9.5` | `4.23` |
| `faker.Number.Float64` | `0.123456789` |
| `faker.Number.Float64Range 100.0 999.0` | `567.89` |

::: v-pre
```yaml
output:
  data:
    score: "{{faker.Number.IntN 100}}"
    rating: "{{faker.Number.Float32}}"
    max_users: "{{faker.Number.IntRange 10 100}}"
    price: "{{faker.Number.Float64Range 9.99 999.99}}"
```
:::

## 10. Identity

| Key | Example |
| --- | --- |
| `faker.Identity.UUID` | `3f8b6a6e-3f34-41e2-a06f-e6a8b8db7a4d` |
| `faker.Identity.SSN` | `513-84-3901` |
| `faker.Identity.EIN` | `26-9182736` |

SSN is a US Social Security Number, EIN a US Employer Identification Number.
Both are synthetic here — never treat a generated value as a real identifier.

::: v-pre
```yaml
output:
  data:
    user_id: "{{faker.Identity.UUID}}"
    ssn: "{{faker.Identity.SSN}}"
    ein: "{{faker.Identity.EIN}}"
```
:::

## Assertions

Faker values change on every evaluation, so assert on format or range rather
than on an exact value. Where a test needs to trace a response back to its
request, mix in a request-bound field: <code v-pre>{{.Request.id}}</code>.

## Full Stub Example

::: v-pre
```yaml
- service: example.UserService
  method: GetProfile
  input:
    matches:
      id: "\\d+"
  output:
    data:
      id: "{{.Request.id}}"
      first_name: "{{faker.Person.FirstName}}"
      last_name: "{{faker.Person.LastName}}"
      full_name: "{{faker.Person.Name}}"
      email: "{{faker.Contact.Email}}"
      phone: "{{faker.Contact.Phone}}"
      city: "{{faker.Geo.City}}"
      country: "{{faker.Geo.Country}}"
      lat: "{{faker.Geo.Latitude}}"
      lon: "{{faker.Geo.Longitude}}"
      ip: "{{faker.Network.IPv4}}"
      user_agent: "{{faker.Network.UserAgent}}"
      company: "{{faker.Company.Company}}"
      product: "{{faker.Commerce.ProductName}}"
      price: "{{faker.Commerce.Price 10 500}}"
      bio: "{{faker.Text.Paragraph 1}}"
      created_at: "{{faker.DateTime.PastDate}}"
      score: "{{faker.Number.IntN 100}}"
      account_id: "{{faker.Identity.UUID}}"
```
:::

Values come from
[`github.com/brianvoe/gofakeit/v7`](https://github.com/brianvoe/gofakeit).

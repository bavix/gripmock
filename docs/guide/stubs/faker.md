---
title: Faker Reference
---

# Faker Reference <VersionTag version="v3.10.0" />

Built-in faker generates realistic dynamic values directly in stub templates.

## How It Works

Faker is available as template object:

- **Key**: `faker.DOMAIN.METHOD`
- **Template usage**: <code v-pre>{{faker.DOMAIN.METHOD}}</code>
- **Result**: generated at runtime for each evaluation

## 1. Person Domain

### Keys
- Key: `faker.Person.FirstName` - Example: `Emma`
- Key: `faker.Person.LastName` - Example: `Johnson`
- Key: `faker.Person.Name` - Example: `Dr. Emma Johnson`
- Key: `faker.Person.Prefix` - Example: `Dr.`
- Key: `faker.Person.Suffix` - Example: `Jr.`
- Key: `faker.Person.Gender` - Example: `female`
- Key: `faker.Person.Age` - Example: `34`

### Stub Example

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

## 2. Contact Domain

### Keys
- Key: `faker.Contact.Email` - Example: `john.smith@example.org`
- Key: `faker.Contact.Phone` - Example: `+1 202-555-0141`
- Key: `faker.Contact.Username` - Example: `silent-river-42`
- Key: `faker.Contact.URL` - Example: `https://api.demo-app.io/users/42`

### Stub Example

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

## 3. Geo Domain

### Keys
- Key: `faker.Geo.Country` - Example: `United States`
- Key: `faker.Geo.CountryCode` - Example: `US`
- Key: `faker.Geo.City` - Example: `San Francisco`
- Key: `faker.Geo.State` - Example: `California`
- Key: `faker.Geo.StateCode` - Example: `CA`
- Key: `faker.Geo.Zip` - Example: `94107`
- Key: `faker.Geo.Street` - Example: `127 Market St`
- Key: `faker.Geo.Latitude` - Example: `37.7749`
- Key: `faker.Geo.Longitude` - Example: `-122.4194`
- Key: `faker.Geo.TimeZone` - Example: `America/Los_Angeles`

### Stub Example

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

## 4. Network Domain

### Keys
- Key: `faker.Network.DomainName` - Example: `customer-api.example.net`
- Key: `faker.Network.DomainSuffix` - Example: `net`
- Key: `faker.Network.IPv4` - Example: `192.168.14.22`
- Key: `faker.Network.IPv6` - Example: `2001:db8:85a3::8a2e:370:7334`
- Key: `faker.Network.MAC` - Example: `3a:8f:52:9d:11:be`
- Key: `faker.Network.UserAgent` - Example: `Mozilla/5.0 (...)`
- Key: `faker.Network.HTTPMethod` - Example: `PATCH`
- Key: `faker.Network.HTTPStatusCode` - Example: `409`

### Stub Example

::: v-pre
```yaml
output:
  data:
    ipv4: "{{faker.Network.IPv4}}"
    ua: "{{faker.Network.UserAgent}}"
    status: "{{faker.Network.HTTPStatusCode}}"
```
:::

## 5. Company Domain

### Keys
- Key: `faker.Company.Company` - Example: `Acme Digital Labs`
- Key: `faker.Company.CompanySuffix` - Example: `LLC`
- Key: `faker.Company.JobTitle` - Example: `Senior Platform Engineer`
- Key: `faker.Company.JobLevel` - Example: `Senior`
- Key: `faker.Company.JobDescriptor` - Example: `Lead`

### Stub Example

::: v-pre
```yaml
output:
  data:
    company: "{{faker.Company.Company}}"
    company_suffix: "{{faker.Company.CompanySuffix}}"
    title: "{{faker.Company.JobTitle}}"
```
:::

## 6. Commerce Domain

### Keys
- Key: `faker.Commerce.ProductName` - Example: `Wireless Noise-Canceling Headphones`
- Key: `faker.Commerce.ProductCategory` - Example: `Electronics`
- Key: `faker.Commerce.ProductDescription` - Example: `Compact over-ear headphones with active noise cancellation.`
- Key: `faker.Commerce.CurrencyLong` - Example: `US Dollar`
- Key: `faker.Commerce.CurrencyShort` - Example: `USD`
- Key: `faker.Commerce.Price 10 500` - Example: `249.99`

### Stub Example

::: v-pre
```yaml
output:
  data:
    product: "{{faker.Commerce.ProductName}}"
    currency: "{{faker.Commerce.CurrencyShort}}"
    price: "{{faker.Commerce.Price 10 500}}"
```
:::

## 7. Text Domain

### Keys
- Key: `faker.Text.Word` - Example: `spectrum`
- Key: `faker.Text.Sentence 8` - Example: `Service health remains stable under peak request load.`
- Key: `faker.Text.Paragraph 2` - Example: `Two short random paragraphs for testing long fields.`
- Key: `faker.Text.Phrase` - Example: `blue horizon`
- Key: `faker.Text.Quote` - Example: `Small steps every day build strong systems.`
- Key: `faker.Text.Question` - Example: `Can we safely retry this request?`

### Stub Example

::: v-pre
```yaml
output:
  data:
    title: "{{faker.Text.Phrase}}"
    summary: "{{faker.Text.Sentence 10}}"
    quote: "{{faker.Text.Quote}}"
```
:::

## 8. DateTime Domain

### Keys
- Key: `faker.DateTime.Date` - Example: `2026-02-17T10:24:51.123456789Z`
- Key: `faker.DateTime.PastDate` - Example: `2021-08-03T14:12:11.987654321Z`
- Key: `faker.DateTime.FutureDate` - Example: `2028-11-29T07:53:02.456789012Z`
- Key: `faker.DateTime.Year` - Example: `2027`
- Key: `faker.DateTime.Month` - Example: `9`
- Key: `faker.DateTime.Day` - Example: `18`
- Key: `faker.DateTime.Hour` - Example: `16`
- Key: `faker.DateTime.Minute` - Example: `42`
- Key: `faker.DateTime.Second` - Example: `5`
- Key: `faker.DateTime.WeekDay` - Example: `Tuesday`

### Stub Example

::: v-pre
```yaml
output:
  data:
    created_at: "{{faker.DateTime.PastDate}}"
    expires_at: "{{faker.DateTime.FutureDate}}"
    weekday: "{{faker.DateTime.WeekDay}}"
```
:::

## 9. Number Domain <VersionTag version="v3.16.1" />

The Number domain generates random numeric values — useful for counters, limits, prices, and range validation.

### Keys
- Key: `faker.Number.Int` - Example: `7249581`
- Key: `faker.Number.IntN 100` - Example: `73`
- Key: `faker.Number.IntRange 10 50` - Example: `27`
- Key: `faker.Number.Int32` - Example: `17249581`
- Key: `faker.Number.Int64` - Example: `38942174958123`
- Key: `faker.Number.Float32` - Example: `0.7425`
- Key: `faker.Number.Float32Range 1.5 9.5` - Example: `4.23`
- Key: `faker.Number.Float64` - Example: `0.123456789`
- Key: `faker.Number.Float64Range 100.0 999.0` - Example: `567.89`

### Stub Example

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

## 10. Identity Domain

### Keys
- Key: `faker.Identity.UUID` - Example: `3f8b6a6e-3f34-41e2-a06f-e6a8b8db7a4d`
- Key: `faker.Identity.SSN` - Example: `513-84-3901`
- Key: `faker.Identity.EIN` - Example: `26-9182736`

### Definitions
- SSN = US Social Security Number (personal tax/identity number).
- EIN = US Employer Identification Number (business tax identifier).
- Values are synthetic test data only.

### Stub Example

::: v-pre
```yaml
output:
  data:
    user_id: "{{faker.Identity.UUID}}"
    ssn: "{{faker.Identity.SSN}}"
    ein: "{{faker.Identity.EIN}}"
```
:::

## Best Practices

- Use faker for realism, not for strict deterministic snapshots.
- Validate format/range instead of exact values in assertions.
- Never store or expose generated identity-like values as real user data.
- Mix faker with request-bound values when traceability is needed.

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

## Thanks

GripMock built-in faker is powered by the excellent open-source library
[`github.com/brianvoe/gofakeit/v7`](https://github.com/brianvoe/gofakeit).

Huge thanks to the library author and contributors.

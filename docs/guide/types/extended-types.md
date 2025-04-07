# Extended Types (`google.type.*`) in Protocol Buffers

Extended types (`google.type.*`) are domain-specific Protobuf types that standardize complex data structures. They are part of the `googleapis` repository and provide reusable definitions for common use cases like money, geolocation, and dates. This documentation covers **all major extended types** with examples, usage guidelines, and best practices.

## 1. `google.type.Money`
Represents a monetary amount with currency precision.

### Syntax
```proto
import "google/type/money.proto";

message BalanceResponse {
  google.type.Money balance = 1;
}
```

### Key Features
- **Currency Code**: ISO 4217 (e.g., `"USD"`, `"EUR"`).
- **Units/Nanos**: `units` (integer part) and `nanos` (fractional part, 0–999,999,999).

### Example: Bank Account Balance
**Proto File (`extended_money.proto`):**
```proto
syntax = "proto3";

import "google/type/money.proto";

package extended;

service MoneyService {
  rpc GetBalance(BalanceRequest) returns (BalanceResponse) {}
}

message BalanceRequest {
  string accountId = 1;
}

message BalanceResponse {
  google.type.Money balance = 1;
}
```

**Stub Configuration (`extended_money.yaml`):**
```yaml
- service: MoneyService
  method: GetBalance
  input:
    equals:
      accountId: "user_123"
  output:
    data:
      balance:
        currencyCode: "USD"
        units: 100
        nanos: 500000000  # $100.50
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"accountId": "user_123"}' localhost:4770 extended.MoneyService/GetBalance
```

**Output:**
```json
{
  "balance": {
    "currencyCode": "USD",
    "units": 100,
    "nanos": 500000000
  }
}
```

## 2. `google.type.LatLng`
Represents geographic coordinates (latitude and longitude).

### Syntax
```proto
import "google/type/latlng.proto";

message LocationResponse {
  google.type.LatLng coordinates = 1;
}
```

### Key Features
- **Precision**: Floating-point values with up to 8 decimal places.
- **Validation**: Latitude must be in `[-90, 90]`, longitude in `[-180, 180]`.

### Example: Geolocation Service
**Proto File (`extended_latlng.proto`):**
```proto
syntax = "proto3";

import "google/type/latlng.proto";

package extended;

service LocationService {
  rpc GetCoordinates(LocationRequest) returns (LocationResponse) {}
}

message LocationRequest {
  string address = 1;
}

message LocationResponse {
  google.type.LatLng coordinates = 1;
}
```

**Stub Configuration (`extended_latlng.yaml`):**
```yaml
- service: LocationService
  method: GetCoordinates
  input:
    equals:
      address: "Eiffel Tower"
  output:
    data:
      coordinates:
        latitude: 48.8584
        longitude: 2.2945
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"address": "Eiffel Tower"}' localhost:4770 extended.LocationService/GetCoordinates
```

**Output:**
```json
{
  "coordinates": {
    "latitude": 48.8584,
    "longitude": 2.2945
  }
}
```

## 3. `google.type.DateTime` and `google.type.TimeZone`
Represents a date, time, and time zone.

### Syntax
```proto
import "google/type/datetime.proto";
import "google/type/timezone.proto";

message EventResponse {
  google.type.DateTime eventTime = 1;
  google.type.TimeZone timeZone = 2;
}
```

### Key Features
- **Date/Time Fields**: Year, month, day, hours, minutes, seconds.
- **Time Zone**: IANA name (e.g., `"America/New_York"`) or UTC offset.

### Example: Event Scheduler
**Proto File (`extended_datetime.proto`):**
```proto
syntax = "proto3";

import "google/type/datetime.proto";
import "google/type/timezone.proto";

package extended;

service DateTimeService {
  rpc GetEventTime(DateTimeRequest) returns (DateTimeResponse) {}
}

message DateTimeRequest {
  string eventId = 1;
}

message DateTimeResponse {
  google.type.DateTime eventTime = 1;
  google.type.TimeZone timeZone = 2;
}
```

**Stub Configuration (`extended_datetime.yaml`):**
```yaml
- service: DateTimeService
  method: GetEventTime
  input:
    equals:
      eventId: "event_123"
  output:
    data:
      eventTime:
        year: 2024
        month: 12
        day: 25
        hours: 18
        minutes: 30
        seconds: 0
      timeZone:
        id: "Europe/London"
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"eventId": "event_123"}' localhost:4770 extended.DateTimeService/GetEventTime
```

**Output:**
```json
{
  "eventTime": {
    "year": 2024,
    "month": 12,
    "day": 25,
    "hours": 18,
    "minutes": 30,
    "seconds": 0
  },
  "timeZone": {
    "id": "Europe/London"
  }
}
```

## 4. `google.type.Date` and `google.type.TimeOfDay`
Represents a date without time and a time without a date.

### Syntax
```proto
import "google/type/date.proto";
import "google/type/timeofday.proto";

message Schedule {
  google.type.Date date = 1;
  google.type.TimeOfDay time = 2;
}
```

### Example: Birthday Reminder
**Proto File (`extended_date.proto`):**
```proto
syntax = "proto3";

import "google/type/date.proto";
import "google/type/timeofday.proto";

package extended;

service ScheduleService {
  rpc GetBirthday(BirthdayRequest) returns (BirthdayResponse) {}
}

message BirthdayRequest {
  string userId = 1;
}

message BirthdayResponse {
  google.type.Date date = 1;
  google.type.TimeOfDay reminderTime = 2;
}
```

**Stub Configuration (`extended_date.yaml`):**
```yaml
- service: ScheduleService
  method: GetBirthday
  input:
    equals:
      userId: "user_123"
  output:
    data:
      date:
        year: 1990
        month: 5
        day: 15
      reminderTime:
        hours: 9
        minutes: 0
        seconds: 0
        nanos: 0
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"userId": "user_123"}' localhost:4770 extended.ScheduleService/GetBirthday
```

**Output:**
```json
{
  "date": {
    "year": 1990,
    "month": 5,
    "day": 15
  },
  "reminderTime": {
    "hours": 9,
    "minutes": 0,
    "seconds": 0
  }
}
```

## 5. `google.type.PostalAddress`
Represents a structured postal address.

### Syntax
```proto
import "google/type/postaladdress.proto";

message AddressResponse {
  google.type.PostalAddress address = 1;
}
```

### Key Features
- **Structured Fields**: Recipient, street, city, region, postal code, etc.
- **Localization**: Supports international addresses.

### Example: Address Validation
**Proto File (`extended_postaladdress.proto`):**
```proto
syntax = "proto3";

import "google/type/postaladdress.proto";

package extended;

service AddressService {
  rpc ValidateAddress(AddressRequest) returns (AddressResponse) {}
}

message AddressRequest {
  string addressId = 1;
}

message AddressResponse {
  google.type.PostalAddress validatedAddress = 1;
}
```

**Stub Configuration (`extended_postaladdress.yaml`):**
```yaml
- service: AddressService
  method: ValidateAddress
  input:
    equals:
      addressId: "addr_123"
  output:
    data:
      validatedAddress:
        regionCode: "US"
        postalCode: "94043"
        administrativeArea: "CA"
        locality: "Mountain View"
        addressLines: ["1600 Amphitheatre Parkway"]
        recipients: ["Google Inc."]
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"addressId": "addr_123"}' localhost:4770 extended.AddressService/ValidateAddress
```

**Output:**
```json
{
  "validatedAddress": {
    "regionCode": "US",
    "postalCode": "94043",
    "administrativeArea": "CA",
    "locality": "Mountain View",
    "addressLines": ["1600 Amphitheatre Parkway"],
    "recipients": ["Google Inc."]
  }
}
```

## 6. `google.type.Color`
Represents a color in RGB/RGBA format.

### Syntax
```proto
import "google/type/color.proto";

message DesignResponse {
  google.type.Color primaryColor = 1;
}
```

### Example: Design Tool
**Proto File (`extended_color.proto`):**
```proto
syntax = "proto3";

import "google/type/color.proto";

package extended;

service DesignService {
  rpc GetThemeColor(ColorRequest) returns (ColorResponse) {}
}

message ColorRequest {
  string themeId = 1;
}

message ColorResponse {
  google.type.Color color = 1;
}
```

**Stub Configuration (`extended_color.yaml`):**
```yaml
- service: DesignService
  method: GetThemeColor
  input:
    equals:
      themeId: "dark_theme"
  output:
    data:
      color:
        red: 0.1
        green: 0.2
        blue: 0.3
        alpha: 0.8
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"themeId": "dark_theme"}' localhost:4770 extended.DesignService/GetThemeColor
```

**Output:**
```json
{
  "color": {
    "red": 0.1,
    "green": 0.2,
    "blue": 0.3,
    "alpha": 0.8
  }
}
```

## 7. `google.type.Interval`
Represents a time interval with start and end times.

### Syntax
```proto
import "google/type/interval.proto";

message BookingResponse {
  google.type.Interval booking_time = 1;
}
```

### Example: Booking System
**Proto File (`extended_interval.proto`):**
```proto
syntax = "proto3";

import "google/type/interval.proto";

package extended;

service BookingService {
  rpc GetBooking(BookingRequest) returns (BookingResponse) {}
}

message BookingRequest {
  string bookingId = 1;
}

message BookingResponse {
  google.type.Interval bookingTime = 1;
}
```

**Stub Configuration (`extended_interval.yaml`):**
```yaml
- service: BookingService
  method: GetBooking
  input:
    equals:
      bookingId: "booking_123"
  output:
    data:
      bookingTime:
        startTime:
          seconds: 1704000000  # 2024-01-01T00:00:00Z
        endTime:
          seconds: 1704086399  # 2024-01-01T23:59:59Z
```

**Test Command:**
```sh
grpcurl -plaintext -d '{"bookingId": "booking_123"}' localhost:4770 extended.BookingService/GetBooking
```

**Output:**
```json
{
  "bookingTime": {
    "startTime": "2024-01-01T00:00:00Z",
    "endTime": "2024-01-01T23:59:59Z"
  }
}
```

## 8. `google.protobuf.Empty` 
Represents an empty request or response (placeholder for no data).  

### Syntax  
```proto  
import "google/protobuf/empty.proto";  
service ExampleService {  
  rpc GetData(google.protobuf.Empty) returns (DataResponse) {}  
}  
```  

### Key Features  
- Used for RPC methods that require no input or return no data.  
- Commonly used for operations like health checks, status resets, or triggering background processes.  

### Example: Service with Empty Input  
**Proto File (`extended_empty.proto`):**  
```proto  
syntax = "proto3";  
import "google/protobuf/empty.proto";  
package extended;  

service EmptyService {  
  rpc GetData(google.protobuf.Empty) returns (DataResponse) {}  
}  

message DataResponse {  
  string content = 1;  
}  
```  

**Stub Configuration (`extended_empty.yaml`):**  
```yaml  
- service: EmptyService  
  method: GetData  
  input:  
    matches: {}  # Required for empty input in Gripmock  
  output:  
    data:  
      content: "test"  
    code: 0  
```  

**Test Command:**  
```sh  
grpcurl -plaintext -d '{}' localhost:4770 extended.EmptyService/GetData  
```  

**Output:**  
```json  
{  
  "content": "test"  
}  
```  

### Gripmock Specific Behavior  
For RPC methods with empty input (e.g., `rpc GetData(google.protobuf.Empty)`), **always specify `input.matches: {}`** in the stub configuration. This ensures Gripmock correctly handles the absence of input data.

## Best Practices
1. **Currency Codes**: Always use ISO 4217 codes (e.g., `"USD"`) with `Money`.
2. **Time Zones**: Prefer IANA names (e.g., `"America/New_York"`) over UTC offsets.
3. **Validation**: Ensure `LatLng` values are within valid ranges.
4. **Postal Addresses**: Use `address_lines` for street-level details and `administrative_area` for regions/states.

## Common Pitfalls
- **Money Precision**: Avoid using `float`/`double` for money; use `units` and `nanos`.
- **DateTime Defaults**: Missing `time_zone` may lead to ambiguous times.
- **Color Alpha**: `alpha` is optional (defaults to 1.0), but always specify it for transparency.

## Further Reading
- [googleapis/googleapis: Extended Types](https://github.com/googleapis/googleapis/tree/master/google/type)
- [Protobuf Well-Known Types](https://protobuf.dev/reference/protobuf/google.protobuf/)

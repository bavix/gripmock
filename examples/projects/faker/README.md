# Faker Example

This project demonstrates how to use the built-in faker functionality in GripMock to generate dynamic, realistic data in your stubs.

## Structure

- `service.proto`: Defines a `UserService` with `GetProfile` and `GetSubscription` methods.
- `stub.yaml`: Contains stubs that use `faker` templates to populate response fields.

## How to run

1. Start GripMock with this project:
   ```bash
   # from repo source (recommended while developing):
   go run main.go --stub examples/projects/faker/stub.yaml examples/projects/faker/service.proto

   # or installed binary:
   gripmock --stub examples/projects/faker/stub.yaml examples/projects/faker/service.proto
   ```

2. Make requests to the `GetProfile` or `GetSubscription` methods:
   ```bash
   # Using grpcurl
   grpcurl -plaintext -d '{"id": "123"}' localhost:4770 example.UserService/GetProfile
   grpcurl -plaintext -d '{"user_id": "42"}' localhost:4770 example.UserService/GetSubscription
   ```

## Faker domains covered

### GetProfile
| Domain | Fields |
|--------|--------|
| **Person** | `first_name`, `last_name` |
| **Contact** | `email` |
| **Geo** | `city`, `lat`, `lon` |
| **Network** | `ip`, `user_agent` |
| **Company** | `company` |
| **Commerce** | `product` |
| **Text** | `bio` |
| **DateTime** | `created_at` |
| **Identity** | `account_id` |

### GetSubscription
| Domain | Fields |
|--------|--------|
| **Identity** | `subscription_id` |
| **Commerce** | `plan_name`, `price`, `currency` |
| **Contact** | `billing_email` |
| **Person** | `billing_name` |
| **Company** | `company_name` |
| **Number** | `max_users`, `trial_days` |
| **Text** | `description` |
| **DateTime** | `start_date`, `next_billing_date`, `cancelled_at` |

## google.protobuf.Timestamp support

The `GetSubscription` endpoint uses `google.protobuf.Timestamp` fields (`start_date`, `next_billing_date`, `cancelled_at`) populated with faker DateTime values. Faker DateTime methods output RFC3339Nano format (e.g., `2026-02-17T10:24:51.123456789Z`), which is compatible with protobuf Timestamp JSON deserialization.

## Test cases

- `case_get_profile_fields.gctf` — validates basic faker fields are non-empty
- `case_get_profile_different_ids.gctf` — verifies faker values differ across requests
- `case_get_profile_same_request_randomized.gctf` — verifies faker re-randomizes on each call
- `case_get_subscription.gctf` — validates subscription with Number domain and Timestamp fields

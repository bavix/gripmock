---
title: "Why You Should Specify IDs in Stubs"
---

# Why You Should Specify IDs in Stubs

Every stub has a UUID. If a stub file does not carry one, GripMock generates a
UUIDv4 on load — so the stub works either way, but its ID changes every time the
file is reloaded. Writing the ID down keeps it stable.

## 1. The ID must be a UUID

```yaml
- id: 7f746310-a470-43dc-9eeb-355dff50d2a9
  service: BookingService
  method: GetBooking
  input:
    equals:
      bookingId: "booking_123"
  output:
    data:
      bookingTime:
        startTime: "2024-01-01T00:00:00Z"
        endTime: "2024-01-01T23:59:59Z"
```

A custom string such as `my_stub_123` fails to parse. Generate one with
`uuidgen` or with the [UUID tools](https://bavix.github.io/uuid-ui/).

## 2. Deep links into the admin panel

```bash
http://localhost:4771/#/stubs/7f746310-a470-43dc-9eeb-355dff50d2a9/show
```

::: warning
Search by ID in the admin panel is not available yet; use the direct URL.
:::

## 3. Deleting a stub by ID

```bash
# Find unused stubs
curl http://localhost:4771/api/stubs/unused

# Delete one
curl -X DELETE http://localhost:4771/api/stubs/7f746310-a470-43dc-9eeb-355dff50d2a9
```

Without a written ID, the value you read from `/api/stubs/unused` is only valid
until the next reload.

## 4. Live reloading

The stub watcher is on by default and reloads changed files without a restart:

```bash
STUB_WATCHER_ENABLED=true   # default: true
STUB_WATCHER_INTERVAL=1s    # default: 1s, timer mode only
STUB_WATCHER_TYPE=fsnotify  # default: fsnotify; other option: timer
```

Reloading works with or without IDs. What IDs change is the outcome. A stub with
a written ID is updated in place. An ID-less stub is re-assigned from the pool of
IDs that file held before, handed out in file order — so inserting or deleting a
stub in the middle of the file shifts every ID after it onto a different stub.
Anything holding the old value — a bookmark, a `DELETE` call, a test fixture —
now points somewhere else.

::: info
In timer mode GripMock does not reload every file each tick. It reloads only
files whose modification time changed.
:::

## 5. Collision prevention

Unique IDs keep two teams from overwriting each other's stubs:

```yaml
# Team A
- id: 6e8b4c2a-3d8f-4a1b-8c9d-0e7f2a9b8c7d
  service: Payments
  method: Process

# Team B
- id: 9f1a2b3c-4d5e-6f7a-8b9c-0d1e2f3a4b5c
  service: Payments
  method: Refund
```

Do not reuse an ID across environments.

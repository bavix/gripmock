# Circular References Example

This example demonstrates GripMock's handling of circular references in Protocol Buffer messages.

## Issue

https://github.com/bavix/gripmock/issues/897

When a proto file contains circular links like:

```
message Reward {
  oneof reward {
    ExprReward expr_reward = 3;
  }
}

message ExprReward {
  Reward reward = 2;
}
```

The `convertToMap` → `convertScalar` → `convertToMap` recursion never terminates, causing a stack overflow.

## Proto Definition

The `service.proto` file defines:
- `RewardService` with `GetReward` and `ProcessReward` unary methods
- `Reward` message containing a `oneof` with `ExprReward` 
- `ExprReward` message containing a reference back to `Reward`
- `ProcessReward(Reward)` triggers the circular reference path in `convertToMap`

## Running the Example

Start the mock server:

```bash
gripmock --stub examples/projects/circular-refs examples/projects/circular-refs/service.proto
```

Or using Docker:

```bash
docker run -p 4770:4770 -p 4771:4771 \
  -v $(pwd)/examples/projects/circular-refs:/proto \
  bavix/gripmock /proto
```

## Testing with grpcurl

```bash
grpcurl -plaintext \
  -d '{"id": "reward_1"}' \
  localhost:4770 circular.RewardService/GetReward
```

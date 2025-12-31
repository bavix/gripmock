---
title: Advanced Plugin Features
---

# Advanced Plugin Features <VersionTag version="v3.5.0" />

## Decorators

```go
plugins.FuncSpec{
	Name:      "add",
	Decorates: "@gripmock/add",
	Fn: func(base func(context.Context, ...any) (any, error)) func(context.Context, ...any) (any, error) {
		return func(ctx context.Context, args ...any) (any, error) {
			val, err := base(ctx, args...)
			if err != nil {
				return nil, err
			}
			switch v := val.(type) {
			case float64:
				return v + 1, nil
			case int:
				return v + 1, nil
			default:
				return val, nil
			}
		}
	},
}
```

## Groups

```go
plugins.FuncSpec{
	Name:  "crc32",
	Fn:    crc32Function,
	Group: "crc",
}
```

## Replacement

```go
plugins.FuncSpec{
	Name:        "md5",
	Fn:          md5Function,
	Replacement: "sha256",
}
```

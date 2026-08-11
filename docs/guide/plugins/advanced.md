---
title: Advanced Plugin Features
---

# Advanced Plugin Features <VersionTag version="v3.5.0" />

Three optional `FuncSpec` fields change how a function is registered.

## Decorators

`Decorates` wraps an existing function instead of registering a new one. The
value names the target as `@plugin/function`, or as a bare function name to
decorate one from the same plugin. Your `Fn` receives the original as `base` and
returns the replacement.

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

A decorator does not collide with the function it wraps. Registering the same
name *without* `Decorates` does collide: the later registration is marked
deactivated and a warning is logged.

## Groups

`Group` tags a function so it can be looked up as a set — `Hooks(group)` returns
every function carrying that group name.

```go
plugins.FuncSpec{
	Name:  "crc32",
	Fn:    crc32Function,
	Group: "crc",
}
```

## Replacement

`Replacement` marks a function as deprecated and names its successor. The
function keeps working; `gripmock info` renders it as
`md5 [deprecated → sha256]`.

```go
plugins.FuncSpec{
	Name:        "md5",
	Fn:          md5Function,
	Replacement: "sha256",
}
```

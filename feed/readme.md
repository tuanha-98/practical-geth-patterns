# Feed Pattern Issue in Viction: Observations from Geth's Implementation
During practice with Geth's Feed pattern, an issue was identified in the Feed implementation of Viction.

## How to run test
```bash
$ cd ./feed/
$ go test -v
```

## Overview
In Geth, `Feed` is initialized through two primary methods: `Subscribe` and `Send`. These methods invoke `init` a single time using `once.Do()` to ensure `init` is executed just once. `init` in Geth accepts an `etype` parameter, which sets the feed’s data type (`feed.etype`). This setup ensures that after `etype` is assigned, any mismatched value sent, or mismatched subscription channel will trigger a `panic` with a custom error.

## Key Differences Between Geth and Viction Feed Implementations

### Feed in Geth
- `init(etype reflect.Type)`: Accepts `etype` to set the data type.
- `Subscribe(channel interface{})`: Validates the channel’s data type against `etype` (set during initialization); mismatches cause a custom `panic`.
- `Send(value interface{})`: Verifies value's data type against `etype`; mismatches also cause a custom `panic`.

### Feed in Viction
- `init()`: Does not assign the data type to `etype` when called.
- `Subscribe(channel interface{})`: Does not validate the channel’s data type against `etype`.
- `Send(value interface{})`: Checks value's data type against `etype` and assigns it (if `etype` = nil) via the `typecheck` method.
- `typecheck(valueType reflect.Type)`: Assigns a data type to `etype` and returns true or false based on whether value matches `etype`.

## Issues in Viction’s Feed

1. Calling `Subscribe` Before `Send`
- If Subscribe is called first, etype has not yet been assigned.
- A later Send call with a mismatched value type causes a runtime panic, as etype has yet to validate against the channel.

2. Calling Send Before Subscribe
- If Send is called first, etype is assigned the data type of value.
- If Subscribe is subsequently called with a channel of a different data type, no immediate panic occurs. However, later Send calls will cause a runtime panic due to the data type mismatch between value and the channel.

## Conclusion
Currently, Viction might avoid this issue if all feeds are initialized with Subscribe first. However, addressing this will ensure long-term reliability and avoid runtime errors.
# Ursa

Ursa is a [Zod](https://github.com/colinhacks/zod) inspired validatin library.

## Key features

- Parse primitives (`int`, `string` etc)
- Parse objects
- Parse JSON
- Parse HTTP requests
  - URL encoded `GET`
  - URL encoded form `POST`
  - Multipart encoded `POST`
  - JSON `POST`

## Features

Supports:

- numbers (int/uint/float) (can parse strings)
  - validate min
  - max
  - non zero
  - integer
- strings
  - validate min length
  - max length
  - regex
  - email
- time.Time (can parse strings)
  - validate not before
  - not after
- uuid.UUID (can parse strings)
  - validate not zero
- Objects (parse from struct, map or HTTPRequest)

## Basic Usage

```go
// Parse a primitive

// define a schema
schema := u.String(
  u.MinLength(5, "String should be at least 5 characters"), // use optional validation message
  u.MaxLength(10), // use default validation message
  u.Matches("^[0-9]*$"))

// call parse to get a result
result := schema.Parse("01234678")

// if valid then extract value
if result.Valid() {
  fmt.Println(result.Value())
}

// if not valid then extract errors
if !result.Valid() {
  errs := result.Errors()
  for _, e := range errs {
    fmt.Printlin(e.Error())
  }
}

// parse an object
// define a schema
schema := u.Object().
  String("Name", u.MinLength(5, "String should be at least 5 characters")).
  Number("Count", u.Int()) // not defining struct data type

errs := schema.Parse(&struct {
  Name  string
  Count int
}{
  Name:  "abcdef",
  Count: 5,
}).Errors()
```

## Gotchas

- the library uses `reflect.ValueOf(...).Convert(...)` to coerce between e.g.
  - Strings: be aware that this can do some surprising coversions e.g. ints to strings.
  - Numbers: will coerce floats to ints and silently drop the fractional part

## TODO

- validate strings against enum
- Multipart file handling
- generate types

# Ursa

Ursa is a [Zod](https://github.com/colinhacks/zod) inspired validatin library.

## Key features

- Parse primitives (`int`, `string` etc)
- Parse objects (`struct` or `map`)
- Parse JSON
- Parse HTTP requests
  - URL encoded `GET`
  - URL encoded form `POST`
  - Multipart form encoded `POST`
  - JSON `POST`
- handle multipart files
- Unmarshal to `struct` or `map`
  - use tags to find field names

## Features

Supports:

- numbers (int/uint/float) (converts strings transparently)
  - validate min
  - max
  - non zero
  - integer
- strings
  - validate min length
  - max length
  - regex
  - email
  - enum
- time.Time (can parse strings)
  - validate not before
  - not after
- uuid.UUID (can parse strings)
  - validate not zero
- Objects (parse from struct, map or HTTPRequest)

## Basic Usage

```go
import u "github.com/jdudmesh/ursa"

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
// ursa wil check tags to find field names
type SignupParams struct {
  Plan            string `json:"plan" form:"plan" query:"plan"`
  Name            string `json:"name" form:"name" query:"name"`
  Email           string `json:"email" form:"email" query:"email"`
  Password        string `json:"password" form:"password" query:"password"`
  ConfirmPassword string `json:"password2" form:"password2" query:"password2"`
}

// define a schema
var signupSchema = u.Object().
  String("plan", u.Enum("personal", "starter", "pro")).
  String("name", u.MinLength(4, "Your name should be at least 4 characters")).
  String("email", u.Email("Please enter a valid email address")).
  String("password", u.MinLength(8, "Your password should be at least 10 characters")).
  String("password2", u.MinLength(8, "The password and confirmation do not match")).
  Refine(func(res u.ObjectParseResult) {
    if res.GetString("password") != res.GetString("password2") {
      res.Append(false, "The password and confirmation do not match", errors.New("password mismatch"))
    }
  })

http.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {

  signupParams := signupSchema.Parse(r)

  if signupParams.Valid() {

    params := &model.SignupParams{}
    signupParams.Unmarshal(params)

    user, err := svc.CreateUser(params)
    if err == nil {
      // e.g. marshal user
      w.Write(...)
      return
    }
  }
  w.Write(...)
})

// if you're using a template engine e.g. Templ you can prepopulate a parse result
// this example uses Labstack Echo
func SignupPage() func(c echo.Context) error {
	return func(c echo.Context) error {
    // this is an initial page load so we need to pass a valid but empty result to the page template

    // extract the pricing plan from the query string
		plan := c.QueryParam("plan")

    // generate a parse result we can pop
		signupParams, err := signupSchema.From(true, &model.SignupParams{Plan: plan})

    // pass it to the template engine
		err = views.SignupPage(signupParams, model.DefaultPlans).Render(c.Request().Context(), c.Response().Writer)
		if err != nil {
			log.Errorf("rendering signup: %s", err)
			return echo.ErrInternalServerError
		}
		return nil
	}
}

// snippet from signup.templ
templ SignupPage(res u.ObjectParseResult) {
...
	<div class="form-control">
		<label class="label" for="name">
			<span class="label-text">Name (*)</span>
		</label>
		<input
 			name="name"
 			type="text"
 			value={ res.GetString("name") }
 			placeholder="Your name"
 			class={ "input input-bordered", templ.KV("input-error", !res.IsFieldValid("name")) }
 			autocomplete="name"
 			aria-label="Your name"
 			if res.IsFieldValid("name") {
				aria-invalid="true"
			}
		/>
	</div>
...
	<div id="signup-form-errors" class="mx-4">
		<ul class="list-disc">
			for _, msg := range res.Errors() {
				<li class="text-sm text-error">{ msg.Error() }</li>
			}
		</ul>
	</div>
...

// handle a file in a multipart form
v := u.Object(u.WithMaxBodySize(1000),
  u.WithFileHandler(func(name string, header *multipart.FileHeader) error {
    log.Infof("file name: %s", header.Filename)
    file, err := header.Open()
    buf := new(bytes.Buffer)
    _, err = buf.ReadFrom(file)
    ... do something with file
    return nil
  }))
```

## Gotchas

- the library uses `reflect.ValueOf(...).Convert(...)` to coerce between e.g.
  - Strings: be aware that this can do some surprising coversions e.g. ints to strings.
  - Numbers: will coerce floats to ints and silently drop the fractional part

package ursa

// ursa is a zod inspired validation library for Go.
// Copyright (C) 2023 John Dudmesh

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
)

type objectValidatorOpt func(o *objectValidator) error

type objectValidator struct {
	validators  map[string]genericValidator[any]
	maxBodySize int64
	err         error
}

type objectParseResult struct {
	parseResult[map[string]*parseResult[any]]
}

func (r *objectParseResult) Set(val any) {
	if val, ok := val.(map[string]*parseResult[any]); ok {
		r.value = val
	}
}

func (r *objectParseResult) Unmarshal(val any) {
	//TODO: implememnt
}

func (o *objectParseResult) GetField(field string) *parseResult[any] {
	return o.value[field]
}

func Object(opts ...any) *objectValidator {
	v := &objectValidator{
		validators:  make(map[string]genericValidator[interface{}]),
		maxBodySize: 1024 * 1024 * 10,
	}
	for _, opt := range opts {
		switch opt := opt.(type) {
		case objectValidatorOpt:
			err := opt(v)
			if err != nil {
				v.err = err
			}
		}
	}
	return v
}

func (o *objectValidator) Parse(val any, opts ...parseOpt[any]) *objectParseResult {
	parseRes := &objectParseResult{
		parseResult: parseResult[map[string]*parseResult[any]]{
			valid:  true,
			value:  make(map[string]*parseResult[any]),
			errors: make([]*parseError, 0),
		},
	}

	if o.err != nil {
		parseRes.errors = []*parseError{InvalidValidatorStateError}
		return parseRes
	}

	switch val := val.(type) {
	case []byte:
		return o.parseJSON(val)
	case *http.Request:
		return o.parseRequest(val)
	}

	for name, validator := range o.validators {
		var fieldResult *parseResult[interface{}]
		fieldVal, err := o.extract(val, name)
		if err != nil {
			fieldResult = &parseResult[interface{}]{
				errors: []*parseError{
					{
						message: "failed to extract value", inner: []error{err},
					},
				},
			}
		} else {
			res := validator.Parse(fieldVal)
			fieldResult = &parseResult[interface{}]{valid: res.Valid(), value: res.Get(), errors: res.Errors()}
		}
		parseRes.value[name] = fieldResult
		parseRes.errors = append(parseRes.errors, fieldResult.errors...)
		if !fieldResult.Valid() {
			parseRes.valid = false
		}
	}

	return parseRes
}

func (o *objectValidator) Error() error {
	return nil
}

func (o *objectValidator) Type() reflect.Type {
	return reflect.TypeOf(map[string]*parseResult[interface{}]{})
}

func (o *objectValidator) extract(val any, name string) (any, error) {
	vo := reflect.ValueOf(val)
	switch {
	case vo.Kind() == reflect.Ptr:
		deref := reflect.Indirect(vo)
		if !(deref.Kind() == reflect.Struct || deref.Kind() == reflect.Map) {
			return nil, InvalidTypeError
		}
		vo = deref
	case !(vo.Kind() == reflect.Struct || vo.Kind() == reflect.Map):
		return nil, InvalidTypeError
	}

	var v reflect.Value
	switch vo.Kind() {
	case reflect.Struct:
		v = vo.FieldByName(name)
	case reflect.Map:
		v = vo.MapIndex(reflect.ValueOf(name))
	}

	if !v.IsValid() {
		return nil, nil
	}

	return v.Interface(), nil
}

func (o *objectValidator) parseJSON(val []byte, opts ...parseOpt[any]) *objectParseResult {
	unpacked := make(map[string]interface{})
	err := json.Unmarshal(val, &unpacked)
	if err != nil {
		return &objectParseResult{
			parseResult: parseResult[map[string]*parseResult[any]]{
				valid:  false,
				errors: []*parseError{{message: "unmarshalling JSON value", inner: []error{err}}},
			},
		}
	}
	return o.Parse(unpacked, opts...)
}

func (o *objectValidator) parseRequest(req *http.Request, opts ...parseOpt[any]) *objectParseResult {
	contentType := strings.TrimSpace(strings.Split(req.Header.Get("Content-Type"), ";")[0])

	body := req.Body
	if body != nil {
		defer body.Close()
	}

	numBytes := req.ContentLength
	if numBytes > o.maxBodySize {
		return &objectParseResult{
			parseResult: parseResult[map[string]*parseResult[any]]{
				errors: []*parseError{{message: "request body too large"}},
			},
		}
	}

	switch contentType {
	case "application/json":
		buf, err := o.readBody(body, int(numBytes))
		if err != nil {
			return &objectParseResult{
				parseResult: parseResult[map[string]*parseResult[any]]{
					errors: []*parseError{err},
				},
			}
		}
		return o.parseJSON(buf, opts...)

	case "application/x-www-form-urlencoded":
		err := req.ParseForm()
		if err != nil {
			return &objectParseResult{
				parseResult: parseResult[map[string]*parseResult[any]]{
					errors: []*parseError{{message: "parsing form", inner: []error{err}}},
				},
			}
		}
		return o.Parse(o.readForm(req.Form), opts...)

	case "multipart/form-data":
		err := req.ParseMultipartForm(o.maxBodySize)
		if err != nil {
			return &objectParseResult{
				parseResult: parseResult[map[string]*parseResult[any]]{
					errors: []*parseError{{message: "parsing multipart form", inner: []error{err}}},
				},
			}
		}
		return o.Parse(o.readForm(req.Form), opts...)

	default:
		if req.Method == "GET" {
			err := req.ParseForm()
			if err != nil {
				return &objectParseResult{
					parseResult: parseResult[map[string]*parseResult[any]]{
						errors: []*parseError{{message: "parsing form", inner: []error{err}}},
					},
				}
			}
			return o.Parse(o.readForm(req.Form), opts...)
		}
		return &objectParseResult{
			parseResult: parseResult[map[string]*parseResult[any]]{
				errors: []*parseError{{message: "unsupported content type"}},
			},
		}
	}
}

func (o *objectValidator) readBody(body io.ReadCloser, size int) ([]byte, *parseError) {
	buf := make([]byte, size)
	numRead, err := io.ReadFull(body, buf)
	if err != nil {
		return nil, &parseError{message: "reading request body", inner: []error{err}}
	}
	if numRead != size {
		return nil, &parseError{message: "request body size mismatch"}
	}
	return buf, nil
}

func (o *objectValidator) readForm(form url.Values) map[string]interface{} {
	output := make(map[string]interface{})
	for k := range form {
		output[k] = form.Get(k)
	}
	return output
}

func (o *objectValidator) String(name string, opts ...any) *objectValidator {
	fv := validatorWrapperFactory[string](opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Int(name string, opts ...any) *objectValidator {
	fv := validatorWrapperFactory[int](opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Int16(name string, opts ...any) *objectValidator {
	fv := validatorWrapperFactory[int16](opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Int32(name string, opts ...any) *objectValidator {
	fv := validatorWrapperFactory[int32](opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Int64(name string, opts ...any) *objectValidator {
	fv := validatorWrapperFactory[int64](opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Uint(name string, opts ...any) *objectValidator {
	fv := validatorWrapperFactory[uint](opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Uint16(name string, opts ...any) *objectValidator {
	fv := validatorWrapperFactory[uint16](opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Uint32(name string, opts ...any) *objectValidator {
	fv := validatorWrapperFactory[uint32](opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Uint64(name string, opts ...any) *objectValidator {
	fv := validatorWrapperFactory[uint64](opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Float32(name string, opts ...any) *objectValidator {
	fv := validatorWrapperFactory[float32](opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Float64(name string, opts ...any) *objectValidator {
	fv := validatorWrapperFactory[float64](opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Time(name string, opts ...any) *objectValidator {
	fv := validatorWrapperFactory[time.Time](opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) UUID(name string, opts ...any) *objectValidator {
	fv := validatorWrapperFactory[uuid.UUID](opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Object(name string, opts ...any) *objectValidator {
	fv := Object(opts...)
	wrapper := &objectValidatorWrapper{validator: fv}
	o.validators[name] = wrapper
	return o
}

type validatorWrapper[T any] struct {
	validator genericValidator[T]
}

func parseOptWrapper[T any](fn parseOpt[interface{}]) parseOpt[T] {
	return func(val T) *parseError {
		res := fn(val)
		return res
	}
}

func (v *validatorWrapper[T]) Parse(val any, opts ...parseOpt[interface{}]) genericParseResult[interface{}] {
	wrappedOpts := make([]parseOpt[T], len(opts))
	for i, opt := range opts {
		wrappedOpts[i] = parseOptWrapper[T](opt)
	}
	res := v.validator.Parse(val, wrappedOpts...)
	wrappedRes := &parseResult[interface{}]{valid: res.Valid(), value: res.Get(), errors: res.Errors()}
	return wrappedRes
}

func (v *validatorWrapper[T]) Error() error {
	return v.validator.Error()
}

func (v *validatorWrapper[T]) Type() reflect.Type {
	return v.validator.Type()
}

func validatorWrapperFactory[T any](opts ...any) genericValidator[interface{}] {
	return &validatorWrapper[T]{validator: newGenerator[T](opts...)}
}

type objectValidatorWrapper struct {
	validator *objectValidator
}

func (v *objectValidatorWrapper) Parse(val any, opts ...parseOpt[interface{}]) genericParseResult[interface{}] {
	res := v.validator.Parse(val, opts...)
	wrappedRes := &parseResult[interface{}]{valid: res.Valid(), value: res.Get(), errors: res.Errors()}
	return wrappedRes
}

func (v *objectValidatorWrapper) Error() error {
	return v.validator.Error()
}

func (v *objectValidatorWrapper) Type() reflect.Type {
	return v.validator.Type()
}

func WithMaxBodySize(size int64) objectValidatorOpt {
	return func(o *objectValidator) error {
		o.maxBodySize = size
		return nil
	}
}

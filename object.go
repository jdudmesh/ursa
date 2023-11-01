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
)

type objectValidatorOpt func(o *objectValidator) error
type objectFieldOpt func(opts ...validatorOpt[interface{}]) genericValidator
type objectValidatorParseOpt func(v *objectValidator, val any) error
type objectValidatorResult map[string]*parseResult[interface{}]

type objectValidator struct {
	validators  map[string]genericValidator
	maxBodySize int64
}

type objectParseResult struct {
	parseResult[objectValidatorResult]
}

func Object(opts ...any) *objectValidator {
	v := &objectValidator{
		validators:  make(map[string]genericValidator),
		maxBodySize: 1024 * 1024 * 10,
	}
	for _, opt := range opts {
		switch opt := opt.(type) {
		case objectValidatorOpt:
			opt(v)
		}
	}
	return v
}

func (o *objectValidator) Parse(val any, opts ...ParseOpt) ParseResult {
	switch val := val.(type) {
	case []byte:
		return o.parseJSON(val)
	case *http.Request:
		return o.parseRequest(val)
	}

	parseRes := &objectParseResult{
		parseResult: parseResult[objectValidatorResult]{
			valid:  true,
			value:  make(objectValidatorResult),
			errors: make([]ParseError, 0),
		},
	}

	for name, validator := range o.validators {
		fieldVal, err := o.extract(val, name)
		if err != nil {
			parseRes.value[name] = &parseResult[interface{}]{errors: []ParseError{&parseError{message: "failed to extract value", inner: []error{err}}}}
			parseRes.valid = false
			continue
		}

		fieldResult := validator.Parse(fieldVal)
		if !fieldResult.Valid() {
			parseRes.valid = false
		}
		parseRes.value[name] = &parseResult[interface{}]{valid: fieldResult.Valid(), value: fieldResult.Value(), errors: fieldResult.Errors()}
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

func (o *objectValidator) parseJSON(val []byte, opts ...ParseOpt) ParseResult {
	unpacked := make(map[string]interface{})
	err := json.Unmarshal(val, &unpacked)
	if err != nil {
		return &parseResult[interface{}]{
			valid:  false,
			errors: []ParseError{&parseError{message: "unmarshalling JSON value", inner: []error{err}}},
		}
	}
	return o.Parse(unpacked, opts...)
}

func (o *objectValidator) parseRequest(req *http.Request, opts ...ParseOpt) ParseResult {
	contentType := strings.TrimSpace(strings.Split(req.Header.Get("Content-Type"), ";")[0])

	body := req.Body
	if body != nil {
		defer body.Close()
	}

	numBytes := req.ContentLength
	if numBytes > o.maxBodySize {
		return &parseResult[interface{}]{errors: []ParseError{&parseError{message: "request body too large"}}}
	}

	switch contentType {
	case "application/json":
		buf, err := o.readBody(body, int(numBytes))
		if err != nil {
			return &parseResult[interface{}]{errors: []ParseError{err}}
		}
		return o.parseJSON(buf, opts...)

	case "application/x-www-form-urlencoded":
		err := req.ParseForm()
		if err != nil {
			return &parseResult[interface{}]{errors: []ParseError{&parseError{message: "parsing form", inner: []error{err}}}}
		}
		return o.Parse(o.readForm(req.Form), opts...)

	case "multipart/form-data":
		err := req.ParseMultipartForm(o.maxBodySize)
		if err != nil {
			return &parseResult[interface{}]{errors: []ParseError{&parseError{message: "parsing multipart form", inner: []error{err}}}}
		}
		return o.Parse(o.readForm(req.Form), opts...)

	default:
		if req.Method == "GET" {
			err := req.ParseForm()
			if err != nil {
				return &parseResult[interface{}]{errors: []ParseError{&parseError{message: "parsing form", inner: []error{err}}}}
			}
			return o.Parse(o.readForm(req.Form), opts...)
		}
		return &parseResult[interface{}]{errors: []ParseError{&parseError{message: "unsupported content type"}}}
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
	fv := String(opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Number(name string, gen numberValidatorGenerator, opts ...any) *objectValidator {
	fv := Number(gen, opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Time(name string, opts ...any) *objectValidator {
	fv := Time(opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) UUID(name string, opts ...any) *objectValidator {
	fv := UUID(opts...)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Object(name string, opts ...any) *objectValidator {
	fv := Object(opts...)
	o.validators[name] = fv
	return o
}

// func Object(opts ...any) *ursaObject {
// 	u := &ursaObject{
// 		validators:  map[string]UrsaObjectOpt{},
// 		maxBodySize: 1024 * 1024 * 10,
// 	}
// 	for _, opt := range opts {
// 		switch opt := opt.(type) {
// 		case UrsaObjectOpt:
// 			opt(u, nil)
// 		case UrsaObjectFieldDefiner:
// 			name, fn := opt()
// 			u.validators[name] = fn
// 		}
// 	}
// 	return u
// }

func WithMaxBodySize(size int64) objectValidatorOpt {
	return func(o *objectValidator) error {
		o.maxBodySize = size
		return nil
	}
}

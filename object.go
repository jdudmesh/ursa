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

type UrsaObjectOpt func(u *ursaObject, val any) *parseResult[interface{}]
type UrsaObjectFieldDefiner func() (string, UrsaObjectOpt)

var ErrFieldAlreadyDefined = parseError{message: "field already defined"}
var ErrUnsupportedContentType = parseError{message: "unsupported content type"}

type objectParseValue map[string]*parseResult[interface{}]

type ursaObject struct {
	validators  map[string]UrsaObjectOpt
	maxBodySize int64
}

func (u *ursaObject) Parse(val any, opts ...ParseOpt) ParseResult {
	switch val := val.(type) {
	case []byte:
		return u.parseJSON(val)
	case *http.Request:
		return u.parseRequest(val)
	}

	res := &parseResult[objectParseValue]{
		valid: true,
		value: make(objectParseValue),
	}

	for _, opt := range opts {
		opt(res)
	}

	for name, validator := range u.validators {
		fieldResult := validator(u, val)
		res.value[name] = fieldResult
	}

	return res
}

func (u *ursaObject) parseJSON(val []byte) ParseResult {
	unpacked := make(map[string]interface{})
	err := json.Unmarshal(val, &unpacked)
	if err != nil {
		return &parseResult[interface{}]{
			valid:  false,
			errors: []ParseError{&parseError{message: "unmarshalling JSON value", inner: []error{err}}},
		}
	}
	return u.Parse(unpacked)
}

func (u *ursaObject) parseRequest(req *http.Request) ParseResult {
	contentType := strings.TrimSpace(strings.Split(req.Header.Get("Content-Type"), ";")[0])

	body := req.Body
	if body != nil {
		defer body.Close()
	}

	numBytes := req.ContentLength
	if numBytes > u.maxBodySize {
		return &parseResult[interface{}]{errors: []ParseError{&parseError{message: "request body too large"}}}
	}

	switch contentType {
	case "application/json":
		buf, err := u.readBody(body, int(numBytes))
		if err != nil {
			return &parseResult[interface{}]{errors: []ParseError{err}}
		}
		return u.parseJSON(buf)
	case "application/x-www-form-urlencoded":
		err := req.ParseForm()
		if err != nil {
			return &parseResult[interface{}]{errors: []ParseError{&parseError{message: "parsing form", inner: []error{err}}}}
		}
		return u.Parse(u.readForm(req.Form))
	case "multipart/form-data":
		err := req.ParseMultipartForm(u.maxBodySize)
		if err != nil {
			return &parseResult[interface{}]{errors: []ParseError{&parseError{message: "parsing multipart form", inner: []error{err}}}}
		}
		return u.Parse(u.readForm(req.Form))
	default:
		if req.Method == "GET" {
			err := req.ParseForm()
			if err != nil {
				return &parseResult[interface{}]{errors: []ParseError{&parseError{message: "parsing form", inner: []error{err}}}}
			}
			return u.Parse(u.readForm(req.Form))
		}
		return &parseResult[interface{}]{errors: []ParseError{&parseError{message: "unsupported content type"}}}
	}
}

func (u *ursaObject) readBody(body io.ReadCloser, size int) ([]byte, *parseError) {
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

func (u *ursaObject) readForm(form url.Values) map[string]interface{} {
	output := make(map[string]interface{})
	for k := range form {
		output[k] = form.Get(k)
	}
	return output
}

func (u *ursaObject) String(name string, opts ...any) *ursaObject {
	n, opt := Field(name, String(opts...))()
	u.validators[n] = opt
	return u
}

func (u *ursaObject) Number(name string, ntype numberType, opts ...any) *ursaObject {
	n, opt := Field(name, Number(ntype, opts...))()
	u.validators[n] = opt
	return u
}

func (u *ursaObject) Date(name string, opts ...any) *ursaObject {
	n, opt := Field(name, Date(opts))()
	u.validators[n] = opt
	return u
}

func (u *ursaObject) UUID(name string, opts ...any) *ursaObject {
	n, opt := Field(name, UUID(opts...))()
	u.validators[n] = opt
	return u
}

func Object(opts ...any) *ursaObject {
	u := &ursaObject{
		validators:  map[string]UrsaObjectOpt{},
		maxBodySize: 1024 * 1024 * 10,
	}
	for _, opt := range opts {
		switch opt := opt.(type) {
		case UrsaObjectOpt:
			opt(u, nil)
		case UrsaObjectFieldDefiner:
			name, fn := opt()
			u.validators[name] = fn
		}
	}
	return u
}

func Field(name string, validator ursaEntity) UrsaObjectFieldDefiner {
	return func() (string, UrsaObjectOpt) {
		return name, func(u *ursaObject, val any) *parseResult[interface{}] {
			s := reflect.ValueOf(val)
			switch {
			case s.Kind() == reflect.Ptr:
				deref := reflect.Indirect(s)
				if !(deref.Kind() == reflect.Struct || deref.Kind() == reflect.Map) {
					return &parseResult[interface{}]{valid: false, errors: []ParseError{UrsaInvalidTypeError}}
				}
				s = deref
			case !(s.Kind() == reflect.Struct || s.Kind() == reflect.Map):
				return &parseResult[interface{}]{valid: false, errors: []ParseError{UrsaInvalidTypeError}}
			}

			var value reflect.Value
			switch s.Kind() {
			case reflect.Struct:
				value = s.FieldByName(name)
			case reflect.Map:
				value = s.MapIndex(reflect.ValueOf(name))
			}

			if !value.IsValid() {
				if d := validator.getDefault(); d != nil {
					return &parseResult[interface{}]{valid: true, value: d}
				}
				if validator.getRequired() {
					return &parseResult[interface{}]{valid: false, errors: []ParseError{UrsaInvalidTypeError}}
				}
			}

			inner := validator.Parse(value.Interface())

			return &parseResult[interface{}]{valid: inner.Valid(), value: inner.Value(), errors: inner.Errors()}
		}
	}
}

func WithMaxBodySize(size int64) UrsaObjectOpt {
	return func(u *ursaObject, val any) *parseResult[interface{}] {
		u.maxBodySize = size
		return nil
	}
}

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
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ObjectParseResult interface {
	genericParseResult[map[string]*parseResult[any]]
	IsFieldValid(field string) bool
	GetError(field string) string
	GetField(field string) *parseResult[any]
	GetString(field string) string
	GetInt(field string) int
	GetBool(field string) bool
}

type File struct {
	Header *multipart.FileHeader
}

type objectValidatorOpt func(o *objectValidator) error
type objectMultipartFileHandler func(name string, file *multipart.FileHeader) error
type objectRefinerFunc func(res ObjectParseResult)

type objectValidator struct {
	fields      []string // use this to preserve order
	validators  map[string]genericValidator[any]
	refiners    []objectRefinerFunc
	maxBodySize int64
	err         error
}

type objectParseResult struct {
	parseResult[map[string]*parseResult[any]]
	fields []string // use this to preserve order
}

func (r *objectParseResult) set(val any) {
	if val, ok := val.(map[string]*parseResult[any]); ok {
		r.value = val
	}
}

func (o *objectParseResult) GetField(field string) *parseResult[any] {
	return o.value[field]
}

func (o *objectParseResult) GetString(field string) string {
	val := o.value[field].Get()
	if val == nil {
		return ""
	}
	vo := reflect.ValueOf(val)
	switch vo.Kind() {
	case reflect.String:
		return vo.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatInt(vo.Int(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(vo.Float(), 'f', -1, 64)
	}
	return o.value[field].Get().(string)
}

func (o *objectParseResult) GetInt(field string) int {
	val := o.value[field].Get()
	if val == nil {
		return 0
	}
	vo := reflect.ValueOf(val)
	switch vo.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return int(vo.Int())
	case reflect.String:
		i, err := strconv.ParseInt(vo.String(), 10, 64)
		if err != nil {
			return 0
		}
		return int(i)
	}
	return 0
}

func (o *objectParseResult) GetBool(field string) bool {
	val := o.value[field].Get()
	if val == nil {
		return false
	}
	vo := reflect.ValueOf(val)
	switch vo.Kind() {
	case reflect.Bool:
		return vo.Bool()
	case reflect.String:
		if len(vo.String()) == 0 {
			return false
		}
		b, err := strconv.ParseBool(vo.String())
		if err != nil {
			return false
		}
		return b
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return vo.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return vo.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return vo.Float() != 0
	default:
		return false
	}
}

func (r *objectParseResult) GetError(field string) string {
	if _, ok := r.value[field]; !ok {
		return ""
	}
	if r.value[field].valid {
		return ""
	}
	errors := make([]string, len(r.value[field].errors))
	for i, err := range r.value[field].errors {
		errors[i] = err.Error()
	}
	return strings.Join(errors, ", ")
}

func (r *objectParseResult) IsFieldValid(field string) bool {
	if _, ok := r.value[field]; !ok {
		return false
	}
	return r.value[field].Valid()
}

func (r *objectParseResult) Unmarshal(target any) error {
	if !r.valid {
		return errors.New("cannot unmarshal invalid value")
	}

	vo := reflect.ValueOf(target)
	if !vo.IsValid() {
		return errors.New("invalid target")
	}

	if vo.Kind() == reflect.Ptr {
		vo = reflect.Indirect(vo)
	}

	switch vo.Kind() {
	case reflect.Struct:
		return r.unmarshalToStruct(target)
	case reflect.Map:
		return r.unmarshalToMap(target.(map[string]interface{}))
	default:
		return errors.New("invalid target")
	}
}

func (r *objectParseResult) unmarshalToStruct(target interface{}) error {
	vo := reflect.Indirect(reflect.ValueOf(target))
	to := vo.Type()

	for i := 0; i < vo.NumField(); i++ {
		field := vo.Field(i)
		if field.CanSet() {
			fieldName := to.Field(i).Name

			sf, _ := reflect.TypeOf(target).Elem().FieldByName(fieldName)
			for _, sourceFieldName := range extractTags(fieldName, sf) {
				if _, ok := r.value[sourceFieldName]; !ok {
					continue
				}
				if field.Kind() == reflect.Struct {
					if res, ok := r.value[sourceFieldName].Get().(*objectParseResult); ok {
						if err := res.Unmarshal(field.Addr().Interface()); err != nil {
							return err
						}
					}
					continue
				}
				field.Set(reflect.ValueOf(r.GetField(sourceFieldName).Get()))
				break
			}
		}
	}

	return nil
}

func (r *objectParseResult) unmarshalToMap(target map[string]interface{}) error {
	for k, v := range r.value {
		target[k] = v.Get()
	}
	return nil
}

func Object(opts ...any) *objectValidator {
	v := &objectValidator{
		fields:      make([]string, 0),
		validators:  make(map[string]genericValidator[interface{}]),
		refiners:    make([]objectRefinerFunc, 0),
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
		fields: o.fields,
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

	// run each validator in order
	for _, name := range o.fields {
		validator := o.validators[name]

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

	for _, refiner := range o.refiners {
		refiner(parseRes)
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

		formData := o.readForm(req.Form)
		for name, fileHeaders := range req.MultipartForm.File {
			if _, ok := formData[name]; !ok {
				formData[name] = make([]File, 0, len(fileHeaders))
			}
			for _, fileHeader := range fileHeaders {
				formData[name] = append(formData[name].([]File), File{Header: fileHeader})
			}
		}

		return o.Parse(formData, opts...)

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
	fv := &validatorWrapper[string]{validator: validatorFactory[string](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Int(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[int]{validator: numericValidatorFactory[int](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Int16(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[int16]{validator: numericValidatorFactory[int16](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Int32(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[int32]{validator: numericValidatorFactory[int32](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Int64(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[int64]{validator: numericValidatorFactory[int64](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Uint(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[uint]{validator: numericValidatorFactory[uint](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Uint16(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[uint16]{validator: numericValidatorFactory[uint16](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Uint32(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[uint32]{validator: numericValidatorFactory[uint32](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Uint64(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[uint64]{validator: numericValidatorFactory[uint64](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Float32(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[float32]{validator: numericValidatorFactory[float32](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Float64(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[float64]{validator: numericValidatorFactory[float64](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Time(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[time.Time]{validator: validatorFactory[time.Time](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) UUID(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[uuid.UUID]{validator: validatorFactory[uuid.UUID](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Bool(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[bool]{validator: validatorFactory[bool](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Object(name string, opts ...any) *objectValidator {
	fv := Object(opts...)
	wrapper := &objectValidatorWrapper{validator: fv}
	o.fields = append(o.fields, name)
	o.validators[name] = wrapper
	return o
}

func (o *objectValidator) File(name string, opts ...any) *objectValidator {
	fv := &validatorWrapper[[]File]{validator: validatorFactory[[]File](opts...)}
	o.fields = append(o.fields, name)
	o.validators[name] = fv
	return o
}

func (o *objectValidator) Refine(fn objectRefinerFunc) *objectValidator {
	o.refiners = append(o.refiners, fn)
	return o
}

func (o *objectValidator) From(valid bool, state any) (*objectParseResult, error) {
	res := &objectParseResult{
		parseResult: parseResult[map[string]*parseResult[any]]{
			valid:  valid,
			errors: make([]*parseError, 0),
			value:  make(map[string]*parseResult[any]),
		},
	}
	vo := reflect.ValueOf(state)
	if !vo.IsValid() {
		return nil, errors.New("invalid state")
	}

	if vo.Kind() == reflect.Ptr {
		if vo.IsNil() {
			return nil, errors.New("invalid state: nil pointer")
		}
		vo = reflect.Indirect(vo)
	}

	switch vo.Kind() {
	case reflect.Struct:
		o.resultFromStruct(valid, state, res)
	case reflect.Map:
		o.resultFromMap(valid, state, res)
	}

	return res, nil
}

func (o *objectValidator) resultFromMap(valid bool, state any, res *objectParseResult) error {
	vo := reflect.ValueOf(state)
	for _, key := range vo.MapKeys() {
		if ix := slices.Index(o.fields, key.String()); ix < 0 {
			continue
		}
		res.value[key.String()] = &parseResult[any]{value: vo.MapIndex(key).Interface(), valid: valid}
	}
	return nil
}

func (o *objectValidator) resultFromStruct(valid bool, state any, res *objectParseResult) error {
	vo := reflect.Indirect(reflect.ValueOf(state))
	to := vo.Type()

	for i := 0; i < vo.NumField(); i++ {
		field := vo.Field(i)
		fieldName := to.Field(i).Name

		sf, _ := reflect.TypeOf(state).Elem().FieldByName(fieldName)
		for _, sourceFieldName := range extractTags(fieldName, sf) {
			if ix := slices.Index(o.fields, sourceFieldName); ix < 0 {
				continue
			}
			res.value[sourceFieldName] = &parseResult[any]{value: field.Interface(), valid: valid}
			break
		}
	}

	return nil
}

type validatorWrapper[T any] struct {
	validator genericValidator[T]
}

func parseOptWrapper[T any](fn parseOpt[interface{}]) parseOpt[T] {
	return func(val *T) *parseError {
		var v interface{}
		v = val
		res := fn(&v)
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

func MaxFileCount(count int, message ...string) parseOpt[[]File] {
	return func(val *[]File) *parseError {
		if val == nil {
			return nil
		}
		if len(*val) > count {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "too many files"}
		}
		return nil
	}
}

func MaxFileSize(size int, message ...string) parseOpt[[]File] {
	return func(val *[]File) *parseError {
		if val == nil {
			return nil
		}
		files := *val
		for _, file := range files {
			if file.Header.Size > int64(size) {
				if len(message) > 0 {
					return &parseError{message: message[0]}
				}
				return &parseError{message: "too many files"}
			}
		}
		return nil
	}
}

package ursa

import "reflect"

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

// type parseOpt[T any] func(res *genericParseResult[T]) error
type parseOpt[T any] func(val T) *parseError
type transformer[T any] func(val any) (T, error)

type validator[T any] struct {
	transformerFn transformer[T]
	options       []parseOpt[T]
	defaultValue  *T
	required      bool
	err           error
}

type genericValidator[T any] interface {
	Parse(val any, opts ...parseOpt[T]) genericParseResult[T]
	Error() error
	Type() reflect.Type
}

type genericParseResult[T any] interface {
	Valid() bool
	Errors() []*parseError
	Get() T
	Set(val T)
}

type genericValidatorOptReceiver interface {
	hasTransformer() bool
	setTransformer(fn transformer[any])
	setDefault(val any)
	setRequired()
}

type validatorWithOpts[T any] interface {
	genericValidator[T]
	genericValidatorOptReceiver
}

type genericValidatorOpt func(v genericValidatorOptReceiver) error

type parseResult[T any] struct {
	valid  bool
	value  T
	errors []*parseError
}

type parseError struct {
	message string
	inner   []error
}

func (r *parseResult[T]) Valid() bool {
	return r.valid
}

func (r *parseResult[T]) Errors() []*parseError {
	return r.errors
}

func (r *parseResult[T]) Get() T {
	return r.value
}

func (r *parseResult[T]) Set(val T) {
	r.value = val
}

func (e *parseError) Inner() []error {
	return e.inner
}

func (e *parseError) Error() string {
	return e.message
}

var InvalidTypeError = &parseError{
	message: "invalid type",
}

var InvalidValueError = &parseError{
	message: "invalid value",
}

var InvalidValidatorStateError = &parseError{
	message: "invalid type",
}

var RequiredPropertyMissingError = &parseError{
	message: "missing required property",
}

var MissingTransformerError = &parseError{
	message: "missing property transformer",
}

func (v *validator[T]) Parse(val any, opts ...parseOpt[T]) genericParseResult[T] {
	res := &parseResult[T]{}
	if v.err != nil {
		res.errors = []*parseError{InvalidValidatorStateError}
		return res
	}

	typedVal, err := v.convert(val)
	if err != nil {
		res.errors = []*parseError{err}
		return res
	}

	if typedVal == nil {
		res.valid = true
		return res
	}

	for _, opt := range v.options {
		err := opt(*typedVal)
		if err != nil {
			res.errors = append(res.errors, err)
		}
	}

	res.valid = len(res.errors) == 0
	if res.valid {
		res.Set(*typedVal)
	}

	return res
}

func (v *validator[T]) convert(val any) (*T, *parseError) {
	var typedVal T
	var err error

	vo := reflect.ValueOf(val)
	switch vo.Kind() {
	case reflect.Ptr:
		vo = vo.Elem()
	case reflect.Invalid:
		if val == nil {
			if v.defaultValue != nil {
				return v.convert(v.defaultValue)
			}
			if v.required {
				return nil, RequiredPropertyMissingError
			}
			return nil, nil
		}
	}

	if vo.Kind() != reflect.TypeOf(typedVal).Kind() {
		if v.transformerFn == nil {
			if !isNumeric(val) && isNumeric(typedVal) {
				val, err = coerceToNumber[float64](val)
				if err != nil {
					return nil, InvalidTypeError
				}
			}
		} else {
			val, err = v.transformerFn(val)
			if err != nil {
				return nil, &parseError{message: "transformer error", inner: []error{err}}
			}
		}

		if reflect.TypeOf(val).ConvertibleTo(reflect.TypeOf(typedVal)) {
			if v, ok := reflect.ValueOf(val).Convert(reflect.TypeOf(typedVal)).Interface().(T); ok {
				typedVal = v
			} else {
				return nil, InvalidTypeError
			}
		} else {
			return nil, MissingTransformerError
		}

	} else {
		typedVal = vo.Interface().(T)
	}

	return &typedVal, nil
}

func (b *validator[T]) setTransformer(fn transformer[any]) {
	b.transformerFn = func(val any) (T, error) {
		var zero T

		val, err := fn(val)
		if err != nil {
			return zero, InvalidValueError
		}

		if !reflect.TypeOf(val).ConvertibleTo(reflect.TypeOf(zero)) {
			return zero, InvalidTypeError
		}
		vo := reflect.ValueOf(val)
		val = vo.Convert(reflect.TypeOf(zero)).Interface().(T)
		return val.(T), err
	}
}

func (b *validator[T]) hasTransformer() bool {
	return b.transformerFn != nil
}

func (b *validator[T]) setDefault(val any) {
	var zero T
	if !reflect.TypeOf(val).ConvertibleTo(reflect.TypeOf(zero)) {
		b.err = InvalidTypeError
		return
	}

	vo := reflect.ValueOf(val)
	if vo.Kind() == reflect.Ptr {
		vo = vo.Elem()
	}

	if vo.Kind() == reflect.Invalid {
		b.err = InvalidTypeError
		return
	}

	d := vo.Convert(reflect.TypeOf(zero)).Interface().(T)

	b.defaultValue = &d
}

func (b *validator[T]) setRequired() {
	b.required = true
}

func (b *validator[T]) Error() error {
	return b.err
}

func (b *validator[T]) Type() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

func WithDefault(val any) genericValidatorOpt {
	return func(v genericValidatorOptReceiver) error {
		v.setDefault(val)
		return nil
	}
}

func Required() genericValidatorOpt {
	return func(v genericValidatorOptReceiver) error {
		v.setRequired()
		return nil
	}
}

func newGenerator[T any](opts ...any) validatorWithOpts[T] {
	v := &validator[T]{
		options: make([]parseOpt[T], 0, len(opts)),
	}

	for _, opt := range opts {
		switch opt := opt.(type) {
		case parseOpt[T]:
			v.options = append(v.options, opt)
		case genericValidatorOpt:
			err := opt(v)
			if err != nil {
				v.err = err
			}
		}
	}

	return v
}

func isNumeric(i interface{}) bool {
	switch reflect.TypeOf(i).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return true
	default:
		return false
	}
}

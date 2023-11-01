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
	"cmp"
	"math"
	"reflect"
	"strconv"

	"golang.org/x/exp/constraints"
)

type UrsaNumberOpt func(u *ursaNumber, val any) *parseError

type ursaNumber struct {
	typeConstraint numberType
	options        []UrsaNumberOpt
	defaultValue   any
	required       bool
}

type NumberType interface {
	constraints.Integer | constraints.Float
	comparable
	cmp.Ordered
}

type numberType interface {
	IsValid(val any) bool
	NewParseResult(value any, errs ...ParseError) ParseResult
}

type numberTypeConstraint[T NumberType] struct{}

func (u numberTypeConstraint[T]) IsValid(val any) bool {
	var zero T
	return reflect.TypeOf(val).ConvertibleTo(reflect.TypeOf(zero))
}

func (u numberTypeConstraint[T]) NewParseResult(value any, errs ...ParseError) ParseResult {
	var zero T
	v := reflect.ValueOf(value).Convert(reflect.TypeOf(zero))
	res := &parseResult[T]{
		value:  v.Interface().(T),
		errors: errs,
		valid:  len(errs) == 0,
	}
	return res
}

func Int() numberType {
	return &numberTypeConstraint[int]{}
}

func Int16() numberType {
	return &numberTypeConstraint[int16]{}
}

func Int32() numberType {
	return &numberTypeConstraint[int32]{}
}

func Int64() numberType {
	return &numberTypeConstraint[int64]{}
}

func Uint() numberType {
	return &numberTypeConstraint[uint]{}
}

func Uint16() numberType {
	return &numberTypeConstraint[uint16]{}
}

func Uint32() numberType {
	return &numberTypeConstraint[uint32]{}
}

func Uint64() numberType {
	return &numberTypeConstraint[uint64]{}
}

func Float32() numberType {
	return &numberTypeConstraint[float32]{}
}

func Float64() numberType {
	return &numberTypeConstraint[float64]{}
}

func Number(constraint numberType, opts ...any) *ursaNumber {
	u := &ursaNumber{
		typeConstraint: constraint,
		options:        make([]UrsaNumberOpt, 0, len(opts)),
	}
	for _, opt := range opts {
		switch opt := opt.(type) {
		case UrsaNumberOpt:
			u.options = append(u.options, opt)
		case EntityOpt:
			opt(u)
		}
	}
	return u
}

func (u *ursaNumber) Parse(val any, opts ...ParseOpt) ParseResult {
	if v, ok := val.(string); ok {
		floatVal, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return u.typeConstraint.NewParseResult(0, UrsaInvalidTypeError)
		}
		return u.Parse(floatVal)
	}

	if !u.typeConstraint.IsValid(val) {
		return u.typeConstraint.NewParseResult(0, UrsaInvalidTypeError)
	}

	errs := make([]ParseError, 0)
	for _, opt := range u.options {
		err := opt(u, val)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return u.typeConstraint.NewParseResult(val, errs...)
}

func (u *ursaNumber) setDefault(val any) {
	u.defaultValue = val
}

func (u *ursaNumber) getDefault() any {
	return u.defaultValue
}

func Min(min float64, message ...string) UrsaNumberOpt {
	return func(u *ursaNumber, val any) *parseError {
		v := reflect.ValueOf(val).Convert(reflect.TypeOf(0.0)).Interface().(float64)
		if v < min {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "number too small"}
		}
		return nil
	}
}

func Max(max float64, message ...string) UrsaNumberOpt {
	return func(u *ursaNumber, val any) *parseError {
		v := reflect.ValueOf(val).Convert(reflect.TypeOf(0.0)).Interface().(float64)
		if v > max {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "number too large"}
		}
		return nil
	}
}

func NonZero(message ...string) UrsaNumberOpt {
	return func(u *ursaNumber, val any) *parseError {
		v := reflect.ValueOf(val).Convert(reflect.TypeOf(0.0)).Interface().(float64)
		if v == 0 {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "number is uero"}
		}
		return nil
	}
}

func MustBeInteger(message ...string) UrsaNumberOpt {
	return func(u *ursaNumber, val any) *parseError {
		v := reflect.ValueOf(val).Convert(reflect.TypeOf(0.0)).Interface().(float64)
		if v != math.Floor(v) {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "number is not integer"}
		}
		return nil
	}
}

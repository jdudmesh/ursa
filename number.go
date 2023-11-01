package ursa

import (
	"math"
	"reflect"
	"strconv"

	"golang.org/x/exp/constraints"
)

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

type numberValidatorOpt = validatorOpt[float64]
type numberValidatorGenerator func(opts ...any) genericValidator

func Number(generatorFn numberValidatorGenerator, opts ...any) genericValidator {
	v := generatorFn(opts...)
	return v
}

func optWrapper[T any](fn numberValidatorOpt) validatorOpt[T] {
	return func(val T) *parseError {
		n := reflect.ValueOf(val).Convert(reflect.TypeOf(0.0)).Interface().(float64)
		return fn(n)
	}
}

func numberValidator[T constraints.Integer | constraints.Signed | constraints.Unsigned | constraints.Float]() numberValidatorGenerator {
	return func(opts ...any) genericValidator {
		wrappedOpts := make([]any, len(opts))
		for i, opt := range opts {
			if fn, ok := opt.(numberValidatorOpt); ok {
				wrappedOpts[i] = optWrapper[T](fn)
			} else {
				wrappedOpts[i] = opt
			}
		}
		return newGenerator[T](wrappedOpts...)
	}
}

func Int() numberValidatorGenerator {
	return numberValidator[int]()
}

func Int16() numberValidatorGenerator {
	return numberValidator[int16]()
}

func Int32() numberValidatorGenerator {
	return numberValidator[int32]()
}

func Int64() numberValidatorGenerator {
	return numberValidator[int64]()
}

func Uint() numberValidatorGenerator {
	return numberValidator[uint]()
}

func Uint16() numberValidatorGenerator {
	return numberValidator[uint16]()
}

func Uint32() numberValidatorGenerator {
	return numberValidator[uint32]()
}

func Uint64() numberValidatorGenerator {
	return numberValidator[uint64]()
}

func Float32() numberValidatorGenerator {
	return numberValidator[float32]()
}

func Float64() numberValidatorGenerator {
	return numberValidator[float64]()
}

func Min(min float64, message ...string) numberValidatorOpt {
	return func(val float64) *parseError {
		if val < min {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "number too small"}
		}
		return nil
	}
}

func Max(max float64, message ...string) numberValidatorOpt {
	return func(val float64) *parseError {
		if val > max {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "number too large"}
		}
		return nil
	}
}

func NonZero(message ...string) numberValidatorOpt {
	return func(val float64) *parseError {
		if val == 0 {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "number is uero"}
		}
		return nil
	}
}

func MustBeInteger(message ...string) numberValidatorOpt {
	return func(val float64) *parseError {
		if val != math.Floor(val) {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "number is not integer"}
		}
		return nil
	}
}

func coerceToNumber(val any) (float64, error) {
	vo := reflect.ValueOf(val)
	if vo.Kind() == reflect.Ptr {
		vo = vo.Elem()
		return coerceToNumber(vo.Interface())
	}
	if vo.Kind() != reflect.String {
		return 0, InvalidValueError
	}
	return strconv.ParseFloat(val.(string), 64)
}

func WithStringTransformer() genericValidatorOpt {
	return func(v genericValidatorOptReceiver) error {
		v.setTransformer(func(val any) (any, error) {
			return coerceToNumber(val)
		})
		return nil
	}
}

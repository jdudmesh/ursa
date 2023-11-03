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

type number interface {
	constraints.Integer | constraints.Float
}

type numberValidatorOpt func(val float64) *parseError

func optWrapper[T any](fn numberValidatorOpt) parseOpt[T] {
	return func(val T) *parseError {
		var zero float64
		zeroType := reflect.TypeOf(zero)
		if reflect.TypeOf(val).ConvertibleTo(zeroType) {
			n := reflect.ValueOf(val).Convert(zeroType).Interface().(float64)
			return fn(n)
		}
		return InvalidTypeError
	}
}

func numberValidator[T number](opts ...any) genericValidator[T] {
	wrappedOpts := make([]any, len(opts))
	for i, opt := range opts {
		if fn, ok := opt.(numberValidatorOpt); ok {
			wrappedOpts[i] = optWrapper[T](fn)
		} else {
			wrappedOpts[i] = opt
		}
	}

	v := newGenerator[T](wrappedOpts...)

	return v
}

func Int(opts ...any) genericValidator[int] {
	return numberValidator[int](opts...)
}

func Int16(opts ...any) genericValidator[int16] {
	return numberValidator[int16](opts...)
}

func Int32(opts ...any) genericValidator[int32] {
	return numberValidator[int32](opts...)
}

func Int64(opts ...any) genericValidator[int64] {
	return numberValidator[int64](opts...)
}

func UInt(opts ...any) genericValidator[uint] {
	return numberValidator[uint](opts...)
}

func UInt16(opts ...any) genericValidator[uint16] {
	return numberValidator[uint16](opts...)
}

func UInt32(opts ...any) genericValidator[uint32] {
	return numberValidator[uint32](opts...)
}

func UInt64(opts ...any) genericValidator[uint64] {
	return numberValidator[uint64](opts...)
}

func Float32(opts ...any) genericValidator[float32] {
	return numberValidator[float32](opts...)
}

func Float64(opts ...any) genericValidator[float64] {
	return numberValidator[float64](opts...)
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
			return &parseError{message: "number is zero"}
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

func coerceToNumber[T number](val any) (T, error) {
	vo := reflect.ValueOf(val)
	if vo.Kind() == reflect.Ptr {
		vo = vo.Elem()
		return coerceToNumber[T](vo.Interface())
	}
	if val, ok := val.(string); !ok {
		return 0, InvalidValueError
	} else {
		floatVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return T(0), InvalidValueError
		}
		return T(floatVal), nil
	}
}

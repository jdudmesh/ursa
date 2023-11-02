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
	"reflect"
	"time"
)

var ErrMissingDateParser = &parseError{message: "missing date parser"}

type timeValidatorOpt = parseOpt[time.Time]

func Time(opts ...any) genericValidator[time.Time] {
	return newGenerator[time.Time](opts...)
}

func coerceToTime(layout string, val any) (time.Time, error) {
	vo := reflect.ValueOf(val)
	if vo.Kind() == reflect.Ptr {
		vo = vo.Elem()
		return coerceToTime(layout, vo.Interface())
	}
	if vo.Kind() != reflect.String {
		return time.Time{}, InvalidValueError
	}
	return time.Parse(layout, val.(string))
}

func WithTimeFormat(layout string) genericValidatorOpt {
	return func(v genericValidatorOptReceiver) error {
		v.setTransformer(func(val any) (any, error) {
			return coerceToTime(layout, val)
		})
		return nil
	}
}

func NotBefore(datum time.Time, message ...string) timeValidatorOpt {
	return func(val time.Time) *parseError {
		if val.Before(datum) {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "date is too early"}
		}
		return nil
	}
}

func NotAfter(datum time.Time, message ...string) timeValidatorOpt {
	return func(val time.Time) *parseError {
		if val.After(datum) {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "date is too late"}
		}
		return nil
	}
}

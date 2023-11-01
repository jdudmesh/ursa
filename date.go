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
	"time"
)

var ErrMissingDateParser = &parseError{message: "missing date parser"}

type UrsaDateOpt func(u *ursaDate, val time.Time) *parseError
type UrsaDateParseFunc func(val string) (time.Time, error)

type ursaDate struct {
	parseFunc    UrsaDateParseFunc
	options      []UrsaDateOpt
	defaultValue time.Time
	required     bool
}

func (u *ursaDate) Parse(val any, opts ...ParseOpt) ParseResult {
	res := &parseResult[time.Time]{}
	var typedVal time.Time
	var err error

	switch val := val.(type) {
	case time.Time:
		typedVal = val
	case *time.Time:
		typedVal = *val
	case string:
		if u.parseFunc == nil {
			res.errors = []ParseError{ErrMissingDateParser}
			return res
		}
		typedVal, err = u.parseFunc(val)
		if err != nil {
			res.errors = []ParseError{&parseError{message: "invalid date", inner: []error{err}}}
			return res
		}
	default:
		res.errors = []ParseError{UrsaInvalidTypeError}
		return res
	}

	for _, opt := range u.options {
		err := opt(u, typedVal)
		if err != nil {
			res.errors = append(res.errors, err)
		}
	}

	res.valid = len(res.errors) == 0
	if res.valid {
		res.value = typedVal
	}

	return res
}

func (u *ursaDate) setDefault(val any) {
	u.defaultValue = val.(time.Time)
}

func (u *ursaDate) getDefault() any {
	return u.defaultValue
}

func (u *ursaDate) WithDateParser(fn UrsaDateParseFunc) {
	u.parseFunc = fn
}

func Date(opts ...any) *ursaDate {
	u := &ursaDate{
		options: make([]UrsaDateOpt, 0, len(opts)),
	}

	for _, opt := range opts {
		switch opt := opt.(type) {
		case UrsaDateOpt:
			u.options = append(u.options, opt)
		case UrsaDateParseFunc:
			u.parseFunc = opt
		case EntityOpt:
			opt(u)
		}
	}

	return u
}

func NotBefore(datum time.Time, message ...string) UrsaDateOpt {
	return func(u *ursaDate, val time.Time) *parseError {
		if val.Before(datum) {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "date is too early"}
		}
		return nil
	}
}

func NotAfter(datum time.Time, message ...string) UrsaDateOpt {
	return func(u *ursaDate, val time.Time) *parseError {
		if val.After(datum) {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "date is too late"}
		}
		return nil
	}
}

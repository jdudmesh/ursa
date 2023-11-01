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
	"net/mail"
	"regexp"
)

type UrsaStringOpt func(u *ursaString, val string) *parseError

type ursaString struct {
	options      []UrsaStringOpt
	defaultValue string
	required     bool
}

func (u *ursaString) Parse(val any, opts ...ParseOpt) ParseResult {
	res := &parseResult[string]{}

	if _, ok := val.(string); !ok {
		res.errors = []ParseError{UrsaInvalidTypeError}
		return res
	}

	typedVal := val.(string)
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

func (u *ursaString) setDefault(val any) {
	u.defaultValue = val.(string)
}

func (u *ursaString) getDefault() any {
	return u.defaultValue
}

func String(opts ...any) *ursaString {
	u := &ursaString{
		options: make([]UrsaStringOpt, 0, len(opts)),
	}
	for _, opt := range opts {
		switch opt := opt.(type) {
		case UrsaStringOpt:
			u.options = append(u.options, opt)
		case EntityOpt:
			opt(u)
		}
	}
	return u
}

func MinLength(min int, message ...string) UrsaStringOpt {
	return func(u *ursaString, val string) *parseError {
		if len(val) < min {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "string too short"}
		}
		return nil
	}
}

func MaxLength(max int, message ...string) UrsaStringOpt {
	return func(u *ursaString, val string) *parseError {
		if len(val) > max {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "string too long"}
		}
		return nil
	}
}

func Matches(patt string, message ...string) UrsaStringOpt {
	re, err := regexp.Compile(patt)
	return func(u *ursaString, val string) *parseError {
		if err != nil {
			return &parseError{message: "invalid regexp pattern", inner: []error{err}}
		}
		if !re.MatchString(val) {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "string does not match pattern"}
		}
		return nil
	}
}

func Email(patt string, message ...string) UrsaStringOpt {
	return func(u *ursaString, val string) *parseError {
		_, err := mail.ParseAddress(val)
		if err != nil {
			if len(message) > 0 {
				return &parseError{message: message[0], inner: []error{err}}
			}
			return &parseError{message: "invalid email address", inner: []error{err}}
		}
		return nil
	}
}

func Enum(values ...string) UrsaStringOpt {
	return func(u *ursaString, val string) *parseError {
		for _, v := range values {
			if v == val {
				return nil
			}
		}
		return &parseError{message: "value not found in enum", inner: []error{}}
	}
}

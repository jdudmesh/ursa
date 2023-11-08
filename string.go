package ursa

import (
	"net/mail"
	"regexp"
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

type stringValidatorOpt = parseOpt[string]

func String(opts ...any) genericValidator[string] {
	return validatorFactory[string](opts...)
}

func MinLength(min int, message ...string) stringValidatorOpt {
	return func(val *string) *parseError {
		if val == nil {
			return nil
		}
		if len(*val) < min {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "string too short"}
		}
		return nil
	}
}

func MaxLength(max int, message ...string) stringValidatorOpt {
	return func(val *string) *parseError {
		if val == nil {
			return nil
		}
		if len(*val) > max {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "string too long"}
		}
		return nil
	}
}

func Matches(patt string, message ...string) stringValidatorOpt {
	re, err := regexp.Compile(patt)
	return func(val *string) *parseError {
		if val == nil {
			return nil
		}
		if err != nil {
			return &parseError{message: "invalid regexp pattern", inner: []error{err}}
		}
		if !re.MatchString(*val) {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "string does not match pattern"}
		}
		return nil
	}
}

func Email(message ...string) stringValidatorOpt {
	return func(val *string) *parseError {
		if val == nil {
			return nil
		}
		_, err := mail.ParseAddress(*val)
		if err != nil {
			if len(message) > 0 {
				return &parseError{message: message[0], inner: []error{err}}
			}
			return &parseError{message: "invalid email address", inner: []error{err}}
		}
		return nil
	}
}

func Enum(values ...string) stringValidatorOpt {
	return func(val *string) *parseError {
		if val == nil {
			return nil
		}
		for _, v := range values {
			if v == *val {
				return nil
			}
		}
		return &parseError{message: "value not found in enum", inner: []error{}}
	}
}

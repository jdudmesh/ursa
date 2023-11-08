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

type boolValidatorOpt = parseOpt[bool]

func Bool(opts ...any) genericValidator[bool] {
	return validatorFactory[bool](opts...)
}

func True(message ...string) boolValidatorOpt {
	return func(val *bool) *parseError {
		if val == nil {
			return nil
		}
		if !*val {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "value should be true"}
		}
		return nil
	}
}

func False(message ...string) boolValidatorOpt {
	return func(val *bool) *parseError {
		if val == nil {
			return nil
		}
		if *val {
			if len(message) > 0 {
				return &parseError{message: message[0]}
			}
			return &parseError{message: "value should be false"}
		}
		return nil
	}
}

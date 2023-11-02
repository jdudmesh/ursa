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

	"github.com/google/uuid"
)

type uuidValidatorOpt = parseOpt[uuid.UUID]

func UUID(opts ...any) genericValidator[uuid.UUID] {
	v := newGenerator[uuid.UUID](opts...)
	v.setTransformer(func(val any) (any, error) {
		return coerceToUUID(val)
	})
	return v
}

func coerceToUUID(val any) (uuid.UUID, error) {
	vo := reflect.ValueOf(val)
	if vo.Kind() == reflect.Ptr {
		vo = vo.Elem()
		return coerceToUUID(vo.Interface())
	}
	if vo.Kind() != reflect.String {
		return uuid.Nil, InvalidValueError
	}
	return uuid.Parse(val.(string))
}

func NonNullUUID(message ...string) uuidValidatorOpt {
	return func(val uuid.UUID) *parseError {
		for _, v := range val {
			if v > 0 {
				return nil
			}
		}
		if len(message) > 0 {
			return &parseError{message: message[0]}
		}
		return &parseError{message: "uuid is zero"}
	}
}

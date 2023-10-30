package u

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
	"github.com/google/uuid"
)

type UrsaUUIDOpt func(u *ursaUUID, val uuid.UUID) *parseError

type ursaUUID struct {
	options []UrsaUUIDOpt
}

func (u *ursaUUID) Parse(val any) ParseResult {
	var err error
	res := &parseResult[uuid.UUID]{}

	typedVal := uuid.UUID{}
	switch val := val.(type) {
	case uuid.UUID:
		typedVal = val
	case *uuid.UUID:
		if val != nil {
			typedVal = *val
		}
	case string:
		typedVal, err = uuid.Parse(val)
		if err != nil {
			res.valid = false
			res.errors = append(res.errors, UrsaInvalidTypeError)
			return res
		}
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

func UUID(opts ...UrsaUUIDOpt) *ursaUUID {
	return &ursaUUID{
		options: opts,
	}
}

func NonNullUUID(message ...string) UrsaUUIDOpt {
	return func(u *ursaUUID, val uuid.UUID) *parseError {
		for _, v := range val {
			if v > 0 {
				return nil
			}
		}
		if len(message) > 0 {
			return &parseError{message: message[0]}
		}
		return &parseError{message: "string too short"}
	}
}

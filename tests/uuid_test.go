package tests

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
	"testing"

	"github.com/google/uuid"
	u "github.com/jdudmesh/ursa"
	"github.com/stretchr/testify/assert"
)

func TestUUID(t *testing.T) {
	assert := assert.New(t)

	v := u.UUID(u.NonNullUUID())

	u := uuid.New()

	res := v.Parse(u)
	assert.True(res.Valid())
	assert.Equal(u, res.Get())

	res = v.Parse(u.String())
	assert.True(res.Valid())

	sz := "not a uuid"
	res = v.Parse(sz)
	assert.False(res.Valid())

	//uuid.MustParse("00000000-0000-0000-0000-000000000000")
	sz = "00000000-0000-0000-0000-000000000000"
	res = v.Parse(sz)
	assert.False(res.Valid())
	assert.Equal("uuid is zero", res.Errors()[0].Error())
}

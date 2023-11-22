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

	u "github.com/jdudmesh/ursa"
	"github.com/stretchr/testify/assert"
)

func TestNumber(t *testing.T) {
	assert := assert.New(t)

	v := u.Int(
		u.Min(5, "Number should be >= 5"),
		u.Max(10))

	res := v.Parse(7)
	assert.Equal(true, res.IsValid())

	res = v.Parse("7")
	assert.Equal(true, res.IsValid())
	assert.Equal(7, res.Get())

	errs := v.Parse("ursa").Errors()
	assert.Equal(1, len(errs))
	assert.EqualError(errs[0], u.InvalidTypeError.Error())

	errs = v.Parse(3.14).Errors()
	assert.Equal(errs[0].Error(), "Number should be >= 5")

	errs = v.Parse(1).Errors()
	assert.Equal(errs[0].Error(), "Number should be >= 5")

	errs = v.Parse(100).Errors()
	assert.Equal(errs[0].Error(), "number too large")

	u2 := u.Int64(u.NonZero())
	errs = u2.Parse(0).Errors()
	assert.Equal(errs[0].Error(), "number is zero")
}

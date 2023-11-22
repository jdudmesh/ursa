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
	"slices"
	"testing"

	u "github.com/jdudmesh/ursa"
	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	assert := assert.New(t)

	v := u.String(
		u.MinLength(5, "String should be at least 5 characters"),
		u.MaxLength(10),
		u.Matches("^[0-9]*$"))

	res := v.Parse("01234678")
	assert.True(res.IsValid())
	assert.Equal(0, len(res.Errors()))
	assert.Equal("01234678", res.Get())

	errs := v.Parse(1).Errors()
	assert.Equal(2, len(errs))
	msgs := make([]string, 0, len(errs))
	for _, m := range errs {
		msgs = append(msgs, m.Error())
	}
	assert.True(slices.Contains(msgs, "String should be at least 5 characters"))
	assert.True(slices.Contains(msgs, "string does not match pattern"))

	errs = v.Parse("0123").Errors()
	assert.Equal(1, len(errs))
	assert.Equal(errs[0].Error(), "String should be at least 5 characters")

	errs = v.Parse("01234678901").Errors()
	assert.Equal(1, len(errs))
	assert.Equal(errs[0].Error(), "string too long")

	errs = v.Parse("notvalid").Errors()
	assert.Equal(1, len(errs))
	assert.Equal(errs[0].Error(), "string does not match pattern")

}

func TestStringPtr(t *testing.T) {
	assert := assert.New(t)

	v := u.String(
		u.MinLength(5, "String should be at least 5 characters"),
		u.MaxLength(10),
		u.Matches("^[0-9]*$"))

	t.Run("valid", func(t *testing.T) {
		testVal := "01234678"
		res := v.Parse(&testVal)
		assert.True(res.IsValid())
	})

	t.Run("nil", func(t *testing.T) {
		res := v.Parse(nil)
		assert.True(res.IsValid())
	})
}

func TestStringRequired(t *testing.T) {
	assert := assert.New(t)

	v := u.String(
		u.MinLength(5, "String should be at least 5 characters"),
		u.MaxLength(10),
		u.Matches("^[0-9]*$"),
		u.Required("required"))

	t.Run("nil", func(t *testing.T) {
		errs := v.Parse(nil).Errors()
		assert.Equal(1, len(errs))
		assert.Equal(errs[0].Error(), "required")
	})
}

func TestStringDefault(t *testing.T) {
	assert := assert.New(t)

	v := u.String(
		u.MinLength(5, "String should be at least 5 characters"),
		u.MaxLength(10),
		u.Matches("^[0-9]*$"),
		u.WithDefault("01234678"),
		u.Required())

	t.Run("nil", func(t *testing.T) {
		res := v.Parse(nil)
		assert.True(res.IsValid())
		assert.Equal("01234678", res.Get())
	})
}

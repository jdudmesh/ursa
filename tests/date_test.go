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
	"time"

	u "github.com/jdudmesh/ursa"
	"github.com/stretchr/testify/assert"
)

func TestDate(t *testing.T) {
	assert := assert.New(t)

	dt := time.Now()
	szdt := dt.Format(time.RFC3339)

	v := u.Time(
		u.NotBefore(dt.Add(-1*time.Hour)),
		u.NotAfter(dt.Add(1*time.Hour)),
		u.WithTimeFormat(time.RFC3339),
	)

	errs := v.Parse(dt).Errors()
	assert.Equal(0, len(errs))

	errs = v.Parse(&dt).Errors()
	assert.Equal(0, len(errs))

	errs = v.Parse(szdt).Errors()
	assert.Equal(0, len(errs))

	errs = v.Parse(dt.Add(-2 * time.Hour)).Errors()
	assert.Equal(1, len(errs))
	assert.Equal(errs[0].Error(), "date is too early")

	errs = v.Parse(dt.Add(2 * time.Hour)).Errors()
	assert.Equal(1, len(errs))
	assert.Equal(errs[0].Error(), "date is too late")
}

func TestDateMissingParser(t *testing.T) {
	assert := assert.New(t)

	dt := time.Now()
	szdt := dt.Format(time.RFC3339)

	v := u.Time(
		u.NotBefore(dt.Add(-1*time.Hour)),
		u.NotAfter(dt.Add(1*time.Hour)),
	)

	errs := v.Parse(szdt).Errors()
	assert.Equal(1, len(errs))
	assert.ErrorIs(errs[0], u.MissingTransformerError)
}

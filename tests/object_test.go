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
	"bytes"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	u "github.com/jdudmesh/ursa"
	"github.com/stretchr/testify/assert"
)

func TestObject(t *testing.T) {
	assert := assert.New(t)

	v := u.Object().
		String("Name", u.MinLength(5, "String should be at least 5 characters")).
		Number("Count", u.Int())

	errs := v.Parse(&struct {
		Name  string
		Count int
	}{
		Name:  "abcdef",
		Count: 5,
	}).Errors()
	assert.Equal(0, len(errs))

	errs = v.Parse(&struct {
		Name string
	}{
		Name: "abc",
	}).Errors()
	assert.Equal(1, len(errs))
	// assert.Equal("Name", errs[0].Field())
	// assert.Equal("String should be at least 5 characters", errState.Inner()[0].Error())

	errs = v.Parse(map[string]string{
		"NotName": "abcdefgh",
	}).Errors()
	assert.Equal(1, len(errs))
	// assert.ErrorAs(errs[0], &errState)
	// assert.Equal("Name", errState.Field())
	// assert.Equal("not found", errState.Error())

}

func TestObjectJSON(t *testing.T) {
	assert := assert.New(t)

	v := u.Object().
		String("Name", u.MinLength(5, "String should be at least 5 characters")).
		Number("Count", u.Int())

	data := `{ "Name": "abcdef", "Count": 5 }`
	errs := v.Parse([]byte(data)).Errors()
	assert.Equal(0, len(errs))
}

func TestObjectHTTP(t *testing.T) {
	assert := assert.New(t)

	v := u.Object(u.WithMaxBodySize(1000)).
		String("Name", u.MinLength(5, "String should be at least 5 characters")).
		Number("Count", u.Int())

	t.Run("json", func(t *testing.T) {
		data := `{ "Name": "abcdef", "Count": 5 }`

		req, _ := http.NewRequest("POST", "http://localhost:8080/upload", strings.NewReader(data))
		req.Header.Set("Content-Type", "application/json")

		errs := v.Parse(req).Errors()
		assert.Equal(0, len(errs))
	})

	t.Run("url encoded", func(t *testing.T) {
		data := `Name=abcdef&Count=5`

		req, _ := http.NewRequest("POST", "http://localhost:8080/upload", strings.NewReader(data))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		errs := v.Parse(req).Errors()
		assert.Equal(0, len(errs))
	})

	t.Run("multipart form", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("Name", "abcdef")
		writer.WriteField("Count", "5")
		writer.Close()

		req, _ := http.NewRequest("POST", "http://localhost:8080/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		errs := v.Parse(req).Errors()
		assert.Equal(0, len(errs))
	})

	t.Run("generic get", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://localhost:8080/upload?Name=abcdef&Count=5", nil)

		errs := v.Parse(req).Errors()
		assert.Equal(0, len(errs))
	})
}

func TestObjectMissingField(t *testing.T) {
	assert := assert.New(t)

	v := u.Object().
		String("Name", u.MinLength(5, "String should be at least 5 characters")).
		Number("Count", u.Int(), u.WithDefault(5))

	errs := v.Parse(map[string]string{
		"Name": "abcdefgh",
	}).Errors()
	assert.Equal(0, len(errs))

}

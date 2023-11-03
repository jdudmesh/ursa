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
		String("Name", u.MinLength(5, "String should be at least 5 characters"), u.Required()).
		Int("Count")

	res := v.Parse(&struct {
		Name  string
		Count int
	}{
		Name:  "abcdef",
		Count: 5,
	})
	assert.Equal(0, len(res.Errors()))
	assert.Equal("abcdef", res.GetField("Name").Get())
	assert.Equal(5, res.GetField("Count").Get())

	res = v.Parse(&struct {
		Name string
	}{
		Name: "abc",
	})
	assert.Equal(1, len(res.Errors()))
	assert.Equal("String should be at least 5 characters", res.Errors()[0].Error())
	assert.Equal("String should be at least 5 characters", res.GetField("Name").Errors()[0].Error())

	errs := v.Parse(map[string]string{
		"NotName": "abcdefgh",
	}).Errors()
	assert.Equal(1, len(errs))
	assert.Equal("missing required property", errs[0].Error())

}

func TestObjectJSON(t *testing.T) {
	assert := assert.New(t)

	v := u.Object().
		String("Name", u.MinLength(5, "String should be at least 5 characters")).
		Int("Count")

	data := `{ "Name": "abcdef", "Count": 5 }`
	errs := v.Parse([]byte(data)).Errors()
	assert.Equal(0, len(errs))
}

func TestObjectHTTP(t *testing.T) {
	assert := assert.New(t)

	v := u.Object(u.WithMaxBodySize(1000)).
		String("Name", u.MinLength(5, "String should be at least 5 characters")).
		Int("Count")

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
		_ = writer.WriteField("Name", "abcdef")
		_ = writer.WriteField("Count", "5")
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
		Int("Count", u.WithDefault(5))

	errs := v.Parse(map[string]string{
		"Name": "abcdefgh",
	}).Errors()
	assert.Equal(0, len(errs))

}

type testStruct struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestUnmarshal(t *testing.T) {
	v := u.Object().
		String("name", u.MinLength(5, "String should be at least 5 characters"), u.Required()).
		Int("count")

	data := map[string]interface{}{
		"name":  "abcdef",
		"count": 5,
	}

	res := v.Parse(data)

	assert.True(t, res.Valid())

	t.Run("unpack to map", func(t *testing.T) {
		tgt := make(map[string]interface{})
		err := res.Unmarshal(tgt)
		assert.NoError(t, err)
		assert.Equal(t, "abcdef", tgt["name"])
		assert.Equal(t, 5, tgt["count"])
	})

	t.Run("unpack to struct", func(t *testing.T) {
		tgt := testStruct{}
		err := res.Unmarshal(&tgt)
		assert.NoError(t, err)
		assert.Equal(t, "abcdef", tgt.Name)
		assert.Equal(t, 5, tgt.Count)
	})
}

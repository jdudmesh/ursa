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

type ursaEntity interface {
	Parse(val any) ParseResult
}

type ParseResult interface {
	Valid() bool
	Errors() []ParseError
	Value() interface{}
}

type ParseError interface {
	error
	Inner() []error
}

type parseResult[T any] struct {
	valid  bool
	value  T
	field  string
	errors []ParseError
}

func (r *parseResult[T]) Valid() bool {
	return r.valid
}

func (r *parseResult[T]) Errors() []ParseError {
	return r.errors
}

func (r *parseResult[T]) Field() string {
	return r.field
}

func (r *parseResult[T]) Value() interface{} {
	return r.value
}

type parseError struct {
	message string
	inner   []error
}

func (e *parseError) Inner() []error {
	return e.inner
}

func (e *parseError) Error() string {
	return e.message
}

var UrsaInvalidTypeError = &parseError{
	message: "invalid type",
}

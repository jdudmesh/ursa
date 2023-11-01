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

type EntityOpt func(u ursaEntity) error
type ParseOpt func(res ParseResult) error

type ursaEntityBase[T any] struct {
	defaultValue T
	required     bool
}

type ursaEntity interface {
	Parse(val any, opts ...ParseOpt) ParseResult
}

type ParseResult interface {
	Valid() bool
	Errors() []ParseError
	Value() interface{}
	Name() string
	AppendError(message string, inner ...error)
}

type ParseError interface {
	error
	Inner() []error
}

type parseResult[T any] struct {
	valid  bool
	value  T
	name   string
	errors []ParseError
}

func (r *parseResult[T]) Valid() bool {
	return r.valid
}

func (r *parseResult[T]) Errors() []ParseError {
	return r.errors
}

func (r *parseResult[T]) AppendError(message string, inner ...error) {
	r.errors = append(r.errors, &parseError{
		message: message,
		inner:   inner,
	})
}

func (r *parseResult[T]) Name() string {
	return r.name
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

func WithDefault(val any) EntityOpt {
	return func(u ursaEntity) error {
		u.setDefault(val)
		return nil
	}
}

func WithRequired() EntityOpt {
	return func(u *ursaEntityBase[T]) error {
		u.setRequired()
		return nil
	}
}

func (b *ursaEntityBase[T]) setDefault(val T) {
	b.defaultValue = val
}

func (b *ursaEntityBase[T]) getDefault() T {
	return b.defaultValue
}

func (b *ursaEntityBase[T]) setRequired() {
	b.required = true
}

func (b *ursaEntityBase[T]) getRequired() bool {
	return b.required
}

type ursaValidator[T any] struct {
	options      []UrsaValidatorOpt[T]
	defaultValue T
	required     bool
}

type UrsaValidatorOpt[T any] func(u *ursaValidator[T], val T) *parseError

type UrsaStringValidatorOpt = UrsaValidatorOpt[string]

type ursaStringValidator struct {
	validator ursaValidator[string]
}

package storage

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
)

// UUIDFieldIndex is used to extract a field from an object
// using reflection and builds an index on that field by treating
// it as a UUID. This is an optimization to using a StringFieldIndex
// as the UUID can be more compactly represented in byte form.
type UUIDFieldIndex struct {
	Field string
}

func (u *UUIDFieldIndex) FromObject(obj interface{}) (bool, []byte, error) {
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v) // Dereference the pointer if any

	fv := v.FieldByName(u.Field)
	if !fv.IsValid() {
		return false, nil, fmt.Errorf("field '%s' for %#v is invalid", u.Field, obj) //nolint:goerr113
	}

	if u, ok := fv.Interface().(*uuid.UUID); ok {
		return true, u[:], nil
	}

	if u, ok := fv.Interface().(uuid.UUID); ok {
		return true, u[:], nil
	}

	return false, nil, nil
}

func (u *UUIDFieldIndex) FromArgs(args ...interface{}) ([]byte, error) {
	const uuidBinLen = 16

	if len(args) != 1 {
		return nil, fmt.Errorf("must provide only a single argument") //nolint:goerr113
	}

	switch arg := args[0].(type) {
	case uuid.UUID:
		return arg[:], nil
	case *uuid.UUID:
		return arg[:], nil
	case string:
		return u.parseString(arg, true)
	case []byte:
		if len(arg) != uuidBinLen {
			return nil, fmt.Errorf("byte slice must be 16 characters") //nolint:goerr113
		}

		return arg, nil
	default:
		return nil,
			fmt.Errorf("argument must be a string or byte slice: %#v", args[0]) //nolint:goerr113
	}
}

// parseString parses a UUID from the string. If enforceLength is false, it will
// parse a partial UUID. An error is returned if the input, stripped of hyphens,
// is not even length.
func (u *UUIDFieldIndex) parseString(s string, enforceLength bool) ([]byte, error) {
	const (
		hyphensLen = 4
		uuidLen    = 36
	)

	// Verify the length
	l := len(s)
	if enforceLength && l != uuidLen {
		return nil, fmt.Errorf("UUID must be 36 characters") //nolint:goerr113
	} else if l > uuidLen {
		return nil, fmt.Errorf("invalid UUID length. UUID have 36 characters; got %d", l) //nolint:goerr113
	}

	hyphens := strings.Count(s, "-")
	if hyphens > hyphensLen {
		return nil, fmt.Errorf(`UUID should have maximum of 4 "-"; got %d`, hyphens) //nolint:goerr113
	}

	// The sanitized length is the length of the original string without the "-".
	sanitized := strings.Replace(s, "-", "", -1)
	sanitizedLength := len(sanitized)
	if sanitizedLength%2 != 0 {
		return nil, fmt.Errorf("input (without hyphens) must be even length") //nolint:goerr113
	}

	dec, err := hex.DecodeString(sanitized)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID: %v", err) //nolint:goerr113,errorlint
	}

	return dec, nil
}

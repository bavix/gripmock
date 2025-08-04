package uuidconv

import (
	"unsafe"

	"github.com/google/uuid"
)

// UUID2DoubleInt converts a UUID to two 64-bit integers.
//
// Parameters:
// - v: the UUID to convert.
//
// Returns:
// - high: the high part of the UUID as a 64-bit integer.
// - low: the low part of the UUID as a 64-bit integer.
func UUID2DoubleInt(v uuid.UUID) (int64, int64) {
	//nolint:gosec // This is a safe conversion from UUID bytes to int64
	return *(*int64)(unsafe.Pointer(&v[0])), *(*int64)(unsafe.Pointer(&v[8]))
}

// DoubleInt2UUID converts two 64-bit integers to a UUID.
//
// Parameters:
// - highValue: the high part of the UUID as a 64-bit integer.
// - lowValue: the low part of the UUID as a 64-bit integer.
//
// Returns:
// - uuid: the UUID constructed from the high and low integers.
func DoubleInt2UUID(highValue int64, lowValue int64) uuid.UUID {
	var uuidValue uuid.UUID

	//nolint:gosec // This is a safe conversion from int64 to UUID bytes
	*(*int64)(unsafe.Pointer(&uuidValue[0])) = highValue
	//nolint:gosec // This is a safe conversion from int64 to UUID bytes
	*(*int64)(unsafe.Pointer(&uuidValue[8])) = lowValue

	return uuidValue
}

package uuidconv_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/bavix/gripmock/v3/internal/infra/uuidconv"
)

func TestUUID2DoubleInt(t *testing.T) {
	// Test with a known UUID
	testUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	high, low := uuidconv.UUID2DoubleInt(testUUID)

	// Verify the conversion is reversible
	convertedUUID := uuidconv.DoubleInt2UUID(high, low)
	assert.Equal(t, testUUID, convertedUUID)
}

func TestDoubleInt2UUID(t *testing.T) {
	// Test with known high and low values
	high := int64(0x550e8400e29b41d4)
	low := int64(-0x58e9bb99aabac000)

	uuidValue := uuidconv.DoubleInt2UUID(high, low)

	// Verify the conversion is reversible
	convertedHigh, convertedLow := uuidconv.UUID2DoubleInt(uuidValue)
	assert.Equal(t, high, convertedHigh)
	assert.Equal(t, low, convertedLow)
}

func TestUUIDConversionRoundTrip(t *testing.T) {
	// Test round-trip conversion with multiple UUIDs
	testUUIDs := []uuid.UUID{
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
		uuid.MustParse("6ba7b811-9dad-11d1-80b4-00c04fd430c8"),
		uuid.New(), // Random UUID
	}

	for _, testUUID := range testUUIDs {
		t.Run(testUUID.String(), func(t *testing.T) {
			high, low := uuidconv.UUID2DoubleInt(testUUID)
			convertedUUID := uuidconv.DoubleInt2UUID(high, low)
			assert.Equal(t, testUUID, convertedUUID)
		})
	}
}

func TestUUIDConversionWithZeroValues(t *testing.T) {
	// Test with zero UUID
	zeroUUID := uuid.UUID{}

	high, low := uuidconv.UUID2DoubleInt(zeroUUID)
	assert.Equal(t, int64(0), high)
	assert.Equal(t, int64(0), low)

	convertedUUID := uuidconv.DoubleInt2UUID(high, low)
	assert.Equal(t, zeroUUID, convertedUUID)
}

func TestUUIDConversionWithMaxValues(t *testing.T) {
	// Test with maximum values
	maxHigh := int64(0x7fffffffffffffff)
	maxLow := int64(0x7fffffffffffffff)

	uuidValue := uuidconv.DoubleInt2UUID(maxHigh, maxLow)
	convertedHigh, convertedLow := uuidconv.UUID2DoubleInt(uuidValue)

	assert.Equal(t, maxHigh, convertedHigh)
	assert.Equal(t, maxLow, convertedLow)
}

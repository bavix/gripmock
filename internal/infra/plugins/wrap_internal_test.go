package plugins

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

const (
	testString = "test"
	baseString = "base"
)

func TestWrapFunc_AlreadyFunc(t *testing.T) {
	t.Parallel()

	// Arrange
	fn := func(ctx context.Context, args ...any) (any, error) {
		return testString, nil
	}

	// Act
	wrapped := wrapFunc(fn)
	result, err := wrapped(t.Context())

	// Assert
	require.NotNil(t, wrapped)
	require.NoError(t, err)
	require.Equal(t, testString, result)
}

func TestWrapFunc_SimpleFunc(t *testing.T) {
	t.Parallel()

	// Arrange
	fn := func() string {
		return testString
	}

	// Act
	wrapped := wrapFunc(fn)
	result, err := wrapped(t.Context())

	// Assert
	require.NotNil(t, wrapped)
	require.NoError(t, err)
	require.Equal(t, testString, result)
}

func TestWrapFunc_FuncWithArgs(t *testing.T) {
	t.Parallel()

	// Arrange
	fn := func(a, b int) int {
		return a + b
	}
	arg1, arg2 := 2, 3

	// Act
	wrapped := wrapFunc(fn)
	result, err := wrapped(t.Context(), arg1, arg2)

	// Assert
	require.NotNil(t, wrapped)
	require.NoError(t, err)
	require.Equal(t, 5, result)
}

func TestWrapFunc_FuncWithError(t *testing.T) {
	t.Parallel()

	// Arrange
	fn := func() (string, error) {
		return testString, nil
	}

	// Act
	wrapped := wrapFunc(fn)
	result, err := wrapped(t.Context())

	// Assert
	require.NotNil(t, wrapped)
	require.NoError(t, err)
	require.Equal(t, testString, result)
}

func TestWrapFunc_FuncWithErrorReturn(t *testing.T) {
	t.Parallel()

	// Arrange
	expectedErr := errors.New("test error") //nolint:err113

	fn := func() (string, error) {
		return "", expectedErr
	}

	// Act
	wrapped := wrapFunc(fn)
	result, err := wrapped(t.Context())

	// Assert
	require.NotNil(t, wrapped)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
	require.Empty(t, result)
}

func TestWrapFunc_UnsupportedResultCount(t *testing.T) {
	t.Parallel()

	fn := func() (int, int, int) {
		return 1, 2, 3
	}

	wrapped := wrapFunc(fn)
	result, err := wrapped(t.Context())

	require.NotNil(t, wrapped)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported result count")
	require.Nil(t, result)
}

func TestWrapFunc_WithContext(t *testing.T) {
	t.Parallel()

	// Arrange
	fn := func(ctx context.Context) string {
		return testString
	}

	// Act
	wrapped := wrapFunc(fn)
	result, err := wrapped(t.Context())

	// Assert
	require.NotNil(t, wrapped)
	require.NoError(t, err)
	require.Equal(t, testString, result)
}

func TestWrapFunc_InvalidType(t *testing.T) {
	t.Parallel()

	// Arrange
	invalidInput := "not a function"

	// Act
	wrapped := wrapFunc(invalidInput)

	// Assert
	require.Nil(t, wrapped)
}

func TestWrapDecorator_ValidDecorator(t *testing.T) {
	t.Parallel()

	// Arrange
	decorator := func(base pkgplugins.Func) pkgplugins.Func {
		return func(ctx context.Context, args ...any) (any, error) {
			result, err := base(ctx, args...)
			if err != nil {
				return nil, err
			}

			str, ok := result.(string)
			if !ok {
				return nil, errors.New("result is not a string") //nolint:err113
			}

			return "decorated: " + str, nil
		}
	}
	base := func(ctx context.Context, args ...any) (any, error) {
		return baseString, nil
	}

	// Act
	wrapped := wrapDecorator(decorator)
	decorated := wrapped(base)
	result, err := decorated(t.Context())

	// Assert
	require.NotNil(t, wrapped)
	require.NoError(t, err)
	require.Equal(t, "decorated: "+baseString, result)
}

func TestWrapDecorator_InvalidType(t *testing.T) {
	t.Parallel()

	// Arrange
	invalidInput := "not a decorator"

	// Act
	wrapped := wrapDecorator(invalidInput)

	// Assert
	require.Nil(t, wrapped)
}

func TestIsNilAssignable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  reflect.Type
		want bool
	}{
		{"chan", reflect.TypeFor[chan int](), true},
		{"func", reflect.TypeFor[func()](), true},
		{"interface", reflect.TypeFor[any](), true},
		{"map", reflect.TypeFor[map[string]int](), true},
		{"pointer", reflect.TypeFor[*int](), true},
		{"slice", reflect.TypeFor[[]int](), true},
		{"string", reflect.TypeFor[string](), false},
		{"int", reflect.TypeFor[int](), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, isNilAssignable(tt.typ))
		})
	}
}

func TestCoerceArg(t *testing.T) {
	t.Parallel()

	// Arrange
	args := []any{1, testString, 3.14}
	idx := 0
	paramType := reflect.TypeFor[int]()
	fnType := reflect.TypeFor[func(int)]()

	// Act
	val, err := coerceArg(args, &idx, paramType, fnType, 0)

	// Assert
	require.NoError(t, err)
	require.Equal(t, 1, val.Interface())
	require.Equal(t, 1, idx)
}

func TestCoerceArg_NilValue(t *testing.T) {
	t.Parallel()

	// Arrange
	args := []any{nil}
	idx := 0
	paramType := reflect.TypeFor[*int]()
	fnType := reflect.TypeFor[func(*int)]()

	// Act
	val, err := coerceArg(args, &idx, paramType, fnType, 0)

	// Assert
	require.NoError(t, err)
	require.True(t, val.IsNil())
}

func TestCoerceArg_NotEnoughArgs(t *testing.T) {
	t.Parallel()

	// Arrange
	args := []any{}
	idx := 0
	paramType := reflect.TypeFor[int]()
	fnType := reflect.TypeFor[func(int)]()

	// Act
	_, err := coerceArg(args, &idx, paramType, fnType, 0)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "not enough arguments")
}

func TestCoerceArg_TypeMismatch(t *testing.T) {
	t.Parallel()

	// Arrange
	args := []any{"string"}
	idx := 0
	paramType := reflect.TypeFor[int]()
	fnType := reflect.TypeFor[func(int)]()

	// Act
	_, err := coerceArg(args, &idx, paramType, fnType, 0)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "want")
}

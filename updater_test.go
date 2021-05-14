package updater

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockReleaseLocator struct {
	mock.Mock
}

func (m *mockReleaseLocator) ListReleases(ctx context.Context, amount int) ([]Release, error) {
	args := m.Called(ctx, amount)
	arg0 := args.Get(0)
	if arg0 == nil {
		return nil, args.Error(1)
	}

	return arg0.([]Release), args.Error(1)
}

func TestStableRelease(t *testing.T) {
	assert.False(t, StableRelease("foo", true, false))
	assert.False(t, StableRelease("foo", false, true))
	assert.False(t, StableRelease("foo", true, true))
	assert.True(t, StableRelease("foo", false, false))
}

func TestLatestRelease(t *testing.T) {
	locator := new(mockReleaseLocator)
	locator.On("ListReleases", mock.Anything, mock.Anything).Return(nil, ErrNoRepository)

	_, err := LatestRelease(context.Background(), locator)
	require.Error(t, err)
	assert.NotEqual(t, ErrNoRepository, err)
	assert.Equal(t, ErrNoRepository, errors.Unwrap(err))
}

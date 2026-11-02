package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextReleaseCandidate(t *testing.T) {
	var gotPrefix string
	list := func(existing []string, err error) func(string) ([]string, error) {
		return func(prefix string) ([]string, error) {
			gotPrefix = prefix
			return existing, err
		}
	}

	v, err := nextReleaseCandidate(list(nil, nil), "1.4.0")
	require.NoError(t, err)
	require.Equal(t, "1.4.0-rc.1", v)
	require.Equal(t, "1.4.0-rc.", gotPrefix)

	v, err = nextReleaseCandidate(list([]string{"1.4.0-rc.1", "1.4.0-rc.2"}, nil), "1.4.0")
	require.NoError(t, err)
	require.Equal(t, "1.4.0-rc.3", v)

	_, err = nextReleaseCandidate(list(nil, errors.New("boom")), "1.4.0")
	require.ErrorContains(t, err, "boom")
}

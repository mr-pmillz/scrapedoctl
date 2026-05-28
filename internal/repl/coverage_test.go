package repl_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/repl"
	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

type MockErrorReader struct {
	err error
}

func (m *MockErrorReader) Readline() (string, error) {
	return "", m.err
}

func TestREPL_Run_Errors(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	s := repl.NewShell(client)

	t.Run("custom error", func(t *testing.T) {
		s.SetReader(&MockErrorReader{err: errors.New("custom failure")})
		err := s.Run(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "custom failure")
	})
}

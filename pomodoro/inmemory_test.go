//go:build inmemory

package pomodoro_test

import (
	"testing"

	"github.com/snirkop89/pomo/pomodoro"
	"github.com/snirkop89/pomo/pomodoro/repository"
)

func getRepo(t *testing.T) (pomodoro.Repository, func()) {
	t.Helper()
	return repository.NewInMemoryRepo(), func() {}
}

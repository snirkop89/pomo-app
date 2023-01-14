//go:build inmemory

package cmd

import (
	"github.com/snirkop89/pomo/pomodoro"
	"github.com/snirkop89/pomo/pomodoro/repository"
)

func getRepo() (pomodoro.Repository, error) {
	return repository.NewInMemoryRepo(), nil
}

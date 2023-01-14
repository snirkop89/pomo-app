//go:build !inmemory

package cmd

import (
	"github.com/snirkop89/pomo/pomodoro"
	"github.com/snirkop89/pomo/pomodoro/repository"
	"github.com/spf13/viper"
)

func getRepo() (pomodoro.Repository, error) {
	repo, err := repository.NewSQLite3Repo(viper.GetString("db"))
	if err != nil {
		return nil, err
	}

	return repo, nil
}

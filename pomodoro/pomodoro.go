// Package pomodoro provides the logic of pomodoro technique
package pomodoro

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Category constants
const (
	CategoryPomodoro   = "Pomodoro"
	CategoryShortBreak = "ShortBreak"
	CategoryLongBreak  = "LongBreak"
)

// State constants
const (
	StateNotStarted = iota
	StateRunning
	StatePaused
	StateDone
	StateCancelled
)

type Interval struct {
	ID              int64
	StartTime       time.Time
	PlannedDuration time.Duration
	ActualDuration  time.Duration
	Category        string
	State           int
}

type Repository interface {
	Create(i Interval) (int64, error)
	Update(i Interval) error
	ByID(id int64) (Interval, error)
	Last() (Interval, error)
	Breaks(n int) ([]Interval, error)
	CategorySummary(day time.Time, filter string) (time.Duration, error)
}

var (
	ErrNoIntervals        = errors.New("no intervals")
	ErrIntervalNotRunning = errors.New("nterval not running")
	ErrIntervalCompleted  = errors.New("interval is completed or cancelled")
	ErrInvalidState       = errors.New("invalid state")
	ErrInvalidID          = errors.New("invalid ID")
)

type IntervalConfig struct {
	repo               Repository
	PomodoroDuration   time.Duration
	ShortBreakDuration time.Duration
	LongBreakDuration  time.Duration
}

func NewConfig(repo Repository, pomodoro, shortBreak, longBreak time.Duration) *IntervalConfig {
	c := &IntervalConfig{
		repo:               repo,
		PomodoroDuration:   25 * time.Minute,
		ShortBreakDuration: 5 * time.Minute,
		LongBreakDuration:  15 * time.Minute,
	}

	if pomodoro > 0 {
		c.PomodoroDuration = pomodoro
	}

	if shortBreak > 0 {
		c.ShortBreakDuration = shortBreak
	}

	if longBreak > 0 {
		c.LongBreakDuration = longBreak
	}

	return c
}

func nextCategory(r Repository) (string, error) {
	li, err := r.Last()
	if err != nil && err == ErrNoIntervals {
		return CategoryPomodoro, nil
	}
	if err != nil {
		return "", err
	}

	// If we are in a break, next stage is work
	if li.Category == CategoryLongBreak || li.Category == CategoryShortBreak {
		return CategoryPomodoro, nil
	}
	lastBreaks, err := r.Breaks(3)
	if err != nil {
		return "", err
	}
	// After every 4th work interval, there should be a long break
	if len(lastBreaks) < 3 {
		return CategoryShortBreak, nil
	}
	// If there was already a long break in the cycle, we'll return a short break
	for _, i := range lastBreaks {
		if i.Category == CategoryLongBreak {
			return CategoryShortBreak, nil
		}
	}
	return CategoryLongBreak, nil
}

type Callback func(Interval)

func tick(ctx context.Context, id int64, config *IntervalConfig, start, periodic, end Callback) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	i, err := config.repo.ByID(id)
	if err != nil {
		return err
	}
	expire := time.After(i.PlannedDuration - i.ActualDuration)

	start(i)

	for {
		select {
		case <-ticker.C:
			i, err := config.repo.ByID(id)
			if err != nil {
				return err
			}
			if i.State == StatePaused {
				return nil
			}
			i.ActualDuration += time.Second
			if err := config.repo.Update(i); err != nil {
				return err
			}
			periodic(i)
		case <-expire:
			i, err := config.repo.ByID(id)
			if err != nil {
				return err
			}
			i.State = StateDone
			end(i)
			return config.repo.Update(i)
		case <-ctx.Done():
			i, err := config.repo.ByID(id)
			if err != nil {
				return err
			}
			i.State = StateCancelled
			return config.repo.Update(i)
		}
	}
}

func newInterval(config *IntervalConfig) (Interval, error) {
	category, err := nextCategory(config.repo)
	if err != nil {
		return Interval{}, err
	}

	var pd time.Duration
	switch category {
	case CategoryPomodoro:
		pd = config.PomodoroDuration
	case CategoryShortBreak:
		pd = config.ShortBreakDuration
	case CategoryLongBreak:
		pd = config.LongBreakDuration
	}

	i := Interval{
		PlannedDuration: pd,
		Category:        category,
	}

	if i.ID, err = config.repo.Create(i); err != nil {
		return Interval{}, err
	}

	return i, nil
}

func GetInterval(config *IntervalConfig) (Interval, error) {
	i, err := config.repo.Last()
	if err != nil && err != ErrNoIntervals {
		return Interval{}, err
	}
	// If there's a current running interval
	if err == nil && i.State != StateCancelled && i.State != StateDone {
		return i, nil
	}

	return newInterval(config)
}

func (i Interval) Start(ctx context.Context, config *IntervalConfig, start, periodic, end Callback) error {
	switch i.State {
	case StateRunning:
		return nil
	case StateNotStarted:
		i.StartTime = time.Now()
		fallthrough
	case StatePaused:
		i.State = StateRunning
		if err := config.repo.Update(i); err != nil {
			return err
		}
		return tick(ctx, i.ID, config, start, periodic, end)
	case StateCancelled, StateDone:
		return fmt.Errorf("%w: cannot start", ErrIntervalCompleted)
	default:
		return fmt.Errorf("%w: %d", ErrInvalidState, i.State)
	}
}

func (i Interval) Pause(config *IntervalConfig) error {
	if i.State != StateRunning {
		return ErrIntervalNotRunning
	}
	i.State = StatePaused
	return config.repo.Update(i)
}

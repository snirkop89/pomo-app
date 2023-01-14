package pomodoro_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/snirkop89/pomo/pomodoro"
)

func TestNewConfig(t *testing.T) {
	testCases := []struct {
		name   string
		input  [3]time.Duration
		expect pomodoro.IntervalConfig
	}{
		{
			name: "Default",
			expect: pomodoro.IntervalConfig{
				PomodoroDuration:   25 * time.Minute,
				ShortBreakDuration: 5 * time.Minute,
				LongBreakDuration:  15 * time.Minute,
			},
		},
		{
			name: "SingleInput",
			input: [3]time.Duration{
				20 * time.Minute,
			},
			expect: pomodoro.IntervalConfig{
				PomodoroDuration:   20 * time.Minute,
				ShortBreakDuration: 5 * time.Minute,
				LongBreakDuration:  15 * time.Minute,
			},
		},
		{
			name: "MultiInput",
			input: [3]time.Duration{
				20 * time.Minute,
				10 * time.Minute,
				12 * time.Minute,
			},
			expect: pomodoro.IntervalConfig{
				PomodoroDuration:   20 * time.Minute,
				ShortBreakDuration: 10 * time.Minute,
				LongBreakDuration:  12 * time.Minute,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var repo pomodoro.Repository
			config := pomodoro.NewConfig(
				repo,
				tt.input[0],
				tt.input[1],
				tt.input[2],
			)
			if config.PomodoroDuration != tt.expect.PomodoroDuration {
				t.Errorf("expected PomodoroDuration %v, got %v", tt.expect.PomodoroDuration, config.PomodoroDuration)
			}
			if config.LongBreakDuration != tt.expect.LongBreakDuration {
				t.Errorf("expected LongBreakDuration %v, got %v", tt.expect.LongBreakDuration, config.LongBreakDuration)
			}
			if config.ShortBreakDuration != tt.expect.ShortBreakDuration {
				t.Errorf("expected ShortBreakDuration %v, got %v", tt.expect.ShortBreakDuration, config.ShortBreakDuration)
			}
		})
	}
}

func TestGetInterval(t *testing.T) {
	repo, cleanup := getRepo(t)
	defer cleanup()

	const duration = 1 * time.Millisecond
	config := pomodoro.NewConfig(repo, 3*duration, duration, 2*duration)

	for i := 1; i <= 16; i++ {
		var (
			expCategory string
			expDuration time.Duration
		)
		switch {
		case i%2 != 0:
			expCategory = pomodoro.CategoryPomodoro
			expDuration = 3 * duration
		case i%8 == 0:
			expCategory = pomodoro.CategoryLongBreak
			expDuration = 2 * duration
		case i%2 == 0:
			expCategory = pomodoro.CategoryShortBreak
			expDuration = duration
		}

		testName := fmt.Sprintf("%d-%s", i, expCategory)
		t.Run(testName, func(t *testing.T) {
			res, err := pomodoro.GetInterval(config)
			if err != nil {
				t.Errorf("expected no error, got %q\n", err)
			}

			noop := func(pomodoro.Interval) {}
			if err := res.Start(context.Background(), config, noop, noop, noop); err != nil {
				t.Fatal(err)
			}
			if res.Category != expCategory {
				t.Errorf("expected category %q, got %q", expCategory, res.Category)
			}
			if res.PlannedDuration != expDuration {
				t.Errorf("expected PlannedDuration %q, got %q", expDuration, res.PlannedDuration)
			}
			if res.State != pomodoro.StateNotStarted {
				t.Errorf("expected state = %q, got %q", pomodoro.StateNotStarted, res.State)
			}
			ui, err := repo.ByID(res.ID)
			if err != nil {
				t.Errorf("expected no error, got %q", err)
			}
			if ui.State != pomodoro.StateDone {
				t.Errorf("expected state = %q, got %q", pomodoro.StateDone, ui.State)
			}
		})
	}
}

func TestPause(t *testing.T) {
	const duration = 2 * time.Second

	repo, cleanup := getRepo(t)
	defer cleanup()

	config := pomodoro.NewConfig(repo, duration, duration, duration)
	testCases := []struct {
		name        string
		start       bool
		expState    int
		expDuration time.Duration
	}{
		{name: "NotStarted", start: false, expState: pomodoro.StateNotStarted, expDuration: 0},
		{name: "Paused", start: true, expState: pomodoro.StatePaused, expDuration: duration / 2},
	}
	expError := pomodoro.ErrIntervalNotRunning

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())

			i, err := pomodoro.GetInterval(config)
			if err != nil {
				t.Fatal(err)
			}
			start := func(pomodoro.Interval) {}
			end := func(pomodoro.Interval) {
				t.Errorf("End callback should be executed")
			}
			periodic := func(i pomodoro.Interval) {
				if err := i.Pause(config); err != nil {
					t.Fatal(err)
				}
			}

			if tt.start {
				if err := i.Start(ctx, config, start, periodic, end); err != nil {
					t.Fatal(err)
				}
			}
			i, err = pomodoro.GetInterval(config)
			if err != nil {
				t.Fatal(err)
			}
			err = i.Pause(config)
			if err != nil {
				if !errors.Is(err, expError) {
					t.Fatalf("expected error %q, got %q", expError, err)
				}
			}
			if err == nil {
				t.Errorf("expected error %q, got nil", expError)
			}
			i, err = repo.ByID(i.ID)
			if err != nil {
				t.Fatal(err)
			}
			if i.State != tt.expState {
				t.Errorf("expected state %d, got %d", tt.expState, i.State)
			}
			if i.ActualDuration != tt.expDuration {
				t.Errorf("expected duration %q, got %q", tt.expDuration, i.ActualDuration)
			}
			cancel()
		})

	}
}

func TestStart(t *testing.T) {
	const duration = 2 * time.Second

	repo, cleanup := getRepo(t)
	defer cleanup()

	config := pomodoro.NewConfig(repo, duration, duration, duration)

	testCases := []struct {
		name        string
		cancel      bool
		expState    int
		expDuration time.Duration
	}{
		{name: "Finish", cancel: false, expState: pomodoro.StateDone, expDuration: duration},
		{name: "Cancel", cancel: true, expState: pomodoro.StateCancelled, expDuration: duration / 2},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())

			i, err := pomodoro.GetInterval(config)
			if err != nil {
				t.Fatal(err)
			}

			start := func(i pomodoro.Interval) {
				if i.State != pomodoro.StateRunning {
					t.Errorf("expected state %d, got %d", pomodoro.StateRunning, i.State)
				}
				if i.ActualDuration >= i.PlannedDuration {
					t.Errorf("expected actualDuration %q, less than pllaned %q", i.ActualDuration, i.PlannedDuration)
				}
			}

			end := func(i pomodoro.Interval) {
				if i.State != tt.expState {
					t.Errorf("expected state %d, got %d", tt.expState, i.State)
				}
				if tt.cancel {
					t.Errorf("End callback should not be executed")
				}
			}

			periodic := func(i pomodoro.Interval) {
				if i.State != pomodoro.StateRunning {
					t.Errorf("expected state %d, got %d", pomodoro.StateRunning, i.State)
				}
				if tt.cancel {
					cancel()
				}
			}

			if err := i.Start(ctx, config, start, periodic, end); err != nil {
				t.Fatal(err)
			}
			i, err = repo.ByID(i.ID)
			if err != nil {
				t.Fatal(err)
			}
			if i.State != tt.expState {
				t.Errorf("Expected state %d, got %d.\n",
					tt.expState, i.State)
			}
			if i.ActualDuration != tt.expDuration {
				t.Errorf("Expected ActualDuration %q, got %q.\n",
					tt.expDuration, i.ActualDuration)
			}
			cancel()
		})
	}
}

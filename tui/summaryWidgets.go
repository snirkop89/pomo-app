package tui

import (
	"context"
	"math"
	"time"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/widgets/barchart"
	"github.com/mum4k/termdash/widgets/linechart"
	"github.com/snirkop89/pomo/pomodoro"
)

type summary struct {
	bcDay        *barchart.BarChart
	lcWeekly     *linechart.LineChart
	updateDaily  chan bool
	updateWeekey chan bool
}

func (s *summary) update(redrawCh chan<- bool) {
	s.updateDaily <- true
	s.updateWeekey <- true
	redrawCh <- true
}

func newSummary(ctx context.Context, config *pomodoro.IntervalConfig, redrawCh chan<- bool, errorCh chan<- error) (*summary, error) {
	var s summary
	var err error

	s.updateDaily = make(chan bool)
	s.updateWeekey = make(chan bool)

	s.bcDay, err = newBarChar(ctx, config, s.updateDaily, errorCh)
	if err != nil {
		return nil, err
	}

	s.lcWeekly, err = newLineChart(ctx, config, s.updateWeekey, errorCh)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func newBarChar(ctx context.Context, config *pomodoro.IntervalConfig, update <-chan bool, errorCh chan<- error) (*barchart.BarChart, error) {
	bc, err := barchart.New(
		barchart.ShowValues(),
		barchart.BarColors([]cell.Color{
			cell.ColorBlue,
			cell.ColorYellow,
		}),
		barchart.ValueColors([]cell.Color{
			cell.ColorBlack,
			cell.ColorBlack,
		}),
		barchart.Labels([]string{
			"Pomodoro",
			"Break",
		}),
	)
	if err != nil {
		return nil, err
	}

	updateWidget := func() error {
		ds, err := pomodoro.DailySummary(time.Now(), config)
		if err != nil {
			return err
		}

		return bc.Values(
			[]int{
				int(ds[0].Minutes()),
				int(ds[1].Minutes()),
			},
			int(math.Max(ds[0].Minutes(), ds[1].Minutes())*1.1)+1,
		)
	}

	go func() {
		for {
			select {
			case <-update:
				errorCh <- updateWidget()
			case <-ctx.Done():
				return
			}
		}
	}()

	if err := updateWidget(); err != nil {
		return nil, err
	}
	return bc, nil
}

func newLineChart(ctx context.Context, config *pomodoro.IntervalConfig, update <-chan bool, errorCh chan<- error) (*linechart.LineChart, error) {
	// Initialize LineChart

	lc, err := linechart.New(
		linechart.AxesCellOpts(cell.FgColor(cell.ColorRed)),
		linechart.YLabelCellOpts(cell.FgColor(cell.ColorBlue)),
		linechart.XLabelCellOpts(cell.FgColor(cell.ColorCyan)),
		linechart.YAxisFormattedValues(
			linechart.ValueFormatterSingleUnitDuration(time.Second, 0),
		),
	)
	if err != nil {
		return nil, err
	}

	updateWidget := func() error {
		ws, err := pomodoro.RangeSummary(time.Now(), 7, config)
		if err != nil {
			return err
		}

		err = lc.Series(ws[0].Name, ws[0].Values,
			linechart.SeriesCellOpts(cell.FgColor(cell.ColorBlue)),
			linechart.SeriesXLabels(ws[0].Labels),
		)
		if err != nil {
			return err
		}

		return lc.Series(ws[1].Name, ws[1].Values,
			linechart.SeriesCellOpts(cell.FgColor(cell.ColorYellow)),
			linechart.SeriesXLabels(ws[1].Labels),
		)
	}

	go func() {
		for {
			select {
			case <-update:
				errorCh <- updateWidget()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Force update lineChart at start
	if err := updateWidget(); err != nil {
		return nil, err
	}
	return lc, nil
}

package tui

import (
	"context"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/widgets/donut"
	"github.com/mum4k/termdash/widgets/segmentdisplay"
	"github.com/mum4k/termdash/widgets/text"
)

type widgets struct {
	donTimer       *donut.Donut
	disType        *segmentdisplay.SegmentDisplay
	txtInfo        *text.Text
	txtTimer       *text.Text
	updateDonTimer chan []int
	updateTxtInfo  chan string
	updateTxtTimer chan string
	updateTxtType  chan string
}

func newWidgets(ctx context.Context, errorCh chan<- error) (*widgets, error) {
	donTimerCh := make(chan []int)
	txtTypeCh := make(chan string)
	txtInfoCh := make(chan string)
	txtTimerCh := make(chan string)

	donTimer, err := newDonut(ctx, donTimerCh, errorCh)
	if err != nil {
		return nil, err
	}

	disType, err := newSegmentDisplay(ctx, txtTypeCh, errorCh)
	if err != nil {
		return nil, err
	}

	txtInfo, err := newText(ctx, txtInfoCh, errorCh)
	if err != nil {
		return nil, err
	}
	txtTimer, err := newText(ctx, txtTimerCh, errorCh)
	if err != nil {
		return nil, err
	}

	return &widgets{
		donTimer:       donTimer,
		disType:        disType,
		txtInfo:        txtInfo,
		txtTimer:       txtTimer,
		updateDonTimer: donTimerCh,
		updateTxtInfo:  txtInfoCh,
		updateTxtTimer: txtTimerCh,
		updateTxtType:  txtTypeCh,
	}, nil
}

func newText(ctx context.Context, updateText <-chan string, errorCh chan<- error) (*text.Text, error) {
	txt, err := text.New()
	if err != nil {
		return nil, err
	}

	// Goroutine to update text
	go func() {
		for {
			select {
			case t := <-updateText:
				txt.Reset()
				errorCh <- txt.Write(t)
			case <-ctx.Done():
				return
			}
		}
	}()

	return txt, nil
}

func newDonut(ctx context.Context, donUpdater <-chan []int, errorCh chan<- error) (*donut.Donut, error) {
	don, err := donut.New(donut.Clockwise(), donut.CellOpts(cell.FgColor(cell.ColorBlue)))
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case d := <-donUpdater:
				if d[0] <= d[1] {
					errorCh <- don.Absolute(d[0], d[1])
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return don, nil
}

func newSegmentDisplay(ctx context.Context, updateText <-chan string, errorCh chan<- error) (*segmentdisplay.SegmentDisplay, error) {
	sd, err := segmentdisplay.New()
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case t := <-updateText:
				if t == "" {
					t = " "
				}
				errorCh <- sd.Write([]*segmentdisplay.TextChunk{
					segmentdisplay.NewChunk(t),
				})
			case <-ctx.Done():
				return
			}
		}
	}()
	return sd, nil
}

func (w *widgets) update(timer []int, txtType, txtInfo, txtTimer string, redrawCh chan<- bool) {
	if txtInfo != "" {
		w.updateTxtInfo <- txtInfo
	}
	if txtType != "" {
		w.updateTxtType <- txtType
	}
	if txtTimer != "" {
		w.updateTxtTimer <- txtTimer
	}
	if len(timer) > 0 {
		w.updateDonTimer <- timer
	}
	redrawCh <- true
}

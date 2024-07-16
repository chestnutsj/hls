package display

import (
	"os"
	"time"

	isatty "github.com/mattn/go-isatty"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

func isTerminal(fd uintptr) bool {
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

var Disable = false

func init() {
	//Disable = !isTerminal(os.Stdout.Fd())
}

type Display struct {
	p *mpb.Progress
}

func NewDisplay() *Display {

	// passed wg will be accounted at p.Wait() call
	p := mpb.New(
		mpb.WithOutput(os.Stdout),
		mpb.WithAutoRefresh())

	return &Display{
		p: p,
	}

}

func (d *Display) AddBar(name string, total int64, desc string) *mpb.Bar {
	//
	if Disable {
		return nil
	}

	bar := d.p.AddBar(int64(total),
		mpb.PrependDecorators(
			// simple name decorator
			decor.Name(name),
			// decor.DSyncWidth bit enables column width synchronization
			decor.Counters(decor.SizeB1024(0), " [% .2f / % .2f]"),
		),
		mpb.AppendDecorators(
			decor.OnComplete(
				// ETA decorator with ewma age of 30
				decor.EwmaETA(decor.ET_STYLE_GO, 30, decor.WCSyncWidth), desc,
			),
			decor.EwmaSpeed(decor.SizeB1024(0), " [% .2f]", 60, decor.WC{W: 14}),
			decor.Percentage(decor.WCSyncSpace),
		),
	)

	return bar
}

func InCr(by *mpb.Bar, incr int, t time.Duration) {
	if by == nil {
		return
	}
	by.EwmaIncrBy(incr, t)
}

func DynInrTotal(by *mpb.Bar) {
	if by == nil {
		return
	}
	by.SetTotal(by.Current()+2048, false)
}

func DynComplete(by *mpb.Bar) {
	if by == nil {
		return
	}
	by.SetTotal(-1, true)
}

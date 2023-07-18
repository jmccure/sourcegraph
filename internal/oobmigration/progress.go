package oobmigration

import (
	"fmt"
	"os"
	"time"

	"github.com/sourcegraph/log"

	"github.com/sourcegraph/sourcegraph/lib/output"
)

const MaxTime = 60 * 60 * 10

// makeOutOfBandMigrationProgressUpdater returns a two functions: `update` should be called
// when the updates to the progress of an out-of-band migration are made and should be reflected
// in the output; and `cleanup` should be called on defer when the progress object should be
// disposed.
func MakeProgressUpdater(out *output.Output, ids []int, animateProgress bool) (
	update func(i int, m Migration),
	cleanup func(),
) {
	if !animateProgress || shouldDisableProgressAnimation() {
		var percentages []float64

		update = func(i int, m Migration) {
			percentages = append(percentages, m.Progress*100)
			if len(percentages) > MaxTime {
				percentages = percentages[len(percentages)-MaxTime:]
			}

			// jesus christ take a stats class
			n := len(percentages)
			diff := percentages[len(percentages)-1] - percentages[0]
			remainingPercentage := 100 - (m.Progress * 100)
			remainingTime := time.Duration(remainingPercentage/diff*float64(n)) * time.Second

			out.WriteLine(output.Linef("", output.StyleReset,
				"Migration #%d is %.8f%% complete (%s remains, about %3d days, est from %.8f%% over %ds)",
				m.ID,
				m.Progress*100,
				fmt.Sprintf("%15s", remainingTime),
				int(remainingTime/time.Hour)/24,
				diff,
				n,
			))
		}
		return update, func() {}
	}

	bars := make([]output.ProgressBar, 0, len(ids))
	for _, id := range ids {
		bars = append(bars, output.ProgressBar{
			Label: fmt.Sprintf("Migration #%d", id),
			Max:   1.0,
		})
	}

	progress := out.Progress(bars, nil)
	return func(i int, m Migration) { progress.SetValue(i, m.Progress) }, progress.Destroy
}

// shouldDisableProgressAnimation determines if progress bars should be avoided because the log level
// will create output that interferes with a stable canvas. In effect, this adds the -disable-animation
// flag when SRC_LOG_LEVEL is info or debug.
func shouldDisableProgressAnimation() bool {
	switch log.Level(os.Getenv(log.EnvLogLevel)) {
	case log.LevelDebug:
		return true
	case log.LevelInfo:
		return true

	default:
		return false
	}
}

package main

import (
	"time"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

func (j *Job) newProgressWithCtx() *mpb.Progress {
	return mpb.NewWithContext(
		j.ctx,
		RefreshRate,
	)
}

// newWritingBar 写入进度条
func (j *Job) newWritingBar() *mpb.Bar {
	bar := j.progress.New(int64(j.size),
		BarStyleMain,
		mpb.PrependDecorators(
			decor.OnComplete(decor.Name("Writing: "), "Written: "),
			decor.OnComplete(decor.CountersKibiByte("% .2f / % .2f"), FormatBytes(j.size)),
			decor.OnComplete(decor.NewPercentage(" [%.1f]"), ""),
		),
	)
	bar.SetPriority(0)
	return bar
}

// newTotalBar 总下载进度条
func (j *Job) newTotalBar() *mpb.Bar {
	bar := j.progress.New(int64(len(j.Blocks)),
		BarStyleMain,
		mpb.PrependDecorators(
			Spinner,
			ETA,
		),
	)
	bar.SetPriority(1)
	return bar
}

// newThreadBar 线程进度条
func (j *Job) newThreadBar(block *Block) *mpb.Bar {
	bar := j.progress.New(int64(block.end-block.start+1),
		BarStyleSecondary,
		mpb.AppendDecorators(
			decor.NewPercentage("%.1f"),
		),
		mpb.BarRemoveOnComplete(),
	)
	bar.SetPriority(2 + block.index)
	return bar
}

// newUnknownSizeBar 未知文件大小进度条
func (j *Job) newUnknownSizeBar() *mpb.Bar {
	bar := j.progress.New(0,
		BarStyleMain,
		mpb.PrependDecorators(
			Spinner,
		),
	)
	return bar
}

var RefreshRate = mpb.WithRefreshRate(100 * time.Millisecond)
var BarStyleMain = mpb.BarStyle().Lbound("⡇").Filler("⣿").Rbound("⢸").Tip("⡇", "⣇", "⡧", "⡗", "⡏").Padding("+")
var BarStyleSecondary = mpb.BarStyle().Lbound("[").Filler("=").Rbound("]").Tip(">").Padding(" ")

var Spinner = decor.OnComplete(SpinnerProgress, SpinnerComplete)
var ETA = decor.OnComplete(decor.AverageETA(decor.ET_STYLE_GO, decor.WC{C: decor.DextraSpace}), "DONE") // with Bar.EwmaIncrement

var SpinnerComplete = "⣏⣹"
var SpinnerProgress = decor.Spinner(Spinners)
var Spinners = []string{
	`⠁⠀`,
	`⠈⠁`, `⠀⠙`, `⠀⢸`, `⢀⣰`, `⣄⣠`,
	`⣇⣀`, `⣏⡁`, `⣏⠙`, `⡏⢹`, `⢏⣹`,
	`⣏⣹`,
	`⣏⣩`, `⣏⡙`, `⡏⠹`, `⠋⢹`, `⠈⣹`,
	`⢀⣸`, `⣀⣠`, `⣄⡀`, `⡆⠀`, `⠃⠀`,

	`⠈⠀`,
	`⠀⠉`, `⠀⠸`, `⠀⣰`, `⣀⣠`, `⣆⣀`,
	`⣏⡀`, `⣏⠉`, `⡏⠹`, `⠏⣹`, `⣋⣹`,
	`⣏⣹`,
	`⣏⣙`, `⣏⠹`, `⠏⢹`, `⠉⣹`, `⢀⣹`,
	`⣀⣰`, `⣄⣀`, `⣆⠀`, `⠇⠀`, `⠉⠀`,

	`⠀⠁`,
	`⠀⠘`, `⠀⢰`, `⢀⣠`, `⣄⣀`, `⣇⡀`,
	`⣏⠁`, `⡏⠙`, `⠏⢹`, `⢋⣹`, `⣍⣹`,
	`⣏⣹`,
	`⣏⡹`, `⡏⢹`, `⠋⣹`, `⢈⣹`, `⣀⣸`,
	`⣄⣠`, `⣆⡀`, `⡇⠀`, `⠋⠀`, `⠈⠁`,

	`⠀⠈`,
	`⠀⠰`, `⠀⣠`, `⣀⣀`, `⣆⡀`, `⣏⠀`,
	`⡏⠉`, `⠏⠹`, `⠋⣹`, `⣉⣹`, `⣎⣹`,
	`⣏⣹`,
	`⣏⢹`, `⠏⣹`, `⢉⣹`, `⣀⣹`, `⣄⣰`,
	`⣆⣀`, `⣇⠀`, `⠏⠀`, `⠉⠁`, `⠀⠉`,

	`⠀⠐`,
	`⠀⢠`, `⢀⣀`, `⣄⡀`, `⣇⠀`, `⡏⠁`,
	`⠏⠙`, `⠋⢹`, `⢉⣹`, `⣌⣹`, `⣇⣹`,
	`⣏⣹`,
	`⡏⣹`, `⢋⣹`, `⣈⣹`, `⣄⣸`, `⣆⣠`,
	`⣇⡀`, `⡏⠀`, `⠋⠁`, `⠈⠉`, `⠀⠘`,

	`⠀⠠`,
	`⠀⣀`, `⣀⡀`, `⣆⠀`, `⡏⠀`, `⠏⠉`,
	`⠋⠹`, `⠉⣹`, `⣈⣹`, `⣆⣹`, `⣏⣸`,
	`⣏⣹`,
	`⢏⣹`, `⣉⣹`, `⣄⣹`, `⣆⣰`, `⣇⣀`,
	`⣏⠀`, `⠏⠁`, `⠉⠉`, `⠀⠙`, `⠀⠰`,

	`⠀⢀`,
	`⢀⡀`, `⣄⠀`, `⡇⠀`, `⠏⠁`, `⠋⠙`,
	`⠉⢹`, `⢈⣹`, `⣄⣹`, `⣇⣸`, `⣏⣱`,
	`⣏⣹`,
	`⣋⣹`, `⣌⣹`, `⣆⣸`, `⣇⣠`, `⣏⡀`,
	`⡏⠁`, `⠋⠉`, `⠈⠙`, `⠀⠸`, `⠀⢠`,

	`⠀⡀`,
	`⣀⠀`, `⡆⠀`, `⠏⠀`, `⠋⠉`, `⠉⠹`,
	`⠈⣹`, `⣀⣹`, `⣆⣸`, `⣏⣰`, `⣏⣩`,
	`⣏⣹`,
	`⣍⣹`, `⣆⣹`, `⣇⣰`, `⣏⣀`, `⣏⠁`,
	`⠏⠉`, `⠉⠙`, `⠀⠹`, `⠀⢰`, `⠀⣀`,

	`⢀⠀`,
	`⡄⠀`, `⠇⠀`, `⠋⠁`, `⠉⠙`, `⠈⢹`,
	`⢀⣹`, `⣄⣸`, `⣇⣰`, `⣏⣡`, `⣏⣙`,
	`⣏⣹`,
	`⣎⣹`, `⣇⣸`, `⣏⣠`, `⣏⡁`, `⡏⠉`,
	`⠋⠙`, `⠈⠹`, `⠀⢸`, `⠀⣠`, `⢀⡀`,

	`⡀⠀`,
	`⠆⠀`, `⠋⠀`, `⠉⠉`, `⠈⠹`, `⠀⣹`,
	`⣀⣸`, `⣆⣰`, `⣏⣠`, `⣏⣉`, `⣏⡹`,
	`⣏⣹`,
	`⣇⣹`, `⣏⣰`, `⣏⣁`, `⣏⠉`, `⠏⠙`,
	`⠉⠹`, `⠀⢹`, `⠀⣰`, `⢀⣀`, `⣀⠀`,

	`⠄⠀`,
	`⠃⠀`, `⠉⠁`, `⠈⠙`, `⠀⢹`, `⢀⣸`,
	`⣄⣰`, `⣇⣠`, `⣏⣁`, `⣏⡙`, `⣏⢹`,
	`⣏⣹`,
	`⣏⣸`, `⣏⣡`, `⣏⡉`, `⡏⠙`, `⠋⠹`,
	`⠈⢹`, `⠀⣸`, `⢀⣠`, `⣀⡀`, `⡄⠀`,

	`⠂⠀`,
	`⠉⠀`, `⠈⠉`, `⠀⠹`, `⠀⣸`, `⣀⣰`,
	`⣆⣠`, `⣏⣀`, `⣏⡉`, `⣏⠹`, `⡏⣹`,
	`⣏⣹`,
	`⣏⣱`, `⣏⣉`, `⣏⠙`, `⠏⠹`, `⠉⢹`,
	`⠀⣹`, `⢀⣰`, `⣀⣀`, `⣄⠀`, `⠆⠀`,
}

// Package system implements the System Stats module: CPU, memory and disk
// utilization of the host machine, sampled on a background ticker.
package system

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"

	"framego/engine"
	"framego/fonts"
	"framego/modules"
	"framego/modules/opt"
	"framego/render"
)

// System renders host utilization rows with mini progress bars.
type System struct {
	interval time.Duration
	diskPath string
	showCPU  bool
	showMem  bool
	showDisk bool

	cpuColor   color.RGBA
	memColor   color.RGBA
	diskColor  color.RGBA
	labelColor color.RGBA
	barTrack   color.RGBA

	done chan struct{}
	wg   sync.WaitGroup

	mu       sync.Mutex
	cpuPct   float64
	memUsed  uint64
	memTotal uint64
	diskUsed uint64
	diskTot  uint64
	lastErr  error
}

// New constructs a system stats module.
func New() engine.Module { return &System{} }

func init() { modules.Register("system", New) }

// Name identifies the module.
func (s *System) Name() string { return "system" }

// Configure applies module options.
func (s *System) Configure(opts map[string]any) error {
	s.interval = opt.Duration(opts, "interval", 5)
	s.diskPath = opt.Str(opts, "diskPath", "/")
	s.showCPU = opt.Bool(opts, "showCPU", true)
	s.showMem = opt.Bool(opts, "showMem", true)
	s.showDisk = opt.Bool(opts, "showDisk", true)

	s.cpuColor = opt.Color(opts, "cpuColor", color.RGBA{R: 87, G: 199, B: 172, A: 255})
	s.memColor = opt.Color(opts, "memColor", color.RGBA{R: 87, G: 160, B: 220, A: 255})
	s.diskColor = opt.Color(opts, "diskColor", color.RGBA{R: 220, G: 190, B: 100, A: 255})
	s.labelColor = opt.Color(opts, "labelColor", color.RGBA{R: 154, G: 167, B: 184, A: 255})
	s.barTrack = opt.Color(opts, "barColor", color.RGBA{R: 38, G: 48, B: 62, A: 255})

	if s.interval <= 0 {
		s.interval = time.Second
	}
	return nil
}

func (s *System) Schema() *engine.Schema {
	return &engine.Schema{
		Name:        "system",
		Description: "CPU, memory, and disk utilization stats",
		Fields: []engine.Field{
			{Key: "interval", Label: "Sample Interval (s)", Kind: engine.FieldDuration, Default: "5", Hint: "Seconds between system samples"},
			{Key: "diskPath", Label: "Disk Path", Kind: engine.FieldText, Default: "/", Placeholder: "/"},
			{Key: "showCPU", Label: "Show CPU", Kind: engine.FieldBoolean, Default: "true"},
			{Key: "showMem", Label: "Show Memory", Kind: engine.FieldBoolean, Default: "true"},
			{Key: "showDisk", Label: "Show Disk", Kind: engine.FieldBoolean, Default: "true"},
			{Key: "cpuColor", Label: "CPU Color", Kind: engine.FieldColor, Default: "#57c7ac"},
			{Key: "memColor", Label: "Memory Color", Kind: engine.FieldColor, Default: "#57a0dc"},
			{Key: "diskColor", Label: "Disk Color", Kind: engine.FieldColor, Default: "#dcbe64"},
			{Key: "labelColor", Label: "Label Color", Kind: engine.FieldColor, Default: "#9aa7b8"},
			{Key: "barColor", Label: "Bar Track Color", Kind: engine.FieldColor, Default: "#26303e"},
		},
	}
}

// Start launches the background sampling ticker.
func (s *System) Start(_ *engine.Bus, log *engine.Logger) error {
	if s.interval <= 0 {
		s.interval = time.Second
	}
	s.done = make(chan struct{})
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.refresh()
		t := time.NewTicker(s.interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				if err := s.refresh(); err != nil && s.lastErr == nil {
					log.Errorf("system: %v", err)
				}
			case <-s.done:
				return
			}
		}
	}()
	return nil
}

// Stop halts the sampling ticker.
func (s *System) Stop() error {
	if s.done != nil {
		close(s.done)
		s.wg.Wait()
	}
	return nil
}

// refresh samples CPU, memory and disk, guarding the shared snapshot.
func (s *System) refresh() error {
	ctx := context.Background()
	per, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return fmt.Errorf("cpu: %w", err)
	}
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return fmt.Errorf("mem: %w", err)
	}
	du, err := disk.UsageWithContext(ctx, s.diskPath)
	if err != nil {
		return fmt.Errorf("disk %q: %w", s.diskPath, err)
	}

	var cp float64
	if len(per) > 0 {
		cp = per[0]
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cpuPct = cp
	s.memUsed = vm.Used
	s.memTotal = vm.Total
	s.diskUsed = du.Used
	s.diskTot = du.Total
	s.lastErr = nil
	return nil
}

// Draw renders utilization rows, or a collecting placeholder before the first
// successful sample.
func (s *System) Draw(cv *render.Canvas, bounds image.Rectangle, _ time.Time) error {
	s.mu.Lock()
	rows := make([]row, 0, 3)
	if s.showCPU {
		rows = append(rows, row{label: "CPU", pct: s.cpuPct, detail: fmt.Sprintf("%.0f%%", s.cpuPct), col: s.cpuColor})
	}
	if s.showMem {
		rows = append(rows, row{label: "MEM", pct: pctOf(s.memUsed, s.memTotal), detail: fmt.Sprintf("%s / %s", bytesFmt(s.memUsed), bytesFmt(s.memTotal)), col: s.memColor})
	}
	if s.showDisk {
		rows = append(rows, row{label: "DISK", pct: pctOf(s.diskUsed, s.diskTot), detail: fmt.Sprintf("%s / %s", bytesFmt(s.diskUsed), bytesFmt(s.diskTot)), col: s.diskColor})
	}
	lastErr := s.lastErr
	s.mu.Unlock()

	if s.memTotal == 0 && s.diskTot == 0 {
		return drawCollecting(cv, bounds, lastErr)
	}

	lf := fonts.Scaled(bounds, 15, fonts.Medium)
	df := fonts.Scaled(bounds, 13, fonts.Regular)
	rowH := 30
	labelW := 52
	barH := 8
	barW := bounds.Dx() - labelW - 120
	if barW < 40 {
		barW = 40
	}
	startX := bounds.Min.X + (bounds.Dx()-labelW-barW-120)/2
	if startX < bounds.Min.X {
		startX = bounds.Min.X
	}
	y := bounds.Min.Y + (bounds.Dy()-rowH*len(rows))/2

	ascent := cv.Ascent(lf)
	_, _, dfH := cv.FaceMetrics(df)
	for _, r := range rows {
		cv.DrawText(image.Pt(startX, y+ascent), r.label, lf, s.labelColor)
		barTop := y + (rowH-barH)/2
		track := image.Rect(startX+labelW, barTop, startX+labelW+barW, barTop+barH)
		cv.FillRoundRect(track, barH/2, s.barTrack)
		if r.pct > 0 {
			fillW := int(float64(track.Dx()) * r.pct / 100)
			if fillW > 0 {
				fill := image.Rect(track.Min.X, track.Min.Y, track.Min.X+fillW, track.Max.Y)
				cv.FillRoundRect(fill, barH/2, r.col)
			}
		}
		cv.DrawText(image.Pt(startX+labelW+barW+10, y+ascent+(rowH-dfH)/2), r.detail, df, s.labelColor)
		y += rowH
	}
	return nil
}

type row struct {
	label  string
	pct    float64
	detail string
	col    color.RGBA
}

func pctOf(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(used) / float64(total) * 100
}

func bytesFmt(b uint64) string {
	const gb = 1024 * 1024 * 1024
	if b >= gb {
		return fmt.Sprintf("%.1fG", float64(b)/gb)
	}
	return fmt.Sprintf("%dM", b/(1024*1024))
}

// drawCollecting paints a placeholder before the first sample arrives.
func drawCollecting(cv *render.Canvas, bounds image.Rectangle, err error) error {
	f := fonts.Scaled(bounds, 16, fonts.Regular)
	msg := "system: collecting…"
	if err != nil {
		msg = "system: unavailable"
	}
	w, _ := cv.TextSize(f, msg)
	ascent := cv.Ascent(f)
	cv.DrawText(image.Pt(bounds.Min.X+(bounds.Dx()-w)/2, bounds.Min.Y+(bounds.Dy()-ascent)/2), msg, f, color.RGBA{R: 154, G: 167, B: 184, A: 255})
	return nil
}

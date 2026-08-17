// Package weather implements the Weather module using the Open-Meteo API,
// which requires no API key. Current conditions and a short forecast are
// fetched on a background ticker; Draw renders the latest snapshot without
// blocking on the network.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"framego/engine"
	"framego/fonts"
	"framego/modules"
	"framego/modules/opt"
	"framego/render"
)

// Open-Meteo endpoints. Overridable on the module for tests.
const (
	defaultGeocodeURL  = "https://geocoding-api.open-meteo.com/v1/search"
	defaultForecastURL = "https://api.open-meteo.com/v1/forecast"
)

// Weather renders current conditions and a short forecast.
type Weather struct {
	city        string
	units       string
	forecastDay int
	interval    time.Duration
	geoURL      string
	forecastURL string

	tempColor  color.RGBA
	condColor  color.RGBA
	hiLoColor  color.RGBA
	locColor   color.RGBA
	dayColor   color.RGBA

	client *http.Client

	done chan struct{}
	wg   sync.WaitGroup

	mu       sync.Mutex
	loc      string
	snap     *snapshot
	lastErr  error
	fetched  bool
}

type snapshot struct {
	cond       string
	temp       float64
	feels      float64
	humidity   int
	wind       float64
	hi         float64
	lo         float64
	days       []dayInfo
	location   string
}

type dayInfo struct {
	label string
	cond  string
	hi    float64
	lo    float64
}

// New constructs a weather module.
func New() engine.Module { return &Weather{} }

func init() { modules.Register("weather", New) }

// Name identifies the module.
func (w *Weather) Name() string { return "weather" }

// Configure applies module options.
func (w *Weather) Configure(opts map[string]any) error {
	w.city = opt.Str(opts, "city", "")
	if w.city == "" {
		return fmt.Errorf("weather: option \"city\" is required")
	}
	w.units = opt.Str(opts, "units", "metric")
	if w.units != "metric" && w.units != "imperial" {
		w.units = "metric"
	}
	w.forecastDay = opt.Int(opts, "forecastDays", 5)
	if w.forecastDay < 1 {
		w.forecastDay = 1
	}
	if w.forecastDay > 7 {
		w.forecastDay = 7
	}
	w.interval = opt.Duration(opts, "update", 600)
	if w.interval <= 0 {
		w.interval = time.Minute
	}
	w.geoURL = opt.Str(opts, "geoURL", defaultGeocodeURL)
	w.forecastURL = opt.Str(opts, "forecastURL", defaultForecastURL)
	if w.client == nil {
		w.client = &http.Client{Timeout: 15 * time.Second}
	}

	w.tempColor = opt.Color(opts, "tempColor", color.RGBA{R: 245, G: 247, B: 250, A: 255})
	w.condColor = opt.Color(opts, "condColor", color.RGBA{R: 154, G: 167, B: 184, A: 255})
	w.hiLoColor = opt.Color(opts, "hiLoColor", color.RGBA{R: 154, G: 167, B: 184, A: 255})
	w.locColor = opt.Color(opts, "locColor", color.RGBA{R: 154, G: 167, B: 184, A: 255})
	w.dayColor = opt.Color(opts, "dayColor", color.RGBA{R: 154, G: 167, B: 184, A: 255})
	return nil
}

func (w *Weather) Schema() *engine.Schema {
	return &engine.Schema{
		Name:        "weather",
		Description: "Current conditions and forecast via Open-Meteo (no API key needed)",
		Fields: []engine.Field{
			{Key: "city", Label: "City", Kind: engine.FieldText, Required: true, Placeholder: "New York", Hint: "City name for geocoding lookup"},
			{Key: "units", Label: "Units", Kind: engine.FieldSelect, Default: "metric", Options: []string{"metric", "imperial"}},
			{Key: "forecastDays", Label: "Forecast Days", Kind: engine.FieldNumber, Default: "5", Min: 1, Max: 7},
			{Key: "update", Label: "Update Interval (s)", Kind: engine.FieldDuration, Default: "600", Hint: "Seconds between API refreshes"},
			{Key: "tempColor", Label: "Temperature Color", Kind: engine.FieldColor, Default: "#f5f7fa"},
			{Key: "condColor", Label: "Condition Color", Kind: engine.FieldColor, Default: "#9aa7b8"},
			{Key: "hiLoColor", Label: "Hi/Lo Color", Kind: engine.FieldColor, Default: "#9aa7b8"},
			{Key: "locColor", Label: "Location Color", Kind: engine.FieldColor, Default: "#9aa7b8"},
			{Key: "dayColor", Label: "Day Label Color", Kind: engine.FieldColor, Default: "#9aa7b8"},
		},
	}
}

// Start launches the background fetch loop.
func (w *Weather) Start(_ *engine.Bus, log *engine.Logger) error {
	w.done = make(chan struct{})
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		if err := w.fetch(); err != nil {
			w.setErr(err)
			log.Errorf("weather: %v", err)
		}
		t := time.NewTicker(w.interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				if err := w.fetch(); err != nil {
					w.setErr(err)
					log.Errorf("weather: %v", err)
				}
			case <-w.done:
				return
			}
		}
	}()
	return nil
}

// Stop halts the background fetch loop.
func (w *Weather) Stop() error {
	if w.done != nil {
		close(w.done)
		w.wg.Wait()
	}
	return nil
}

func (w *Weather) setErr(err error) {
	w.mu.Lock()
	w.lastErr = err
	w.mu.Unlock()
}

// fetch resolves the city and pulls the forecast into a snapshot.
func (w *Weather) fetch() error {
	loc, lat, lon, err := w.geocode(context.Background())
	if err != nil {
		return err
	}
	snap, err := w.forecast(context.Background(), lat, lon)
	if err != nil {
		return err
	}
	snap.location = loc

	w.mu.Lock()
	defer w.mu.Unlock()
	w.loc = loc
	w.snap = snap
	w.lastErr = nil
	w.fetched = true
	return nil
}

// geocode resolves cityName into a display name and coordinates.
func (w *Weather) geocode(ctx context.Context) (string, float64, float64, error) {
	u, err := url.Parse(w.geoURL)
	if err != nil {
		return "", 0, 0, err
	}
	q := u.Query()
	q.Set("name", w.city)
	q.Set("count", "1")
	q.Set("language", "en")
	q.Set("format", "json")
	u.RawQuery = q.Encode()

	var resp geoResp
	if err := w.getJSON(ctx, u.String(), &resp); err != nil {
		return "", 0, 0, err
	}
	if len(resp.Results) == 0 {
		return "", 0, 0, fmt.Errorf("city %q not found", w.city)
	}
	r := resp.Results[0]
	name := r.Name
	if r.Admin1 != "" {
		name += ", " + r.Admin1
	}
	return name, r.Latitude, r.Longitude, nil
}

// forecast fetches current and daily conditions for the coordinates.
func (w *Weather) forecast(ctx context.Context, lat, lon float64) (*snapshot, error) {
	u, err := url.Parse(w.forecastURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("latitude", strconv.FormatFloat(lat, 'f', 5, 64))
	q.Set("longitude", strconv.FormatFloat(lon, 'f', 5, 64))
	q.Set("forecast_days", strconv.Itoa(w.forecastDay))
	q.Set("timezone", "auto")
	q.Set("current", "temperature_2m,relative_humidity_2m,apparent_temperature,weather_code,wind_speed_10m")
	q.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min")
	if w.units == "imperial" {
		q.Set("temperature_unit", "fahrenheit")
		q.Set("wind_speed_unit", "mph")
	} else {
		q.Set("temperature_unit", "celsius")
		q.Set("wind_speed_unit", "kmh")
	}
	u.RawQuery = q.Encode()

	var resp forecastResp
	if err := w.getJSON(ctx, u.String(), &resp); err != nil {
		return nil, err
	}
	snap := &snapshot{
		temp:     resp.Current.Temp,
		feels:    resp.Current.Feels,
		humidity: resp.Current.Humidity,
		wind:     resp.Current.Wind,
		cond:     codeLabel(resp.Current.Code),
	}
	if len(resp.Daily.Max) > 0 {
		snap.hi = resp.Daily.Max[0]
	}
	if len(resp.Daily.Min) > 0 {
		snap.lo = resp.Daily.Min[0]
	}
	for i := 0; i < len(resp.Daily.Dates) && i < w.forecastDay; i++ {
		hi, lo := resp.Daily.Max[i], resp.Daily.Min[i]
		label := ""
		if d, err := time.Parse("2006-01-02", resp.Daily.Dates[i]); err == nil {
			label = d.Format("Mon")
		}
		snap.days = append(snap.days, dayInfo{
			label: label,
			cond:  codeLabel(resp.Daily.Codes[i]),
			hi:    hi,
			lo:    lo,
		})
	}
	return snap, nil
}

// getJSON GETs u and decodes the JSON body into v.
func (w *Weather) getJSON(ctx context.Context, u string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", resp.Status, body)
	}
	return json.Unmarshal(body, v)
}

// Draw renders the latest conditions, or a placeholder before the first fetch.
func (w *Weather) Draw(cv *render.Canvas, bounds image.Rectangle, _ time.Time) error {
	w.mu.Lock()
	snap := w.snap
	loc := w.loc
	fetched := w.fetched
	w.mu.Unlock()

	if !fetched || snap == nil {
		msg := "weather: collecting…"
		if !fetched && w.snapshotErr() != "" {
			msg = "weather: unavailable"
		}
		return drawCenter(cv, bounds, msg, w.condColor)
	}

	locF := fonts.Face(15, fonts.Regular)
	locW, _ := cv.TextSize(locF, loc)
	ascentLoc := cv.Ascent(locF)

	tempF := fonts.Face(56, fonts.Medium)
	tempStr := fmt.Sprintf("%s°", w.formatTemp(snap.temp))
	tempW, tempH := cv.TextSize(tempF, tempStr)
	ascentTemp := cv.Ascent(tempF)

	condF := fonts.Face(18, fonts.Regular)
	feelsStr := fmt.Sprintf("%s · %s · %d%% · %s", snap.cond, w.formatTemp(snap.feels), snap.humidity, w.formatWind(snap.wind))
	feelsW, _ := cv.TextSize(condF, feelsStr)
	ascentFeels := cv.Ascent(condF)

	hiLoF := fonts.Face(16, fonts.Regular)
	hiLoStr := fmt.Sprintf("H %s   L %s", w.formatTemp(snap.hi), w.formatTemp(snap.lo))
	hiLoW, _ := cv.TextSize(hiLoF, hiLoStr)

	_, _, locH := cv.FaceMetrics(locF)
	_, _, condH := cv.FaceMetrics(condF)
	_, _, hiLoH := cv.FaceMetrics(hiLoF)
	spacing := 4
	blockH := locH + spacing + tempH + spacing + condH + spacing + hiLoH
	blockTop := bounds.Min.Y + (bounds.Dy()-blockH)/2

	// Forecast strip below, if there is room.
	var stripY int
	if bounds.Dy()-blockH > 46 && len(snap.days) > 1 {
		stripY = bounds.Min.Y + bounds.Dy() - 30
		w.drawDays(cv, bounds, stripY, snap.days)
	}

	y := blockTop
	cv.DrawText(image.Pt(bounds.Min.X+(bounds.Dx()-locW)/2, y+ascentLoc), loc, locF, w.locColor)
	y += locH + spacing
	cv.DrawText(image.Pt(bounds.Min.X+(bounds.Dx()-tempW)/2, y+ascentTemp), tempStr, tempF, w.tempColor)
	y += tempH + spacing
	cv.DrawText(image.Pt(bounds.Min.X+(bounds.Dx()-feelsW)/2, y+ascentFeels), feelsStr, condF, w.condColor)
	y += condH + spacing
	cv.DrawText(image.Pt(bounds.Min.X+(bounds.Dx()-hiLoW)/2, y+ascentFeels), hiLoStr, hiLoF, w.hiLoColor)
	return nil
}

// drawDays renders per-day forecast columns across the bottom of bounds.
func (w *Weather) drawDays(cv *render.Canvas, bounds image.Rectangle, y int, days []dayInfo) {
	f := fonts.Face(13, fonts.Regular)
	colW := bounds.Dx() / len(days)
	ascent := cv.Ascent(f)
	for i, d := range days {
		x := bounds.Min.X + i*colW
		cx := x + colW/2
		label := d.label
		if label == "" {
			label = "–"
		}
		dayW, _ := cv.TextSize(f, label)
		cv.DrawText(image.Pt(cx-dayW/2, y+ascent), label, f, w.dayColor)
		condW, _ := cv.TextSize(f, d.cond)
		cv.DrawText(image.Pt(cx-condW/2, y+ascent+16), d.cond, f, w.dayColor)
		hiLo := fmt.Sprintf("%s/%s", w.formatTemp(d.hi), w.formatTemp(d.lo))
		hiLoW, _ := cv.TextSize(f, hiLo)
		cv.DrawText(image.Pt(cx-hiLoW/2, y+ascent+32), hiLo, f, w.dayColor)
	}
}

func (w *Weather) formatTemp(v float64) string {
	if math.IsNaN(v) {
		return "–"
	}
	return strconv.FormatInt(int64(math.Round(v)), 10)
}

func (w *Weather) formatWind(v float64) string {
	if w.units == "imperial" {
		return fmt.Sprintf("%.0f mph", v)
	}
	return fmt.Sprintf("%.0f km/h", v)
}

func (w *Weather) snapshotErr() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lastErr == nil {
		return ""
	}
	return w.lastErr.Error()
}

// drawCenter renders a single centered status line.
func drawCenter(cv *render.Canvas, bounds image.Rectangle, msg string, col color.RGBA) error {
	f := fonts.Face(16, fonts.Regular)
	lines := render.WrapText(f, msg, bounds.Dx())
	_, _, lh := cv.FaceMetrics(f)
	ascent := cv.Ascent(f)
	y := bounds.Min.Y + (bounds.Dy()-len(lines)*lh)/2
	for _, line := range lines {
		w, _ := cv.TextSize(f, line)
		cv.DrawText(image.Pt(bounds.Min.X+(bounds.Dx()-w)/2, y+ascent), line, f, col)
		y += lh
	}
	return nil
}

// WMO weather-code mapping.
func codeLabel(code int) string {
	switch code {
	case 0:
		return "Clear"
	case 1:
		return "Mainly clear"
	case 2:
		return "Partly cloudy"
	case 3:
		return "Overcast"
	case 45, 48:
		return "Fog"
	case 51, 53, 55:
		return "Drizzle"
	case 56, 57:
		return "Freezing drizzle"
	case 61, 63, 65:
		return "Rain"
	case 66, 67:
		return "Freezing rain"
	case 71, 73, 75:
		return "Snow"
	case 77:
		return "Snow grains"
	case 80, 81, 82:
		return "Rain showers"
	case 85, 86:
		return "Snow showers"
	case 95:
		return "Thunderstorm"
	case 96, 99:
		return "Storm + hail"
	default:
		return "Unknown"
	}
}

type geoResp struct {
	Results []struct {
		Name      string  `json:"name"`
		Admin1    string  `json:"admin1"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"results"`
}

type forecastResp struct {
	Current struct {
		Temp   float64 `json:"temperature_2m"`
		Feels  float64 `json:"apparent_temperature"`
		Humidity int    `json:"relative_humidity_2m"`
		Wind   float64 `json:"wind_speed_10m"`
		Code   int     `json:"weather_code"`
	} `json:"current"`
	Daily struct {
		Dates []string  `json:"time"`
		Codes []int     `json:"weather_code"`
		Max   []float64 `json:"temperature_2m_max"`
		Min   []float64 `json:"temperature_2m_min"`
	} `json:"daily"`
}

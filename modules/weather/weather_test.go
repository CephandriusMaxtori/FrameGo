package weather

import (
	"image"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"framego/render"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/geo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"name":"Berlin","admin1":"Berlin","latitude":52.52,"longitude":13.41}]}`))
	})
	mux.HandleFunc("/fc", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"current":{"temperature_2m":21.3,"relative_humidity_2m":56,"apparent_temperature":20.9,"weather_code":2,"wind_speed_10m":11.2},"daily":{"time":["2026-08-15","2026-08-16"],"weather_code":[2,61],"temperature_2m_max":[24.1,19.5],"temperature_2m_min":[14.2,12.0]}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestConfigureRequiresCity(t *testing.T) {
	w := New().(*Weather)
	if err := w.Configure(nil); err == nil {
		t.Error("expected error when city missing")
	}
}

func TestFetch(t *testing.T) {
	srv := testServer(t)
	w := New().(*Weather)
	if err := w.Configure(map[string]any{"city": "Berlin"}); err != nil {
		t.Fatal(err)
	}
	w.geoURL = srv.URL + "/geo"
	w.forecastURL = srv.URL + "/fc"
	if err := w.fetch(); err != nil {
		t.Fatal(err)
	}
	w.mu.Lock()
	snap := w.snap
	fetched := w.fetched
	loc := w.loc
	w.mu.Unlock()
	if !fetched || snap == nil {
		t.Fatal("no snapshot after fetch")
	}
	if loc != "Berlin, Berlin" {
		t.Errorf("loc = %q", loc)
	}
	if snap.temp != 21.3 || snap.cond != "Partly cloudy" {
		t.Errorf("snap = %+v", snap)
	}
	if len(snap.days) != 2 || snap.days[1].hi != 19.5 {
		t.Errorf("days = %+v", snap.days)
	}
}

func TestFetchCityNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/geo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	w := New().(*Weather)
	if err := w.Configure(map[string]any{"city": "Atlantis"}); err != nil {
		t.Fatal(err)
	}
	w.geoURL = srv.URL + "/geo"
	w.forecastURL = srv.URL + "/fc"
	if err := w.fetch(); err == nil {
		t.Error("expected error for unknown city")
	}
}

func TestDrawAfterFetch(t *testing.T) {
	srv := testServer(t)
	w := New().(*Weather)
	if err := w.Configure(map[string]any{"city": "Berlin", "forecastDays": 2}); err != nil {
		t.Fatal(err)
	}
	w.geoURL = srv.URL + "/geo"
	w.forecastURL = srv.URL + "/fc"
	if err := w.fetch(); err != nil {
		t.Fatal(err)
	}
	cv := render.NewCanvas(260, 200)
	if err := w.Draw(cv, image.Rect(0, 0, 260, 200), time.Now()); err != nil {
		t.Fatal(err)
	}
	lit := 0
	for y := 0; y < 200; y += 2 {
		for x := 0; x < 260; x += 2 {
			if _, _, _, a := cv.Img.At(x, y).RGBA(); a > 0 {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Error("no text pixels rendered")
	}
}

func TestDrawBeforeFetch(t *testing.T) {
	w := New().(*Weather)
	if err := w.Configure(map[string]any{"city": "Berlin"}); err != nil {
		t.Fatal(err)
	}
	cv := render.NewCanvas(200, 60)
	if err := w.Draw(cv, image.Rect(0, 0, 200, 60), time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestCodeLabel(t *testing.T) {
	if codeLabel(0) != "Clear" || codeLabel(61) != "Rain" || codeLabel(95) != "Thunderstorm" {
		t.Error("code mapping broken")
	}
}

func TestFormatTempImperial(t *testing.T) {
	w := New().(*Weather)
	_ = w.Configure(map[string]any{"city": "Berlin", "units": "imperial"})
	if w.units != "imperial" {
		t.Fatal("units not imperial")
	}
	if got := w.formatTemp(72.4); got != "72" {
		t.Errorf("formatTemp = %q", got)
	}
	if got := w.formatWind(11.2); !strings.Contains(got, "mph") {
		t.Errorf("formatWind = %q", got)
	}
}

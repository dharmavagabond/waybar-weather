package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/carlmjohnson/requests"
	"github.com/tidwall/gjson"
	"github.com/ybbus/httpretry"
)

type WeatherSettings struct {
	Key           string `json:"key"`
	UseFahrenheit bool   `json:"use_fahrenheit"`
	IconPos       string `json:"icon_pos"`
	Parameters    string `json:"parameters"`
	OnlyIcon      bool   `json:"only_icon"`
	URL           string `json:"url"`
	Unit          string
}

type WeatherResponse struct {
	Current struct {
		IsDay     int     `json:"is_day"`
		TempC     float64 `json:"temp_c"`
		TempF     float64 `json:"temp_f"`
		Condition struct {
			Text string `json:"text"`
		} `json:"condition"`
	} `json:"current"`
}

type WaybarResponse struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
}

func init() {
	h := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(h)
	slog.SetDefault(logger)
}

func main() {
	jsonBytes, err := getWeather()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(string(jsonBytes))
}

func getWeather() ([]byte, error) {
	var (
		settings  *WeatherSettings
		weather   WeatherResponse
		jsonBytes []byte
		err       error
	)

	cl := httpretry.NewDefaultClient()
	settingsPath := flag.String("settings", "./weather-settings.json", "path/to/weather-settings.json")

	flag.Parse()

	if jsonBytes, err = readJSONFile(*settingsPath); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(jsonBytes, &settings); err != nil {
		slog.Error("Couldn't unmarshal the settings", slog.String("path", *settingsPath))
		return nil, err
	}

	settings.Unit = "°C"

	if settings.UseFahrenheit {
		settings.Unit = "°F"
	}

	if err = requests.URL(settings.URL).
		Client(cl).
		Param("key", settings.Key).
		Param("q", settings.Parameters).
		ToJSON(&weather).
		Fetch(context.Background()); err != nil {
		err = fmt.Errorf("[requests.Fetch]: %w", err)
		slog.Error("WeatherAPI", slog.String("%s", settings.URL), slog.String("%s", settings.Parameters))
		return nil, err
	}

	basedir := filepath.Dir(*settingsPath)
	iconsPath := path.Join(basedir, "weather-icons.json")

	if jsonBytes, err = readJSONFile(iconsPath); err != nil {
		return nil, err
	}

	icon := getIcon(jsonBytes, &weather)
	temp := weather.Current.TempC

	if settings.Unit == "°F" {
		temp = weather.Current.TempF
	}

	fmtTemp := fmt.Sprintf("%d%s", int(math.Round(temp)), settings.Unit)
	text := fmt.Sprintf("%s %s", fmtTemp, icon)

	if settings.IconPos == "left" {
		text = fmt.Sprintf("%s %s", icon, fmtTemp)
	}

	wr := WaybarResponse{
		text,
		text,
	}

	if settings.OnlyIcon && icon != "" {
		wr.Text = icon
	}

	if jsonBytes, err = json.Marshal(wr); err != nil {
		slog.Error("Couldn't marshal json", slog.String("text", wr.Text), slog.String("tooltip", wr.Tooltip))
		return nil, err
	}

	return jsonBytes, nil
}

func getIcon(icons []byte, weather *WeatherResponse) string {
	condition := strings.ToLower(weather.Current.Condition.Text)
	obj := gjson.GetBytes(icons, fmt.Sprintf("#(day==%s)", condition))

	if !obj.Exists() {
		return ""
	}

	iconKey := "icon"

	if weather.Current.IsDay == 0 {
		iconKey = "icon_night"
	}

	return obj.Get(iconKey).String()
}

func readJSONFile(f string) ([]byte, error) {
	var (
		file  *os.File
		bytes []byte
		err   error
	)

	if _, err = os.Stat(f); errors.Is(err, os.ErrNotExist) {
		slog.Error("File doesn't exist", slog.String("file", f))
		return nil, err
	}

	if file, err = os.Open(f); err != nil {
		slog.Error("Couldn't open file", slog.String("file", f))
		return nil, err
	}

	defer file.Close()

	if bytes, err = io.ReadAll(file); err != nil {
		slog.Error("Couldn't read file", slog.String("file", f))
		return nil, err
	}

	return bytes, nil
}

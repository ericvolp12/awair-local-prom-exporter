package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type App struct {
	ListenAddress     string
	ListenPort        uint64
	AwairAddress      string
	TimeBetweenChecks time.Duration
	TempGauge         prometheus.Gauge
	HumidityGauge     prometheus.Gauge
	Co2Gauge          prometheus.Gauge
	VOCGauge          prometheus.Gauge
	PM25Gauge         prometheus.Gauge
	Logger            *zap.SugaredLogger
}

type AwairStats struct {
	Timestamp      time.Time `json:"timestamp"`
	Score          int       `json:"score"`
	DewPoint       float64   `json:"dew_point"`
	Temp           float64   `json:"temp"`
	Humid          float64   `json:"humid"`
	AbsHumid       float64   `json:"abs_humid"`
	Co2            int       `json:"co2"`
	Co2Est         int       `json:"co2_est"`
	Co2EstBaseline int       `json:"co2_est_baseline"`
	Voc            int       `json:"voc"`
	VocBaseline    int       `json:"voc_baseline"`
	VocH2Raw       int       `json:"voc_h2_raw"`
	VocEthanolRaw  int       `json:"voc_ethanol_raw"`
	Pm25           int       `json:"pm25"`
	Pm10Est        int       `json:"pm10_est"`
}

func main() {

	rawLogger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("Failed to start logger: %+v", err))
	}

	sugaredLogger := rawLogger.Sugar()

	app := App{
		Logger: sugaredLogger,
	}

	// Initialize Flags for configuration
	listenAddress := flag.String("listen", "0.0.0.0", "Listen address")
	listenPort := flag.Uint64("port", 2112, "Listen port number")
	awairAddress := flag.String("awair_address", "http://localhost/air-data/latest", "Awair air-data URL")
	pollFrequency := flag.String("poll_frequency", "30s", "Time (seconds) to wait between polling device")

	flag.Parse()

	app.ListenAddress = *listenAddress
	app.ListenPort = *listenPort
	app.AwairAddress = *awairAddress

	// Parse time duration from poll frequency flag
	app.TimeBetweenChecks, err = time.ParseDuration(*pollFrequency)
	if err != nil {
		app.Logger.Fatalf("Couldn't parse duration from poll_frequency (%+v): %+v", *pollFrequency, err)
	}

	// Initialize the Prometheus Gauges
	app.initializeGauges()

	// Start the metrics recording goroutine
	app.recordMetrics()

	// Register the metrics handler
	http.Handle("/metrics", promhttp.Handler())

	listenString := fmt.Sprintf("%s:%d", app.ListenAddress, app.ListenPort)

	app.Logger.Infof("Awair Poller started on (%+v) polling Awair Device at (%+v) every (%+v)", listenString, app.AwairAddress, app.TimeBetweenChecks)

	err = http.ListenAndServe(listenString, nil)
	if err != nil {
		app.Logger.Fatalf("Failed to start server: %+v", err)
	}
}

func (app *App) initializeGauges() {
	tempGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "awair",
		Subsystem: "climate",
		Name:      "temp_c",
		Help:      "The current temperature in C",
	})

	humidityGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "awair",
		Subsystem: "climate",
		Name:      "relative_humidity",
		Help:      "The current % relative humidity",
	})

	co2Gauge := promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "awair",
		Subsystem: "climate",
		Name:      "co2_ppm",
		Help:      "The current C02 PPM",
	})

	vocGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "awair",
		Subsystem: "climate",
		Name:      "voc_ppb",
		Help:      "The current Volatile Organic Compound reading in parts per billion",
	})

	pm25Gauge := promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "awair",
		Subsystem: "climate",
		Name:      "pm25_ug_m3",
		Help:      "The current concentration of 2.5 micron particles in micrograms per meter cubed",
	})

	app.TempGauge = tempGauge
	app.HumidityGauge = humidityGauge
	app.Co2Gauge = co2Gauge
	app.VOCGauge = vocGauge
	app.PM25Gauge = pm25Gauge
}

func (app *App) recordMetrics() {
	go func() {
		for {
			app.getAwairData()
			time.Sleep(app.TimeBetweenChecks)
		}
	}()
}

func (app *App) getAwairData() {
	resp, err := http.Get(app.AwairAddress)
	if err != nil {
		app.Logger.Errorf("Failed to GET from configured Awair Address (%+v): %+v", app.AwairAddress, err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		app.Logger.Errorf("Failed to read body from Awair GET response: %+v", err)
		return
	}

	awairStats := AwairStats{}

	err = json.Unmarshal(body, &awairStats)
	if err != nil {
		app.Logger.Errorf("Failed to unmarshal Awair GET body into JSON: %+v", err)
		return
	}

	app.TempGauge.Set(awairStats.Temp)
	app.HumidityGauge.Set(awairStats.Humid)
	app.Co2Gauge.Set(float64(awairStats.Co2))
	app.VOCGauge.Set(float64(awairStats.Voc))
	app.PM25Gauge.Set(float64(awairStats.Pm25))
}

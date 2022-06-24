package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/namsral/flag"
	"github.com/nik-johnson-net/go-envoy"
	log "github.com/sirupsen/logrus"
)

func PrintUsageAndExit() {
	flag.Usage()
	os.Exit(1)
}

type config struct {
	envoyIP        string
	influxDbAddr   string
	influxDbToken  string
	influxDbBucket string
	schedule       string
}

func run(cfg *config) error {

	// Log
	log.Infoln("Start query.")
	defer func() {
		log.Infoln("Query complete.")
	}()

	// Contains data on Production and Consumption, if equipped.
	client := envoy.NewClient(cfg.envoyIP)
	productionData, err := client.Production()
	if nil != err {
		return fmt.Errorf("failed to create envoy client: %w", err)
	}

	// Create a new client using an InfluxDB server base URL and an authentication token
	dbClient := influxdb2.NewClient(cfg.influxDbAddr, cfg.influxDbToken)
	defer dbClient.Close()

	// create points to insert
	allReadings := append(productionData.Production, productionData.Consumption...)
	dbPoints := make([]*write.Point, len(allReadings))
	for i, reading := range allReadings {
		createdTime := time.Unix(int64(reading.ReadingTime), 0)
		dbPoints[i] = influxdb2.NewPointWithMeasurement("readings").
			AddTag("type", reading.MeasurementType).
			AddField("watts", reading.WNow).
			SetTime(createdTime)
	}

	// insert points in a batch
	writeAPI := dbClient.WriteAPIBlocking("", cfg.influxDbBucket)
	if err := writeAPI.WritePoint(context.Background(), dbPoints...); nil != err {
		return fmt.Errorf("failed to write influxdb: %w", err)
	}

	return nil
}

func main() {

	// flags
	cfg := config{}
	flag.StringVar(&cfg.envoyIP, "envoy_host", "", "address of Envoy host. e.g. 127.0.0.1")
	flag.StringVar(&cfg.influxDbAddr, "influxdb", "", "address of InfluxDb host. e.g http://127.0.0.1:8086")
	flag.StringVar(&cfg.influxDbToken, "influxdb_token", "", "auth token (optional)")
	flag.StringVar(&cfg.influxDbBucket, "influxdb_db", "solar", "influxdb database")
	flag.StringVar(&cfg.schedule, "schedule", "*/1 * * * *", "cron schedule")
	flag.Parse()

	// verify flags
	if cfg.envoyIP == "" {
		PrintUsageAndExit()
	}
	if cfg.influxDbAddr == "" {
		PrintUsageAndExit()
	}

	log.Infof("Config: %+v\n", cfg)

	// run via scheduler
	s := gocron.NewScheduler(time.UTC)
	s.Cron(cfg.schedule).Do(func() {
		if err := run(&cfg); nil != err {
			log.Errorln(err)
		}
	})
	s.StartBlocking()
}

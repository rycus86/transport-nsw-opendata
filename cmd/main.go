package main

import (
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rycus86/transport-nsw-opendata/pkg/client"
	"github.com/rycus86/transport-nsw-opendata/pkg/server"
	"github.com/rycus86/transport-nsw-opendata/pkg/timetables"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	apiKeyFile = flag.String("apikey", "",
		"TfNSW Open Data API key (in a text file, alternatively set the TFNSW_API_KEY environment variable)")

	currentTimetables sync.Map

	updaters = []func(*timetables.TimetableGTFSClient){
		// Sydney trains
		func(cli *timetables.TimetableGTFSClient) {
			if timetable, err := cli.SydneyTrains(); err != nil {
				fmt.Println("Failed to update Sydney trains timetable:", err)
			} else {
				previous, hadPrevious := currentTimetables.Load("sydneytrains")

				currentTimetables.Store("sydneytrains", timetable)

				if hadPrevious {
					previous.(*timetables.Timetable).Delete()
				}
			}
		},
	}
)

func runUpdates(cli *timetables.TimetableGTFSClient) {
	runAll := func() {
		for _, updater := range updaters {
			updater(cli)
		}

		fmt.Println("Timetables have been updated")
	}

	firstRun := time.After(0)

	for {
		select {
		case <-firstRun:
			// run all of the updaters at startup
			runAll()
		case <-time.Tick(5 * time.Minute):
			// then at every 5 minutes
			runAll()
		}
	}
}

func getApiKey() string {
	flag.Parse()

	secretPath := strings.TrimSpace(*apiKeyFile)
	envvar := strings.TrimSpace(os.Getenv("TFNSW_API_KEY"))

	if secretPath == "" && envvar == "" {
		log.Panicln("missing API key")
	}

	if secretPath == "" {
		return envvar
	} else if f, err := os.Open(*apiKeyFile); err != nil {
		panic(err)
	} else if contents, err := ioutil.ReadAll(f); err != nil {
		panic(err)
	} else {
		return strings.TrimSpace(string(contents))
	}
}

func main() {
	apiKey := getApiKey()
	cli := timetables.NewGTFSClient(client.NewHttpClient(apiKey))

	go runUpdates(cli)

	http.HandleFunc("/sydneytrains/", server.NextTrips(func() *timetables.Timetable {
		if tt, ok := currentTimetables.Load("sydneytrains"); ok {
			return tt.(*timetables.Timetable)
		} else {
			return nil
		}
	}))

	http.Handle("/metrics", promhttp.Handler())

	fmt.Println("Starting HTTP server ...")
	log.Panicln(http.ListenAndServe(":8080", nil))
}

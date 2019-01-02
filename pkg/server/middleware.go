package server

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rycus86/transport-nsw-opendata/pkg/timetables"
	"net/http"
	"strings"
	"time"
)

var (
	tripsHistogram = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:        "req_trips",
		Help:        "Histogram for serving requests related to trips",
		ConstLabels: prometheus.Labels{"endpoint_type": "trips"},
	})
)

func init() {
	prometheus.MustRegister(tripsHistogram)
}

func NextTrips(timetableSupplier func() *timetables.Timetable) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		reqStart := time.Now()
		defer func() { tripsHistogram.Observe(time.Since(reqStart).Seconds()) }()

		parts := strings.Split(request.URL.Path, "/")
		if len(parts) < 2 {
			writer.WriteHeader(400)
			return
		}

		from, to := parts[len(parts)-2], parts[len(parts)-1]
		fmt.Println("Searching trips from", from, "to", to)

		timetable := timetableSupplier()
		if timetable == nil {
			fmt.Println("No timetable available")
			writer.WriteHeader(503)
			return
		}

		fromStation, err := timetable.FindStation(from)
		if err != nil {
			fmt.Println("Station", from, "not found:", err)
			writer.WriteHeader(404)
			return
		}

		toStation, err := timetable.FindStation(to)
		if err != nil {
			fmt.Println("Station", to, "not found:", err)
			writer.WriteHeader(404)
			return
		}

		trips, err := fromStation.FindTripsTo(toStation, 3)
		if err != nil {
			fmt.Println("Trips not found:", err)
			writer.WriteHeader(500)
			return
		}

		accept := request.Header.Get("Accept")

		if strings.Contains(accept, "application/json") {
			writer.Header().Add("Content-Type", "application/json")
			writer.Header().Add("Cache-Control", "public, max-age=60")
			writer.WriteHeader(200)

			json.NewEncoder(writer).Encode(trips)
		} else if strings.Contains(accept, "text/html") {
			writer.Header().Add("Content-Type", "text/html")
			writer.WriteHeader(200)

			content := ""
			for _, trip := range trips {
				content += fmt.Sprintf(`
<p>
	<span>Headsign: %s</span><br/>
	<span>Departing: %s from %s</span><br/>
	<span>Arriving: %s at %s</span><br/>
</p>
`, trip.Headsign, trip.DepartureTime, trip.DepartureStop, trip.ArrivalTime, trip.ArrivalStop)
			}

			writer.Write([]byte(fmt.Sprintf(`<html>
<head>
	<title>Departures from %s to %s</title>
</head>
<body>
<h1>Trips from %s to %s</h1>
<div>
%s
</div>
</body>
</html>`,
				fromStation.Name, toStation.Name,
				fromStation.Name, toStation.Name,
				content)))
		} else {
			writer.Header().Add("Content-Type", "text/plain")
			writer.WriteHeader(200)

			for _, trip := range trips {
				writer.Write([]byte(fmt.Sprintf(
					"%s :: %s - %s\n",
					trip.Headsign, trip.DepartureTime, trip.ArrivalTime)))
			}
		}
	}
}

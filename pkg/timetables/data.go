package timetables

import (
	"archive/zip"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

func (t *Timetable) IsTheSameAs(other *Timetable) bool {
	return t.f.Name() == other.f.Name()
}

func (t *Timetable) Delete() error {
	return os.Remove(t.f.Name())
}

func (t *Timetable) FindStation(targetName string) (*Station, error) {
	r, err := zip.OpenReader(t.f.Name())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	stopsFile, err := findFile(r, "stops.txt")
	if err != nil {
		return nil, err
	}
	defer stopsFile.Close()

	station := Station{
		timetable: t,
	}

	reader := csv.NewReader(stopsFile)
	reader.ReuseRecord = true

	for rec, err := reader.Read(); err == nil; rec, err = reader.Read() {
		stopId, stopName, parentStation := rec[0], rec[2], rec[9]

		if station.Id != "" && parentStation == station.Id {
			station.Stops = append(station.Stops, Stop{
				Id:   stopId,
				Name: stopName,
			})
		} else if strings.Contains(strings.ToLower(stopName), strings.ToLower(targetName)) {
			station.Id = stopId
			station.Name = stopName
		}
	}

	if station.Id == "" {
		return nil, errors.New(fmt.Sprintf("station not found: %s", targetName))
	}

	return &station, nil
}

func (s *Station) FindTripsTo(target *Station, limit int) ([]Trip, error) {
	location, err := time.LoadLocation("Australia/NSW")
	if err != nil {
		return nil, err
	}

	now := time.Now().In(location)
	tnow := now.Format("15:04:05")
	dnow := now.Format("20060102")

	r, err := zip.OpenReader(s.timetable.f.Name())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	calendarFile, err := findFile(r, "calendar.txt")
	if err != nil {
		return nil, err
	}
	defer calendarFile.Close()

	validServiceIds := map[string]bool{}

	reader := csv.NewReader(calendarFile)
	reader.ReuseRecord = true

	for rec, err := reader.Read(); err == nil; rec, err = reader.Read() {
		serviceId, startDate, endDate := rec[0], rec[8], rec[9]

		if startDate <= dnow && dnow <= endDate {
			dayPos := int(now.Weekday())

			if dayPos == 0 {
				dayPos = 7
			}

			if rec[dayPos] == "1" {
				validServiceIds[serviceId] = true
			}
		}
	}

	validTrips := map[string]bool{}

	tripsFile, err := findFile(r, "trips.txt")
	if err != nil {
		return nil, err
	}
	defer tripsFile.Close()

	reader = csv.NewReader(tripsFile)
	reader.ReuseRecord = true

	for rec, err := reader.Read(); err == nil; rec, err = reader.Read() {
		serviceId, tripId := rec[1], rec[2]

		if validServiceIds[serviceId] {
			validTrips[tripId] = true
		}
	}

	stopTimesFile, err := findFile(r, "stop_times.txt")
	if err != nil {
		return nil, err
	}
	defer stopTimesFile.Close()

	isAtStation := func(s *Station, id string) bool {
		if s.Id == id {
			return true
		}

		for _, stop := range s.Stops {
			if stop.Id == id {
				return true
			}
		}

		return false
	}

	getStopName := func(s *Station, id string) string {
		if s.Id == id {
			return s.Name
		}

		for _, stop := range s.Stops {
			if stop.Id == id {
				return stop.Name
			}
		}

		return "Unknown Stop"
	}

	potentialTrips := map[string]Trip{}

	var potentialToTrips [][]string

	reader = csv.NewReader(stopTimesFile)
	reader.ReuseRecord = true

	for rec, err := reader.Read(); err == nil; rec, err = reader.Read() {
		tripId, stopCode := rec[0], rec[3]

		if !validTrips[tripId] {
			continue
		}

		if isAtStation(s, stopCode) {
			if pickupType := rec[6]; pickupType != "0" {
				continue // not pickup stop
			}

			if arrivalTime := rec[1]; arrivalTime < tnow {
				continue // already too late
			}

			potentialTrips[rec[0]] = Trip{
				Id:            rec[0],
				ArrivalTime:   rec[1],
				DepartureTime: rec[2],
				DepartureStop: getStopName(s, rec[3]),
				Headsign:      rec[5],
			}

		} else if isAtStation(target, stopCode) {
			if dropOffType := rec[7]; dropOffType != "0" {
				continue // not drop-off stop
			}

			if arrivalTime := rec[1]; arrivalTime < tnow {
				continue // already too late
			}

			potentialToTrips = append(potentialToTrips, rec)
		}
	}

	if len(potentialTrips) == 0 {
		return []Trip{}, nil
	}

	var toTrips []Trip

	for _, rec := range potentialToTrips {
		fromTrip, ok := potentialTrips[rec[0]]

		if ok && isAtStation(target, rec[3]) {
			if arrivalTime := rec[1]; arrivalTime < fromTrip.DepartureTime {
				continue // arrival time is before the departure time of the departing station
			}

			toTrips = append(toTrips, Trip{
				Id:            rec[0],
				ArrivalTime:   rec[1],
				DepartureTime: fromTrip.DepartureTime,
				ArrivalStop:   getStopName(target, rec[3]),
				DepartureStop: fromTrip.DepartureStop,
				Headsign:      fromTrip.Headsign,
			})
		}
	}

	sort.Slice(toTrips, func(i, j int) bool {
		return toTrips[i].DepartureTime < toTrips[j].DepartureTime
	})

	if limit >= len(toTrips) {
		return toTrips, nil
	}

	return toTrips[0:limit], nil
}

func findFile(r *zip.ReadCloser, name string) (io.ReadCloser, error) {
	for _, f := range r.File {
		if f.Name == name {
			return f.Open()
		}
	}

	return nil, errors.New(fmt.Sprintf("no file found with name: %s", name))
}

package timetables

type Timetable struct {
	dir string
}

type Station struct {
	Id   string
	Name string

	timetable *Timetable

	Stops []Stop
}

type Stop struct {
	Id   string
	Name string
}

type Trip struct {
	Id            string
	DepartureTime string
	ArrivalTime   string
	DepartureStop string
	ArrivalStop   string
	Headsign      string
}

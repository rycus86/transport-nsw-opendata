package timetables

import "github.com/rycus86/transport-nsw-opendata/pkg/client"

const baseUrl = "https://api.transport.nsw.gov.au/v1/gtfs/schedule"

type TimetableGTFSClient struct {
	client client.Client
}

func (c *TimetableGTFSClient) SydneyTrains() (*Timetable, error) {
	f, err := c.client.FetchBinary(baseUrl + "/sydneytrains")
	if err != nil {
		return nil, err
	}

	return &Timetable{f: f}, nil
}

func NewGTFSClient(client client.Client) *TimetableGTFSClient {
	return &TimetableGTFSClient{
		client: client,
	}
}

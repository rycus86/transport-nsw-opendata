package timetables

import (
	"archive/zip"
	"github.com/rycus86/transport-nsw-opendata/pkg/client"
	"io"
	"os"
	"path"
	"regexp"
)

const baseUrl = "https://api.transport.nsw.gov.au/v1/gtfs/schedule"

type TimetableGTFSClient struct {
	client client.Client
}

func (c *TimetableGTFSClient) SydneyTrains() (*Timetable, error) {
	filename, err := c.client.FetchBinary(baseUrl + "/sydneytrains")
	if err != nil {
		return nil, err
	}

	dirname := path.Join(os.TempDir(), "_extracted_"+regexp.MustCompile("[./]").ReplaceAllString(filename, "_"))
	os.Mkdir(dirname, os.ModePerm)

	archive, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer archive.Close()

	for _, f := range archive.File {
		source, err := f.Open()
		if err != nil {
			return nil, err
		}

		target, err := os.OpenFile(path.Join(dirname, f.Name), os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			source.Close()
			return nil, err
		}

		io.Copy(target, source)

		target.Close()
		source.Close()
	}

	return &Timetable{dir: dirname}, nil
}

func NewGTFSClient(client client.Client) *TimetableGTFSClient {
	return &TimetableGTFSClient{
		client: client,
	}
}

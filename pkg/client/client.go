package client

import "os"

type Client interface {
	FetchBinary(url string) (*os.File, error)
}

package client

type Client interface {
	FetchBinary(url string) (string, error)
}

package client

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type HttpClient struct {
	client      *http.Client
	cachedItems map[string]cachedItem
}

type cachedItem struct {
	lastModified string
	value        interface{}
}

func (c *HttpClient) FetchBinary(url string) (*os.File, error) {
	if cached := c.getCachedItem(url); cached != nil {
		return cached.(*os.File), nil
	}

	response, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("failed to fetch data from %s: HTTP %d", url, response.StatusCode))
	}

	tmpFile, err := ioutil.TempFile("", "*.apibin")
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(tmpFile, response.Body); err != nil {
		return nil, err
	}

	if lastModified := response.Header.Get("Last-Modified"); lastModified != "" {
		c.cachedItems[url] = cachedItem{
			lastModified: lastModified,
			value:        tmpFile,
		}
	}

	return tmpFile, nil
}

func (c *HttpClient) getCachedItem(url string) interface{} {
	cached, ok := c.cachedItems[url]
	if !ok {
		return nil
	}

	response, err := c.client.Head(url)
	if err != nil {
		return cached.value // use the cached item if this has failed
	}

	lastModified := response.Header.Get("Last-Modified")
	if lastModified == cached.lastModified {
		return cached.value
	}

	return nil
}

func NewHttpClient(apiKey string) Client {
	return &HttpClient{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: newTransport(apiKey),
		},
		cachedItems: map[string]cachedItem{},
	}
}

type apiTransport struct {
	ApiKey    string
	UserAgent string
}

func (t *apiTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request.Header.Add("User-Agent", t.UserAgent)
	request.Header.Add("Authorization", fmt.Sprintf("apikey %s", t.ApiKey))

	return http.DefaultTransport.RoundTrip(request)
}

func newTransport(apiKey string) http.RoundTripper {
	return &apiTransport{
		ApiKey:    apiKey,
		UserAgent: "Transport NSW OpenData client (https://github.com/rycus86/transport-nsw-opendata)",
	}
}

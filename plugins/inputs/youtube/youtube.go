package youtube

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal"
	// "github.com/influxdata/telegraf/metric"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/tidwall/gjson"
)

var (
	utf8BOM = []byte("\xef\xbb\xbf")
)

// YouTube struct
type YouTube struct {
	Name               string
	PlaylistId         string
	MaxResults         int
	PlaylistItemsURI   string
	VideoStatisticsURI string
	ApiKey             string
	TagKeys            []string
	Method             string
	ResponseTimeout    internal.Duration
	Parameters         map[string]string
	Headers            map[string]string

	client HTTPClient
}

type HTTPClient interface {
	// Returns the result of an http request
	//
	// Parameters:
	// req: HTTP request object
	//
	// Returns:
	// http.Response:  HTTP respons object
	// error        :  Any error that may have occurred
	MakeRequest(req *http.Request) (*http.Response, error)

	SetHTTPClient(client *http.Client)
	HTTPClient() *http.Client
}

type RealHTTPClient struct {
	client *http.Client
}

func (c *RealHTTPClient) MakeRequest(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

func (c *RealHTTPClient) SetHTTPClient(client *http.Client) {
	c.client = client
}

func (c *RealHTTPClient) HTTPClient() *http.Client {
	return c.client
}

var sampleConfig = `
  interval = "15m"

  playlist_id = "my-playlist-id"
  max_results = 5

  api_key = "my-api-key"
  
  ## Set response_timeout (default 5 seconds)
  response_timeout = "5s"

  ## HTTP method to use: GET or POST (case-sensitive)
  method = "GET"
  
  fieldpass = ["*statistics_*"]
`

func (y *YouTube) SampleConfig() string {
	return sampleConfig
}

func (y *YouTube) Description() string {
	return "Read flattened metrics from YouTube API HTTP endpoints"
}

// Gathers data for all videos in a playlist.
func (y *YouTube) Gather(accumulator telegraf.Accumulator) error {
	var wg sync.WaitGroup

	if y.client.HTTPClient() == nil {
		tr := &http.Transport{
			ResponseHeaderTimeout: y.ResponseTimeout.Duration,
		}
		client := &http.Client{
			Transport: tr,
			Timeout:   y.ResponseTimeout.Duration,
		}
		y.client.SetHTTPClient(client)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		accumulator.AddError(y.gatherPlaylist(accumulator, ""))
	}()

	wg.Wait()

	return nil
}

// Gathers data from a youtube endpoints for videos in a playlist
// Parameters:
//     acc      	: The telegraf Accumulator to use
//	   pageToken	: The page token to request (if paging through results)
//
// Returns:
//     error: Any error that may have occurred
func (y *YouTube) gatherPlaylist(
	acc telegraf.Accumulator,
	pageToken string,
) error {
	uri := y.PlaylistItemsURI + "&maxResults=" + strconv.Itoa(y.MaxResults) + "&playlistId=" + y.PlaylistId + "&key=" + y.ApiKey

	if pageToken != "" {
		uri = uri + "&pageToken=" + pageToken
	}
	resp, _, err := y.sendRequest(uri)
	if err != nil {
		return err
	}

	// msrmnt_name := "youtube"
	// tags := map[string]string{}

	// extract the nextPageToken if it exists
	nextPageToken := gjson.Get(resp, "nextPageToken")
	// if there is a "nextPageToken", then there are still more pages of
	// results to process, so call them recursively now
	if nextPageToken.Exists() {
		acc.AddError(y.gatherPlaylist(acc, nextPageToken.String()))
	}

	// extract all of the videoIds here
	videoIds := gjson.Get(resp, "items.#.snippet.resourceId.videoId")

	// for each video id, request the stats for that video
	for _, videoId := range videoIds.Array() {
		// get stats
		resp, _, err := y.sendRequest(y.VideoStatisticsURI + "&id=" + videoId.String() + "&key=" + y.ApiKey)
		if err != nil {
			return err
		}

		result := gjson.Get(resp, "items.#.statistics.*Count")
		for _, stat := range result.Array() {
			fmt.Println(stat.String())
		}

		// fmt.Println(strconv.ParseFloat(gjson.Get(resp, "items.#.statistics.viewCount").String(), 64))
		fields := make(map[string]float64)

		// fields["viewCount"] = gjson.Get(resp, "items.#.statistics.viewCount").Float()
		// fields["likeCount"] = gjson.Get(resp, "items.#.statistics.likeCount").Num
		// fields["dislikeCount"] = gjson.Get(resp, "items.#.statistics.dislikeCount").Num
		// fields["favoriteCount"] = gjson.Get(resp, "items.#.statistics.favoriteCount").Num
		// fields["commentCount"] = gjson.Get(resp, "items.#.statistics.commentCount").Num

		fmt.Println(fields)

		//
		// 			m, err := metric.New(msrmnt_name, tags, fields, time.Now().UTC())
		// 			if err != nil {
		// 				return err
		// 			}
		// 			m.AddTag("videoId", videoId.String())
		// 			acc.AddFields(m.Name(), fields, m.Tags())
		// }
	}

	return nil
}

// Sends an HTTP request to the server using the HttpJson object's HTTPClient.
// This request will be a GET.
// Parameters:
//     serverURL: endpoint to send request to
//
// Returns:
//     string: body of the response
//     error : Any error that may have occurred
func (y *YouTube) sendRequest(serverURL string) (string, float64, error) {
	// Prepare URL
	requestURL, err := url.Parse(serverURL)
	if err != nil {
		return "", -1, fmt.Errorf("Invalid server URL \"%s\"", serverURL)
	}

	data := url.Values{}
	switch {
	case y.Method == "GET":
		params := requestURL.Query()
		for k, v := range y.Parameters {
			params.Add(k, v)
		}
		requestURL.RawQuery = params.Encode()

	case y.Method == "POST":
		requestURL.RawQuery = ""
		for k, v := range y.Parameters {
			data.Add(k, v)
		}
	}

	// Create + send request
	req, err := http.NewRequest(y.Method, requestURL.String(),
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", -1, err
	}

	start := time.Now()
	resp, err := y.client.MakeRequest(req)
	if err != nil {
		return "", -1, err
	}

	defer resp.Body.Close()
	responseTime := time.Since(start).Seconds()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return string(body), responseTime, err
	}
	body = bytes.TrimPrefix(body, utf8BOM)

	// Process response
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Response from url \"%s\" has status code %d (%s), expected %d (%s)",
			requestURL.String(),
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
			http.StatusOK,
			http.StatusText(http.StatusOK))
		return string(body), responseTime, err
	}

	return string(body), responseTime, err
}

func init() {
	inputs.Add("youtube", func() telegraf.Input {
		return &YouTube{
			client: &RealHTTPClient{},
			ResponseTimeout: internal.Duration{
				Duration: 5 * time.Second,
			},
			PlaylistItemsURI:   "https://www.googleapis.com/youtube/v3/playlistItems?part=snippet",
			VideoStatisticsURI: "https://www.googleapis.com/youtube/v3/videos?part=statistics,snippet",
		}
	})
}

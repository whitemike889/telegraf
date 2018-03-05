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
	"github.com/influxdata/telegraf/metric"
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
}

func NewYouTube() *YouTube {
	return &YouTube{
		PlaylistItemsURI:   "https://www.googleapis.com/youtube/v3/playlistItems?part=snippet",
		VideoStatisticsURI: "https://www.googleapis.com/youtube/v3/videos?part=statistics,snippet",
	}
}

var sampleConfig = `
  interval = "15m"

  playlist_id = "my-playlist-id"
  max_results = 5

  api_key = "my-api-key"
  
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
	//
	// if y.client.HTTPClient() == nil {
	// 	tr := &http.Transport{
	// 		ResponseHeaderTimeout: y.ResponseTimeout.Duration,
	// 	}
	// 	client := &http.Client{
	// 		Transport: tr,
	// 		Timeout:   y.ResponseTimeout.Duration,
	// 	}
	// 	y.client.SetHTTPClient(client)
	// }

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
	uri := y.PlaylistItemsURI + "&playlistId=" + y.PlaylistId + "&key=" + y.ApiKey
	mr := strconv.Itoa(y.MaxResults)
	uri = uri + "&maxResults=" + mr

	if pageToken != "" {
		uri = uri + "&pageToken=" + pageToken
	}
	resp, err := y.sendRequest(uri)
	if err != nil {
		return err
	}

	msrmnt_name := "youtube"
	tags := map[string]string{}

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
		resp, err := y.sendRequest(y.VideoStatisticsURI + "&id=" + videoId.String() + "&key=" + y.ApiKey)
		if err != nil {
			return err
		}

		fields := make(map[string]interface{})
		// the stats from Google Data API come in as quoted strings, and when the gjson lib parses them out,
		// it goes one step further, wrapping them in [] and then escaping the quotes. Strip it all back!

		vc := strings.Trim(gjson.Get(resp, "items.#.statistics.viewCount").String(), "[]\"")
		if vc != "" {
			vcf, err := strconv.ParseFloat(vc, 64)
			if err != nil {
				return err
			}
			fields["viewCount"] = vcf
		}

		lc := strings.Trim(gjson.Get(resp, "items.#.statistics.likeCount").String(), "[]\"")
		if lc != "" {
			lcf, err := strconv.ParseFloat(lc, 64)
			if err != nil {
				return err
			}
			fields["likeCount"] = lcf
		}

		dc := strings.Trim(gjson.Get(resp, "items.#.statistics.dislikeCount").String(), "[]\"")
		if dc != "" {
			dcf, err := strconv.ParseFloat(dc, 64)
			if err != nil {
				return err
			}
			fields["dislikeCount"] = dcf
		}

		fc := strings.Trim(gjson.Get(resp, "items.#.statistics.favoriteCount").String(), "[]\"")
		if fc != "" {
			fcf, err := strconv.ParseFloat(fc, 64)
			if err != nil {
				return err
			}
			fields["favoriteCount"] = fcf
		}

		cc := strings.Trim(gjson.Get(resp, "items.#.statistics.commentCount").String(), "[]\"")
		if cc != "" {
			ccf, err := strconv.ParseFloat(cc, 64)
			if err != nil {
				return err
			}
			fields["commentCount"] = ccf
		}

		if len(fields) > 0 {
			m, err := metric.New(msrmnt_name, tags, fields, time.Now().UTC())
			if err != nil {
				return err
			}
			m.AddTag("videoId", videoId.String())
			acc.AddFields(m.Name(), fields, m.Tags())
		}
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
func (y *YouTube) sendRequest(serverURL string) (string, error) {
	// Prepare URL
	requestURL, err := url.Parse(serverURL)
	if err != nil {
		return "", fmt.Errorf("Invalid server URL \"%s\"", serverURL)
	}

	tr := &http.Transport{
		ResponseHeaderTimeout: y.ResponseTimeout.Duration,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   y.ResponseTimeout.Duration,
	}

	var b bytes.Buffer
	req, err := http.NewRequest("GET", requestURL.String(), &b)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return string(body), err
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
		return string(body), err
	}

	return string(body), err
}

func init() {
	inputs.Add("youtube", func() telegraf.Input {
		return NewYouTube()
	})
}

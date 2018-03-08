package youtube

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/tidwall/gjson"
)

const (
	playlist_items_endpoint   string = "/youtube/v3/playlistItems"
	video_statistics_endpoint string = "/youtube/v3/videos"
)

// YouTube struct
type YouTube struct {
	PlaylistId string
	ApiKey     string
	TagKeys    []string

	Transport http.RoundTripper

	sync.Mutex

	playlist_url *url.URL
	stats_url    *url.URL
}

func NewYouTube() *YouTube {
	u := &url.URL{}
	u.Scheme = "https"
	u.Host = "www.googleapis.com"

	return &YouTube{playlist_url: u, stats_url: u}
}

var sampleConfig = `
  interval = "15m"

  playlist_id = "my-playlist-id"

  api_key = "my-api-key"
  
  fieldpass = ["*statistics_*"]
`

func (y *YouTube) SampleConfig() string {
	return sampleConfig
}

func (y *YouTube) Description() string {
	return "Read flattened metrics from YouTube API HTTP endpoints"
}

func (y *YouTube) Gather(accumulator telegraf.Accumulator) error {
	y.Lock()
	defer y.Unlock()
	videoIds, err := y.gatherPlaylist("")
	if err != nil {
		return err
	}
	accumulator.AddError(y.gatherVideoStatistics(accumulator, videoIds))
	return nil
}

func (y *YouTube) gatherPlaylist(
	pageToken string,
) ([]string, error) {
	y.playlist_url.Path = playlist_items_endpoint
	parameters := url.Values{}
	parameters.Add("part", "snippet")
	parameters.Add("playlistId", y.PlaylistId)
	parameters.Add("key", y.ApiKey)

	if pageToken != "" {
		parameters.Add("pageToken", pageToken)
	}
	y.playlist_url.RawQuery = parameters.Encode()

	playlist_resp, err := sendRequest(y, y.playlist_url.String())
	if err != nil {
		return nil, err
	}

	playlist_resp_str := string(playlist_resp)

	resultsPerPage := gjson.Get(playlist_resp_str, "pageInfo.resultsPerPage").Int()
	videoIds := make([]string, resultsPerPage)

	// extract the nextPageToken if it exists
	nextPageToken := gjson.Get(playlist_resp_str, "nextPageToken")
	// if there is a "nextPageToken", then there are still more pages of
	// results to process, so call them recursively now
	if nextPageToken.Exists() {
		nextPageVideoIds, err := y.gatherPlaylist(nextPageToken.String())
		if err != nil {
			return nil, err
		}
		videoIds = append(videoIds, nextPageVideoIds...)
	}

	// extract all of the videoIds here
	result := gjson.Get(playlist_resp_str, "items.#.snippet.resourceId.videoId")
	for i, videoId := range result.Array() {
		videoIds[i] = videoId.String()
	}

	return videoIds, nil
}

func (y *YouTube) gatherVideoStatistics(
	acc telegraf.Accumulator,
	videoIds []string,
) error {
	for _, videoId := range videoIds {
		// get stats
		y.stats_url.Path = video_statistics_endpoint
		params := url.Values{}
		params.Add("part", "statistics")
		params.Add("key", y.ApiKey)
		params.Add("id", videoId)
		y.stats_url.RawQuery = params.Encode()

		stats_resp, err := sendRequest(y, y.stats_url.String())
		if err != nil {
			return err
		}

		fields := make(map[string]interface{})
		extractAllStats(string(stats_resp), videoId, fields)

		tags := map[string]string{}

		if len(fields) > 0 {
			m, err := metric.New("youtube", tags, fields, time.Now().UTC())
			if err != nil {
				return err
			}
			m.AddTag("videoId", videoId)
			acc.AddFields(m.Name(), fields, m.Tags())
		}
	}

	return nil
}

func extractAllStats(json string, videoId string, fields map[string]interface{}) error {
	result := gjson.Get(json, "items.#[id==\""+videoId+"\"].statistics")
	result.ForEach(func(key, value gjson.Result) bool {
		viewCount := result.Get("viewCount").Int()
		likeCount := result.Get("likeCount").Int()
		dislikeCount := result.Get("dislikeCount").Int()
		favoriteCount := result.Get("favoriteCount").Int()
		commentCount := result.Get("commentCount").Int()

		fields["viewCount"] = viewCount
		fields["likeCount"] = likeCount
		fields["dislikeCount"] = dislikeCount
		fields["favoriteCount"] = favoriteCount
		fields["commentCount"] = commentCount

		return true // keep iterating
	})

	return nil
}

func sendRequest(y *YouTube, url string) ([]byte, error) {
	client := &http.Client{
		Transport: y.Transport,
		Timeout:   time.Duration(4 * time.Second),
	}

	var b bytes.Buffer
	req, err := http.NewRequest("GET", url, &b)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func init() {
	inputs.Add("youtube", func() telegraf.Input {
		return NewYouTube()
	})
}

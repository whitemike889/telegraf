package youtube

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

// Gathers data for all videos in a playlist.
func (y *YouTube) Gather(accumulator telegraf.Accumulator) error {
	y.Lock()
	defer y.Unlock()
	accumulator.AddError(y.gatherPlaylist(accumulator, ""))
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
		return err
	}

	tags := map[string]string{}

	playlist_resp_str := string(playlist_resp)

	// extract the nextPageToken if it exists
	nextPageToken := gjson.Get(playlist_resp_str, "nextPageToken")
	// if there is a "nextPageToken", then there are still more pages of
	// results to process, so call them recursively now
	if nextPageToken.Exists() {
		acc.AddError(y.gatherPlaylist(acc, nextPageToken.String()))
	}

	// extract all of the videoIds here
	videoIds := gjson.Get(playlist_resp_str, "items.#.snippet.resourceId.videoId")

	// for each video id, request the stats for that video
	for _, videoId := range videoIds.Array() {
		// get stats
		y.stats_url.Path = video_statistics_endpoint
		params := url.Values{}
		params.Add("part", "statistics")
		params.Add("key", y.ApiKey)
		params.Add("id", videoId.String())
		y.stats_url.RawQuery = params.Encode()

		stats_resp, err := sendRequest(y, y.stats_url.String())
		if err != nil {
			return err
		}

		fields, err := extractAllStats(string(stats_resp))

		if len(fields) > 0 {
			m, err := metric.New("youtube", tags, fields, time.Now().UTC())
			if err != nil {
				return err
			}
			m.AddTag("videoId", videoId.String())
			acc.AddFields(m.Name(), fields, m.Tags())
		}
	}

	return nil
}

func extractAllStats(json string) (map[string]interface{}, error) {
	fields := make(map[string]interface{})

	vcf, err := extractStat("viewCount", json)
	if err != nil {
		return nil, err
	} else if vcf >= 0 {
		fields["viewCount"] = vcf
	}

	lcf, err := extractStat("likeCount", json)
	if err != nil {
		return nil, err
	} else if lcf >= 0 {
		fields["likeCount"] = lcf
	}

	dcf, err := extractStat("dislikeCount", json)
	if err != nil {
		return nil, err
	} else if dcf >= 0 {
		fields["dislikeCount"] = dcf
	}

	fcf, err := extractStat("favoriteCount", json)
	if err != nil {
		return nil, err
	} else if fcf >= 0 {
		fields["favoriteCount"] = fcf
	}

	ccf, err := extractStat("commentCount", json)
	if err != nil {
		return nil, err
	} else if ccf >= 0 {
		fields["commentCount"] = ccf
	}

	return fields, nil
}

func extractStat(statName string, json string) (float64, error) {
	stat := strings.Trim(gjson.Get(json, "items.#.statistics."+statName).String(), "[]\"")
	if stat != "" {
		stat_f, err := strconv.ParseFloat(stat, 64)
		if err != nil {
			return -1, err
		}
		return stat_f, nil
	}
	return -1, nil
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

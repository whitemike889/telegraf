package youtube

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/influxdata/telegraf/testutil"
	"github.com/stretchr/testify/require"
)

func TestYouTubeGatherPlaylist(t *testing.T) {
	playlistHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, testPlaylistJSON)
		},
	)
	fakeServer := httptest.NewServer(playlistHandler)
	defer fakeServer.Close()

	u, err := url.ParseRequestURI(fakeServer.URL)
	require.NoError(t, err)

	plugin := &YouTube{playlist_url: u, stats_url: u}
	plugin.PlaylistId = "test-playlist-id"
	plugin.ApiKey = "test-api-key"

	videoIds, _ := plugin.gatherPlaylist("")

	var expectedResult = []string{"XDgC4FMftpg"}

	if !reflect.DeepEqual(expectedResult, videoIds) {
		t.Fatalf("Expected %s but got %s", expectedResult, videoIds)
	}
}

func TestYouTubeGatherVideoStats(t *testing.T) {
	var acc testutil.Accumulator

	videoStatsHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, testVideoStatsJSON)
		},
	)
	fakeServer := httptest.NewServer(videoStatsHandler)
	defer fakeServer.Close()

	u, err := url.ParseRequestURI(fakeServer.URL)
	require.NoError(t, err)

	plugin := &YouTube{playlist_url: u, stats_url: u}
	plugin.PlaylistId = "test-playlist-id"
	plugin.ApiKey = "test-api-key"

	var expectedVideoIds = []string{"XDgC4FMftpg"}
	plugin.gatherVideoStatistics(&acc, expectedVideoIds)

	expectedTags := make(map[string]string)
	expectedTags["videoId"] = "XDgC4FMftpg"

	expectedFields := map[string]interface{}{
		"viewCount":     int64(5574069),
		"likeCount":     int64(843),
		"dislikeCount":  int64(74),
		"favoriteCount": int64(0),
		"commentCount":  int64(128),
	}

	acc.Get("youtube")
	acc.AssertContainsTaggedFields(t, "youtube", expectedFields, expectedTags)
}

const testPlaylistJSON = `
	{
	  "kind": "youtube#playlistItemListResponse",
	  "etag": "\"_gJQceDMxJ8gP-8T2HLXUoURK8c/np7Fp7dG3wEN7yn3BurjyJGFKYI\"",
	  "pageInfo": {
	    "totalResults": 1,
	    "resultsPerPage": 1
	  },
	  "items": [
	    {
	      "kind": "youtube#playlistItem",
	      "etag": "\"_gJQceDMxJ8gP-8T2HLXUoURK8c/x8utLR6SK07clGh9DQzv80Mczyw\"",
	      "id": "UExCQ0YyREFDNkZGQjU3NERFLkMyQjUzQkM1OTFFRTNFMEQ=",
	      "snippet": {
	        "publishedAt": "2011-11-22T15:29:40.000Z",
	        "channelId": "UCvceBgMIpKb4zK1ss-Sh90w",
	        "title": "Mark Kempton: Neighbors In Need",
	        "description": "Follow Mark on Google+: https://profiles.google.com/u/0/105705606437451864842\r\n\r\nWhen floodwaters hit northeast Australia, Mark's innovative search became the difference between life and death for many of his neighbors.",
	        "thumbnails": {
	          "default": {
	            "url": "https://i.ytimg.com/vi/XDgC4FMftpg/default.jpg",
	            "width": 120,
	            "height": 90
	          },
	          "medium": {
	            "url": "https://i.ytimg.com/vi/XDgC4FMftpg/mqdefault.jpg",
	            "width": 320,
	            "height": 180
	          },
	          "high": {
	            "url": "https://i.ytimg.com/vi/XDgC4FMftpg/hqdefault.jpg",
	            "width": 480,
	            "height": 360
	          },
	          "standard": {
	            "url": "https://i.ytimg.com/vi/XDgC4FMftpg/sddefault.jpg",
	            "width": 640,
	            "height": 480
	          }
	        },
	        "channelTitle": "Google Search Stories",
	        "playlistId": "PLBCF2DAC6FFB574DE",
	        "position": 4,
	        "resourceId": {
	          "kind": "youtube#video",
	          "videoId": "XDgC4FMftpg"
	        }
	      }
	    }
	  ]
	}
`
const testVideoStatsJSON = `
	{
	  "kind": "youtube#videoListResponse",
	  "etag": "\"_gJQceDMxJ8gP-8T2HLXUoURK8c/grhISBuUMf9BbGpHC3pF9NjoPBI\"",
	  "pageInfo": {
	    "totalResults": 1,
	    "resultsPerPage": 1
	  },
	  "items": [
	    {
	      "kind": "youtube#video",
	      "etag": "\"_gJQceDMxJ8gP-8T2HLXUoURK8c/pdhyD3KpgzajB3BV4Te1i_NNfkU\"",
	      "id": "XDgC4FMftpg",
	      "statistics": {
	        "viewCount": "5574069",
	        "likeCount": "843",
	        "dislikeCount": "74",
	        "favoriteCount": "0",
	        "commentCount": "128"
	      }
	    }
	  ]
	}
`

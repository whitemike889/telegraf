# YouTube Input Plugin

The YouTube plugin collects data from YouTube Data API v3 endpoints which respond with JSON. It flattens the JSON and finds all numeric values, treating them as floats.

Refer to the [YouTube Data API v3](https://developers.google.com/youtube/v3/docs/), specifically the [Playlist Items](https://developers.google.com/youtube/v3/docs/playlistitems), and the [Videos](https://developers.google.com/youtube/v3/docs/videos).

### Configuration:

```toml
[[inputs.youtube]]
  interval = "15m"

  playlist_id = "my-playlist-id"
  max_results = 5

  api_key = "my-api-key"
```


### Measurements & Fields:

- youtube
	- tags:
		- videoId (string): the ID that YouTube uses to uniquely identify the playlist item
	- fields:
		- viewCount (float): the number of times the video has been viewed
		- likeCount (float): the number of times the video has been liked
		- dislikeCount (float): the number of times the video has been disliked
		- favoriteCount (float): the number of times the video has been favorited
		- commentCount (float): the number of times the video has been commented on

		
### Example Output:

This section shows example output in Line Protocol format.  You can often use
`telegraf --input-filter youtube --test` or use the `file` output to get
this information.

```
youtube,videoId=18TknKGC7tY favoriteCount=0,commentCount=57,viewCount=418712,likeCount=376,dislikeCount=40 1520265899000000000
youtube,videoId=7hakGJU9xco viewCount=1884390,likeCount=440,dislikeCount=36,favoriteCount=0,commentCount=25 1520265900000000000
youtube,videoId=x9-F6dbCIHw dislikeCount=73,favoriteCount=0,commentCount=115,viewCount=451370,likeCount=678 1520265900000000000
youtube,videoId=RQbmXxU2dkg dislikeCount=23,favoriteCount=0,commentCount=48,viewCount=1079043,likeCount=673 1520265900000000000
```
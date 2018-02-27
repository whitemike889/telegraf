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
  
  ## Set response_timeout (default 5 seconds)
  response_timeout = "5s"

  ## HTTP method to use: GET or POST (case-sensitive)
  method = "GET"
```


### Measurements & Fields:

- youtube
	- tags:
		- videoId (string): the ID that YouTube uses to uniquely identify the playlist item
	- fields:
		- viewCount (float): the number of times the video has been viewed

		
### Example Output:

This section shows example output in Line Protocol format.  You can often use
`telegraf --input-filter youtube --test` or use the `file` output to get
this information.

```
youtube,videoId=-aQOGsHm_bo viewCount=2908 1519753668000000000
youtube,videoId=qCxYjq7EBHc viewCount=11316 1519753668000000000
youtube,videoId=qr_gro2TCGU viewCount=597 1519753668000000000
youtube,videoId=khMbosLRuFo viewCount=925 1519753668000000000

```
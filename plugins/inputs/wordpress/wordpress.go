package wordpress

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/parsers"
)

var (
	utf8BOM = []byte("\xef\xbb\xbf")
)

// Wordpress struct
type Wordpress struct {
	Name                string
	WpApiURI            string
	SiteId              string
	MaxTopPosts         int
	TopPostsStatsURI    string
	SummaryStatsURI     string
	PostsURI            string
	TagStatsURI         string
	TopPostsTagKeys     []string
	SummaryStatsTagKeys []string
	PostsTagKeys        []string
	TagStatsTagKeys     []string
	Method              string
	ResponseTimeout     internal.Duration
	Parameters          map[string]string
	Headers             map[string]string

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
  
  site_id = "my-site-id"
  max_top_posts = 100
  
  wp_api_URI = "https://public-api.wordpress.com/rest/v1.1/sites/"

  top_posts_stats_URI = "/stats/top-posts"
  summary_stats_URI = "/stats/summary"
  posts_URI = "/posts"
  tag_stats_URI = "/stats/tags"
  
  ## Set response_timeout (default 5 seconds)
  response_timeout = "5s"

  ## List of tag names to extract from top-level of JSON server response
  top_posts_tag_keys = ["id",]
  summary_stats_tag_keys = ["date","period"]
  posts_tag_keys = ["ID","author_ID","date","slug"]
  tag_stats_tag_keys = []
  
  fielddrop = ["*attachment*", "*thumbnail*", "menu_order"]
  
  ## HTTP Headers (all values must be strings)
  [inputs.wordpress.headers]
     authorization = "Bearer YOUR_BEARER_TOKEN"
  
`

func (w *Wordpress) SampleConfig() string {
	return sampleConfig
}

func (w *Wordpress) Description() string {
	return "Read flattened metrics from Wordpress HTTP endpoints"
}

// Gathers data for blog posts.
func (w *Wordpress) Gather(acc telegraf.Accumulator) error {
	var wg sync.WaitGroup

	if w.client.HTTPClient() == nil {
		tr := &http.Transport{
			ResponseHeaderTimeout: w.ResponseTimeout.Duration,
		}
		client := &http.Client{
			Transport: tr,
			Timeout:   w.ResponseTimeout.Duration,
		}
		w.client.SetHTTPClient(client)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if w.TopPostsStatsURI != "" {
			acc.AddError(w.gatherTopPostsStats(acc))
		}
		if w.SummaryStatsURI != "" {
			acc.AddError(w.gatherSummaryStats(acc))
		}
		if w.PostsURI != "" {
			acc.AddError(w.gatherPosts(acc))
		}
		if w.TagStatsURI != "" {
			acc.AddError(w.gatherTagStats(acc))
		}
	}()

	wg.Wait()

	return nil
}

/*
Gathers data from a wordpress stats endpoint about top posts.
	Parameters:
		acc      : The telegraf Accumulator to use
	Returns:
		error: Any error that may have occurred
*/
func (w *Wordpress) gatherTopPostsStats(
	acc telegraf.Accumulator,
) error {
	resp, _, err := w.sendRequest(w.WpApiURI + w.SiteId + w.TopPostsStatsURI + "?max=" + strconv.Itoa(w.MaxTopPosts))
	if err != nil {
		return err
	}

	msrmnt_name := "wordpress_topposts"
	tags := map[string]string{}

	parser, err := parsers.NewJSONParser(msrmnt_name, w.TopPostsTagKeys, tags)
	if err != nil {
		return err
	}

	reStr := regexp.MustCompile("^(.*?)(\\[.*\\])(,\"total_views\":.*)$")
	repStr := "$2"
	resp = reStr.ReplaceAllString(resp, repStr)

	metrics, err := parser.Parse([]byte(resp))
	if err != nil {
		return err
	}

	for _, metric := range metrics {
		fields := make(map[string]interface{})
		for k, v := range metric.Fields() {
			fields[k] = v
		}
		acc.AddFields(metric.Name(), fields, metric.Tags())
	}

	return nil
}

/*
Gathers data from a wordpress posts endpoint about posts. JSON return format is an array with
a nesting that must be transformed from this:
	Parameters:
		acc      : The telegraf Accumulator to use
	Returns:
		error: Any error that may have occurred
*/
func (w *Wordpress) gatherPosts(
	acc telegraf.Accumulator,
) error {
	resp, _, err := w.sendRequest(w.WpApiURI + w.SiteId + w.PostsURI)
	if err != nil {
		return err
	}

	msrmnt_name := "wordpress_posts"
	tags := map[string]string{}

	parser, err := parsers.NewJSONParser(msrmnt_name, w.PostsTagKeys, tags)
	if err != nil {
		return err
	}

	// Strip out all of the "meta" blocks
	reStr := regexp.MustCompile(",\"meta\":\\s*{.*?}}")
	repStr := ""
	resp = reStr.ReplaceAllString(resp, repStr)

	// Trim off the leading preamble
	reStr = regexp.MustCompile("^.*?(\\[.*\\])")
	repStr = "$1"
	resp = reStr.ReplaceAllString(resp, repStr)

	metrics, err := parser.Parse([]byte(resp))
	if err != nil {
		return err
	}

	// var categories_val string
	// var tags_val string
	for _, metric := range metrics {
		fields := make(map[string]interface{})
		for k, v := range metric.Fields() {
			fields[k] = v
			// if strings.HasPrefix(k, "categories") && strings.HasSuffix(k, "slug") {
			// 	categories_val = v.(string) + "," + categories_val
			// } else if strings.HasPrefix(k, "tags") && strings.HasSuffix(k, "slug") {
			// 	tags_val = v.(string) + "," + tags_val
			// } else if !strings.HasPrefix(k, "categories") && !strings.HasPrefix(k, "tags") {
			// 	fields[k] = v
			// }
		}
		// if len(categories_val) > 0 {
		// 	fields["categories"] = strings.TrimSuffix(categories_val, ",")
		// }
		// if len(tags_val) > 0 {
		// 	fields["tags"] = strings.TrimSuffix(tags_val, ",")
		// }
		acc.AddFields(metric.Name(), fields, metric.Tags())
		// categories_val = ""
		// tags_val = ""
	}

	return nil
}

/*
Gathers data from a wordpress stats endpoint about site summary.
	Parameters:
		acc      : The telegraf Accumulator to use
	Returns:
		error: Any error that may have occurred
*/
func (w *Wordpress) gatherSummaryStats(
	acc telegraf.Accumulator,
) error {
	resp, _, err := w.sendRequest(w.WpApiURI + w.SiteId + w.SummaryStatsURI)
	if err != nil {
		return err
	}

	msrmnt_name := "wordpress_summary"
	tags := map[string]string{}

	parser, err := parsers.NewJSONParser(msrmnt_name, w.SummaryStatsTagKeys, tags)
	if err != nil {
		return err
	}
	metrics, err := parser.Parse([]byte(resp))
	if err != nil {
		return err
	}

	for _, metric := range metrics {
		fields := make(map[string]interface{})
		for k, v := range metric.Fields() {
			fields[k] = v
		}
		acc.AddFields(metric.Name(), fields, metric.Tags())
	}

	return nil
}

/*
Gather data from a wordpress stats endpoint about tags & categories.
	Parameters:
    	acc     : The telegraf Accumulator to use
	Returns:
    	error	: Any error that may have occurred
*/
func (w *Wordpress) gatherTagStats(
	acc telegraf.Accumulator,
) error {
	resp, _, err := w.sendRequest(w.WpApiURI + w.SiteId + w.TagStatsURI)
	if err != nil {
		return err
	}

	msrmnt_name := "wordpress_tagstats"
	tags := map[string]string{}

	parser, err := parsers.NewJSONParser(msrmnt_name, w.TagStatsTagKeys, tags)
	if err != nil {
		return err
	}

	// strip everything through the "date" field and the top-level "tags" field,
	// leaving only the surrounding brackets [].
	// also strip the trailing brace }
	// reStr := regexp.MustCompile("^[^\\[]+(.*\\])\\}")
	// repStr := "$1"
	// resp = reStr.ReplaceAllString(resp, repStr)

	metrics, err := parser.Parse([]byte(resp))
	if err != nil {
		return err
	}

	for _, metric := range metrics {
		fields := make(map[string]interface{})
		for k, v := range metric.Fields() {
			fields[k] = v
		}
		acc.AddFields(metric.Name(), fields, metric.Tags())
	}
	// metrics, err := parser.Parse([]byte(resp))
	// if err != nil {
	// 	return err
	// }
	//
	// for _, metric := range metrics {
	// 	fields := make(map[string]interface{})
	// 	for k, v := range metric.Fields() {
	// 		// k = strings.Replace(k, "tags_0_", "", 1)
	// 		// if k == "type" || k == "name" {
	// 		// 	metric.AddTag(k, v.(string))
	// 		// } else {
	// 		fields[k] = v
	// 		// }
	// 	}
	// 	acc.AddFields(metric.Name(), fields, metric.Tags())
	// }

	return nil
}

// Sends an HTTP request to the server using the HttpJson object's HTTPClient.
// This request can be either a GET or a POST.
// Parameters:
//     serverURL: endpoint to send request to
//
// Returns:
//     string: body of the response
//     error : Any error that may have occurred
func (w *Wordpress) sendRequest(serverURL string) (string, float64, error) {
	// Prepare URL
	requestURL, err := url.Parse(serverURL)
	if err != nil {
		return "", -1, fmt.Errorf("Invalid server URL \"%s\"", serverURL)
	}

	data := url.Values{}
	switch {
	case w.Method == "GET":
		params := requestURL.Query()
		for k, v := range w.Parameters {
			params.Add(k, v)
		}
		requestURL.RawQuery = params.Encode()

	case w.Method == "POST":
		requestURL.RawQuery = ""
		for k, v := range w.Parameters {
			data.Add(k, v)
		}
	}

	// Create + send request
	req, err := http.NewRequest(w.Method, requestURL.String(),
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", -1, err
	}

	// Add header parameters
	for k, v := range w.Headers {
		if strings.ToLower(k) == "host" {
			req.Host = v
		} else {
			req.Header.Add(k, v)
		}
	}

	start := time.Now()
	resp, err := w.client.MakeRequest(req)
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
	inputs.Add("wordpress", func() telegraf.Input {
		return &Wordpress{
			client: &RealHTTPClient{},
			ResponseTimeout: internal.Duration{
				Duration: 5 * time.Second,
			},
		}
	})
}

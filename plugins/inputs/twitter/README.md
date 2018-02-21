# Twitter Input Plugin

The twitter plugin collects data from Twitter API endpoints which respond with JSON. It flattens the JSON and finds all numeric values, treating them as floats.

Refer to the Twitter API, specifically for the User [User Object](https://developer.twitter.com/en/docs/tweets/data-dictionary/overview/user-object) and [User Timelines](https://developer.twitter.com/en/docs/tweets/timelines/api-reference/get-statuses-user_timeline.html).

### Configuration:

```toml
[[inputs.twitter]]
  ## NOTE This plugin only reads numerical measurements, strings and booleans
  ## will be ignored.

  interval = "15m"

  consumer_key = "my_consumer_key"
  consumer_secret = "my_consumer_secret"
  access_token  = "my_access_token"
  access_secret = "my_access_secret"

  ## Parameters used when for getting User Timeline
  ## https://developer.twitter.com/en/docs/tweets/timelines/api-reference/get-statuses-user_timeline.html
  screen_name = "twitter"
  count = 5
  trim_user = false
  exclude_replies = true
  include_retweets = true

  ## List of tag names to extract from top-level of JSON server response
  tag_keys = [
    "id_str",
	"retweeted",
	"favorited",
  ]

  fieldpass = ["user_statuses_count", "user_favourites_count", "user_followers_count", "user_friends_count", "user_listed_count", "retweet_count", "favorite_count"]

```toml


### Measurements & Fields:

- twitter
	- tags:
		- id_str (string): string representation of the Tweet Id
		- retweeted (boolean): whether or not this Tweet is a retweet
		- favorited (boolean): whether or not this user has Liked (favorited) this Tweet
	- fields:
		- user_friends_count (float): number of friends the user currently follows
		- user_followers_count (float): number of followers of the user
		- user_statuses_count (float): number of Tweets (statuses) the user has made
		- user_listed_count (float): the number of lists the user shows up in
		- user_favorites_count (float): the number of times this user has been Liked (favorited)
		- retweet_count (float): the number of times this Tweet has been retweeted
		- favorite_count (float): the number of times this Tweet has been Liked (favorited)

### Sample Queries:

This section should contain some useful InfluxDB queries that can be used to
get started with the plugin or to generate dashboards.  For each query listed,
describe at a high level what data is returned.

Get the max, mean, and min for the measurement in the last hour:
```
SELECT max(field1), mean(field1), min(field1) FROM measurement1 WHERE tag1=bar AND time > now() - 1h GROUP BY tag
```

### Example Output:

This section shows example output in Line Protocol format.  You can often use
`telegraf --input-filter <plugin-name> --test` or use the `file` output to get
this information.

```
twitter,id_str=966018857503723521,retweeted=false,favorited=false retweet_count=305,user_followers_count=62351642,user_friends_count=145,favorite_count=0,user_listed_count=90902,user_favourites_count=5501,user_statuses_count=6575 1519230765000000000
twitter,id_str=965492398393516033,retweeted=false,favorited=false user_friends_count=145,user_listed_count=90902,favorite_count=0,retweet_count=323,user_favourites_count=5501,user_followers_count=62351642,user_statuses_count=6575 1519230765000000000
twitter,id_str=965371971482431490,retweeted=false,favorited=false retweet_count=106,user_friends_count=145,user_statuses_count=6575,favorite_count=0,user_followers_count=62351642,user_favourites_count=5501,user_listed_count=90902 1519230765000000000
twitter,id_str=965368695265533952,retweeted=false,favorited=false user_friends_count=145,user_followers_count=62351642,user_statuses_count=6575,user_listed_count=90902,retweet_count=64,favorite_count=0,user_favourites_count=5501 1519230765000000000
```


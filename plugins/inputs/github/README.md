# GitHub Input Plugin

The GitHub plugin collects data from GitHub REST API v3 endpoints which respond with JSON. It flattens the JSON and finds all numeric values, treating them as floats.

Refer to the [GitHub REST API v3](https://developer.github.com/v3/), specifically the [User Repositories] (https://developer.github.com/v3/repos/#list-user-repositories), and the 

### Configuration:

```toml
[[inputs.github]]
  interval = "24h"

  org_id = "YOUR_ORG_ID"
  
  ## Set response_timeout (default 5 seconds)
  response_timeout = "5s"
  
  get_contributors = true

  ## List of tag names to extract from top-level of JSON server response
  tag_keys = [
    "id",
	"name",
	"full_name",
	"created_at",
	"updated_at",
	"pushed_at",
	"login",
  ]
    
  ## HTTP Headers (all values must be strings)
  [inputs.github.headers]
     Authorization = "token YOUR_AUTH_TOKEN"
  
  fieldpass = ["forks", "open_issues", "watchers", "stargazers_count", "contributions"]
```


### Measurements & Fields:

- github
	- tags:
		- id (integer): the GitHub repo ID
		- name (string): the repo name
		- full_name (string): the full repo name, including the org name
		- created_at (date): the date the repo was created at
		- updated_at (date): the date when the repo was last update at
		- pushed_at (date): the date when the repo was last pushed at
	- fields:
		- forks (float): the number of Forks of this repo
		- stargazers_count (float): the number of Stargazers of this repo
		- watchers (float): the number of Watchers of this repo
		- open_issues (float): the number of Open Issues of this repo

- github_contributors
	- tags:
		- id (integer): the GitHub user ID
		- login (string): the GitHub user name
		- repo (string): the full repo name, including the org name
		- repo_id (integer): the GitHub repo ID
	- fields:
		- contributions (float): the number of Contributions this user has made to the repo
		
### Example Output:

This section shows example output in Line Protocol format.  You can often use
`telegraf --input-filter github --test` or use the `file` output to get
this information.

```
github_contributors,id=101085,login=akutz,repo=thecodeteam/goxtremio,repo_id=36694266 contributions=4 1519752795000000000
github_contributors,repo_id=50635537,id=1395761,login=clintkitson,repo=thecodeteam/heat contributions=1 1519752795000000000
github,id=49217195,name=ansible-role-rexray,full_name=thecodeteam/ansible-role-rexray,created_at=2016-01-07T16:47:49Z,updated_at=2018-02-09T18:33:27Z,pushed_at=2017-05-10T00:12:48Z forks=8,watchers=8,stargazers_count=8,open_issues=5 1519752539000000000
github,created_at=2016-01-29T03:50:22Z,updated_at=2018-02-01T06:15:32Z,pushed_at=2016-01-29T04:15:12Z,id=50635723,name=cloudformation,full_name=thecodeteam/cloudformation forks=1,stargazers_count=1,watchers=1,open_issues=0 1519752539000000000
github,pushed_at=2017-06-23T17:59:50Z,id=23676370,name=codedellemc.github.io,full_name=thecodeteam/codedellemc.github.io,created_at=2014-09-04T19:31:01Z,updated_at=2018-02-07T15:35:37Z watchers=45,forks=22,stargazers_count=45,open_issues=1 1519752539000000000

```

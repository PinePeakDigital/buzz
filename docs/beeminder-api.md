---
title: Beeminder API Reference

language_tabs:
  - shell 
  - ruby

toc_footers:
  - <a href='https://www.beeminder.com'>Beeminder</a>
  - <a href='https://www.beeminder.com/apps/new'>Get an API key</a>
  - <a href='https://github.com/beeminder/apidocs'>Contribute to these docs!</a>
  - <a href='https://github.com/lord/slate'>Docs powered by Slate</a>

includes:
  - errors

search: false
---


# Beeminder API Reference

## Introduction

> See 
> <a href="https://github.com/beeminder" title="The Beeminder org on GitHub">github.com/beeminder</a> 
> for API libraries in various languages.
> Examples here are currently just Curl and Ruby. 



In case you're here to automate adding data to Beeminder, there's a good chance 
we've got you covered with our 
[Zapier integration](http://beeminder.com/zapier "Zapier is a service like IFTTT that connects hundreds of disparate webservices. In the case of Beeminder you can create triggers in other webservices that automatically cause data to be added to Beeminder graphs.")
or our
[IFTTT integration](http://ifthisMINDthat.com "IFTTT = If This Then That").

The
[tech category of our forum](http://forum.beeminder.com/c/tech "Yay Discourse.org!")
is a good place to ask questions and show off what you're working on.
It's really important to us that this API be easy for developers to use so please don't be shy about asking us questions.
Whether you post in
[the forum](http://forum.beeminder.com "The above link is the Tech subset of the forum; this link is to the main page for the forum")
or email us at **support@beeminder.com** we've invariably found that questions people avoided asking for fear they were dumb turned out to point to things we needed to improve in the API or the documentation.
So lean on us heavily as you're hacking away with our API &mdash; it helps us a lot when you do!

If you're looking for ideas for things to do with the Beeminder API, we have a
[blog post with lots of examples](http://blog.beeminder.com/api "This was the blog post that originally announced our API in 2012").
While we're talking about things that people have done with the Beeminder API, we want to ladle out some extra praise for the
[Intend](https://intend.do/features?utm_source=beeminder&utm_medium=link&utm_campaign=beem-api-docs#beeminder "Describing how Beeminder integration complements Intend") 
and 
[TaskRatchet](https://docs.taskratchet.com/integrations.html#beeminder "Describing how Beeminder works with TaskRatchet") integrations.
Malcolm and Narthur, respectively, have built useful Beeminder integrations without us having to lift a finger. 
A beautiful example of our API earning its keep. 



## Preliminaries

### API Base URL
The base URL for all requests is `https://www.beeminder.com/api/v1/`.

You may also consume the Beeminder API via
[RapidAPI](https://rapidapi.com/beeminder/api/beeminder "RapidAPI (formerly Mashape) is a hub for cloud APIs (is how Wikipedia puts it)").


### Troubleshooting
A common mistake is to use the wrong URL, e.g., using an `http` protocol instead of `https`, or leaving out the `www` subdomain. 
We redirect insecure and non `www` requests to the canonical Beeminder URL, but do not forward parameters for `POST` requests, so some things will break opaquely if you don't use exactly the above base URL.
(And even worse, others will not.)

Also, please pay attention to the type of HTTP request an endpoint is expecting.
For example, the `Goal#create` endpoint and `Goal#update` endpoint differ primarily in whether you are making a `POST` or a `PUT` request (respectively).


### Backwards Compatibility
A note of caution for the reader: We document the User and Goal resources below, explaining various attributes that you will find in the API outputs. If you inspect the API outputs you get, however, you'll probably notice a bunch of info that's not included here. You can use any of that info you like, but it may change at our whim down the road. (That's why we don't document it.) 


[Back to top](#)

<h1 id="auth">Authentication</h1>

All API endpoints require authentication.
There are two ways to authenticate.
Both ultimately give you a token which you must then include with every API request.

<aside class="notice">
Note: A common mistake is to pass the personal auth token but call the parameter
access_token, or vice-versa. 
The parameter name for your personal auth token should be `auth_token`.
</aside>

## Personal authentication token

> For example, if your username is "alice" and your token is "abc123" you can 
> query information about your "weight" goal like so:

```shell
  curl https://www.beeminder.com/api/v1/users/alice/goals/weight.json?auth_token=abc123
```


This authentication pattern is for making API calls just to your own Beeminder account.


After you
[log in to Beeminder](https://www.beeminder.com/users/sign_in ),
visit
<a href="https://www.beeminder.com/api/v1/auth_token.json">`https://www.beeminder.com/api/v1/auth_token.json`</a>
to get your personal auth token.
Append it to API requests you make as an additional GET or POST parameter.


## Client OAuth


This authentication pattern is for clients (applications) accessing the Beeminder API on a user's behalf.
Beeminder implements the
[OAuth](http://oauth.net/ "Specifically Oauth2")
provider protocol to allow access reasonably securely.

There are four steps to build a client:

<h3 id="registerapp">1. Register your app</h3>

Register your app at
[beeminder.com/apps/new](https://www.beeminder.com/apps/new ).
Application name and redirect URL are required.
The redirect URL is where the user is sent after authorizing your app.

<h3 id="authurl">2. Send your users to the Beeminder authorization URL</h3>

> Example authorization URL:

```
  https://www.beeminder.com/apps/authorize?\
   client_id=xyz456&redirect_uri=http&#58;//foo.com/auth_callback\
   &response_type=token
```

The base URL is the same for all apps:
`https://www.beeminder.com/apps/authorize`.
You'll need to add the following parameters:

* `client_id`: Your application's client ID.
You can see a list of your registered apps and retrieve their `client_id`s at
[beeminder.com/apps](https://www.beeminder.com/apps "List of apps you've registered, not to be confused with the list of apps you've authorized to access your Beeminder account, or the list of services you've authorized Beeminder to access").
* `redirect_uri`: This is where Beeminder will send the user after they have authorized your app.
This *must match* the redirect URL you supplied when you registered your app above.
Make sure to
[url-encode](http://en.wikipedia.org/wiki/Percent-encoding "Where you replace characters that have special meaning in URLs with a percent sign and their ascii number (or plus signs for spaces)")
this if it contains any special characters like question marks or ampersands.
* `response_type`: Currently this should just always be set to the value "`token`".


<h3 id="storetoken">3. Receive and store user's access token</h3>

> For example, if the user "alice" has access token "abc123" then the following string would be appended to the URL when the user is redirected there:

```
  ?access_token=abc123&username=alice
```

After the user authorizes your application they'll be redirected to the `redirect_uri` that you specified, with two additional parameters,
`access_token` and
`username`, in the
[query string](http://en.wikipedia.org/wiki/Query_string "The query string is the parameters that come after the question mark in a URL").


You should set up your server to handle this GET request and have it remember each user's access token.
The access token uniquely identifies the user's authorization for your app.

The username is provided here for convenenience.
You can retrieve the username for a given access token at any time by sending a GET request for `/api/v1/me.json` with the token appended as a parameter.

<h3 id="tokenparam">4. Include access token in your request</h3>

```shell
  curl https://www.beeminder.com/api/v1/users/me.json?access_token=abc123

  or

  curl -H "Authorization: Bearer abc123" https://www.beeminder.com/api/v1/users/me.json
```

Append the access token as a parameter on any API requests you make on behalf of that user, or include it in the request headers using the `Authorization: Bearer` scheme.
For example, your first request will probably be to get information about the
[User](#user)
who just authorized your app.


You can literally use "me" in place of the username for any endpoint and it will be macro-expanded to the username of the authorized user.

<h3 id="deauth">5. Optional: De-authorization callback</h3>

If you provide a Post De-Authorization Callback URL when you register your client, we will make a POST to your endpoint when a user removes your app. The POST will include a single parameter, `access_token` in the body of the request. The value of this parameter will be the token that was de-authorized.

<h3 id="autofetch">6. Optional: Autofetch callback</h3>

The autofetch callback URL is also optional.
We will POST to this URL if provided, including the parameters `username` and `slug` in the body of the request when the user wants new data from you.
E.g., when the user pushes the manual refresh button, or prior to sending alerts to the user, and before derailing the goal at the end of a beemergency day.

[Back to top](#)

<h1 id="user">User Resource</h1>

A User object ("object" in the
[JSON](http://json.org "JavaScript Objection Notation aka how data is passed around on the internet")
sense) includes information about a user, like their list of goals.

### Attributes

* `username` (string)
* `timezone` (string)
* `updated_at` (number):
[Unix timestamp](http://en.wikipedia.org/wiki/Unix_time "Number of seconds since 1970-01-01 at midnight GMT")
(in seconds) of the last update to this user or any of their goals or datapoints.
* `goals` (array):
A list of slugs for each of the user's goals, or an array of goal hashes (objects) if `diff_since` or `associations` is sent.
* `deadbeat` (boolean):
True if the user's payment info is out of date, or an attempted payment has failed.
* `urgency_load` (number):
The idea of Urgency Load is to construct a single number that captures how edge-skatey you are across all your goals. A lower number means fewer urgently due goals. A score of 0 means that you have >= 7 days of buffer on all of your active goals.
* `deleted_goals` (array):
An array of hashes, each with one key/value pair for the id of the deleted goal.
Only returned if `diff_since` is sent.


<h2 id="getuser">Get information about a user</h2>

> Examples

```shell
  curl https://www.beeminder.com/api/v1/users/alice.json?auth_token=abc123

  { "username": "alice",
    "timezone": "America/Los_Angeles",
    "updated_at": 1343449880,                       
    "goals": ["gmailzero", "weight"] }
```

```ruby
  require 'beeminder'

  bee = Beeminder::User.new "yourtoken"
  bee.info
```

```shell
  curl https://www.beeminder.com/api/v1/users/alice.json?diff_since=1352561989&auth_token=abc123
```

```json
  { "username": "alice",
    "timezone": "America/Los_Angeles",
    "updated_at": 1343449880,                       
    "goals": [ {"slug": "weight", ...,
               "datapoints": [{"timestamp": 1325523600,    
                    "value": 70.45,            
                    "comment": "blah blah",     
                    "id": "4f9dd9fd86f22478d3"},
                   {"timestamp": 1325610000,
                    "value": 70.85,
                    "comment": "blah blah",
                    "id": "5f9d79fd86f33468d4"}],
               "title": "Weight Loss", ...},
               { another goal }, ... ],
    "deleted_goals": [{ "id": "519279fd86f33468ne"}, ... ] }
```

### HTTP Request

`GET /users/`*u*`.json`

Retrieves information and a list of goalnames for the user with username *u*.

Since appending an `access_token` to the request uniquely identifies a user, you can alternatively make the request to /users/me.json (without the username).

### Parameters


* \[`associations`\] (boolean): Convenience method to fetch all information about a user. Please use sparingly and see also the `diff_since` parameter.
Default: false  
Send `true` if you want to receive all of the user's goal and datapoints as attributes of the user object.

* \[`diff_since`\] (number): Unix timestamp in seconds.
Default: null, which will return all goals and datapoints  
Send a Unix timestamp to receive a filtered list of the user's goals and datapoints.
Only goals and datapoints that have been created or updated since the timestamp will be returned.
Sending `diff_since` implies that you want the user's associations, so you don't need to send both.

* \[`skinny`\] (boolean): Convenience method to only get a subset of goal attributes and the most recent datapoint for the goal.
Default: false, which will return all goal attributes and all datapoints created or updated since `diff_since`.  
`skinny` must be sent along with `diff_since`.
If `diff_since` is not present, `skinny` is ignored.
Some goal attributes, as well as fetching all datapoints, can take some additional time to compute on the server side, so you can send `skinny` if you only need the latest datapoint and the following subset of attributes:
`slug,
title,
description,
goalval,
rate,
goaldate,
svg_url,
graph_url,
thumb_url,
goal_type,
autodata,
losedate,
urgencykey,
deadline,
leadtime,
alertstart,
id,
queued,
updated_at,
burner,
yaw,
lane,
delta,
runits,
limsum,
frozen,
lost,
won,
contract,
delta_text,
safebump,
gunits,
todayta,
timey
hhmmformat
`
Instead of a `datapoints` attribute, sending `skinny` will replace that attribute with a `last_datapoint` attribute. Its value is a Datapoint hash.

* \[`emaciated`\] (boolean):
If included the goal attributes called `road`, `roadall`, and `fullroad` will be stripped from any goal objects returned with the user. 
Default: false.

* \[`datapoints_count`\] (number): number of datapoints.
Default: null, which will return all goals and datapoints.
Send a number `n` to only recieve the `n` most recently added datapoints, sorted by `updated_at`.
Note that the most recently added datapoint could have been a datapoint whose timestamp is well in the past and therefore before other datapoints in that respect.
For example, my datapoints might look like:  
<br>
12 1  
14 1  
15 1  
16 1  
<br>
If I go back and realize that I forgot to enter data on the 13th, the datapoint for the 13th will be sorted ahead of the one on the 16th:  
<br>
12 1  
14 1  
15 1  
16 1  
13 1  

### Returns

A [User](#user) object.


Use the `updated_at` field to be a good Beeminder API citizen and avoid unnecessary requests for goals and datapoints.
Any updates to a user, their goals, or any datapoints on any of their goals will cause this field to be updated to the current unix timestamp.
If you store the returned value and, on your next call to this endpoint, the value is the same, there's no need to make requests to other endpoints.

Checking the timestamp is an order of magnitude faster than retrieving all the data, so it's definitely wise to use this approach.


<h2 id="redirectuser">Authenticate and redirect the user</h2>

> Examples

```shell
  curl https://www.beeminder.com/api/v1/users/alice.json?auth_token=abc123&redirect_to_url=https%3A%2F%2Fwww.beeminder.com%2Fpledges
```

### HTTP Request

`GET /users/`*u*`.json`

Attempts to authenticate the user and if successful redirects to the given URL.
Allows third-party apps to send the user to a specific part of the website without getting intercepted by a login screen, for doing things not available through the API.

### Parameters

* \[`redirect_to_url`\] (string): Url to redirect the user to.



[Back to top](#)



<h1 id="goal">Goal Resource</h1>

A Goal object includes everything about a specific goal for a specific user, including the target value and date, the steepness of the bright red line, the graph image, and various settings for the goal. 

### Attributes

* `slug` (string): The final part of the URL of the goal, used as an identifier. E.g., if user "alice" has a goal at beeminder.com/alice/weight then the goal's slug is "weight".
* `updated_at` (number): [Unix timestamp](http://en.wikipedia.org/wiki/Unix_time ) of the last time this goal was updated.
* `title` (string): The title that the user specified for the goal. E.g., "Weight Loss".
* `fineprint` (string): The user-provided description of what exactly they are committing to.
* `yaxis` (string): The label for the y-axis of the graph. E.g., "Cumulative total hours".
* `goaldate` (number): Unix timestamp (in seconds) of the goal date. NOTE: this may be null; [see below](#one-of-three).
* `goalval` (number): Goal value &mdash; the number the bright red line will eventually reach. E.g., 70 kilograms. NOTE: this may be null; [see below](#one-of-three).
* `rate` (number): The slope of the (final section of the) bright red line. You must also consider `runits` to fully specify the rate. NOTE: this may be null; [see below](#one-of-three). 
* `runits` (string): Rate units. One of `y`, `m`, `w`, `d`, `h` indicating that the rate of the bright red line is yearly, monthly, weekly, daily, or hourly.
* `svg_url` (string): URL for the goal's graph svg. E.g., "http://static.beeminder.com/alice/weight.svg".
* `graph_url` (string): URL for the goal's graph image. E.g., "http://static.beeminder.com/alice/weight.png".
* `thumb_url` (string): URL for the goal's graph thumbnail image. E.g., "http://static.beeminder.com/alice/weight-thumb.png".
* `autodata` (string): The name of automatic data source, if this goal has one. Will be null for manual goals.
* `goal_type` (string): One of the following symbols (detailed info [below](#goal-types)):
 - `hustler`: Do More
 - `biker`: Odometer
 - `fatloser`: Weight loss
 - `gainer`: Gain Weight
 - `inboxer`: Inbox Fewer
 - `drinker`: Do Less
 - `custom`: Full access to the underlying goal parameters
* `losedate` (number): Unix timestamp of derailment. When you'll cross the bright red line if nothing is reported.
* `urgencykey` (string): Sort by this key to put the goals in order of decreasing urgency. (Case-sensitive ascii or unicode sorting is assumed). This is the order the goals list comes in. Detailed info [on the blog](https://blog.beeminder.com/urgency).
* `queued` (boolean): Whether the graph is currently being updated to reflect new data.
* `secret` (boolean): Whether you have to be logged in as owner of the goal to view it. Default: `false`.
* `datapublic` (boolean): Whether you have to be logged in as the owner of the goal to view the datapoints. Default: `false`.
* `datapoints` (array of [Datapoints](#datapoint)): The datapoints for this goal.
* `numpts` (number): Number of datapoints.
* `pledge` (number): Amount pledged (USD) on the goal.
* `initday` (number): Unix timestamp (in seconds) of the start of the bright red line.
* `initval` (number): The y-value of the start of the bright red line.
* `curday` (number): Unix timestamp (in seconds) of the end of the bright red line, i.e., the most recent (inferred) datapoint.
* `curval` (number): The value of the most recent datapoint.
* `currate` (number): The rate of the red line at time `curday`; if there's a rate change on that day, take the limit from the left.
* `lastday` (number): Unix timestamp (in seconds) of the last (explicitly entered) datapoint.
* `yaw` (number): Good side of the bright red line. I.e., the side of the line (+1/-1 = above/below) that makes you say "yay".
* `dir` (number): Direction the bright red line is sloping, usually the same as yaw.
* `lane` (number): Deprecated. See `losedate` and `safebuf`.
* `mathishard` (array of 3 numbers): The goaldate, goalval, and rate &mdash; all filled in. (The commitment dial specifies 2 out of 3 and you can check this if you want Beeminder to do the math for you on inferring the third one.) Note: this field may be null if the goal is in an error state such that the graph image can't be generated.
* `headsum` (string): Deprecated. Summary text blurb saying how much safety buffer you have.
* `limsum` (string): Summary of what you need to do to eke by, e.g., "+2 within 1 day".
* `kyoom` (boolean): Cumulative; plot values as the sum of all those entered so far, aka auto-summing.
* `odom` (boolean): Treat zeros as accidental odometer resets.
* `aggday` (string): How to aggregate points on the same day, eg, min/max/mean.
* `steppy` (boolean): Join dots with purple steppy-style line.
* `rosy` (boolean): Show the rose-colored dots and connecting line.
* `movingav` (boolean): Show moving average line superimposed on the data.
* `aura` (boolean): Show turquoise swath, aka blue-green aura.
* `frozen` (boolean): Whether the goal is currently frozen and therefore must be restarted before continuing to accept data.
* `won` (boolean): Whether the goal has been successfully completed.
* `lost` (boolean): Whether the goal is currently off track.
* `maxflux` (Integer): Max daily fluctuation for weight goals. Used as an absolute buffer amount after a derail. Also shown on the graph as a thick guiding line.
* `contract` (dictionary): Dictionary with two attributes. `amount` is the amount at risk on the contract, and `stepdown_at` is a Unix timestamp of when the contract is scheduled to revert to the next lowest pledge amount. `null` indicates that it is not scheduled to revert.
* `road` (array): Array of tuples that can be used to construct the Bright Red Line (formerly "Yellow Brick Road"). This field is also known as the graph matrix. Each tuple specifies 2 out of 3 of \[`time`, `goal`, `rate`\]. To construct `road`, start with a known starting point (time, value) and then each row of the graph matrix specifies 2 out of 3 of {t,v,r} which gives the segment ending at time t. You can walk forward filling in the missing 1-out-of-3 from the (time, value) in the previous row.
* `roadall` (array): Like `road` but with an additional initial row consisting of \[`initday`, `initval`, null\] and an additional final row consisting of \[`goaldate`, `goalval`, `rate`\].
* `fullroad` (array): Like `roadall` but with the nulls filled in.
* `rah` (number): Red line value (y-value of the bright red line) at the akrasia horizon (today plus one week).
* `delta` (number): Distance from the bright red line to today's datapoint (`curval`).
* `delta_text` (string): Deprecated.
* `safebuf` (number): The integer number of safe days. If it's a beemergency this will be zero.
* `safebump` (number): The absolute y-axis number you need to reach to get one additional day of safety buffer.
* `autoratchet` (number): The goal's autoratchet setting. If it's not set or they don't have permission to autoratchet, its value will be nil. This represents the maximum number of days of safety buffer the goal is allowed to accrue, or in the case of a Do-Less goal, the max buffer in terms of the goal's units. Read-only. 
* `id` (string of hex digits): We prefer using user/slug as the goal identifier, however, since we began allowing users to change slugs, this id is useful!
* `callback_url` (string): Callback URL, as
[discussed in the forum](http://forum.beeminder.com/t/webhook-callback-documentation/313 "In short: you can add a callback to your own server whenever data is added on Beeminder").
WARNING: If different apps change this they'll step on each other's toes.
* `description` (string): Deprecated. User-supplied description of goal (listed in sidebar of graph page as "Goal Statement").
* `graphsum` (string): Deprecated. Text summary of the graph, not used in the web UI anymore.
* `lanewidth` (number): Deprecated. Now always zero.
* `deadline` (number): Seconds by which your deadline differs from midnight. Negative is before midnight, positive is after midnight.
Allowed range is -17*3600 to 6*3600 (7am to 6am).
* `leadtime` (number): Days before derailing we start sending you reminders. Zero means we start sending them on the beemergency day, when you will derail later that day.
* `alertstart` (number): Seconds after midnight that we start sending you reminders (on the day that you're scheduled to start getting them, see `leadtime` above).
* `plotall` (boolean): Whether to plot all the datapoints, or only the `aggday`'d one. So if false then only the official datapoint that's counted is plotted.
* `last_datapoint` ([Datapoint](#datapoint)): The last datapoint entered for this goal.
* `integery` (boolean): Assume that the units must be integer values.  Used for things like `limsum`.
* `gunits` (string): Goal units, like "hours" or "pushups" or "pages".
* `timey` (boolean): Whether to show data in a "timey" way, with colons.  For example, this would make a 1.5 show up as 1:30. This replaces `hhmmformat`.
* `hhmmformat` (boolean): DEPRECATED. Will be removed in a future version. Use `timey` instead.
* `todayta` (boolean): Whether there are any datapoints for today
* `weekends_off` (boolean): If the goal has weekends automatically scheduled.
* `tmin` (string): Lower bound on x-axis; don't show data before this date; using yyyy-mm-dd date format. (In Graph Settings this is 'X-min')
* `tmax` (string): Upper bound on x-axis; don't show data after this date; using yyyy-mm-dd date format. (In Graph Settings this is 'X-max')
* `tags` (array): A list of the goal's tags.

<em id="one-of-three">A note about rate, date, and val:</em> One of the three fields `goaldate`, `goalval`, and `rate` will return a null value.
This indicates that the value is calculated based on the other two fields, as selected by the user.

<em id="goal-types">A detailed note about goal types:</em> The goal types are shorthand for a collection of settings of more fundamental goal attributes.
Note that changing the goal type of an already-created goal has no effect on those fundamental goal attributes.
The following table lists what those attributes are.

parameter | `hustler` | `biker` | `fatloser` | `gainer` | `inboxer` | `drinker`
--------- | --------- | ------- | ---------- | -------- | --------- | ---------
`yaw` | 1 | 1 | -1 | 1 | -1 | -1
`dir` | 1 | 1 | -1 | 1 | -1 | 1
`kyoom`| true| false| false| false| false| true
`odom` | false |true |false |false |false| false
`edgy` | false |false |false |false |false |true
`aggday`| "sum" |"last" |"min" |"max" |"min" |"sum"
`steppy`| true |true |false |false |true |true
`rosy`| false |false| true| true| false| false
`movingav`| false |false |true |true |false |false
`aura`|false|false|true| true |false |false

There are four broad, theoretical categories &mdash; called the platonic goal types &mdash; that goals fall into, defined by `dir` and `yaw`:

`MOAR = dir +1 & yaw +1`: "go up, like work out more"<br>
`PHAT = dir -1 & yaw -1`: "go down, like weightloss or gmailzero"<br>
`WEEN = dir +1 & yaw -1`: "go up less, like quit smoking"<br>
`RASH = dir -1 & yaw +1`: "go down less, ie, rationing, <a href="http://beeminder.com/d/contacts" title="The Beeminder CEO with an early Beeminder graph to ration his supply of 'daily' contact lenses to last for 2 years, till 2013">for example</a>"

The `dir` parameter, for which direction the bright red line is expected to go, is mostly just for the above categorization, but is used specifically in the following ways:

1. Where to draw the watermarks (amount pledged and number of safe days)
2. How to phrase things like "bare min of +123 in 4 days" and the status line (also used in bot email subjects)
3. Which direction is the optimistic one for the rosy dots algorithm

<aside class="notice">
Clearing up confusion about WEEN and RASH goal types: Beeminder generally plots the cumulative total of your metric, such as total cigarettes smoked. So even a quit-smoking goal will slope up (dir&gt;0). Just that it will slope up less and less steeply as you wean yourself. When you actually quit, the slope will be zero. That's why "WEEN" goals are sloping up but good side is down. The opposite case &mdash; sloping down but good side's up &mdash; is called "RASH" and is rarely used. It's for beeminding a number that you want to go down slowly. Maybe cigarettes remaining in a carton that you want to be your last, or bottles of fresh water remaining post-apocalypse &mdash; someday this goal type will be useful!
</aside>

If you just want the dot color, here's how to infer it from `safebuf` (see code in sidebar).

```
color = (safebuf < 1 ? "red"    :
         safebuf < 2 ? "orange" :
         safebuf < 3 ? "blue"   : 
         safebuf < 7 ? "green"  : "gray")
```

Finally, the way to tell if a goal has finished successfully is `now >= goaldate && goaldate < losedate`.
That is, you win if you hit the goal date before hitting `losedate`.
You don't have to actually reach the goal value &mdash; staying on the right side of the bright red line till the end suffices.




<h2 id="getgoal">Get information about a goal</h2>

> Examples


```shell
  curl https://www.beeminder.com/api/v1/users/alice/goals/weight.json?auth_token=abc123&datapoints=true
```

```json
  { "slug": "weight",               
    "title": "Weight Loss",         
    "goaldate": 1358524800,         
    "goalval": 166,                 
    "rate": null,                   
    "svg_url":   "http://static.beeminder.com/alice+weight.svg",
    "graph_url": "http://static.beeminder.com/alice+weight.png",
    "thumb_url": "http://static.beeminder.com/alice+weight-thumb.png",    
    "goal_type": "fatloser",            
    "losedate": 1358524800,        
    "queued": false,                
    "updated_at": 1337479214,       
    "datapoints": [{"timestamp": 1325523600,    
                    "value": 70.45,            
                    "comment": "blah blah",     
                    "id": "4f9dd9fd86f22478d3"},
                   {"timestamp": 1325610000,
                    "value": 70.85,
                    "comment": "blah blah",
                    "id": "5f9d79fd86f33468d4"}]}

```

### HTTP Request

`GET /users/`*u*`/goals/`*g*`.json`

Gets goal details for user *u*'s goal *g* &mdash; beeminder.com/*u*/*g*.

### Parameters

* \[`datapoints`\] (boolean): Whether to send the goal's datapoints in the response. Default: `false`.
* \[`emaciated`\] (boolean):
If included the goal attributes called `road`, `roadall`, and `fullroad` will be stripped from the goal object. 
Default: false.

### Returns

A [Goal](#goal) object, possibly without the datapoints attribute.


<h2 id="getgoals">Get all goals for a user</h2>


> Examples

```shell
  curl https://www.beeminder.com/api/v1/users/alice/goals.json?auth_token=abc123
```

```json
  [ { "slug": "gmailzero",
      "title": "Inbox Zero",
      "goal_type": "inboxer",
      "svg_url": "http://static.beeminder.com/alice+gmailzero.svg",
      "graph_url": "http://static.beeminder.com/alice+gmailzero.png",
      "thumb_url": "http://static.beeminder.com/alice+weight-thumb.png",
      "losedate": 1347519599,
      "goaldate": 0,
      "goalval": 25.0,
      "rate": -0.5,
      "updated_at": 1345774578,
      "queued": false },
    { "slug": "fitbit-me",
      "title": "Never stop moving",
      "goal_type": "hustler",
      "svg_url": "http://static.beeminder.com/alice+fitbit-me.svg",
      "graph_url": "http://static.beeminder.com/alice+fitbit-me.png",
      "thumb_url": "http://static.beeminder.com/alice+fitbit-thumb.png",
      "losedate": 1346482799,
      "goaldate": 1349582400,
      "goalval": null,
      "rate": 8.0,
      "updated_at": 1345771188,
      "queued": false } ]
```

### HTTP Request

`GET /users/`*u*`/goals.json`

Get user *u*'s list of goals.

### Parameters

* \[`emaciated`\] (boolean):
If included the goal attributes called `road`, `roadall`, and `fullroad` will be stripped from the goal objects. 
Default: false.

### Returns

A list of [Goal](#goal) objects for the user.
Goals are sorted in descending order of urgency, i.e., increasing order of time to derailment.
(There's actually a very tiny caveat to this involving the long-deprecated "sort threshold" parameter.
If you don't know what that is then you can ignore this parenthetical!)

<h2 id="getarchivedgoals">Get archived goals for a user</h2>


> Examples

```shell
  curl https://www.beeminder.com/api/v1/users/alice/goals/archived.json?auth_token=abc123
```

```json
  [ { "slug": "gmailzero",
      "title": "Inbox Zero",
      "goal_type": "inboxer",
      "svg_url": "http://static.beeminder.com/alice+gmailzero.svg",
      "graph_url": "http://static.beeminder.com/alice+gmailzero.png",
      "thumb_url": "http://static.beeminder.com/alice+weight-thumb.png",
      "losedate": 1347519599,
      "goaldate": 0,
      "goalval": 25.0,
      "rate": -0.5,
      "updated_at": 1345774578,
      "queued": false },
    { "slug": "fitbit-me",
      "title": "Never stop moving",
      "goal_type": "hustler",
      "svg_url": "http://static.beeminder.com/alice+fitbit-me.svg",
      "graph_url": "http://static.beeminder.com/alice+fitbit-me.png",
      "thumb_url": "http://static.beeminder.com/alice+fitbit-thumb.png",
      "losedate": 1346482799,
      "goaldate": 1349582400,
      "goalval": null,
      "rate": 8.0,
      "updated_at": 1345771188,
      "queued": false } ]
```

### HTTP Request

`GET /users/`*u*`/goals/archived.json`

Get user *u*'s archived goals.

### Parameters

* \[`emaciated`\] (boolean):
If included the goal attributes called `road`, `roadall`, and `fullroad` will be stripped from the goal objects. 
Default: false.

### Returns

A list of [Goal](#goal) objects representing the user's archived goals.

<h2 id="creategoal">Create a goal for a user</h2>


> Examples

```shell
  curl -X POST https://www.beeminder.com/api/v1/users/alice/goals.json \
    -d auth_token=abc123 \
    -d slug=exercise \
    -d title=Work+Out+More \
    -d goal_type=hustler \
    -d goaldate=1400000000 \
    -d gunits=workouts \
    -d rate=5 \
    -d goalval=null
```

```json
  { "slug": "exercise",
    "title": "Work Out More",
    "goal_type": "hustler",
    "svg_url": "http://static.beeminder.com/alice+exercise.svg",
    "graph_url": "http://static.beeminder.com/alice+exercise.png",
    "thumb_url": "http://static.beeminder.com/alice+exercise-thumb.png",
    "losedate": 1447519599,
    "goaldate": 1400000000,
    "goalval": null,
    "rate": 5,
    "updated_at": 1345774578,
    "queued": false }
```

### HTTP Request

`POST /users/`*u*`/goals.json`

Create a new goal for user *u*.

### Parameters

* `slug` (string)
* `title` (string)
* `goal_type` (string)
* `gunits` (string)
* `goaldate` (number or null)
* `goalval` (number or null)
* `rate` (number or null)
* `initval` (number): Initial value for today's date. Default: 0.
* \[`secret`\] (boolean)
* \[`datapublic`\] (boolean)
* \[`datasource`\] (string): one of {"api", "ifttt", "zapier", or `clientname`\}. Default: none (i.e., "manual").
* \[`dryrun`\] (boolean). Pass this to test the endpoint without actually creating a goal. Defaults to false.
* \[`tags`\] (array). An optional list of tags to add to the new goal. Each tag must be an alphanumeric string.

[Exactly](http://youtu.be/QM9Bynjh2Lk?t=4m14s) two out of three of `goaldate`, `goalval`, and `rate` are required.

If you pass in your API client's registered name for the `datasource`, and your client has a registered `autofetch_callback_url`, we will POST to your callback when this goal wants new data, as outlined in [Client OAuth](#6-optional-autofetch-callback ).

### Returns

The newly created [Goal](#goal) object.


<h2 id="putgoal">Update a goal for a user</h2>

> Examples

```shell
  curl -X PUT https://www.beeminder.com/api/v1/users/alice/goals/exercise.json \
    -d auth_token=abc124 \
    -d title=Work+Out+Even+More \
    -d secret=true
```

```json
  { "slug": "exercise",
    "title": "Work Out Even More",
    "goal_type": "hustler",
    "svg_url": "http://static.beeminder.com/alice+exercise.svg",
    "graph_url": "http://static.beeminder.com/alice+exercise.png",
    "thumb_url": "http://static.beeminder.com/alice+exercise-thumb.png",
    "secret": true,
    "losedate": 1447519599,
    "goaldate": 1400000000,
    "goalval": null,
    "rate": 5,
    "updated_at": 1345774578,
    "queued": false }
```

### HTTP Request

`PUT /users/`*u*`/goals/`*g*`.json`

Update user *u*'s goal with slug *g*.
This is similar to the call to create a new goal, but the goal type (`goal_type`) cannot be changed.
To change any of {`goaldate`, `goalval`, `rate`} use `roadall`.

### Parameters

* \[`title`\] (string)
* \[`yaxis`\] (string)
* \[`tmin`\] (string) date format "yyyy-mm-dd"
* \[`tmax`\] (string) date format "yyyy-mm-dd"
* \[`secret`\] (boolean)
* \[`datapublic`\] (boolean)
* \[`roadall`\] (array of arrays like `[date::int, value::float, rate::float]` each with exactly one field null)
  * This must not make the goal easier between now and the akrasia horizon (unless you are an admin).
  * Use `roadall` returned by [goal GET](#getgoal), not `road` &mdash; the latter is missing the first and last rows (for the sake of backwards compatibility).
  * The first row must be `[date, value, null]` and gives the start of the bright red line, same as `initday` and `initval` in [goal GET](#getgoal).
  * The last row can be `[null, value, rate]` but no other row can be.
  * You can also send a `roadall` with dates specified as either a daystamp or date string, e.g., "20170727" or "2017-07-27".
  * This is a superset of `dial_road` (which changes just the last row of this `roadall`).
  * If you change rate units in the same call, the bright red line will be updated first, and rate units second, so make adjustments to the bright red line in terms of the original rate units, or make two separate calls, first updating rate units, then sending your adjusted `roadall`.
* \[`datasource`\] (string): one of {"api", "ifttt", "zapier", or `clientname`\}. Default: none.
  * If you pass in your API client's registered name for the `datasource`, and your client has a registered `autofetch_callback_url`, we will POST to your callback when this goal wants new data, as outlined in [Client OAuth](#6-optional-autofetch-callback ).
  * To unset the datasource, (i.e., return to manual entry) pass in the empty string `""`.
* \[`tags`\] (array). A list of tags for the goal. Each tag must be an alphanumeric string. NOTE: if you pass this parameter, it will replace the existing tags for the goal. If you pass an empty array, or an explicit nil value, it will remove all tags from the goal. 

### Returns

The updated [Goal](#goal) object.


<h2 id="refresh">Force a fetch of autodata and graph refresh</h2>

> Example Request

```shell
  curl https://www.beeminder.com/api/v1/users/alice/goals/weight/refresh_graph.json?auth_token=abc123
```

```json
  true
```

### HTTP Request

`GET /users/`*u*`/goals/`*g*`/refresh_graph.json`

Analagous to the refresh button on the goal page. Forces a refetch of autodata for goals with automatic data sources. Refreshes the graph image regardless.
***Please be extremely conservative with this endpoint!***

### Parameters

None.

### Returns

This is an asynchronous operation, so this endpoint simply returns **true** if the goal was queued and **false** if not.
It is up to you to watch for an updated graph image.


<h2 id="dialroad">[deprecated] Update a yellow brick road aka bright red line</h2>


```shell
  // Example request

  curl -X POST https://www.beeminder.com/api/v1/users/alice/goals/weight/dial_road.json \
    -d auth_token=abc124 \
    -d rate=-0.5 \
    -d goalval=166 \
    -d goaldate=null
```

```json
  // Example result

  { "slug": "weight",                       
    "title": "Weight Loss",                 
    "goal_type": "fatloser",                    
    "svg_url": "http://static.beeminder.com/alice+weight.svg",
    "graph_url": "http://static.beeminder.com/alice+weight.png",
    "thumb_url": "http://static.beeminder.com/alice+weight-thumb.png",
    "goaldate": null,                 
    "goalval": 166,                         
    "rate": -0.5,                           
    "losedate": 1358524800 }
```

### HTTP Request

`POST /users/`*u*`/goals/`*g*`/dial_road.json`

<aside class="notice">
Note: the dial_road endpoint is deprecated in favor of
<a href="#putgoal" title="on the goal update endpoint">roadall</a> which, despite its highly confusing state, is the future.
</aside>

Change the slope of the yellow brick road aka bright red line (starting after the one-week
[Akrasia Horizon](http://blog.beeminder.com/dial ))
for beeminder.com/*u*/*g*.

### Parameters

* `rate` (number or null)
* `goaldate` (number or null)
* `goalval` (number or null)

Exactly two of `goaldate`, `goalval`, and `rate` should be specified &mdash; setting two implies the third.

### Returns

The updated [Goal](#goal) object.


<h2 id="shortcircuit">Short circuit a goal's pledge</h2>

### HTTP Request

`POST /users/`*u*`/goals/`*g*`/shortcircuit.json`

Increase the goal's pledge level and **charge the user the amount of the current pledge**.

### Parameters

None

### Returns

The updated [Goal](#goal) object.


<h2 id="stepdown">Step down a goal's pledge</h2>

### HTTP Request

`POST /users/`*u*`/goals/`*g*`/stepdown.json`

Decrease the goal's pledge level **subject to the akrasia horizon**, i.e., not immediately.
After a successful request the goal will have a countdown to when it will revert to the lower pledge level.

### Parameters

None

### Returns

The updated [Goal](#goal) object.


<h2 id="cancelstepdown">Cancel a scheduled step down</h2>

### HTTP Request

`POST /users/`*u*`/goals/`*g*`/cancel_stepdown.json`

Cancel a pending stepdown of a goal's pledge.
The pledge will remain at the current amount.

### Parameters

None

### Returns

The updated [Goal](#goal) object.


<h2 id="unclebutton">Call "Uncle" (i.e. instant derail)</h2>

> Example
```shell
  curl https://www.beeminder.com/api/v1/users/alice/goals/blah/uncleme.json?auth_token=abc123
```

```json
  // Example success:
  // updated goal object
  { "slug": "blah",                       
    "goal_type": "hustler",                    
    "svg_url": "http://static.beeminder.com/alice+blah.svg",
    "graph_url": "http://static.beeminder.com/alice+blah.png",
    "thumb_url": "http://static.beeminder.com/alice+blah-thumb.png",
    "goaldate": null,                 
    "goalval": 166,                         
    "rate": 0.5,                           
    ...
    "losedate": 1358524800 }

  // Example error:
  {"errors": "Can't uncle a goal that's not in the red."}
```

### HTTP Request

`POST /users/`*u*`/goals/`*g*`/uncleme.json`

Call "Uncle" on a goal that is imminently going to derail (aka is in a beemergency, or "is red").
Sometimes there's just no way you're going to complete a goal, despite it being in the red, and you'd rather just derail it now, pay the pledge, and get your post-derail-respite.
That's what this endpoint is for.

Posting to this endpoint insta-derails the goal (stopping all alerts), charges you the pledge amount, and inserts your post-derail respite into the graph. 

This endpoint will fail if the goal has more than 0 days of buffer. 

This endpoint will charge you -- and all Groupies of the goal -- immediately for the derail.

### Parameters

None

### Returns

The updated [Goal](#goal) object, or an error if the goal is not red.

[Back to top](#)



<h1 id="datapoint">Datapoint Resource</h1>

A Datapoint consists of a timestamp and a value, an optional comment, and meta information.
A Datapoint belongs to a [Goal](#goal), which has many Datapoints.

### Attributes

* `id` (string): A unique ID, used to identify a datapoint when deleting or editing it.
* `timestamp` (number): The [unix time](http://en.wikipedia.org/wiki/Unix_time ) (in seconds) of the datapoint.
* `daystamp` (string): The date of the datapoint (e.g., "20150831"). Sometimes timestamps are surprising due to goal deadlines, so if you're looking at Beeminder data, you're probably interested in the daystamp.
* `value` (number): The value, e.g., how much you weighed on the day indicated by the timestamp.
* `comment` (string): An optional comment about the datapoint.
* `updated_at` (number): The unix time that this datapoint was entered or last updated.
* `requestid` (string): If a datapoint was created via the API and this parameter was included, it will be echoed back.
* `origin` (string): A short code related to where the datapoint came from. E.g. if it was added from the website, it would be "web"; if it was added by an autodata integration, e.g. Duolingo, it would be "duolingo".
* `creator` (string): Similar to origin, but for users. Especially in context of group goals, should resolve to the member who added the data, assuming the member is still around etc. When there isn't a logical `creator` this will be null.
* `is_dummy` (boolean): Not a logical datapoint, e.g. a "#DERAIL" datapoint, or Pessimistic Presumptive datapoint, added by Beeminder.
* `is_initial` (boolean): The initial datapoint added at goal creation time. Depending on the goal type this can be semantically slightly different from a "dummy" datapoint, e.g. in the case of an Odometer goal, it's a meaningful datapoint because it sets your starting count, which is "actual" data, and meaningful to the goal, but in the case of a Do More goal, it's more of a placeholder. 
* `created_at` (time): This is the timestamp at which the datapoint was created, which may differ from the datapoint's `timestamp` because of Reasons.



<h2 id="dataall">Get all the datapoints</h2>

> Examples

```shell
  curl https://www.beeminder.com/api/v1/users/alice/goals/weight/datapoints.json?auth_token=abc123
```

```json
  [{"id":"1", "timestamp":1234567890, "daystamp":"20090213", "value":7, "comment":"", "updated_at":123, "requestid":"a"},
   {"id":"2", "timestamp":1234567891, "daystamp":"20090214", "value":8, "comment":"", "updated_at":123, "requestid":"b"}]
```

### HTTP Request

`GET /users/`*u*`/goals/`*g*`/datapoints.json`

Get the list of datapoints for user *u*'s goal *g* &mdash; beeminder.com/*u*/*g*.

### Parameters

* \[`sort`\] (string): Which attribute to sort on, descending. Defaults to `id` if none given.
* \[`count`\] (integer): Limit results to count number of datapoints. Must be non-negative. Defaults to all datapoints if parameter is missing. Ignored when `page` is specified.
* \[`page`\] (integer): Used to paginate results, 1-indexed, meaning page 1 is the first page.
* \[`per`\] (integer): Number of results per page. Default 25. Ignored without `page` parameter. Must be non-negative.

### Returns

The list of [Datapoint](#datapoint) objects.


<h2 id="postdata">Create a datapoint</h2>

> Examples

```shell
  curl -X POST https://www.beeminder.com/api/v1/users/alice/goals/weight/datapoints.json \
    -d auth_token=abc123 \
    -d timestamp=1325523600 \
    -d value=130.1 \
    -d comment=sweat+a+lot+today
```

```json
  { "timestamp": 1325523600,
    "daystamp": "20120102",
    "value": 130.1,         
    "comment": "sweat a lot today",   
    "id": "4f9dd9fd86f22478d3000008",
    "requestid":"abcd182475925" }
```

### HTTP Request

`POST /users/`*u*`/goals/`*g*`/datapoints.json`

Add a new datapoint to user *u*'s goal *g* &mdash; beeminder.com/*u*/*g*.

### Parameters

* `value` (number)  
* \[`timestamp`\] (number). Defaults to "now" if none is passed in, or the existing timestamp if the datapoint is being updated rather than created (see `requestid` below).
* \[`daystamp`\] (string). Optionally you can include daystamp instead of the timestamp. If both are included, timestamp takes precedence.
* \[`comment`\] (string)
* \[`requestid`\] (string):
String to uniquely identify this datapoint (scoped to this goal. The same `requestid` can be used for different goals without being considered a duplicate).
Clients can use this to verify that Beeminder received a datapoint (important for clients with spotty connectivity).
Using requestids also means clients can safely resend datapoints without accidentally creating duplicates.
If `requestid` is included and the datapoint is identical to the existing datapoint with that requestid then the datapoint will be ignored (the API will return "duplicate datapoint").
If `requestid` is included and the datapoint differs from the existing one with the same requestid then the datapoint will be updated.
If no datapoint with the requestid exists then the datapoint is simply created.
In other words, this is an upsert endpoint and requestid is an idempotency key.

### Returns

The updated [Datapoint](#datapoint) object.


<h2 id="postdatas">Create multiple datapoints</h2>

> Examples

```shell
  curl -X POST https://www.beeminder.com/api/v1/users/alice/goals/weight/datapoints/create_all.json \
    -d auth_token=abc123 \
    -d datapoints=[{"timestamp":1343577600,"value":220.6,"comment":"blah+blah", "requestid":"abcd182475929"}, {"timestamp":1343491200,"value":220.7, "requestid":"abcd182475930"}]
```

```json
  [ { "id": "5016fa9adad11576ad00000f",
      "timestamp": 1343577600,
      "daystamp": "20120729",
      "value": 220.6,
      "comment": "blah blah",
      "updated_at": 1343577600,
      "requestid":"abcd182475923"},
    { "id": "5016fa9bdad11576ad000010",
      "timestamp": 1343491200,
      "daystamp": "20120728",
      "value": 220.7,
      "comment": "",
      "updated_at": 1343491200,
      "requestid":"abcd182475923" } ]
```

### HTTP Request

`POST /users/`*u*`/goals/`*g*`/datapoints/create_all.json`

Create multiple new datapoints for beeminder.com/*u*/*g*.

### Parameters

* `datapoints` (array of Datapoints):
Each Datapoint should be a JSON object, and must include at minimum a `value`.
Other parameters are the same as for the single-create method above.  

### Returns

A list of successfully created [Datapoints](#datapoint). 
Or, in the case of any errors, you will receive an object with two lists, `successes`, and `errors`.


<h2 id="putdata">Update a datapoint</h2>

> Examples

```shell
  curl -X PUT https://www.beeminder.com/api/v1/users/alice/goals/weight/datapoints/5016fa9adad11576ad00000f.json \
    -d auth_token=abc123 \
    -d comment=a+real+comment
```

```json
  { "id": "5016fa9adad11576ad00000f",
    "value": 220.6,
    "comment": "a real comment",
    "timestamp": 1343577600,
    "daystamp": "20120729",
    "updated_at": 1343577609 }
```

### HTTP Request

`PUT /users/`*u*`/goals/`*g*`/datapoints/`*id*`.json`

Update the datapoint with ID *id* for user *u*'s goal *g* (beeminder.com/*u*/*g*).

### Parameters

* \[`timestamp`\] (number)
* \[`value`\] (number)
* \[`comment`\] (string)

### Returns

The updated [Datapoint](#datapoint) object.


<h2 id="deletedata">Delete a datapoint</h2>

> Examples

```shell
  curl -X DELETE https://www.beeminder.com/api/v1/users/alice/goals/weight/datapoints/5016fa9adad11576ad00000f.json?auth_token=abc123
```

```json
  { "id": "5016fa9adad11576ad00000f",
    "value": 220.6,
    "comment": "a real comment",
    "timestamp": 1343577600,
    "daystamp": "20120729",
    "updated_at": 1343577609 }
```

### HTTP Request

`DELETE /users/`*u*`/goals/`*g*`/datapoints/`*id*`.json`

Delete the datapoint with ID *id* for user *u*'s goal *g* (beeminder.com/*u*/*g*).

### Parameters

None.

### Returns

The deleted [Datapoint](#datapoint) object.



[Back to top](#)



<h1 id="charge">Charge Resource</h1>

Beeminder provides an endpoint to charge an arbitrary amount to a Beeminder user. The user is inferred from the `access_token` or `auth_token` provided.
A `Charge` object has the following attributes:


### Attributes

* `amount` (number): The amount to charge the user, in US dollars.  Must be positive, >=1.00
* `note` (string): An explanation of why the charge was made.
* `username` (string): The Beeminder username of the user being charged.

<h2 id="postcharge">Create a charge</h2>

> Example request


```shell
  curl -X POST 'https://www.beeminder.com/api/v1/charges.json' \
    -d auth_token=abc123 \
    -d user_id=alice \
    -d amount=10 \
    -d note=I%27m+not+worthy%3B+charge+myself+%2410 \
```

> Example response

```json
  { "id": "5016fa9adad11576ad00000f",
    "amount": 10,
    "note": "I'm not worthy; charge myself $10",
    "username": "alice" }
```

### HTTP request

`POST /charges`

Create a charge of a given amount and optionally add a note.

### Parameters

* `user_id` (string): Username of the user who is getting charged.
* `amount` (number): The amount to charge the user, in US dollars.  Minimum value is 1.00
* `note` (string)
* \[`dryrun`\] (string): If passed, the Charge is not actually created, but the JSON for it is returned as if it were. Default: false.

### Returns

The Charge object, or an object with the error message(s) if the request was not successful.


<h2 id="goalcharge">Charge a goal</h2>

You're probably thinking of calling Uncle on a goal that's about to derail to get it over with.
See the [Uncle](#unclebutton) endpoint.


[Back to top](#)



<h1 id="webhooks">Webhooks</h1>

> Example of POSTed data

```json
  { "goal":
    { "id": "5016fa9adad11576ad00000f",
    "slug": "example", ... }  
  }
```

You can configure Beeminder to remind you about goals that are about to derail via webhook, either on the individual goal settings page or on your reminder settings page.

Beeminder will remind you via POST request to the URL you specify with a JSON body with all the attributes specified in the description of the
[Goal Resource](#goal).

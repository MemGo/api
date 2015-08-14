package main

import (
  "fmt"
  "net/http"
  "os"
  "github.com/memgo/api/meetup"
  "encoding/json"
  "github.com/pmylund/go-cache"
  "github.com/gorilla/mux"
  "time"
  "log"
  "strings"
)

var c *cache.Cache

func main() {
	port := "8080"
	ip := "0.0.0.0"

	if len(os.Getenv("PORT")) > 0 {
		port = os.Getenv("PORT")
	}
	if len(os.Getenv("IP")) > 0 {
		ip = os.Getenv("IP")
	}
  c = cache.New(5*time.Minute, 30*time.Second)

  r := mux.NewRouter()
  r.Handle("/", http.RedirectHandler("/calendar.json?keyword=memphis+ruby", 302))
  r.Handle("/favicon.ico", http.NotFoundHandler())
  r.HandleFunc("/calendar.json", calendarJson)
  r.HandleFunc("/{meetup}", meetupRedir)
  http.Handle("/", r)

	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%s", ip, port), nil))
}

func meetupRedir(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    keyword := string(vars["meetup"])

    events := filterEventsByKeyword(keyword)

    url := "http://www.meetup.com/memphis-technology-user-groups/"
    if len(events) > 0 {
      url = events[0].EventUrl
    }

    http.Redirect(w, r, url, 302)
}

func calendarJson(w http.ResponseWriter, r *http.Request) {
  keyword := string(r.FormValue("keyword"))

  events := filterEventsByKeyword(keyword)

  type CalendarJsonResponse struct {
    PR string `json:_pull_requests_appreciated`
    Meetups []meetup.Event
  }

  data := CalendarJsonResponse{"https://github.com/memgo/api",events}
  marsh, _ := json.Marshal(data)
  w.Write(marsh)
}

func filterEventsByKeyword(keyword string) (results []meetup.Event) {
  keyword = strings.ToLower(keyword)
  r, found := c.Get(keyword)
  if !found {
    log.Println("Cache miss for [", keyword, "]")
    latest, err := getLatestEvents()
    if err != nil {
      log.Println(err)
      return results
    }

    if err != nil {
      log.Println(err)
    }

    for _, element := range latest.Results {
      if strings.Index(strings.ToLower(element.Name), keyword) > -1 {
        results = append(results, element)
      }
    }

    c.Set(keyword, results, cache.DefaultExpiration)
  } else {
    results = r.([]meetup.Event)
  }

  return results
}

func getLatestEvents() (meetup.Events, error) {
  group_id := os.Getenv("MEETUP_GROUP_ID")
  api_key := os.Getenv("MEETUP_API_KEY")
  url := fmt.Sprintf("https://api.meetup.com/2/events?group_id=%s&key=%s", group_id, api_key)
  resp, err := http.Get(url)

  events := new(meetup.Events)
  decoder := json.NewDecoder(resp.Body)
  decoder.Decode(events)

  if err != nil {
    return *events, err
  }

  return *events, nil
}
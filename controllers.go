package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/unrolled/render"
	"github.com/zenazn/goji/web"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/urlfetch"
	"gopkg.in/validator.v2"
	//	"google.golang.org/appengine/user"
)

func indexHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	ren := render.New(render.Options{
		Layout:     "layout",
		Extensions: []string{".html"},
	})
	expiration := time.Now()
	expiration = expiration.Add(1 * time.Hour)
	sessionIDCookie := sessionID()
	cookie := http.Cookie{Name: "sessionid", Value: sessionIDCookie, Expires: expiration}
	http.SetCookie(w, &cookie)
	ren.HTML(w, http.StatusOK, "index", map[string]interface{}{"state": sessionIDCookie})
}

func registerHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	ren := render.New(render.Options{
		Layout:     "layout",
		Extensions: []string{".html"},
	})
	stateToBe, err := r.Cookie("sessionid")
	if err != nil {
		log.Infof(ctx, "Bad request, cannot find session id value in cookie :%v", err)
		ren.HTML(w, http.StatusBadRequest, "400", nil)
		return
	}
	stateReturned := r.URL.Query().Get("state")
	if stateToBe.Value != stateReturned {
		log.Infof(ctx, "Bad request, cannot session id value and state value doesnot match")
		log.Infof(ctx, "Session ID: %v", stateToBe.Value)
		log.Infof(ctx, "State Value: %v", stateReturned)
		ren.HTML(w, http.StatusBadRequest, "400", nil)
		return
	}
	ren.HTML(w, http.StatusOK, "register", nil)
}

func requestTokenHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	ren := render.New()
	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)
	code := r.URL.Query().Get("code")
	req, err := http.NewRequest("GET", "https://slack.com/api/oauth.access?code="+code, nil)
	if err != nil {
		ren.JSON(w, http.StatusBadRequest, map[string]interface{}{"message": "cannnot make http request"})
		return
	}
	req.Header.Add("Authorization", "Basic "+basicAuth())
	resp, err := client.Do(req)
	if err != nil {
		ren.JSON(w, http.StatusInternalServerError, map[string]interface{}{"message": "failed to get response from Oauth2 request"})
		return
	}
	dec := json.NewDecoder(resp.Body)
	var jsonData AccessToken
	dec.Decode(&jsonData)
	if len(jsonData.Error) > 0 {
		ren.JSON(w, http.StatusBadRequest, map[string]interface{}{"message": "Slack API does return error:" + jsonData.Error})
		return
	}
	var rn ReviewNotify
	rn.Code = code
	rn.AccessToken = jsonData.AccessToken
	rn.WebhookURL = jsonData.IncomingWebhook.URL
	rn.Channel = jsonData.IncomingWebhook.Channel
	rn.ConfigurationURL = jsonData.IncomingWebhook.ConfigurationURL
	rn.TeamName = jsonData.TeamName
	rn.TeamID = jsonData.TeamID
	_, err = rn.Create(ctx)
	if err != nil {
		ren.JSON(w, http.StatusInternalServerError, map[string]interface{}{"message": "cannot create new entity"})
		return
	}
	ren.JSON(w, http.StatusOK, map[string]interface{}{"message": "successfully fetch webhook settings"})
}

func parseStoreURLHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	ren := render.New()
	ctx := appengine.NewContext(r)
	url := r.URL.Query().Get("url")
	appID, countryCode, countryName, err := parseURL(url)
	if err != nil {
		ren.JSON(w, http.StatusBadRequest, map[string]interface{}{"message": "url is invalid"})
		return
	}
	client := urlfetch.Client(ctx)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		ren.JSON(w, http.StatusBadRequest, map[string]interface{}{"message": "cannot get the url"})
		return
	}
	var title string
	doc, _ := goquery.NewDocumentFromResponse(resp)
	doc.Find("body").Each(func(i int, s *goquery.Selection) {
		title = s.Find("h1").Text()
	})
	ren.JSON(w, http.StatusOK, map[string]interface{}{"app_id": appID, "country_code": countryCode, "country_name": countryName, "title": title})
}

func setNotificationHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	ren := render.New()
	ctx := appengine.NewContext(r)
	code := r.URL.Query().Get("code")
	appID := r.URL.Query().Get("app_id")
	appTitle := r.URL.Query().Get("title")
	countryCode := r.URL.Query().Get("country_code")
	var rn ReviewNotify
	rn.Code = code
	err := datastore.Get(ctx, rn.key(ctx), &rn)
	if err != nil {
		ren.JSON(w, http.StatusNotFound, map[string]interface{}{"message": "invalid request"})
		return
	}
	rn.AppID = appID
	rn.CountryCode = countryCode
	rn.Title = appTitle
	rn.SetUpCompleted = true
	err = validator.Validate(rn)
	if err != nil {
		errs := err.(validator.ErrorMap)
		var errOuts []string
		for f, e := range errs {
			errOuts = append(errOuts, fmt.Sprintf("\t - %s (%v)\n", f, e))
		}
		// Again this part is extraneous and you should not need this in real
		// code.
		for _, str := range errOuts {
			log.Infof(ctx, str)
		}
		ren.JSON(w, http.StatusBadRequest, map[string]interface{}{"message": "insufficient request"})
		return
	}
	_, err = rn.Update(ctx)
	if err != nil {
		ren.JSON(w, http.StatusInternalServerError, map[string]interface{}{"message": "cannot set notification"})
		return
	}
	ren.JSON(w, http.StatusOK, map[string]interface{}{"message": "successfully set appstore review notification"})
}

func getReviewSettingsHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	ren := render.New()
	ctx := appengine.NewContext(r)
	reviews := []ReviewNotify{}
	_, err := datastore.NewQuery("ReviewNotify").Filter("SetUpCompleted =", true).Order("-UpdatedAt").GetAll(ctx, &reviews)
	if err != nil {
		log.Infof(ctx, "failed to query datastore")
		return
	}
	for i := 0; i < len(reviews); i++ {
		log.Infof(ctx, reviews[i].AppID)
		t := taskqueue.NewPOSTTask("/admin/task/getreview/"+reviews[i].Code, nil)
		if _, err := taskqueue.Add(ctx, t, ""); err != nil {
			log.Infof(ctx, "failed to post new task")
			return
		}
	}
	listDataSet := map[string]interface{}{"items": reviews}
	ren.JSON(w, http.StatusOK, listDataSet)
}

func getReviewHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	code := c.URLParams["code"]
	ctx := appengine.NewContext(r)
	log.Infof(ctx, "code: %v", code)
	var rn ReviewNotify
	rn.Code = code
	err := datastore.Get(ctx, rn.key(ctx), &rn)
	if err != nil {
		log.Infof(ctx, "cannot get AppReview setting: %v", err)
		return
	}
	client := urlfetch.Client(ctx)
	req, err := http.NewRequest("GET", "https://itunes.apple.com/WebObjects/MZStore.woa/wa/viewContentsUserReviews?pageNumber=0&sortOrdering=4&onlyLatestVersion=false&type=Purple+Software&id="+rn.AppID, nil)
	req.Header.Add("X-Apple-Store-Front", rn.CountryCode)
	req.Header.Add("User-Agent", "iTunes/12.3 (Macintosh; U; Mac OS X 10.11)")
	resp, err := client.Do(req)
	if err != nil {
		log.Infof(ctx, "fail to request appstore review feed: %v", err)
		return
	}
	defer resp.Body.Close()
	regex_str := "([0-9]{4,}$)"
	re, err := regexp.Compile(regex_str)
	if err != nil {
		log.Infof(ctx, "regex compile error: %v", err)
	}
	doc, _ := goquery.NewDocumentFromResponse(resp)
	doc.Find("Document View VBoxView View MatrixView VBoxView:nth-child(1) VBoxView VBoxView VBoxView").Each(func(_ int, s *goquery.Selection) {
		titleNode := s.Find("HBoxView>TextView>SetFontStyle>b").First()
		title := titleNode.Text()
		if title != "" {
			reviewIDURL, idExists := s.Find("HBoxView VBoxView GotoURL").First().Attr("url")
			if idExists {
				reviewID := re.FindString(reviewIDURL)
				var content string
				var versionAndDate string
				if len(reviewID) > 4 {
					num := 0
					log.Infof(ctx, title)
					log.Infof(ctx, reviewID)
					s.Find("TextView SetFontStyle").Each(func(_ int, sc *goquery.Selection) {
						num = num + 1
						if num == 4 {
							content = sc.Text()
							log.Infof(ctx, content)
						}
					})
					userProfileNode := s.Find("HBoxView TextView SetFontStyle GotoURL").First()
					versionAndDate = userProfileNode.Parent().Text()
					versionAndDate = strings.Replace(versionAndDate, "\n", "", -1)
					versionAndDate = strings.Replace(versionAndDate, " ", "", -1)
					log.Infof(ctx, "version and date: %v", versionAndDate)
					var appreview AppReview
					appreview.AppID = rn.AppID
					appreview.Code = rn.Code
					appreview.ReviewID = reviewID
					appreview.Title = title
					appreview.Content = content
					appreview.Version = versionAndDate
					_, err = appreview.Create(ctx)
					if err != nil {
						log.Infof(ctx, "cannot create review an entity", err)
						return
					}
				}

			}
		}

	})
}

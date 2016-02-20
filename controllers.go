package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/go.net/context"
	"github.com/PuerkitoBio/goquery"
	"github.com/unrolled/render"
	"github.com/zenazn/goji/web"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/urlfetch"
	"gopkg.in/validator.v2"
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

func privacyHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	ren := render.New(render.Options{
		Layout:     "layout",
		Extensions: []string{".html"},
	})
	ren.HTML(w, http.StatusOK, "privacy", nil)
}

func supportHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	ren := render.New(render.Options{
		Layout:     "layout",
		Extensions: []string{".html"},
	})
	ren.HTML(w, http.StatusOK, "support", nil)
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
	defer resp.Body.Close()
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
	ess, err := getExistingSettings(ctx, code)
	if err != nil {
		ren.JSON(w, http.StatusInternalServerError, map[string]interface{}{"message": "cannot check existing settings"})
		return
	}
	ren.JSON(w, http.StatusOK, map[string]interface{}{"message": "successfully fetch webhook settings", "existing_settings": ess})
}

func getExistingSettings(ctx context.Context, code string) ([]ExistingSetting, error) {
	// The Query type and its methods are used to construct a query.
	var rn ReviewNotify
	rn.Code = code
	err := datastore.Get(ctx, rn.key(ctx), &rn)
	if err != nil {
		return nil, err
	}
	q := datastore.NewQuery("ReviewNotify").Filter("SetUpCompleted =", true).Filter("TeamID =", rn.TeamID).Order("-UpdatedAt")

	// To retrieve the results,
	// you must execute the Query using its GetAll or Run methods.
	var rns []ReviewNotify
	if _, err := q.GetAll(ctx, &rns); err != nil {
		return nil, err
	}
	var ess []ExistingSetting
	for i := 0; i < len(rns); i++ {
		var es ExistingSetting
		es.Channel = rns[i].Channel
		es.Title = rns[i].Title
		es.AppID = rns[i].AppID
		ess = append(ess, es)
	}
	return ess, nil
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
	defer resp.Body.Close()
	var title string
	doc, _ := goquery.NewDocumentFromResponse(resp)
	doc.Find("body").Each(func(i int, s *goquery.Selection) {
		title = s.Find("h1").Text()
	})
	ren.JSON(w, http.StatusOK, map[string]interface{}{"app_id": appID, "country_code": countryCode, "country_name": countryName, "title": title})
}

func removeNotificationHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	ren := render.New()
	ctx := appengine.NewContext(r)
	code := r.URL.Query().Get("code")
	appID := r.URL.Query().Get("app_id")
	channel := r.URL.Query().Get("channel")
	var rn ReviewNotify
	rn.Code = code
	err := datastore.Get(ctx, rn.key(ctx), &rn)
	if err != nil {
		ren.JSON(w, http.StatusNotFound, map[string]interface{}{"message": "invalid request"})
		return
	}
	now := time.Now()
	expiration := rn.CreatedAt.Add(3 * time.Hour)
	if !expiration.After(now) {
		ren.JSON(w, http.StatusBadRequest, map[string]interface{}{"message": "code expired, retry auth process of Slack"})
		return
	}
	q := datastore.NewQuery("ReviewNotify").
		Filter("SetUpCompleted =", true).
		Filter("TeamID =", rn.TeamID).
		Filter("AppID =", appID).
		Filter("Channel =", channel).
		Order("-UpdatedAt")
	var rns []ReviewNotify
	if _, err := q.GetAll(ctx, &rns); err != nil {
		ren.JSON(w, http.StatusInternalServerError, map[string]interface{}{"message": "failed to query data"})
		return
	}
	if len(rns) < 1 {
		ren.JSON(w, http.StatusNotFound, map[string]interface{}{"message": "setting not found"})
		return
	}
	for i := 0; i < len(rns); i++ {
		err := datastore.Delete(ctx, rns[i].key(ctx))
		if err != nil {
			ren.JSON(w, http.StatusInternalServerError, map[string]interface{}{"message": "failed to delete data"})
			return
		}
	}
	ren.JSON(w, http.StatusOK, map[string]interface{}{"message": "successfully delete setting"})
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
	cursorParam := c.URLParams["cursor"]
	ren := render.New()
	ctx := appengine.NewContext(r)
	q := datastore.NewQuery("ReviewNotify").Filter("SetUpCompleted =", true).Order("-UpdatedAt").Limit(1000)
	if cursorParam != "" {
		cursor, err := datastore.DecodeCursor(cursorParam)
		if err == nil {
			q = q.Start(cursor)
		}
	}
	t := q.Run(ctx)
	for {
		var rn ReviewNotify
		_, err := t.Next(&rn)
		if err == datastore.Done {
			break
		}
		if err != nil {
			log.Errorf(ctx, "fetching next Review Setting: %v", err)
			break
		}
		task := taskqueue.NewPOSTTask("/admin/task/getreview/"+rn.Code, nil)
		log.Infof(ctx, "add task: %v", rn.Title)
		if _, err := taskqueue.Add(ctx, task, ""); err != nil {
			log.Errorf(ctx, "failed to post new task")
			break
		}
	}
	if cursor, err := t.Cursor(); err == nil {
		if cursor.String() == cursorParam {
			returnMessage := map[string]interface{}{"message": "no more query"}
			ren.JSON(w, http.StatusOK, returnMessage)
			return
		}
		task := taskqueue.NewPOSTTask("/admin/task/getreviews/"+cursor.String(), nil)
		if _, err := taskqueue.Add(ctx, task, ""); err != nil {
			log.Errorf(ctx, "failed to post new task")
			return
		}
		log.Infof(ctx, "post next query task: %v", cursor.String())
	}
	returnMessage := map[string]interface{}{"message": "notification check fired"}
	ren.JSON(w, http.StatusOK, returnMessage)
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
	regexStr := "([0-9]{4,}$)"
	re, err := regexp.Compile(regexStr)
	if err != nil {
		log.Infof(ctx, "regex compile error: %v", err)
	}
	regexNum := "([0-9])"
	reNum, err := regexp.Compile(regexNum)
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
					var starString string
					starCheckCount := 0
					s.Find("HBoxView HBoxView PictureView").Each(func(_ int, st *goquery.Selection) {
						if starCheckCount == 0 {
							url, _ := st.Attr("url")
							log.Infof(ctx, url)
							starString, _ = st.Parent().Attr("alt")
							log.Infof(ctx, starString)
						}
						starCheckCount = starCheckCount + 1
					})
					starCount, _ := strconv.Atoi(reNum.FindString(starString))
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
					appreview.Star = starCount
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

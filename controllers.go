package main

import (
	"encoding/json"
	"net/http"
	//	"strconv"
	"time"
	"fmt"

	"gopkg.in/validator.v2"
	"github.com/PuerkitoBio/goquery"
	"github.com/unrolled/render"
	"github.com/zenazn/goji/web"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
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

/*
func spotHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	ren := render.New()
	ctx := appengine.NewContext(r)
	spots := []SpotGet{}
	_, err := datastore.NewQuery("Spot").Order("-UpdatedAt").GetAll(ctx, &spots)
	if err != nil {
		ren.JSON(w, http.StatusInternalServerError, map[string]interface{}{"message": "error"})
		return
	}
	ren.JSON(w, http.StatusOK, map[string]interface{}{"items": spots})
}

func spotGetHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	ren := render.New()
	spotCode, err := strconv.ParseInt(c.URLParams["spotCode"], 10, 64)
	if err != nil {
		log.Infof(ctx, "failed to parse int :%v", err)
		ren.JSON(w, http.StatusBadRequest, map[string]interface{}{"message": "error, bad resource id"})
		return
	}
	var spot SpotGet
	spot.SpotCode = spotCode
	err = datastore.Get(ctx, spot.key(ctx), &spot)
	if err != nil {
		ren.JSON(w, http.StatusNotFound, map[string]interface{}{"message": "error, entity not found"})
		return
	}
	ren.JSON(w, http.StatusOK, map[string]interface{}{"item": spot})
}

func spotCreateHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	var spot Spot
	ctx := appengine.NewContext(r)
	ren := render.New()
	err := json.NewDecoder(r.Body).Decode(&spot)
	if err != nil {
		ren.JSON(w, http.StatusBadRequest, map[string]interface{}{"message": "error, cannot decode json"})
		log.Infof(ctx, "failed to parse JSON:%v", err)
		return
	}
	_, err = spot.Create(ctx)
	if err != nil {
		ren.JSON(w, http.StatusBadRequest, map[string]interface{}{"message": "error, cannot create new entity"})
		return
	}
	var spotGet SpotGet
	spotGet.SpotCode = spot.SpotCode
	err = datastore.Get(ctx, spotGet.key(ctx), &spotGet)
	ren.JSON(w, http.StatusCreated, map[string]interface{}{"message": "new entity created", "item": spotGet})
}

func spotUpdateHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	ren := render.New()
	spotCode, err := strconv.ParseInt(c.URLParams["spotCode"], 10, 64)
	if err != nil {
		ren.JSON(w, http.StatusBadRequest, map[string]interface{}{"message": "error, bad resource id"})
		return
	}
	var checkSpot Spot
	checkSpot.SpotCode = spotCode
	currentUser := user.Current(ctx)
	err = datastore.Get(ctx, checkSpot.key(ctx), &checkSpot)
	if err != nil {
		ren.JSON(w, http.StatusNotFound, map[string]interface{}{"message": "error, entity not found"})
		return
	}
	if currentUser.ID != checkSpot.Editor {
		ren.JSON(w, http.StatusForbidden, map[string]interface{}{"message": "error, you don't have right to write this entity"})
		return
	}
	var updateSpot Spot
	err = json.NewDecoder(r.Body).Decode(&updateSpot)
	if err != nil {
		ren.JSON(w, http.StatusBadRequest, map[string]interface{}{"message": "error, cannot decode json"})
		log.Infof(ctx, "failed to parse JSON:%v", err)
		return
	}
	updateSpot.Editor = checkSpot.Editor
	_, err = updateSpot.Update(ctx, spotCode)
	if err != nil {
		ren.JSON(w, http.StatusForbidden, map[string]interface{}{"message": "error, you cannot edit this spot"})
		return
	}
	err = datastore.Get(ctx, checkSpot.key(ctx), &checkSpot)
	if err != nil {
		ren.JSON(w, http.StatusNotFound, map[string]interface{}{"message": "error, entity not found"})
		return
	}
	ren.JSON(w, http.StatusOK, map[string]interface{}{"message": "entity updated", "item": checkSpot})
}
*/

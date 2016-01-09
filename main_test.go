package main

import (
	//	"bytes"
	//	"net/http/httptest"
	//	"strconv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	//	"github.com/zenazn/goji/web"

	//	"net/http"

	"github.com/zenazn/goji/web"
	"google.golang.org/appengine"
	"google.golang.org/appengine/aetest"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
)

func TestIndex(t *testing.T) {
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	defer inst.Close()

	req, err := inst.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("Failed to create req1: %v", err)
	}
	_ = appengine.NewContext(req)
	res := httptest.NewRecorder()
	c := web.C{}
	indexHandler(c, res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("Fail to request index")
	}
}

func TestPrivacy(t *testing.T) {
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	defer inst.Close()

	req, err := inst.NewRequest("GET", "/privacy", nil)
	if err != nil {
		t.Fatalf("Failed to create req1: %v", err)
	}
	_ = appengine.NewContext(req)
	res := httptest.NewRecorder()
	c := web.C{}
	privacyHandler(c, res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("Fail to request index")
	}
}

func TestRegister(t *testing.T) {
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	defer inst.Close()
	sessionIDCookie := sessionID()
	req, err := inst.NewRequest("GET", "/register?state="+sessionIDCookie, nil)
	if err != nil {
		t.Fatalf("Failed to create req: %v", err)
	}
	_ = appengine.NewContext(req)
	res := httptest.NewRecorder()
	c := web.C{}
	expiration := time.Now()
	expiration = expiration.Add(1 * time.Hour)
	cookie := http.Cookie{Name: "sessionid", Value: sessionIDCookie, Expires: expiration}
	req.AddCookie(&cookie)
	registerHandler(c, res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("Fail to request spots list")
	}
}

type ParseResult struct {
	AppID       string `json:"app_id"`
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	Title       string `json:"title"`
}

func TestParseURL(t *testing.T) {
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	defer inst.Close()
	req, err := inst.NewRequest("GET", "/parse/store/url?url=https%3A%2F%2Fitunes.apple.com%2Fus%2Fapp%2Ffacebook%2Fid284882215%3Fmt%3D8", nil)
	if err != nil {
		t.Fatalf("Failed to create req: %v", err)
	}
	_ = appengine.NewContext(req)
	res := httptest.NewRecorder()
	c := web.C{}
	parseStoreURLHandler(c, res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("Fail to request spots list")
	}
	dec := json.NewDecoder(res.Body)
	var jsonData ParseResult
	dec.Decode(&jsonData)
	if jsonData.AppID != "284882215" {
		t.Fatalf("Parse result is invalid")
	}
}

func TestSetNotification(t *testing.T) {
	opt := aetest.Options{AppID: "bell-apps", StronglyConsistentDatastore: true}
	inst, err := aetest.NewInstance(&opt)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	defer inst.Close()
	req, err := inst.NewRequest("GET", "/set/notification?code=test&app_id=284882215&title=Facebook&country_code=143441", nil)
	if err != nil {
		t.Fatalf("Failed to create req: %v", err)
	}
	ctx := appengine.NewContext(req)
	var rn ReviewNotify
	rn.Code = "test"
	rn.AccessToken = "test"
	rn.WebhookURL = "https://hoge.com/slack"
	rn.Channel = "#test"
	rn.ConfigurationURL = "https://hoge.com/slack"
	rn.TeamName = "TestTeam"
	rn.TeamID = "testest"
	_, err = rn.Create(ctx)
	res := httptest.NewRecorder()
	c := web.C{}
	setNotificationHandler(c, res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("Fail to request set notification")
	}
	var checkReviewNotify ReviewNotify
	checkReviewNotify.Code = "test"
	err = datastore.Get(ctx, checkReviewNotify.key(ctx), &checkReviewNotify)
	if err != nil {
		t.Fatalf("Fail to get data from datastore: %v", err)
	}
	if checkReviewNotify.AppID != "284882215" {
		t.Fatalf("AppID should be 284882215")
	}
}

func TestGetReviewSettings(t *testing.T) {
	opt := aetest.Options{AppID: "bell-apps", StronglyConsistentDatastore: true}
	inst, err := aetest.NewInstance(&opt)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	defer inst.Close()
	req, err := inst.NewRequest("GET", "/admin/task/getreviews", nil)
	if err != nil {
		t.Fatalf("Failed to create req: %v", err)
	}
	loginUser := user.User{Email: "hoge@gmail.com", Admin: true, ID: "111111"}
	aetest.Login(&loginUser, req)
	ctx := appengine.NewContext(req)
	var rn ReviewNotify
	rn.Code = "test"
	rn.AccessToken = "test"
	rn.WebhookURL = "https://hoge.com/slack"
	rn.Channel = "#test"
	rn.ConfigurationURL = "https://hoge.com/slack"
	rn.TeamName = "TestTeam"
	rn.TeamID = "testest"
	rn.AppID = "284882215"
	rn.Title = "Facebook"
	rn.CountryCode = "143441"
	_, err = rn.Create(ctx)
	res := httptest.NewRecorder()
	c := web.C{}
	getReviewSettingsHandler(c, res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("Fail to request review settings")
	}
}

/*
type GetResponse struct {
	Message string `json:"message"`
	Item    SpotGet   `json:"item"`
}

func TestGetSpot(t *testing.T) {
	opt := aetest.Options{AppID: "t2jp-2015", StronglyConsistentDatastore: true}
	inst, err := aetest.NewInstance(&opt)
	defer inst.Close()
	input, err := json.Marshal(Spot{SpotName: "foo", Body: "bar"})
	req, err := inst.NewRequest("POST", "/edit/v1/spots", bytes.NewBuffer(input))
	if err != nil {
		t.Fatalf("Failed to create req: %v", err)
	}
	loginUser := user.User{Email: "hoge@gmail.com", Admin: false, ID: "111111"}
	aetest.Login(&loginUser, req)
	// ctx := appengine.NewContext(req)
	res := httptest.NewRecorder()
	c := web.C{}
	spotCreateHandler(c, res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("Fail to request spots create, status code: %v", res.Code)
	}
	var getResponse GetResponse
	err = json.NewDecoder(res.Body).Decode(&getResponse)
	spotCode := getResponse.Item.SpotCode
	t.Logf("spot code: %v", strconv.FormatInt(spotCode, 10))
	getReq, err := inst.NewRequest("GET", "/edit/v1/spots/"+strconv.FormatInt(spotCode, 10), nil)
	if err != nil {
		t.Fatalf("Failed to create req: %v", err)
	}
	getRes := httptest.NewRecorder()
	getC := web.C{URLParams: map[string]string{"spotCode": strconv.FormatInt(spotCode, 10)}}
	spotGetHandler(getC, getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("Fail to request spot get, status code: %v", getRes.Code)
	}
}

func TestUpdateSpot(t *testing.T) {
	opt := aetest.Options{AppID: "t2jp-2015", StronglyConsistentDatastore: true}
	inst, err := aetest.NewInstance(&opt)
	defer inst.Close()
	input, err := json.Marshal(Spot{SpotName: "foo", Body: "bar"})
	req, err := inst.NewRequest("POST", "/edit/v1/spots", bytes.NewBuffer(input))
	if err != nil {
		t.Fatalf("Failed to create req: %v", err)
	}
	loginUser := user.User{Email: "hoge@gmail.com", Admin: false, ID: "111111"}
	aetest.Login(&loginUser, req)
	// ctx := appengine.NewContext(req)
	res := httptest.NewRecorder()
	c := web.C{}
	spotCreateHandler(c, res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("Fail to request spots create, status code: %v", res.Code)
	}
	var getResponse GetResponse
	err = json.NewDecoder(res.Body).Decode(&getResponse)
	if getResponse.Item.Status != "draft" {
		t.Fatalf("not saved as draft on creation!")
	}
	spotCodeString := strconv.FormatInt(getResponse.Item.SpotCode, 10)
	t.Logf("spot code: %v", spotCodeString)
	patchInput, err := json.Marshal(Spot{SpotName: "foo2", Body: "barbar"})
	patchReq, err := inst.NewRequest("PATCH", "/edit/v1/spots/"+spotCodeString, bytes.NewBuffer(patchInput))
	aetest.Login(&loginUser, patchReq)
	if err != nil {
		t.Fatalf("Failed to create req: %v", err)
	}
	patchRes := httptest.NewRecorder()
	patchC := web.C{URLParams: map[string]string{"spotCode": spotCodeString}}
	spotUpdateHandler(patchC, patchRes, patchReq)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("Fail to request spot patch, status code: %v", patchRes.Code)
	}
	var checkSpot Spot
	ctx := appengine.NewContext(patchReq)
	checkSpot.SpotCode = getResponse.Item.SpotCode
	err = datastore.Get(ctx, checkSpot.key(ctx), &checkSpot)
	if err != nil {
		t.Fatalf("Fail to get data from datastore: %v", err)
	}
	if checkSpot.RevisionNumber != 1 {
		t.Fatalf("RevisionNumber should be 1")
	}
}
*/

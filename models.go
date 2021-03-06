package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/delay"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

// IncomingWebhook is a child element of oauth2 response
type IncomingWebhook struct {
	URL              string `json:"url"`
	Channel          string `json:"channel"`
	ConfigurationURL string `json:"configuration_url"`
}

// AccessToken is a struct for oauth2 response
type AccessToken struct {
	AccessToken     string          `json:"access_token"`
	Scope           string          `json:"scope"`
	TeamName        string          `json:"team_name"`
	TeamID          string          `json:"team_id"`
	IncomingWebhook IncomingWebhook `json:"incoming_webhook"`
	OK              bool            `json:"ok"`
	Error           string          `json:"error"`
}

// ExistingSetting is a struct for existing setting
type ExistingSetting struct {
	Channel string `json:"channel" validate:"regexp=^#.+$"`
	Title   string `json:"title"`
	AppID   string `json:"app_id" validate:"min=5,max=20,regexp=^[0-9]+$"`
}

// ReviewNotify is a Struct for AppStore review notification setting, definition of datastore kind
type ReviewNotify struct {
	Code             string    `json:"code"`
	AppID            string    `json:"app_id" validate:"min=5,max=20,regexp=^[0-9]+$"`
	CountryCode      string    `json:"country_code" validate:"min=5,max=6,regexp=^[0-9]+$"`
	Title            string    `json:"title"`
	AccessToken      string    `json:"access_token"`
	WebhookURL       string    `json:"webhook_url" validate:"regexp=^https.+$"`
	TeamName         string    `json:"team_name"`
	TeamID           string    `json:"team_id"`
	Channel          string    `json:"channel" validate:"regexp=^#.+$"`
	ConfigurationURL string    `json:"configuration_url" validate:"regexp=^https.+$"`
	SetUpCompleted   bool      `json:"-"`
	UpdatedAt        time.Time `json:"updated_at"`
	CreatedAt        time.Time `json:"created_at"`
}

func (rn *ReviewNotify) key(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, "ReviewNotify", rn.Code, 0, nil)
}

//Create new ReviewNotify Entity
func (rn *ReviewNotify) Create(ctx context.Context) (*ReviewNotify, error) {
	rn.SetUpCompleted = false
	rn.CreatedAt = time.Now()
	rn.UpdatedAt = time.Now()
	_, err := datastore.Put(ctx, rn.key(ctx), rn)
	if err != nil {
		return nil, err
	}
	return rn, nil
}

//Update existing ReviewNotify Entity
func (rn *ReviewNotify) Update(ctx context.Context) (*ReviewNotify, error) {
	rn.UpdatedAt = time.Now()
	_, err := datastore.Put(ctx, rn.key(ctx), rn)
	if err != nil {
		return nil, err
	}
	return rn, nil
}

// AppStoreID is a struct for AppStoreID setting
type AppStoreID struct {
	CountryDomain string
	CountryName   string
	CountryCode   string
}

// AppReview is a kind which stores reviews of a app, a entity == a review
type AppReview struct {
	KeyName   string    `json:"key_name" datastore:"KeyName"`
	AppID     string    `json:"app_id" datastore:"AppID"`
	Code      string    `json:"code" datastore:"Code"`
	ReviewID  string    `json:"review_id" datastore:"ReviewID"`
	Star      int       `json:"star" datastore:"Star"`
	Title     string    `json:"title" datastore:"Title,noindex"`
	Content   string    `json:"content" datastore:"Content,noindex"`
	Version   string    `json:"version" datastore:"Version,noindex"`
	CreatedAt time.Time `json:"created_at" datastore:"CreatedAt"`
}

func (ar *AppReview) key(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, "AppReview", ar.KeyName, 0, nil)
}

// NotifyReviewToSlack is a function to send notification of new reviews to Slack channel
func NotifyReviewToSlack(ctx context.Context, ar *AppReview) {
	var rn ReviewNotify
	key := datastore.NewKey(ctx, "ReviewNotify", ar.Code, 0, nil)
	err := datastore.Get(ctx, key, &rn)
	if err != nil {
		log.Infof(ctx, "%v", err)
		return
	}
	client := urlfetch.Client(ctx)
	iconURL := "https://bell-apps.appspot.com/static/icon57.png"
	text := "[" + rn.Title + "]\n" + ar.Title + ":\n" + ar.Content + "\n" + ar.Version
	var fields []map[string]interface{}
	starEmoji := ""
	for i := 0; i < ar.Star; i++ {
		starEmoji = starEmoji + ":star:"
	}
	fields = append(fields, map[string]interface{}{
		"title": "Star:",
		"value": starEmoji,
		"short": false,
	})
	fields = append(fields, map[string]interface{}{
		"title": "Meta:",
		"value": ar.Version,
		"short": false,
	})
	var attachments []map[string]interface{}
	attachments = append(attachments, map[string]interface{}{
		"fallback": text,
		"pretext":  rn.Title,
		"color":    "#8EFCD3",
		"title":    ar.Title,
		"text":     ar.Content + "\n",
		"fields":   fields,
	})
	payload := map[string]interface{}{"attachments": attachments, "username": "Bell Apps App Review Notification", "icon_url": iconURL, "mrkdwn": false}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Infof(ctx, "%v", err)
		return
	}
	b := bytes.NewBuffer(payloadJSON)
	req, _ := http.NewRequest("POST", rn.WebhookURL, b)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(req)
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		rn.SetUpCompleted = false
		_, err = rn.Update(ctx)
	}
}

var notifyToSlackAsync = delay.Func("put", NotifyReviewToSlack)

// Create AppReview Entity if same review does not exist
func (ar *AppReview) Create(ctx context.Context) (*AppReview, error) {
	var appreview AppReview
	ar.KeyName = ar.Code + "_" + ar.AppID + "_" + ar.ReviewID
	err := datastore.Get(ctx, ar.key(ctx), &appreview)
	if err == nil {
		log.Infof(ctx, "already registered")
		return &appreview, nil
	}
	ar.CreatedAt = time.Now()
	_, err = datastore.Put(ctx, ar.key(ctx), ar)
	if err != nil {
		return nil, err
	}
	notifyToSlackAsync.Call(ctx, ar)
	return ar, nil
}

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
	Star      string    `json:"star" datastore:"Star"`
	Title     string    `json:"title" datastore:"Tite,noindex"`
	Content   string    `json:"content" datastore:"Content,noindex"`
	Version   string    `json:"version" datastore:"Version,noindex"`
	CreatedAt time.Time `json:"created_at" datastore:"CreatedAt"`
}

func (ar *AppReview) key(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, "AppReview", ar.KeyName, 0, nil)
}

func NotifyReviewToSlack(ctx context.Context, ar *AppReview) {
	var rn ReviewNotify
	key := datastore.NewKey(ctx, "ReviewNotify", ar.Code, 0, nil)
	err := datastore.Get(ctx, key, &rn)
	if err != nil {
		log.Infof(ctx, "%v", err)
		return
	}
	client := urlfetch.Client(ctx)
	iconURL := "https://bell-apps.appspot.com/statici/icon57.png"
	text := "[" + rn.Title + "]\n" + ar.Title + ":\n" + ar.Content + "\n" + ar.Version
	payload := map[string]string{"text": text, "username": "app review", "icon_url": iconURL}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Infof(ctx, "%v", err)
		return
	}
	b := bytes.NewBuffer(payloadJSON)
	req, err := http.NewRequest("POST", rn.WebhookURL, b)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	defer resp.Body.Close()
}

var notifyToSlackAsync = delay.Func("put", NotifyReviewToSlack)

// Save puts to datastore
func (ar *AppReview) Create(ctx context.Context) (*AppReview, error) {
	var appreview AppReview
	ar.KeyName = ar.AppID + "_" + ar.ReviewID
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

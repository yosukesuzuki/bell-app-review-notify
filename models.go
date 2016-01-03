package main

import (
	"time"

	"golang.org/x/net/context"
	//	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	//	"google.golang.org/appengine/user"
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

//Create new Spot Entity
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

//Update existing Spot Entity
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

/*
type AppStoreSettings struct {
	Stores []AppStoreID
}
*/

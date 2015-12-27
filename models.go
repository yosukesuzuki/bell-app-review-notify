package main

import (
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
)

// T2JPUser is the kind which stores data of users who comment and have favourite list
type T2JPUser struct {
	KeyName    string    `json:"key_name" datastore:"KeyName"`
	Name       string    `json:"name" datastore:"Name"`
	Email      string    `json:"email" datastore:"Email"`
	IconURL    string    `json:"icon_url" datastore:"IconURL"`
	Favourites []string  `json:"favourites" datastore:"Favourites"` // list of SpotCode
	IsEditor   bool      `json:"is_editor" datastore:"IsEditor"`
	IsAdmin    bool      `json:"is_admin" datastore:"IsAdmin"`
	UpdatedAt  time.Time `json:"updated_at" datastore:"UpdatedAt"`
	CreatedAt  time.Time `json:"created_at" datastore:"CreatedAt"`
}

func (u *T2JPUser) key(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, "T2JPUser", u.KeyName, 0, nil)
}

//Create new T2JPUser entity
func (u *T2JPUser) Create(ctx context.Context) (*T2JPUser, error) {
	currentUser := user.Current(ctx)
	u.KeyName = currentUser.ID
	u.Email = currentUser.Email
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	_, err := datastore.Put(ctx, u.key(ctx), u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

//Update existing T2JPUser entity
func (u *T2JPUser) Update(ctx context.Context) (*T2JPUser, error) {
	u.UpdatedAt = time.Now()
	_, err := datastore.Put(ctx, u.key(ctx), u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Spot is the kind which stores sightseeing spot information
// Set SpotCode as KeyName
type Spot struct {
	SpotCode       int64              `json:"-" datastore:"SpotCode"`
	RevisionNumber int                `json:"-" datastore:"Revision"` // increment on update
	Status         string             `json:"status" datastore:"Status"`     // 'post' or 'revision' or 'draft'
	SpotName       string             `json:"spot_name" datastore:"SpotName"`
	Body           string             `json:"body" datastore:"Body,noindex"`
	Photos         []string           `json:"photos" datastore:"Photos"`
	GeoInfo        appengine.GeoPoint `json:"geo_info" datastore:"GeoInfo"`
	Phone          string             `json:"phone" datastore:"Phone"`
	Editor         string             `json:"-" datastore:"Editor"`
	UpdatedAt      time.Time          `json:"-" datastore:"UpdatedAt"`
	CreatedAt      time.Time          `json:"-" datastore:"CreatedAt"`
}

// Spot Struct for Listing
type SpotGet struct {
	SpotCode       int64              `json:"spot_code" datastore:"SpotCode"`
	RevisionNumber int                `json:"revision_number" datastore:"Revision"` // increment on update
	Status         string             `json:"status" datastore:"Status"`     // 'post' or 'revision' or 'draft'
	SpotName       string             `json:"spot_name" datastore:"SpotName"`
	Body           string             `json:"body" datastore:"Body,noindex"`
	Photos         []string           `json:"photos" datastore:"Photos"`
	GeoInfo        appengine.GeoPoint `json:"geo_info" datastore:"GeoInfo"`
	Phone          string             `json:"phone" datastore:"Phone"`
	Editor         string             `json:"editor" datastore:"Editor"`
	UpdatedAt      time.Time          `json:"updated_at" datastore:"UpdatedAt"`
	CreatedAt      time.Time          `json:"created_at" datastore:"CreatedAt"`
}

func (s *Spot) key(ctx context.Context) *datastore.Key {
	if s.SpotCode == 0 {
		low, _, err := datastore.AllocateIDs(ctx, "Spot", nil, 1)
		if err != nil {
			return nil
		}
		return datastore.NewKey(ctx, "Spot", "", low, nil)
	}
	return datastore.NewKey(ctx, "Spot", "", s.SpotCode, nil)
}

func (s *SpotGet) key(ctx context.Context) *datastore.Key {
	if s.SpotCode == 0 {
		low, _, err := datastore.AllocateIDs(ctx, "Spot", nil, 1)
		if err != nil {
			return nil
		}
		return datastore.NewKey(ctx, "Spot", "", low, nil)
	}
	return datastore.NewKey(ctx, "Spot", "", s.SpotCode, nil)
}

//Create new Spot Entity
func (s *Spot) Create(ctx context.Context) (*Spot, error) {
	currentUser := user.Current(ctx)
	s.Editor = currentUser.ID
	s.Status = "draft"
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	newKey := s.key(ctx)
	s.SpotCode = newKey.IntID()
	_, err := datastore.Put(ctx, newKey, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

//Update existing Spot Entity
func (s *Spot) Update(ctx context.Context, spotCode int64) (*Spot, error) {
	s.SpotCode = spotCode
	s.RevisionNumber = s.RevisionNumber + 1
	s.UpdatedAt = time.Now()
	_, err := datastore.Put(ctx, s.key(ctx), s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

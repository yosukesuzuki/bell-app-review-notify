package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/zenazn/goji/web"

	"net/http"

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

	_, err = inst.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("Failed to create req1: %v", err)
	}
}

func TestSpot(t *testing.T) {
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	defer inst.Close()
	req, err := inst.NewRequest("GET", "/edit/v1/spots", nil)
	if err != nil {
		t.Fatalf("Failed to create req: %v", err)
	}
	loginUser := user.User{Email: "hoge@gmail.com", Admin: false, ID: "111111"}
	aetest.Login(&loginUser, req)
	_ = appengine.NewContext(req)
	res := httptest.NewRecorder()
	c := web.C{}
	spotHandler(c, res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("Fail to request spots list")
	}
}

func TestCreateSpot(t *testing.T) {
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
	ctx := appengine.NewContext(req)
	res := httptest.NewRecorder()
	c := web.C{}
	spotCreateHandler(c, res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("Fail to request spots create, status code: %v", res.Code)
	}
	spots := []Spot{}
	_, err = datastore.NewQuery("Spot").Order("-UpdatedAt").GetAll(ctx, &spots)
	for i := 0; i < len(spots); i++ {
		t.Logf("SpotCode:%v", spots[i].SpotCode)
		t.Logf("SpotName:%v", spots[i].SpotName)
	}
	if spots[0].SpotName != "foo" {
		t.Fatalf("not expected value! :%v", spots[0].SpotName)
	}

}

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

package main

import (
	"net/http"

	"github.com/zenazn/goji"
)

func init() {
	http.Handle("/", goji.DefaultMux)

	goji.Get("/", indexHandler)
	goji.Get("/register", registerHandler)
	goji.Get("/request/token", requestTokenHandler)
	goji.Get("/parse/store/url", parseStoreURLHandler)
	goji.Get("/set/notification", setNotificationHandler)
	goji.Get("/admin/task/getreviews", getReviewSettingsHandler)
	goji.Post("/admin/task/getreview/:code", getReviewHandler)
	/*
		goji.Get("/api/v1/spots", spotHandler)
		goji.Get("/api/v1/spots/:spotCode", spotGetHandler)
		goji.Get("/edit/", indexHandler)
		goji.Get("/edit/v1/spots", spotHandler)
		goji.Get("/edit/v1/spots/:spotCode", spotGetHandler)
		goji.Post("/edit/v1/spots", spotCreateHandler)
		goji.Patch("/edit/v1/spots/:spotCode", spotUpdateHandler)
	*/
	//	goji.Delete("/edit/v1/spots/:spotCode", spotDeleteHandler)
}

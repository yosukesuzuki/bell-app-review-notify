package main

import (
	"net/http"

	"github.com/zenazn/goji"
)

func init() {
	http.Handle("/", goji.DefaultMux)

	goji.Get("/", indexHandler)
	goji.Get("/privacy", privacyHandler)
	goji.Get("/register", registerHandler)
	goji.Get("/request/token", requestTokenHandler)
	goji.Get("/parse/store/url", parseStoreURLHandler)
	goji.Get("/set/notification", setNotificationHandler)
	goji.Get("/admin/task/getreviews", getReviewSettingsHandler)
	goji.Post("/admin/task/getreviews/:cursor", getReviewSettingsHandler)
	goji.Post("/admin/task/getreview/:code", getReviewHandler)
}

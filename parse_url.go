package main

import (
	"regexp"
)


func parseURL(url string) (string, string) {
	regexURL := regexp.MustCompile(`https://itunes.apple.com/([a-zA-Z]{2})/app/id(\d+)`)
	result := regexURL.FindStringSubmatch(url)
	appStoreSetting := AppStoreSetting(result[1])
	return result[2], appStoreSetting.CountryCode
}

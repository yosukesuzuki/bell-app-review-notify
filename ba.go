package main

import (
	"encoding/base64"
)

func basicAuth() string {
	// 文字列をbyte配列にしてbase64にする
	data := clientId + ":" + clientSecret
	sEnc := base64.StdEncoding.EncodeToString([]byte(data))
	return sEnc
}

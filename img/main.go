package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	spinhttp "github.com/fermyon/spin/sdk/go/v2/http"
)

func init() {
	spinhttp.Handle(func(w http.ResponseWriter, r *http.Request) {

		replaceDict := map[string]string{
			"https://github.com": "https://ghmirror.fermyon.app",
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		url := "https://github.com" + r.URL.Path

		req, err := http.NewRequest(r.Method, url, bytes.NewBuffer([]byte{}))
		if err != nil {
			fmt.Println("Request error:", err)
			return
		}

		// copy body
		if r.Method == "POST" {
			req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
			req.Body = r.Body
		}

		req.Header.Set("Referer", "https://github.com")

		client := spinhttp.NewClient()
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Request error:", err)
			return
		}
		defer resp.Body.Close()

		// Websocket
		if isUpgrade(req.Header) {
			fmt.Fprintln(w, resp)
			return
		}

		for k, v := range resp.Header {
			w.Header()[k] = v
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Del("Content-Security-Policy")
		w.Header().Del("Content-Disposition")

		//获取Content-Type
		ctype := resp.Header.Get("Content-Type")
		if ctype == "" {
			contentType := "text/html; charset=UTF-8"
			w.Header().Set("Content-Type", contentType)
		}

		var body string
		isZip := strings.Contains(resp.Header.Get("Content-Encoding"), "gzip")
		if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
			body, err = replaceResponseText(*resp, replaceDict, isZip)
			if err != nil {
				fmt.Fprintln(w, err)
				return
			}
		}

		// res := &http.Response{
		// 	StatusCode: resp.StatusCode,
		// 	Header:     newResHeaders,
		// 	Body:       io.NopCloser(strings.NewReader(body)),
		// }

		// b, err := io.ReadAll(resp.Body)
		fmt.Fprintln(w, body)
	})
}

func main() {
}

func isUpgrade(headers http.Header) bool {
	connection_upgrade := headers.Get("Upgrade")
	if len(connection_upgrade) > 1 && strings.ToLower(connection_upgrade) == "websocket" {
		return true
	}
	// 检查是否为 WebSocket 升级请求
	return false
}

func isMobileDevice(userAgent string) bool {
	agents := []string{"Android", "iPhone", "SymbianOS", "Windows Phone", "iPad", "iPod"}
	for _, agent := range agents {
		if strings.Contains(userAgent, agent) {
			return false
		}
	}
	return true
}

func replaceResponseText(response http.Response, replaceDict map[string]string, gz bool) (string, error) {

	if !gz {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return "", err
		}

		text := string(body)
		for i, j := range replaceDict {
			re := regexp.MustCompile(i)
			text = re.ReplaceAllString(text, j)
		}
		return text, nil
	}

	gzipReader, err := gzip.NewReader(response.Body)
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()

	body, err := io.ReadAll(gzipReader)
	if err != nil {
		return "", err
	}

	text := string(body)
	// log.Println(text)

	for i, j := range replaceDict {
		// if i == "$upstream" {
		// 	i = upstreamDomain
		// } else if i == "$custom_domain" {
		// 	i = hostname
		// }

		// if j == "$upstream" {
		// 	j = upstreamDomain
		// } else if j == "$custom_domain" {
		// 	j = hostname
		// }

		re := regexp.MustCompile(i)
		text = re.ReplaceAllString(text, j)
	}

	var buf bytes.Buffer
	zip := gzip.NewWriter(&buf)
	_, err = zip.Write([]byte(text))
	if err != nil {
		return "", err
	}

	text = buf.String()

	return text, nil
}

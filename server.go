package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flosch/pongo2/v6"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/microcosm-cc/bluemonday"
	gomail "gopkg.in/mail.v2"
)

const (
	siteVerifyURL = "https://www.google.com/recaptcha/api/siteverify"
	captchaSecret = "6LcINqIiAAAAAEVLgj2sGPRDmSx5euUDdPwCEsii"
)

type siteVerifyResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

func main() {
	r := chi.NewRouter()
	p := bluemonday.UGCPolicy()
	r.Use(middleware.Logger)

	indexTemplate := pongo2.Must(pongo2.FromFile("www/index.html"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		context := pongo2.Context{"currentPage": "home", "title": "Welcome"}
		err := indexTemplate.ExecuteWriter(context, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.Post("/contact", func(w http.ResponseWriter, r *http.Request) {
		email := p.Sanitize(r.FormValue("email"))
		name := p.Sanitize(r.FormValue("name"))
		message := p.Sanitize(r.FormValue("message"))
		subject := p.Sanitize(r.FormValue("subject"))
		recaptchaResponse := r.FormValue("g-recaptcha-response")

		// Check and verify the recaptcha response token.
		if err := CheckRecaptcha(captchaSecret, recaptchaResponse); err != nil {
			http.Error(w, "Bad captcha result", http.StatusBadRequest)
			return
		}

		fmt.Fprintf(w, "OK")
		go SendEmail(email, name, message, subject)
	})

	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "www/"))
	FileServer(r, "/", filesDir)

	http.ListenAndServe(":3001", r)
}

// FileServer will serve the static files
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

// SendEmail will send the content of the form via email
func SendEmail(fromSender string, name string, body string, subject string) {
	if len(body) > 300 {
		body = body[:300]
	}
	m := gomail.NewMessage()

	m.SetHeader("From", "site@trifectasupport.com")
	m.SetHeader("To", "milosh.spasikj@gmail.com")
	m.SetHeader("Subject", fmt.Sprintf("Email from site (%s): %s", fromSender, subject))
	m.SetBody("text/plain", fmt.Sprintf("From: %s (%s)\n\n%s", name, fromSender, body))

	d := gomail.NewDialer("mail.trifectasupport.com", 587, "site@trifectasupport.com", "showcase-nickname-regional")
	d.TLSConfig = &tls.Config{
		ServerName:         "mail.trifectasupport.com",
		InsecureSkipVerify: false,
	}

	// Now send E-Mail
	if err := d.DialAndSend(m); err != nil {
		fmt.Println(err)
		panic(err)
	}
}

// CheckRecaptcha will verify the result with google
func CheckRecaptcha(secret, response string) error {
	req, err := http.NewRequest(http.MethodPost, siteVerifyURL, nil)
	if err != nil {
		return err
	}

	// Add necessary request parameters.
	q := req.URL.Query()
	q.Add("secret", secret)
	q.Add("response", response)
	req.URL.RawQuery = q.Encode()

	// Make request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode response.
	var body siteVerifyResponse
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}

	// Check recaptcha verification success.
	if !body.Success {
		return errors.New("unsuccessful recaptcha verify request")
	}

	return nil
}

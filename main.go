package main

import (
	"net/http"
	"fmt"
	"html"
	"path"
	"path/filepath"
	"regexp"
	"encoding/base64"
	"crypto/rand"
)

func prehtml(w http.ResponseWriter, title, lang string) {
	fmt.Fprint(w, `
<!doctype html>
<html lang="` + lang + `">
	<head>
		<meta charset="utf-8">
		<title>` + html.EscapeString(title) + `</title>
		<link rel="stylesheet" type="text/css" href="/static/styles.css">
	</head>
	<body>
		<div class="wrap">`)

}

func posthtml(w http.ResponseWriter) {
	fmt.Fprint(w, `
			<div class="bottom hidden" lang="en">
				<div class="about">
					This is part of The Grey Log, written by nils.
				</div>
			</div>
			<div class="bottom" lang="en">
				<div class="about">
					This is part of The Grey Log, written by nils.
				</div>
			</div>
		</div>
	</body>
</html>`)

}

func isindir(path, dir string) bool {
	if len(path) < len(dir) {
		return false
	}

	return dir == path[:len(dir)]
}

func serve(w http.ResponseWriter, req *http.Request) {
	path := path.Clean(req.URL.Path)

	switch {
	case isindir(path, "/static/"):
		serverstatic.ServeHTTP(w, req)
	case path == "/":
		servemain(w, req)
	case path == "/login":
		servelogin(w, req)
	default:
		goto parse
	}
	return
parse:
	check, err := regexp.MatchString("^/[0-9]+/[a-z]+$", path)
	if err != nil { panic(err) }
	if check {
		servestoryversion(w, req, path)
		return
	}
	http.NotFound(w, req)
}

var sessions map[string]string

func init() {
	sessions = make(map[string]string)
}

func servelogin(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		goto process
	}

	prehtml(w, "Login page", "en")
	fmt.Fprint(w, `
			<form method="post" action="/login" class="login">
				<input type="text" placeholder="Username" name="username"><br>
				<input type="password" placeholder="Password" name="password"><br>
				<input type="submit" value="Log in">
			</div>`)
	posthtml(w)
	return

process:
	var message string

	username, password := req.FormValue("username"), req.FormValue("password")

	err := dbauthenticate(username, password)

	if err != nil {
		message = err.Error()
	} else {
		message = "Login successful!"
	}

	rbytes := make([]byte, 30)
	rand.Read(rbytes) // unchecked error
	sessid := base64.URLEncoding.EncodeToString(rbytes)
	sessions[sessid] = username

	w.Header().Add("Set-Cookie", "sessid=" + sessid)

	prehtml(w, message, "en")
	fmt.Fprint(w, `
		<p>` + html.EscapeString(message) + `</p>`)

	posthtml(w)
}

func servestoryversion(w http.ResponseWriter, req *http.Request, path string) {
	var (
		story int
		lang string
	)

	fmt.Sscanf(path, "/%d/%s", &story, &lang)

	sv := dbgetstoryversion(story, lang)
	if sv == nil {
		http.NotFound(w, req)
	}
	prehtml(w, sv.Title, lang)
	fmt.Fprint(w, `
			<div class="storyversion">
				<h1>` + html.EscapeString(sv.Title) + `</h1>
				<p class="date">` + html.EscapeString(sv.Date) + `</p>
				` + sv.Content + `
			</div>`)
	posthtml(w)
}

var serverstatic http.Handler

func init() {
	serverstatic = http.FileServer(http.Dir("."))
}

func servestatic(w http.ResponseWriter, req *http.Request) {
	stylepath, _ := filepath.Abs("styles.css")
	http.ServeFile(w, req, stylepath)
}

func servemain(w http.ResponseWriter, req *http.Request) {
	prehtml(w, "The Grey Log", "en")
	defer posthtml(w)

	itemch := itemchannel()
	fmt.Fprint(w, `
			<div class="maintop">
				<div class="maintopbottom">
					<img src="/static/images/mainpic.png" alt="">
				</div>
			</div>
			<h1 class="maintitle">The Grey Log</h1>`)
	for v := range itemch {
		serveitem(w, v)
	}
}

func itemchannel() chan *mainitem {
	ch := make(chan *mainitem)
	go dbfeeditems("no", ch)
	return ch
}

type mainitem struct {
	StoryID string
	Date string
	Title string
	Summary string
	Langs []string
}

type storyversion struct {
	Date, Title, Content string
}

func serveitem(w http.ResponseWriter, item *mainitem) {
	fmt.Fprint(w, `
			<div class="item">
				<div class="itemdate">` + html.EscapeString(item.Date) + `</div>
				<div class="itemtext">
					<h2 class="itemtitle">
						<a href="/` + item.StoryID + "/" + item.Langs[0] + `/">
							` + item.Title + `
						</a>
					</h2>
					` + item.Summary + `
				</div>
				<div class="itemlangs">`)

	for _, v := range item.Langs {
		fmt.Fprint(w, `
					<a class="itemlang" href="/` + item.StoryID + "/" + v + `/">
						<img src="/static/images/lang/` + v + `.png">
					</a>`)
	}

	fmt.Fprint(w, `
				</div>
				<div class="clearboth"></div>
			</div>`)
}

func main() {
	http.ListenAndServe(":1111", http.HandlerFunc(serve))
}

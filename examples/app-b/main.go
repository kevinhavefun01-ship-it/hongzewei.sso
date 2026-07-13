// Package main 是 demo 子应用 B(端口 5002)。
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	appName   = "App B"
	appPort   = ":5002"
	clientID  = "demo-app"
	clientSec = "demo-secret"
	ssoPub    = "http://localhost:4446"
	callback  = "http://localhost:5002/callback"
)

var sessions sync.Map

type session struct {
	Username string
	Name     string
	LoginAt  time.Time
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/callback", callbackHandler)
	mux.HandleFunc("/logout", logoutHandler)
	fmt.Printf(" %s 启动: http://localhost%s\n", appName, appPort)
	http.ListenAndServe(appPort, mux)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	sess := getSession(r)
	if sess == nil {
		fmt.Fprintf(w, `<h1>%s</h1><a href="/login">SSO 登录</a><p>提示:已在 App A 登录则免登</p>`, appName)
		return
	}
	fmt.Fprintf(w, `<h1>已登录 %s(SSO)</h1><p>用户:%s</p><a href="/logout">退出</a>`, appName, sess.Username)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	state := randomHex(16)
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Value: state, Path: "/", MaxAge: 300})
	http.Redirect(w, r, fmt.Sprintf("%s/oauth2/auth?client_id=%s&response_type=code&redirect_uri=%s&scope=openid&state=%s", ssoPub, clientID, callback, state), http.StatusFound)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", 400)
		return
	}
	data := fmt.Sprintf("grant_type=authorization_code&code=%s&redirect_uri=%s&client_id=%s&client_secret=%s", code, callback, clientID, clientSec)
	resp, _ := http.Post(ssoPub+"/oauth2/token", "application/x-www-form-urlencoded", strings.NewReader(data))
	if resp != nil && resp.StatusCode == 200 {
		var t struct {
			AccessToken string `json:"access_token"`
		}
		json.NewDecoder(resp.Body).Decode(&t)
		sessionID := randomHex(16)
		sessions.Store(sessionID, &session{Username: "demo", Name: "Demo User", LoginAt: time.Now()})
		http.SetCookie(w, &http.Cookie{Name: "session_id", Value: sessionID, Path: "/", MaxAge: 86400})
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("session_id"); err == nil {
		sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusFound)
}

func getSession(r *http.Request) *session {
	c, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}
	v, ok := sessions.Load(c.Value)
	if !ok {
		return nil
	}
	return v.(*session)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

package web

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/example/resy-scheduler/internal/application/usecases"
	"github.com/example/resy-scheduler/internal/domain/user"
	"github.com/example/resy-scheduler/internal/internaltypes"
)

type Server struct {
	addr     string
	sessions *SessionManager
	auth     usecases.AuthService
	creds    usecases.CredentialsService
	tmpl     *template.Template
}

func New(addr string, sessions *SessionManager, auth usecases.AuthService, creds usecases.CredentialsService, tmpl *template.Template) *Server {
	return &Server{addr: addr, sessions: sessions, auth: auth, creds: creds, tmpl: tmpl}
}

func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/logout", s.handleLogout)
	mux.HandleFunc("/", s.requireAuth(s.handleHome))
	mux.HandleFunc("/credentials", s.requireAuth(s.handleCredentials))

	srv := &http.Server{
		Addr:              s.addr,
		Handler:           logging(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("listening on %s", s.addr)
	return srv.ListenAndServe()
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

type ctxKeyUserID struct{}

func userIDFromCtx(r *http.Request) string {
	if v := r.Context().Value(ctxKeyUserID{}); v != nil {
		if s, ok := v.(string); ok { return s }
	}
	return ""
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := s.sessions.GetUserID(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		c, err := s.creds.Get(ctx, uid)
		if err != nil && err != internaltypes.ErrNotFound {
			writeErr(w, err, http.StatusInternalServerError); return
		}
		if !(c.HasResy() || c.HasOpenTable()) && !strings.HasPrefix(r.URL.Path, "/credentials") {
			http.Redirect(w, r, "/credentials", http.StatusFound)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKeyUserID{}, uid)))
	}
}

func writeErr(w http.ResponseWriter, err error, code int) {
	w.WriteHeader(code)
	_, _ = w.Write([]byte(err.Error()))
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("content-type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		writeErr(w, err, http.StatusInternalServerError)
	}
}

type loginData struct {
	Error    string
	Username string
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.render(w, "login.html", loginData{})
		return
	case "POST":
		_ = r.ParseForm()
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		u, err := s.auth.VerifyPassword(ctx, username, password)
		if err != nil {
			s.render(w, "login.html", loginData{Error: "Invalid username or password", Username: username})
			return
		}
		if err := s.sessions.SetUserID(w, r, u.ID); err != nil {
			writeErr(w, err, http.StatusInternalServerError); return
		}
		http.Redirect(w, r, "/", http.StatusFound)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.sessions.Clear(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

type homeData struct {
	HasResy      bool
	HasOpenTable bool
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	uid := userIDFromCtx(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	c, err := s.creds.Get(ctx, uid)
	if err != nil && err != internaltypes.ErrNotFound {
		writeErr(w, err, http.StatusInternalServerError); return
	}
	s.render(w, "home.html", homeData{HasResy: c.HasResy(), HasOpenTable: c.HasOpenTable()})
}

type credData struct {
	Saved           bool
	Error           string
	HasResy         bool
	HasOpenTable    bool
	OpenTablePQHash string
}

func (s *Server) handleCredentials(w http.ResponseWriter, r *http.Request) {
	uid := userIDFromCtx(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	cur, _ := s.creds.Get(ctx, uid)

	switch r.Method {
	case "GET":
		s.render(w, "credentials.html", credData{
			Saved: r.URL.Query().Get("saved") == "1",
			HasResy: cur.HasResy(),
			HasOpenTable: cur.HasOpenTable(),
			OpenTablePQHash: cur.OpenTablePQHash,
		})
		return
	case "POST":
		_ = r.ParseForm()
		resyKey := strings.TrimSpace(r.FormValue("resy_api_key"))
		resyTok := strings.TrimSpace(r.FormValue("resy_auth_token"))
		otTok := strings.TrimSpace(r.FormValue("opentable_token"))
		pq := strings.TrimSpace(r.FormValue("opentable_pq_hash"))

		c := user.Credentials{
			UserID: uid,
			ResyAPIKey: cur.ResyAPIKey,
			ResyAuthToken: cur.ResyAuthToken,
			OpenTableToken: cur.OpenTableToken,
			OpenTablePQHash: cur.OpenTablePQHash,
		}
		if resyKey != "" { c.ResyAPIKey = resyKey }
		if resyTok != "" { c.ResyAuthToken = resyTok }
		if otTok != "" { c.OpenTableToken = otTok }
		c.OpenTablePQHash = pq

		if !(c.HasResy() || c.HasOpenTable()) {
			s.render(w, "credentials.html", credData{
				Error: "Please provide at least one set of credentials (Resy and/or OpenTable).",
				HasResy: cur.HasResy(),
				HasOpenTable: cur.HasOpenTable(),
				OpenTablePQHash: cur.OpenTablePQHash,
			})
			return
		}
		if err := s.creds.Update(ctx, c); err != nil {
			s.render(w, "credentials.html", credData{Error: err.Error()}); return
		}
		http.Redirect(w, r, "/credentials?saved=1", http.StatusFound)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

package web

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/resy-scheduler/internal/auth"
	"github.com/example/resy-scheduler/internal/jobs"
)

//go:embed templates/*.html static/*
var fs embed.FS

type Server struct {
	Auth *auth.Store
	Jobs *jobs.Repo

	BaseURL string
}

type tmplData struct {
	Title string
	User  int64

	Flash string
	Jobs  []jobs.Job
	Job   jobs.Job
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/static/", http.FileServer(http.FS(fs)))

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/logout", s.handleLogout)

	authed := s.Auth.RequireAuth(http.HandlerFunc(s.handleHome))
	mux.Handle("/", authed)
	mux.Handle("/jobs/new", s.Auth.RequireAuth(http.HandlerFunc(s.handleJobNew)))
	mux.Handle("/jobs/create", s.Auth.RequireAuth(http.HandlerFunc(s.handleJobCreate)))

	return mux
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	js, err := s.Jobs.ListByUser(r.Context(), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "templates/jobs.html", tmplData{
		Title: "Jobs",
		User:  uid,
		Jobs:  js,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.render(w, "templates/login.html", tmplData{Title: "Login"})
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")
		id, err := s.Auth.Authenticate(r.Context(), username, password)
		if err != nil {
			s.render(w, "templates/login.html", tmplData{Title: "Login", Flash: "Invalid username/password"})
			return
		}
		if err := s.Auth.SetSession(w, r, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.Auth.ClearSession(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (s *Server) handleJobNew(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	s.render(w, "templates/new_job.html", tmplData{
		Title: "New Job",
		User:  uid,
		Job: jobs.Job{
			Timezone:       "America/New_York",
			IntervalSec:    10,
			PreferredTimes: []string{"19:00:00", "19:15:00", "18:45:00"},
		},
	})
}

func (s *Server) handleJobCreate(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	partySize, _ := strconv.Atoi(r.FormValue("party_size"))
	intervalSec, _ := strconv.Atoi(r.FormValue("interval_seconds"))

	resDate, err := time.Parse("2006-01-02", r.FormValue("reservation_date"))
	if err != nil {
		s.render(w, "templates/new_job.html", tmplData{Title: "New Job", Flash: "Invalid reservation date"})
		return
	}

	tz := strings.TrimSpace(r.FormValue("timezone"))
	if tz == "" {
		tz = "America/New_York"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		s.render(w, "templates/new_job.html", tmplData{Title: "New Job", Flash: "Invalid timezone"})
		return
	}

	// "Optimal window" helper inputs:
	// - days_out: number of days before reservation_date when slots open
	// - release_time: HH:MM (local to timezone)
	// - lead_minutes: start window N minutes before release_time
	// - window_minutes: attempt window length
	daysOut, _ := strconv.Atoi(r.FormValue("days_out"))
	leadMin, _ := strconv.Atoi(r.FormValue("lead_minutes"))
	windowMin, _ := strconv.Atoi(r.FormValue("window_minutes"))
	releaseTime := strings.TrimSpace(r.FormValue("release_time")) // HH:MM
	if releaseTime == "" {
		releaseTime = "00:00"
	}

	openDate := resDate.AddDate(0, 0, -daysOut)
	openAtLocal, err := time.ParseInLocation("2006-01-02 15:04", openDate.Format("2006-01-02")+" "+releaseTime, loc)
	if err != nil {
		s.render(w, "templates/new_job.html", tmplData{Title: "New Job", Flash: "Invalid release time"})
		return
	}
	windowStart := openAtLocal.Add(-time.Duration(leadMin) * time.Minute).UTC()
	windowEnd := openAtLocal.Add(time.Duration(windowMin) * time.Minute).UTC()

	j := jobs.Job{
		UserID:           uid,
		Name:             strings.TrimSpace(r.FormValue("name")),
		VenueID:          strings.TrimSpace(r.FormValue("venue_id")),
		PartySize:        partySize,
		ReservationDate:  resDate,
		PreferredTimes:   splitCSV(r.FormValue("preferred_times")),
		ReservationTypes: strings.TrimSpace(r.FormValue("reservation_types")),
		Timezone:         tz,
		WindowStartAt:    windowStart,
		WindowEndAt:      windowEnd,
		IntervalSec:      intervalSec,
	}

	if err := j.Validate(); err != nil {
		s.render(w, "templates/new_job.html", tmplData{Title: "New Job", Flash: err.Error(), Job: j})
		return
	}

	if _, err := s.Jobs.Create(r.Context(), j); err != nil {
		log.Printf("create job err: %v", err)
		s.render(w, "templates/new_job.html", tmplData{Title: "New Job", Flash: "Failed to create job", Job: j})
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (s *Server) render(w http.ResponseWriter, name string, data tmplData) {
	t, err := template.ParseFS(fs,
		"templates/base.html",
		name,
	)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
	}
}

func Start(ctx context.Context, addr string, h http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	fmt.Printf("listening on %s\n", addr)
	return srv.ListenAndServe()
}

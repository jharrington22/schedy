package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"time"

	"github.com/example/resy-scheduler/internal/db"
	"github.com/gorilla/securecookie"
	"golang.org/x/crypto/bcrypt"
)

type Store struct {
	sc *securecookie.SecureCookie
	db *db.DB
}

type ctxKey string

const userIDKey ctxKey = "userID"

func NewStore(d *db.DB, hashKey, blockKey []byte) *Store {
	sc := securecookie.New(hashKey, blockKey)
	// keep cookie small and secure
	sc.MaxAge(int((14 * 24 * time.Hour).Seconds()))
	return &Store{sc: sc, db: d}
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, pw string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
	return err == nil
}

func (s *Store) CreateUser(ctx context.Context, username, password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	return s.db.Exec(ctx, `INSERT INTO users(username, password_bcrypt) VALUES ($1,$2)`, username, hash)
}

func (s *Store) Authenticate(ctx context.Context, username, password string) (int64, error) {
	var id int64
	var hash string
	err := s.db.QueryRow(ctx, `SELECT id, password_bcrypt FROM users WHERE username=$1`, username).Scan(&id, &hash)
	if err != nil {
		return 0, db.WrapNotFound(err)
	}
	if !CheckPassword(hash, password) {
		return 0, errors.New("invalid credentials")
	}
	return id, nil
}

type Session struct {
	UserID int64
}

const cookieName = "resysched_session"

func (s *Store) SetSession(w http.ResponseWriter, r *http.Request, userID int64) error {
	val := map[string]any{"uid": userID, "v": 1}
	encoded, err := s.sc.Encode(cookieName, val)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil, // ok for local http; secure in https
		MaxAge:   int((14 * 24 * time.Hour).Seconds()),
	})
	return nil
}

func (s *Store) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func (s *Store) GetSession(r *http.Request) (Session, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return Session{}, false
	}
	val := map[string]any{}
	if err := s.sc.Decode(cookieName, c.Value, &val); err != nil {
		return Session{}, false
	}
	uidAny, ok := val["uid"]
	if !ok {
		return Session{}, false
	}
	uid, ok := uidAny.(float64) // json-ish decode
	if !ok || uid <= 0 {
		// sometimes it comes back as int64 when using securecookie; handle both
		if i, ok2 := uidAny.(int64); ok2 && i > 0 {
			return Session{UserID: i}, true
		}
		return Session{}, false
	}
	return Session{UserID: int64(uid)}, true
}

func (s *Store) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := s.GetSession(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, sess.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	uid, ok := ctx.Value(userIDKey).(int64)
	return uid, ok
}

// Constant-time string compare helper (used in tests / future headers)
func secureEq(a, b string) bool { //nolint:unused
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

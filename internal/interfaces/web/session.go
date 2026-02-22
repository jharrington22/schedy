package web

import (
	"net/http"

	"github.com/gorilla/securecookie"
)

const sessionName = "resysched_session"

type SessionManager struct{ sc *securecookie.SecureCookie }

func NewSessionManager(hashKey, blockKey []byte) *SessionManager {
	return &SessionManager{sc: securecookie.New(hashKey, blockKey)}
}

func (s *SessionManager) SetUserID(w http.ResponseWriter, r *http.Request, userID string) error {
	value := map[string]string{"uid": userID}
	encoded, err := s.sc.Encode(sessionName, value)
	if err != nil { return err }
	http.SetCookie(w, &http.Cookie{
		Name: sessionName, Value: encoded, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (s *SessionManager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

func (s *SessionManager) GetUserID(r *http.Request) (string, bool) {
	c, err := r.Cookie(sessionName)
	if err != nil { return "", false }
	value := map[string]string{}
	if err := s.sc.Decode(sessionName, c.Value, &value); err != nil { return "", false }
	uid := value["uid"]
	if uid == "" { return "", false }
	return uid, true
}

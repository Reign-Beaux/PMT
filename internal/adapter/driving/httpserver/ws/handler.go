package ws

import (
	"encoding/json"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	"project-management-tools/internal/domain/shared"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Handler struct {
	hub       *Hub
	jwtSecret []byte
}

func NewHandler(hub *Hub, jwtSecret []byte) *Handler {
	return &Handler{hub: hub, jwtSecret: jwtSecret}
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.authenticate(r)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	c := &client{
		ownerID: ownerID,
		conn:    conn,
		send:    make(chan []byte, 256),
		hub:     h.hub,
	}

	h.hub.register <- c

	go c.writePump()
	go c.readPump()
}

func (h *Handler) authenticate(r *http.Request) (shared.ID, bool) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		if auth := r.Header.Get("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
			tokenStr = auth[7:]
		}
	}
	if tokenStr == "" {
		if cookie, err := r.Cookie("access_token"); err == nil {
			tokenStr = cookie.Value
		}
	}
	if tokenStr == "" {
		return shared.ID{}, false
	}

	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return h.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return shared.ID{}, false
	}

	id, err := shared.ParseID(claims.Subject)
	if err != nil {
		return shared.ID{}, false
	}

	return id, true
}

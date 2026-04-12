package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"project-management-tools/internal/adapter/driving/httpserver/middleware"
	userapp "project-management-tools/internal/application/user"
	"project-management-tools/internal/domain/shared"
	"project-management-tools/internal/domain/user"
)

const accessTokenTTL = 15 * time.Minute

// AuthService is the driving port — defined here, in the consumer.
type AuthService interface {
	Register(ctx context.Context, input userapp.RegisterInput) (user.User, error)
	Authenticate(ctx context.Context, email, password string) (user.User, error)
	GetByID(ctx context.Context, id shared.ID) (user.User, error)
	IssueRefreshToken(ctx context.Context, userID shared.ID) (string, error)
	RotateRefreshToken(ctx context.Context, token string) (user.User, string, error)
	RevokeRefreshToken(ctx context.Context, token string) error
}

type AuthHandler struct {
	svc       AuthService
	jwtSecret []byte
}

func NewAuthHandler(svc AuthService, jwtSecret []byte) *AuthHandler {
	return &AuthHandler{svc: svc, jwtSecret: jwtSecret}
}

// POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := h.svc.Register(r.Context(), userapp.RegisterInput{
		Email:    body.Email,
		Password: body.Password,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}

	if err := h.issueTokens(w, r, u); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, toUserResponse(u))
}

// POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := h.svc.Authenticate(r.Context(), body.Email, body.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	if err := h.issueTokens(w, r, u); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(u))
}

// POST /auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	u, newRefresh, err := h.svc.RotateRefreshToken(r.Context(), cookie.Value)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	accessToken, err := h.generateAccessToken(u.ID().String())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	setAccessTokenCookie(w, accessToken)
	setRefreshTokenCookie(w, newRefresh)

	writeJSON(w, http.StatusOK, toUserResponse(u))
}

// GET /auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	u, err := h.svc.GetByID(r.Context(), userID)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(u))
}

// POST /auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil {
		_ = h.svc.RevokeRefreshToken(r.Context(), cookie.Value)
	}

	clearCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *AuthHandler) issueTokens(w http.ResponseWriter, r *http.Request, u user.User) error {
	accessToken, err := h.generateAccessToken(u.ID().String())
	if err != nil {
		return err
	}

	refreshToken, err := h.svc.IssueRefreshToken(r.Context(), u.ID())
	if err != nil {
		return err
	}

	setAccessTokenCookie(w, accessToken)
	setRefreshTokenCookie(w, refreshToken)
	return nil
}

func (h *AuthHandler) generateAccessToken(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(h.jwtSecret)
}

func setAccessTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		MaxAge:   int(accessTokenTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func setRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/auth/refresh",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "", Path: "/auth/refresh", MaxAge: -1, HttpOnly: true})
}

type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func toUserResponse(u user.User) userResponse {
	return userResponse{
		ID:        u.ID().String(),
		Email:     u.Email().String(),
		CreatedAt: u.CreatedAt(),
	}
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, userapp.ErrEmailAlreadyExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, userapp.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, user.ErrInvalidEmail),
		errors.Is(err, user.ErrInvalidPassword):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, userapp.ErrInvalidRefreshToken),
		errors.Is(err, userapp.ErrRefreshTokenExpired):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

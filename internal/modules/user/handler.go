package user

import (
	"encoding/json"
	"net/http"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/pagination"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// adminSchool returns the school a school_admin caller is limited to, or nil
// for a super_admin, who manages every school. The /users routes only admit
// these two roles (see module.go).
func adminSchool(r *http.Request) (*uuid.UUID, error) {
	claims := httpx.ClaimsFromContext(r.Context())
	if claims == nil {
		return nil, apperr.ErrUnauthorized
	}
	if claims.Role == string(domain.RoleSuperAdmin) {
		return nil, nil
	}
	id, err := uuid.Parse(claims.SchoolID)
	if err != nil {
		return nil, apperr.ErrForbidden
	}
	return &id, nil
}

// managedUser loads the user {id} if the caller may manage them: a
// super_admin may manage anyone, a school_admin only their own school's
// users and never a super_admin. Anyone else's account reads as not found.
func (h *Handler) managedUser(r *http.Request, id uuid.UUID) (*domain.User, error) {
	school, err := adminSchool(r)
	if err != nil {
		return nil, err
	}
	u, err := h.svc.Get(r.Context(), id)
	if err != nil {
		return nil, err
	}
	if school != nil && (u.Role == domain.RoleSuperAdmin || u.SchoolID == nil || *u.SchoolID != *school) {
		return nil, apperr.ErrNotFound
	}
	return u, nil
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in CreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	school, err := adminSchool(r)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if school != nil {
		// A school_admin adds users to their own school only, and can't create a super_admin.
		if in.Role == domain.RoleSuperAdmin {
			httpx.Error(w, http.StatusForbidden, "only a super admin can create super admin accounts")
			return
		}
		in.SchoolID = school
	}
	u, err := h.svc.Create(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, u)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, err := h.managedUser(r, id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if _, err := h.managedUser(r, id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	u, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.managedUser(r, id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	var schoolID *uuid.UUID
	if sid := r.URL.Query().Get("school_id"); sid != "" {
		id, err := uuid.Parse(sid)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid school_id")
			return
		}
		schoolID = &id
	}
	school, err := adminSchool(r)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	if school != nil {
		schoolID = school // a school_admin only ever lists their own school
	}
	status := domain.UserStatus(r.URL.Query().Get("status"))
	items, total, err := h.svc.List(r.Context(), schoolID, status, p.Limit, p.Offset)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, pagination.NewListResponse(items, total, p.Limit, p.Offset))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in LoginInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	resp, err := h.svc.Login(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// LogoutEverywhere invalidates every JWT issued to the caller, including
// the one used for this request -- the frontend clears its own stored
// token immediately after this call succeeds, exactly like a normal logout.
func (h *Handler) LogoutEverywhere(w http.ResponseWriter, r *http.Request) {
	claims := httpx.ClaimsFromContext(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.svc.LogoutEverywhere(r.Context(), userID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

// RequestPasswordReset always responds success, whether or not the email
// belongs to an account -- see Service.RequestPasswordReset.
func (h *Handler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.svc.RequestPasswordReset(r.Context(), in.Email); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "if that email is registered, a reset link has been sent"})
}

func (h *Handler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.svc.ConfirmPasswordReset(r.Context(), in.Token, in.NewPassword); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "password updated"})
}

// Register handles public self-registration. The created account is inactive
// and pending until an admin approves it via Approve.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	u, err := h.svc.Register(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{
		"user":    u,
		"message": "registration submitted; an administrator must approve your account before you can log in",
	})
}

// Approve marks a pending user as approved (admin only, see RequireRole in module.go).
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.managedUser(r, id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	u, err := h.svc.Approve(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

// Reject marks a pending user as rejected (admin only, see RequireRole in module.go).
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.managedUser(r, id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	u, err := h.svc.Reject(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

// Me returns the caller's own profile -- unlike Get, it's not gated by the
// user_management feature or BlockRoles("registrar"), since it only ever
// returns the caller's own record regardless of role.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := httpx.ClaimsFromContext(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	u, err := h.svc.Get(r.Context(), userID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

// RequestPhoneVerification sends an OTP to a phone number the caller wants
// to attach to their own account -- it isn't saved until ConfirmPhoneVerification.
func (h *Handler) RequestPhoneVerification(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.svc.RequestPhoneVerification(r.Context(), in.Phone); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// ConfirmPhoneVerification checks the OTP and, on success, attaches phone to
// the caller's own account (never someone else's).
func (h *Handler) ConfirmPhoneVerification(w http.ResponseWriter, r *http.Request) {
	claims := httpx.ClaimsFromContext(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in struct {
		Phone string `json:"phone"`
		OTP   string `json:"otp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.svc.ConfirmPhoneVerification(r.Context(), userID, in.Phone, in.OTP); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	u, err := h.svc.Get(r.Context(), userID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

// RequestOTPLogin sends a login OTP to phone (must already be a verified,
// usable account -- see Service.RequestOTPLogin for why this can't hide
// whether a number is registered the way the email reset flow does).
func (h *Handler) RequestOTPLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.svc.RequestOTPLogin(r.Context(), in.Phone); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *Handler) VerifyOTPLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Phone string `json:"phone"`
		OTP   string `json:"otp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	resp, err := h.svc.VerifyOTPLogin(r.Context(), in.Phone, in.OTP)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// RequestPasswordResetOTP always responds success, whether or not the
// phone belongs to an account -- see Service.RequestPasswordResetOTP.
func (h *Handler) RequestPasswordResetOTP(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.svc.RequestPasswordResetOTP(r.Context(), in.Phone); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *Handler) ConfirmPasswordResetOTP(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Phone       string `json:"phone"`
		OTP         string `json:"otp"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.svc.ConfirmPasswordResetOTP(r.Context(), in.Phone, in.OTP, in.NewPassword); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "reset"})
}

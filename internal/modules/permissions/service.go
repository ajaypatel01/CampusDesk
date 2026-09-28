package permissions

import (
	"context"
	"fmt"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	userrepo "github.com/ajaypatel01/CampusDesk/internal/modules/user"
	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/google/uuid"
)

type Service struct {
	repo  *Repository
	users *userrepo.Repository
}

func NewService(repo *Repository, users *userrepo.Repository) *Service {
	return &Service{repo: repo, users: users}
}

// FeatureRow is one line of the access-control matrix for a target user:
// the feature, and its current effective view/write access plus whether
// that's an explicit override or just the default.
type FeatureRow struct {
	FeatureKey string `json:"feature_key"`
	Label      string `json:"label"`
	CanView    bool   `json:"can_view"`
	CanWrite   bool   `json:"can_write"`
	IsOverride bool   `json:"is_override"`
}

// canManage reports whether actor may edit target's permissions: a
// super_admin may edit anyone except another super_admin (whose access is
// always full and not something to narrow); a school_admin may only edit
// teacher/registrar/parent accounts within their own school -- never another
// admin's, and never across schools.
func canManage(actor *httpx.Claims, target *domain.User) error {
	if actor == nil {
		return apperr.ErrForbidden
	}
	if target.Role == domain.RoleSuperAdmin {
		return fmt.Errorf("%w: a super_admin's access can't be restricted", apperr.ErrForbidden)
	}
	if actor.Role == string(domain.RoleSuperAdmin) {
		return nil
	}
	if actor.Role != string(domain.RoleSchoolAdmin) {
		return apperr.ErrForbidden
	}
	if target.Role == domain.RoleSchoolAdmin {
		return fmt.Errorf("%w: only a super_admin can restrict another school admin", apperr.ErrForbidden)
	}
	if target.SchoolID == nil || target.SchoolID.String() != actor.SchoolID {
		return fmt.Errorf("%w: that user is not in your school", apperr.ErrForbidden)
	}
	return nil
}

// GetMatrix returns the full feature catalog decorated with targetUserID's
// current effective access, for the admin UI to render as a checkbox grid.
func (s *Service) GetMatrix(ctx context.Context, actor *httpx.Claims, targetUserID uuid.UUID) ([]FeatureRow, error) {
	target, err := s.users.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if err := canManage(actor, target); err != nil {
		return nil, err
	}
	overrides, err := s.repo.ListByUser(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]Override, len(overrides))
	for _, o := range overrides {
		byKey[o.FeatureKey] = o
	}
	rows := make([]FeatureRow, len(domain.Features))
	for i, f := range domain.Features {
		if o, ok := byKey[f.Key]; ok {
			rows[i] = FeatureRow{FeatureKey: f.Key, Label: f.Label, CanView: o.CanView, CanWrite: o.CanWrite, IsOverride: true}
		} else {
			rows[i] = FeatureRow{FeatureKey: f.Key, Label: f.Label, CanView: true, CanWrite: true, IsOverride: false}
		}
	}
	return rows, nil
}

// SetInput is one row of an admin's submitted matrix. A feature key omitted
// entirely from the submitted set resets it to the default (fully allowed).
type SetInput struct {
	FeatureKey string `json:"feature_key"`
	CanView    bool   `json:"can_view"`
	CanWrite   bool   `json:"can_write"`
}

// SetMatrix replaces targetUserID's overrides with exactly the rows in in --
// a row identical to the default (view+write both true) is dropped rather
// than stored, so "reset to default" and "explicitly grant everything" are
// indistinguishable, which is the simplest and safest behavior.
func (s *Service) SetMatrix(ctx context.Context, actor *httpx.Claims, targetUserID uuid.UUID, in []SetInput) error {
	target, err := s.users.GetByID(ctx, targetUserID)
	if err != nil {
		return err
	}
	if err := canManage(actor, target); err != nil {
		return err
	}
	overrides := make([]Override, 0, len(in))
	for _, row := range in {
		if !domain.IsValidFeatureKey(row.FeatureKey) {
			return fmt.Errorf("%w: unknown feature %q", apperr.ErrInvalidInput, row.FeatureKey)
		}
		if row.CanView && row.CanWrite {
			continue // matches the default -- no need to store a row
		}
		overrides = append(overrides, Override{FeatureKey: row.FeatureKey, CanView: row.CanView, CanWrite: row.CanWrite})
	}
	return s.repo.ReplaceForUser(ctx, target.SchoolID, targetUserID, overrides)
}

// EffectiveForRequest builds the httpx.FeatureAccess map for the current
// caller, used by the loader middleware on every request. super_admin
// always gets fullAccess=true (and skips the DB round-trip).
func (s *Service) EffectiveForRequest(ctx context.Context, claims *httpx.Claims) (features map[string]httpx.FeatureAccess, fullAccess bool, err error) {
	if claims == nil {
		return nil, false, nil
	}
	if claims.Role == string(domain.RoleSuperAdmin) {
		return nil, true, nil
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		return nil, false, nil
	}
	overrides, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	features = make(map[string]httpx.FeatureAccess, len(overrides))
	for _, o := range overrides {
		features[o.FeatureKey] = httpx.FeatureAccess{View: o.CanView, Write: o.CanWrite}
	}
	return features, false, nil
}

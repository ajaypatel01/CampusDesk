package staff

import (
	"context"

	"github.com/ajaypatel01/CampusDesk/internal/domain"
	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, schoolID *uuid.UUID, limit, offset int) ([]domain.StaffMember, int, error) {
	return s.repo.List(ctx, schoolID, limit, offset)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.StaffMember, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) UpsertProfile(ctx context.Context, userID uuid.UUID, in UpdateProfileInput) (*domain.StaffProfile, error) {
	p := &domain.StaffProfile{
		UserID:                    userID,
		GuardianName:              in.GuardianName,
		AadharNumber:              in.AadharNumber,
		EducationQualification:    in.EducationQualification,
		ProfessionalQualification: in.ProfessionalQualification,
		Designation:               in.Designation,
		BasicSalary:               in.BasicSalary,
		HRA:                       in.HRA,
		SpecialAllowance:          in.SpecialAllowance,
		Bonus:                     in.Bonus,
		EPF:                       in.EPF,
		ESIC:                      in.ESIC,
		AdditionalDeduction:       in.AdditionalDeduction,
		AdditionalDeductionLabel:  in.AdditionalDeductionLabel,
		BankName:                  in.BankName,
		BankIFSC:                  in.BankIFSC,
		BankBranch:                in.BankBranch,
		BankAccountNumber:         in.BankAccountNumber,
		BankAccountHolder:         in.BankAccountHolder,
		Phone:                     in.Phone,
		StaffType:                 in.StaffType,
		CLQuotaPerYear:            in.CLQuotaPerYear,
	}
	if err := s.repo.UpsertProfile(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

type UpdateProfileInput struct {
	GuardianName              *string `json:"guardian_name"`
	AadharNumber              *string `json:"aadhar_number"`
	EducationQualification    *string `json:"education_qualification"`
	ProfessionalQualification *string `json:"professional_qualification"`
	Designation               *string `json:"designation"`
	BasicSalary               int     `json:"basic_salary"`
	HRA                       int     `json:"hra"`
	SpecialAllowance          int     `json:"special_allowance"`
	Bonus                     int     `json:"bonus"`
	EPF                       int     `json:"epf"`
	ESIC                      int     `json:"esic"`
	AdditionalDeduction       int     `json:"additional_deduction"`
	AdditionalDeductionLabel  *string `json:"additional_deduction_label"`
	BankName                  *string `json:"bank_name"`
	BankIFSC                  *string `json:"bank_ifsc"`
	BankBranch                *string `json:"bank_branch"`
	BankAccountNumber         *string `json:"bank_account_number"`
	BankAccountHolder         *string `json:"bank_account_holder"`
	Phone                     *string `json:"phone"`
	StaffType                 *string `json:"staff_type"`
	CLQuotaPerYear            int     `json:"cl_quota_per_year"`
}

// Package whatsappbot is the inbound half of WhatsApp integration: a parent
// messages the school's WhatsApp Business number from a phone number that
// matches a guardian record, and gets an automatic reply with their
// ward(s)' fee balance and latest result -- no login involved, identified
// purely by phone number (see guardian.Repository.FindStudentIDsByPhone's
// doc comment for why that's the one place in the app this is true, and the
// matching tradeoff that implies).
//
// This is the opposite direction from internal/modules/communications
// (admin-initiated broadcasts out to parents/staff), and composes existing
// modules' repositories/services directly rather than duplicating fee or
// result logic, the same pattern internal/modules/promotion already uses.
package whatsappbot

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajaypatel01/CampusDesk/internal/modules/academic"
	"github.com/ajaypatel01/CampusDesk/internal/modules/fee"
	"github.com/ajaypatel01/CampusDesk/internal/modules/guardian"
	"github.com/ajaypatel01/CampusDesk/internal/modules/results"
	"github.com/ajaypatel01/CampusDesk/internal/modules/student"
	"github.com/ajaypatel01/CampusDesk/internal/platform/whatsapp"
)

type Service struct {
	guardians *guardian.Repository
	students  *student.Repository
	academic  *academic.Repository
	fees      *fee.Service
	results   *results.Repository
	wa        *whatsapp.Client
}

func NewService(guardians *guardian.Repository, students *student.Repository, academicRepo *academic.Repository, fees *fee.Service, resultsRepo *results.Repository, wa *whatsapp.Client) *Service {
	return &Service{guardians: guardians, students: students, academic: academicRepo, fees: fees, results: resultsRepo, wa: wa}
}

func (s *Service) Enabled() bool { return s.wa.WebhookEnabled() }

// HandleInboundMessage looks up which student(s) the sender's phone number
// is a guardian for and replies with a fee/result summary for each. Any
// message text at all triggers the same summary -- there's no menu/keyword
// parsing (yet); this is intentionally the simplest useful version.
// Errors are returned for logging but the caller (the webhook handler)
// should never surface them to Meta as a failure -- an unrecognized number
// or a student with no fee account yet are both expected, common cases,
// not bugs.
func (s *Service) HandleInboundMessage(ctx context.Context, fromPhone string) error {
	if !s.wa.Enabled() {
		return fmt.Errorf("whatsapp client not configured, cannot reply")
	}
	studentIDs, err := s.guardians.FindStudentIDsByPhone(ctx, fromPhone)
	if err != nil {
		return err
	}
	if len(studentIDs) == 0 {
		return s.wa.SendText(fromPhone,
			"This number isn't registered as a parent/guardian contact with the school. "+
				"Please contact the school office if you believe this is a mistake.")
	}

	var reply strings.Builder
	reply.WriteString("Here's the latest for your ward(s):\n")
	for _, studentID := range studentIDs {
		st, err := s.students.GetByID(ctx, studentID)
		if err != nil {
			continue
		}
		year, err := s.academic.GetCurrentYear(ctx, st.SchoolID)
		if err != nil {
			reply.WriteString(fmt.Sprintf("\n*%s %s*: no current academic year set up yet.\n", st.FirstName, st.LastName))
			continue
		}

		reply.WriteString(fmt.Sprintf("\n*%s %s* (%s)\n", st.FirstName, st.LastName, st.StudentCode))

		if fs, err := s.fees.StudentFeeSummary(ctx, studentID, year.ID); err == nil {
			if fs.BalanceRemaining > 0 {
				reply.WriteString(fmt.Sprintf("Fees: Rs.%d outstanding\n", fs.BalanceRemaining))
			} else {
				reply.WriteString("Fees: fully paid\n")
			}
		} else {
			reply.WriteString("Fees: no fee account set up yet for this year\n")
		}

		if rc, err := s.results.GetReportCard(ctx, studentID, year.ID); err == nil && len(rc.Exams) > 0 {
			if rc.OverallGrade != "" {
				reply.WriteString(fmt.Sprintf("Latest result: %.1f%% (Grade %s)\n", rc.OverallPercent, rc.OverallGrade))
			} else {
				reply.WriteString(fmt.Sprintf("Latest result: %.1f%%\n", rc.OverallPercent))
			}
		} else {
			reply.WriteString("Latest result: not published yet\n")
		}
	}

	return s.wa.SendText(fromPhone, reply.String())
}

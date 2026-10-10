package user

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"regexp"
	"strings"
	"time"

	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/fast2sms"
	"github.com/ajaypatel01/CampusDesk/internal/platform/whatsapp"
	"github.com/google/uuid"
)

// OTPSender delivers one-time codes: on WhatsApp (an approved
// Authentication template) when that is set up, otherwise -- or if WhatsApp
// fails -- by SMS through Fast2SMS. OTP features are off until one is set up.
type OTPSender struct {
	Client   *whatsapp.Client
	Template string
	Language string
	// SchoolSenders maps a school to the Phone number ID its codes are sent
	// from, so parents and staff see their own school's number. Other
	// schools use Client's default number.
	SchoolSenders map[uuid.UUID]string
	SMS           *fast2sms.Client
}

func (o OTPSender) whatsappReady() bool {
	return o.Client.HasToken() && o.Template != "" && (o.Client.Enabled() || len(o.SchoolSenders) > 0)
}

func (o OTPSender) enabled() bool { return o.whatsappReady() || o.SMS.Enabled() }

// clientFor returns the client that sends for schoolID, or nil if that
// school has no number and there's no default.
func (o OTPSender) clientFor(schoolID *uuid.UUID) *whatsapp.Client {
	if schoolID != nil {
		if id, ok := o.SchoolSenders[*schoolID]; ok {
			return o.Client.From(id)
		}
	}
	if o.Client.Enabled() {
		return o.Client
	}
	return nil
}

const (
	otpLength      = 6
	otpTTL         = 5 * time.Minute
	otpMaxAttempts = 5
	// At most otpSendLimit codes per number per otpSendWindow, and
	// otpDailyLimit per day, so nobody can run up the WhatsApp/SMS bill or
	// flood someone's phone.
	otpSendLimit  = 3
	otpSendWindow = 15 * time.Minute
	otpDailyLimit = 10
)

const (
	otpPurposeLogin       = "login"
	otpPurposeVerifyPhone = "verify_phone"
	otpPurposeReset       = "password_reset"
)

var errOTPNotConfigured = fmt.Errorf("%w: login by OTP is not set up yet", apperr.ErrInvalidInput)

// Channels a code can go out on (returned to the app so it can say where
// to look for it).
const (
	channelWhatsApp = "whatsapp"
	channelSMS      = "sms"
)

// phoneRun matches one written phone number, allowing a leading + and
// spaces, dashes or brackets between digits ("+91 98765-43210").
var phoneRun = regexp.MustCompile(`\+?[0-9][0-9 ()\-]*[0-9]`)

// mobileNumbers returns every Indian mobile number in s as its 10 digits.
// Guardian phones are free text and may hold more than one number
// ("9876543210 / 9123456789", "98765 43210, 091234 56789").
func mobileNumbers(s string) []string {
	var out []string
	for _, run := range phoneRun.FindAllString(s, -1) {
		digits := onlyDigits(run)
		if looksLikeLandline(run, digits) {
			continue
		}
		if n, ok := mobile10(digits); ok {
			out = append(out, n)
			continue
		}
		// Two numbers separated only by a space or dash read as one run.
		if len(digits) == 20 {
			a, okA := mobile10(digits[:10])
			b, okB := mobile10(digits[10:])
			if okA && okB {
				out = append(out, a, b)
			}
		}
	}
	return out
}

// looksLikeLandline spots "0755-2345678": a 0 plus an STD code split off
// from the rest. A mobile with a leading 0 is written "09876543210",
// "0 9876543210" or "098765 43210".
func looksLikeLandline(run, digits string) bool {
	if len(digits) != 11 || digits[0] != '0' {
		return false
	}
	first := len(onlyDigits(strings.FieldsFunc(run, func(r rune) bool {
		return r == ' ' || r == '-' || r == '(' || r == ')'
	})[0]))
	return first != 1 && first != 6 && first != 11
}

// normalizePhone turns a number the user typed into its 10 digits.
func normalizePhone(s string) (string, error) {
	if n, ok := mobile10(onlyDigits(s)); ok && !looksLikeLandline(s, onlyDigits(s)) {
		return n, nil
	}
	return "", fmt.Errorf("%w: enter a valid 10-digit mobile number", apperr.ErrInvalidInput)
}

// mobile10 accepts 10 digits, or 10 digits behind 0 / 91 / 091 / 0091.
func mobile10(d string) (string, bool) {
	switch {
	case len(d) == 11 && d[0] == '0':
		d = d[1:]
	case len(d) == 12 && strings.HasPrefix(d, "91"):
		d = d[2:]
	case len(d) == 13 && strings.HasPrefix(d, "091"):
		d = d[3:]
	case len(d) == 14 && strings.HasPrefix(d, "0091"):
		d = d[4:]
	}
	if len(d) != 10 || d[0] < '6' {
		return "", false
	}
	return d, true
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *Service) otpHash(phone, purpose, code string) string {
	m := hmac.New(sha256.New, []byte(s.jwtSecret))
	m.Write([]byte(phone + "|" + purpose + "|" + code))
	return hex.EncodeToString(m.Sum(nil))
}

// sendOTP creates a new code for phone (10 digits) and sends it -- on
// WhatsApp from schoolID's number if WhatsApp is set up, else by SMS, and
// by SMS if WhatsApp fails. Any earlier unused code for the same purpose
// stops working. Returns the channel used.
func (s *Service) sendOTP(ctx context.Context, phone, purpose string, schoolID *uuid.UUID) (string, error) {
	var wa *whatsapp.Client
	if s.otp.whatsappReady() {
		wa = s.otp.clientFor(schoolID)
	}
	if wa == nil && !s.otp.SMS.Enabled() {
		return "", errOTPNotConfigured
	}
	now := time.Now()
	recent, today, err := s.repo.CountOTPs(ctx, phone, purpose, now.Add(-otpSendWindow), now.Add(-24*time.Hour))
	if err != nil {
		return "", err
	}
	if recent >= otpSendLimit || today >= otpDailyLimit {
		return "", fmt.Errorf("%w: too many codes requested for this number, please try again later", apperr.ErrTooMany)
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	code := fmt.Sprintf("%0*d", otpLength, n.Int64())
	if err := s.repo.CreateOTP(ctx, phone, purpose, s.otpHash(phone, purpose, code), now.Add(otpTTL)); err != nil {
		return "", err
	}
	if wa != nil {
		err := wa.SendAuthCode("91"+phone, s.otp.Template, s.otp.Language, code)
		if err == nil {
			return channelWhatsApp, nil
		}
		if !s.otp.SMS.Enabled() {
			log.Printf("whatsapp otp failed: %v", err)
			return "", fmt.Errorf("send whatsapp otp: %w", err)
		}
		log.Printf("whatsapp otp failed, sending by sms instead: %v", err)
	}
	if err := s.otp.SMS.SendOTP(phone, code); err != nil {
		log.Printf("sms otp failed: %v", err)
		return "", fmt.Errorf("send sms otp: %w", err)
	}
	return channelSMS, nil
}

// checkOTP uses up the latest live code for phone if code matches it. Each
// try counts against the code's attempt limit, right or wrong.
func (s *Service) checkOTP(ctx context.Context, phone, purpose, code string) error {
	if !s.otp.enabled() {
		return errOTPNotConfigured
	}
	bad := fmt.Errorf("%w: invalid or expired code", apperr.ErrUnauthorized)
	code = strings.TrimSpace(code)
	if len(code) != otpLength {
		return bad
	}
	id, hash, err := s.repo.TakeOTPAttempt(ctx, phone, purpose, otpMaxAttempts)
	if err != nil {
		return bad
	}
	if !hmac.Equal([]byte(hash), []byte(s.otpHash(phone, purpose, code))) {
		return bad
	}
	if used, err := s.repo.MarkOTPUsed(ctx, id); err != nil {
		return err
	} else if !used {
		return bad // the same code was just used by a parallel request
	}
	return nil
}

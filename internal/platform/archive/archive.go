// Package archive keeps a copy of every issued document (TC, marksheet,
// report card) in S3, exactly as it was issued, with a row in
// issued_documents saying who issued it and when. Re-issuing an identical
// PDF (same sha256) is recorded once.
package archive

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	KindTC         = "tc"
	KindMarksheet  = "marksheet"
	KindReportCard = "report_card"
)

type Archiver struct {
	pool    *pgxpool.Pool
	storage *storage.Client
}

func New(pool *pgxpool.Pool, s *storage.Client) *Archiver {
	return &Archiver{pool: pool, storage: s}
}

// Save stores pdf for studentID. It never fails the caller's request: the
// document has already been made, so a storage problem is logged instead.
func (a *Archiver) Save(ctx context.Context, studentID uuid.UUID, kind, title string, pdf []byte) {
	if a == nil || !a.storage.Enabled() || len(pdf) == 0 {
		return
	}
	if err := a.save(ctx, studentID, kind, title, pdf); err != nil {
		log.Printf("archive %s for student %s: %v", kind, studentID, err)
	}
}

func (a *Archiver) save(ctx context.Context, studentID uuid.UUID, kind, title string, pdf []byte) error {
	sum := sha256.Sum256(pdf)
	hash := hex.EncodeToString(sum[:])

	var exists bool
	if err := a.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM issued_documents WHERE student_id=$1 AND kind=$2 AND sha256=$3)`,
		studentID, kind, hash).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	var schoolID uuid.UUID
	var name string
	if err := a.pool.QueryRow(ctx, `SELECT school_id, TRIM(first_name || ' ' || last_name) FROM students WHERE id=$1`, studentID).
		Scan(&schoolID, &name); err != nil {
		return fmt.Errorf("student: %w", err)
	}
	key := fmt.Sprintf("archive/%s/%s/%s/%s-%s.pdf", schoolID, studentID, kind, time.Now().UTC().Format("20060102-150405"), hash[:12])
	if err := a.storage.Upload(key, "application/pdf", pdf); err != nil {
		return err
	}
	// Issued by the logged-in user (downloads are GETs, which carry no
	// audit actor, so read the token's claims).
	var issuedBy *uuid.UUID
	if claims := httpx.ClaimsFromContext(ctx); claims != nil {
		if id, err := uuid.Parse(claims.Sub); err == nil {
			issuedBy = &id
		}
	}
	_, err := a.pool.Exec(ctx, `
		INSERT INTO issued_documents (school_id, student_id, student_name, kind, title, s3_key, sha256, size_bytes, issued_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (student_id, kind, sha256) DO NOTHING`,
		schoolID, studentID, name, kind, title, key, hash, len(pdf), issuedBy)
	return err
}

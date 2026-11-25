package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func validEmail(e string) bool {
	_, err := mail.ParseAddress(e)
	return err == nil && !strings.ContainsAny(e, " <>")
}

func (s *Server) registerInstitute(w http.ResponseWriter, r *http.Request) {
	var in struct {
		InstituteName string `json:"institute_name"`
		FullName      string `json:"full_name"`
		Email         string `json:"email"`
		Password      string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if strings.TrimSpace(in.InstituteName) == "" || strings.TrimSpace(in.FullName) == "" ||
		!validEmail(in.Email) || len(in.Password) < 8 || len([]byte(in.Password)) > 72 {
		fail(w, 400, "school name, your name, a valid email and a password between 8 and 72 bytes are required")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 500, "something went wrong on our side")
		return
	}
	var c Claims
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(r.Context(), `INSERT INTO institutes (name) VALUES ($1) RETURNING id::text`,
			strings.TrimSpace(in.InstituteName)).Scan(&c.InstituteID); err != nil {
			return err
		}
		if err := tx.QueryRow(r.Context(),
			`INSERT INTO users (email, password_hash, full_name) VALUES ($1,$2,$3) RETURNING id::text`,
			in.Email, string(hash), strings.TrimSpace(in.FullName)).Scan(&c.UserID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(),
			`INSERT INTO institute_members (institute_id, user_id, role) VALUES ($1,$2,'owner')`,
			c.InstituteID, c.UserID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `INSERT INTO subscriptions (institute_id) VALUES ($1)`, c.InstituteID); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `INSERT INTO grade_bands (institute_id, min_score, grade, remark)
			SELECT $1, v.m, v.g, v.r FROM (VALUES (70,'A','Excellent'),(60,'B','Very good'),(50,'C','Good'),
			(45,'D','Fair'),(40,'E','Pass'),(0,'F','Fail')) AS v(m,g,r)`, c.InstituteID)
		return err
	})
	if err != nil {
		dbFail(w, err)
		return
	}
	c.Role = "owner"
	c.TokenVersion = 1
	send(w, 201, map[string]string{"token": s.sign(c), "role": c.Role})
}

func (s *Server) tooManyFails(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-15 * time.Minute)
	kept := s.fails[ip][:0]
	for _, t := range s.fails[ip] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	s.fails[ip] = kept
	return len(kept) >= 10
}

func (s *Server) noteFail(ip string) {
	s.mu.Lock()
	s.fails[ip] = append(s.fails[ip], time.Now())
	s.mu.Unlock()
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		InstituteID string `json:"institute_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = host
	}
	if s.tooManyFails(ip) {
		fail(w, 429, "too many failed attempts, try again in 15 minutes")
		return
	}
	if strings.TrimSpace(in.Email) == "" || in.Password == "" || len([]byte(in.Password)) > 72 {
		s.noteFail(ip)
		fail(w, 401, "wrong email or password")
		return
	}
	var c Claims
	var hash string
	var tokenVersion int64
	err := s.db.QueryRow(r.Context(), `SELECT u.id::text, u.password_hash, m.institute_id::text, m.role, u.token_version
		FROM users u JOIN institute_members m ON m.user_id = u.id
		JOIN institutes i ON i.id = m.institute_id
		WHERE u.email = $1 AND u.is_active AND i.status='active'
		  AND ($2 = '' OR m.institute_id::text = $2)
		ORDER BY m.id LIMIT 1`, strings.ToLower(strings.TrimSpace(in.Email)), in.InstituteID).
		Scan(&c.UserID, &hash, &c.InstituteID, &c.Role, &tokenVersion)
	if err != nil {
		// Keep unknown accounts indistinguishable from wrong passwords, but do not
		// disguise a database outage as a credential failure.
		if err != pgx.ErrNoRows {
			dbFail(w, err)
			return
		}
		bcrypt.CompareHashAndPassword([]byte("$2a$10$7EqJtq98hPqEX7fNZaFWoOhi5BNoUgCz1vGnBDWMiCcSMAeRXgZ.W"), []byte(in.Password))
		s.noteFail(ip)
		fail(w, 401, "wrong email or password")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		s.noteFail(ip)
		fail(w, 401, "wrong email or password")
		return
	}
	c.TokenVersion = tokenVersion
	send(w, 200, map[string]string{"token": s.sign(c), "role": c.Role})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, c Claims) {
	var name, email, school, logoURL, motto, address, phone, contactEmail, website, primaryColor, secondaryColor string
	err := s.db.QueryRow(r.Context(), `SELECT u.full_name, COALESCE(u.email,''), i.name,
        COALESCE(i.logo_url,''), COALESCE(i.motto,''), COALESCE(i.address,''), COALESCE(i.phone,''),
        COALESCE(i.contact_email,''), COALESCE(i.website,''), COALESCE(i.primary_color,'#173653'),
        COALESCE(i.secondary_color,'#168c8c')
        FROM users u JOIN institutes i ON i.id = $2::uuid WHERE u.id = $1::uuid`, c.UserID, c.InstituteID).
		Scan(&name, &email, &school, &logoURL, &motto, &address, &phone, &contactEmail, &website, &primaryColor, &secondaryColor)
	if err != nil {
		dbFail(w, err)
		return
	}
	send(w, 200, map[string]string{"full_name": name, "email": email, "institute_name": school,
		"institute_id": c.InstituteID, "role": c.Role, "logo_url": logoURL, "motto": motto,
		"address": address, "phone": phone, "contact_email": contactEmail, "website": website,
		"primary_color": primaryColor, "secondary_color": secondaryColor})
}

// createUser adds a staff, teacher, parent or student login to the caller's school.
func (s *Server) createUser(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if !has([]string{"admin", "bursar", "teacher", "parent", "student"}, in.Role) ||
		(in.Role == "admin" && c.Role != "owner") {
		fail(w, 400, "role must be bursar, teacher, parent or student (only the owner can add an admin)")
		return
	}
	if strings.TrimSpace(in.FullName) == "" || !validEmail(in.Email) || len(in.Password) < 8 || len([]byte(in.Password)) > 72 {
		fail(w, 400, "name, a valid email and a password between 8 and 72 bytes are required")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 500, "could not securely create the account")
		return
	}
	var id string
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(r.Context(),
			`INSERT INTO users (email, password_hash, full_name) VALUES ($1,$2,$3) RETURNING id::text`,
			in.Email, string(hash), strings.TrimSpace(in.FullName)).Scan(&id); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(),
			`INSERT INTO institute_members (institute_id, user_id, role) VALUES ($1,$2,$3)`, c.InstituteID, id, in.Role)
		return err
	})
	if err != nil {
		dbFail(w, err)
		return
	}
	send(w, 201, map[string]string{"id": id})
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT u.id::text AS id, u.full_name, u.email, m.role, u.is_active FROM institute_members m
		JOIN users u ON u.id = m.user_id WHERE m.institute_id = $1 ORDER BY m.role, u.full_name`, c.InstituteID)
}

func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if !decode(w, r, &in) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if validEmail(email) {
		var instituteID, userID string
		err := s.db.QueryRow(r.Context(), `SELECT m.institute_id::text,u.id::text
			FROM users u JOIN institute_members m ON m.user_id=u.id
			WHERE u.email=$1 AND u.is_active LIMIT 1`, email).Scan(&instituteID, &userID)
		if err == nil {
			raw := make([]byte, 32)
			if _, err = rand.Read(raw); err == nil {
				token := hex.EncodeToString(raw)
				hash := sha256.Sum256([]byte(token))
				_, err = s.db.Exec(r.Context(), `INSERT INTO password_reset_tokens
					(institute_id,user_id,token_hash,expires_at)
					VALUES ($1::uuid,$2::uuid,$3,NOW()+INTERVAL '30 minutes')`, instituteID, userID, hex.EncodeToString(hash[:]))
				if err == nil {
					// Delivery is intentionally abstracted; configure SMTP in production.
					fmtTokenHint(w, token)
				}
			}
		}
	}
	send(w, 200, map[string]string{"message": "If an account exists for that email, reset instructions have been generated."})
}

func fmtTokenHint(w http.ResponseWriter, token string) { _ = w; _ = token }

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Password) < 8 || len([]byte(in.Password)) > 72 {
		fail(w, 400, "password must contain 8 to 72 characters")
		return
	}
	hash := sha256.Sum256([]byte(in.Token))
	tokenHash := hex.EncodeToString(hash[:])
	var userID string
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(r.Context(), `UPDATE password_reset_tokens SET used_at=NOW()
			WHERE token_hash=$1 AND used_at IS NULL AND expires_at>NOW()
			RETURNING user_id::text`, tokenHash).Scan(&userID); err != nil {
			return err
		}
		newHash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `UPDATE users SET password_hash=$1,token_version=token_version+1 WHERE id=$2::uuid`, string(newHash), userID)
		return err
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			fail(w, 400, "reset link is invalid or expired")
			return
		}
		dbFail(w, err)
		return
	}
	send(w, 200, map[string]string{"message": "Password updated successfully"})
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		Role     string `json:"role"`
		IsActive *bool  `json:"is_active"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Role != "" && !has([]string{"admin", "bursar", "teacher", "parent"}, in.Role) {
		fail(w, 400, "invalid role")
		return
	}
	if in.Role == "admin" && c.Role != "owner" {
		fail(w, 403, "only the owner can assign the admin role")
		return
	}

	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var targetRole string
		if err := tx.QueryRow(r.Context(), `SELECT role FROM institute_members
			WHERE institute_id=$1 AND user_id::text=$2 FOR UPDATE`, c.InstituteID, r.PathValue("id")).Scan(&targetRole); err != nil {
			return err
		}
		if targetRole == "owner" {
			return errOwnerProtected
		}
		if in.Role != "" {
			if _, err := tx.Exec(r.Context(), `UPDATE institute_members SET role=$3
				WHERE institute_id=$1 AND user_id::text=$2`, c.InstituteID, r.PathValue("id"), in.Role); err != nil {
				return err
			}
		}
		if in.IsActive != nil {
			if _, err := tx.Exec(r.Context(), `UPDATE users SET is_active=$2,token_version=token_version+1
				WHERE id::text=$1`, r.PathValue("id"), *in.IsActive); err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, errOwnerProtected) {
		fail(w, 403, "the school owner account cannot be changed")
		return
	}
	if err != nil {
		if err == pgx.ErrNoRows {
			fail(w, 404, "account not found")
			return
		}
		dbFail(w, err)
		return
	}
	s.sendAudit(r, c, "account.updated", r.PathValue("id"), map[string]any{"role": in.Role, "is_active": in.IsActive})
	send(w, 200, map[string]bool{"ok": true})
}

var errOwnerProtected = fmt.Errorf("owner account protected")

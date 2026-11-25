package api

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type studentIn struct {
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	AdmissionNumber string `json:"admission_number"`
	DateOfBirth     string `json:"date_of_birth"`
	Status          string `json:"status"`
}

func (in *studentIn) ok() bool {
	in.FirstName, in.LastName = strings.TrimSpace(in.FirstName), strings.TrimSpace(in.LastName)
	in.AdmissionNumber = strings.TrimSpace(in.AdmissionNumber)
	if in.Status == "" {
		in.Status = "active"
	}
	return in.FirstName != "" && in.LastName != "" && in.AdmissionNumber != "" &&
		(in.Status == "active" || in.Status == "inactive")
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *Server) listStudents(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT id::text AS id, first_name, last_name, admission_number, date_of_birth, status
		FROM students WHERE institute_id = $1 ORDER BY last_name, first_name`, c.InstituteID)
}

func (s *Server) createStudent(w http.ResponseWriter, r *http.Request, c Claims) {
	var in studentIn
	if !decode(w, r, &in) {
		return
	}
	if !in.ok() {
		fail(w, 400, "first name, last name and admission number are required")
		return
	}
	s.create(w, r, `INSERT INTO students (institute_id, first_name, last_name, admission_number, date_of_birth, status)
		VALUES ($1,$2,$3,$4,$5::date,$6) RETURNING id::text`,
		c.InstituteID, in.FirstName, in.LastName, in.AdmissionNumber, nilIfEmpty(in.DateOfBirth), in.Status)
}

func (s *Server) getStudent(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT id::text AS id, first_name, last_name, admission_number, date_of_birth, status
		FROM students WHERE institute_id = $1 AND id::text = $2`, c.InstituteID, r.PathValue("id"))
}

func (s *Server) updateStudent(w http.ResponseWriter, r *http.Request, c Claims) {
	var in studentIn
	if !decode(w, r, &in) {
		return
	}
	if !in.ok() {
		fail(w, 400, "first name, last name and admission number are required")
		return
	}
	tag, err := s.db.Exec(r.Context(), `UPDATE students SET first_name=$3, last_name=$4, admission_number=$5,
		date_of_birth=$6::date, status=$7 WHERE institute_id=$1 AND id::text=$2`,
		c.InstituteID, r.PathValue("id"), in.FirstName, in.LastName, in.AdmissionNumber, nilIfEmpty(in.DateOfBirth), in.Status)
	if err != nil {
		dbFail(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "student not found")
		return
	}
	s.sendAudit(r, c, "student.updated", r.PathValue("id"), map[string]any{"status": in.Status})
	send(w, 200, map[string]string{"status": "updated"})
}

func (s *Server) deleteStudent(w http.ResponseWriter, r *http.Request, c Claims) {
	tag, err := s.db.Exec(r.Context(), `UPDATE students SET status='inactive'
		WHERE institute_id=$1 AND id::text=$2 AND status!='inactive'`,
		c.InstituteID, r.PathValue("id"))
	if err != nil {
		dbFail(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "student not found or already inactive")
		return
	}
	s.sendAudit(r, c, "student.archived", r.PathValue("id"), map[string]any{"status": "inactive"})
	send(w, 200, map[string]string{"status": "inactive"})
}

// createGuardian records a parent, links their children, and optionally creates their login.
func (s *Server) createGuardian(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		FullName   string   `json:"full_name"`
		Phone      string   `json:"phone"`
		Email      string   `json:"email"`
		Password   string   `json:"password"`
		StudentIDs []string `json:"student_ids"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if strings.TrimSpace(in.FullName) == "" || (in.Email != "" && (!validEmail(in.Email) || len(in.Password) < 8 || len([]byte(in.Password)) > 72)) {
		fail(w, 400, "name is required; a login needs a valid email and a password between 8 and 72 bytes")
		return
	}
	var id string
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var userID any
		if in.Email != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			var uid string
			if err := tx.QueryRow(r.Context(),
				`INSERT INTO users (email, password_hash, full_name) VALUES ($1,$2,$3) RETURNING id::text`,
				in.Email, string(hash), strings.TrimSpace(in.FullName)).Scan(&uid); err != nil {
				return err
			}
			if _, err := tx.Exec(r.Context(),
				`INSERT INTO institute_members (institute_id, user_id, role) VALUES ($1,$2,'parent')`, c.InstituteID, uid); err != nil {
				return err
			}
			userID = uid
		}
		if err := tx.QueryRow(r.Context(), `INSERT INTO guardians (institute_id, user_id, full_name, phone)
			VALUES ($1,$2::uuid,$3,$4) RETURNING id::text`, c.InstituteID, userID, strings.TrimSpace(in.FullName), nilIfEmpty(in.Phone)).Scan(&id); err != nil {
			return err
		}
		for _, sid := range in.StudentIDs {
			if _, err := tx.Exec(r.Context(), `INSERT INTO student_guardians (institute_id, student_id, guardian_id)
				VALUES ($1,$2::uuid,$3::uuid)`, c.InstituteID, sid, id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		dbFail(w, err)
		return
	}
	send(w, 201, map[string]string{"id": id})
}

func (s *Server) listGuardians(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT g.id::text AS id, g.full_name, g.phone,
		COALESCE(string_agg(st.first_name || ' ' || st.last_name, ', '), '') AS children
		FROM guardians g LEFT JOIN student_guardians sg ON sg.guardian_id = g.id
		LEFT JOIN students st ON st.id = sg.student_id
		WHERE g.institute_id = $1 GROUP BY g.id ORDER BY g.full_name`, c.InstituteID)
}

// --- simple setup lists: classes, subjects, sessions, terms ---

func nameOnly(w http.ResponseWriter, r *http.Request) (string, bool) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return "", false
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		fail(w, 400, "name is required")
		return "", false
	}
	return in.Name, true
}

func (s *Server) listClasses(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT id::text AS id, name FROM classes WHERE institute_id=$1 ORDER BY name`, c.InstituteID)
}
func (s *Server) createClass(w http.ResponseWriter, r *http.Request, c Claims) {
	if n, ok := nameOnly(w, r); ok {
		s.create(w, r, `INSERT INTO classes (institute_id, name) VALUES ($1,$2) RETURNING id::text`, c.InstituteID, n)
	}
}
func (s *Server) listSubjects(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT id::text AS id, name FROM subjects WHERE institute_id=$1 ORDER BY name`, c.InstituteID)
}
func (s *Server) createSubject(w http.ResponseWriter, r *http.Request, c Claims) {
	if n, ok := nameOnly(w, r); ok {
		s.create(w, r, `INSERT INTO subjects (institute_id, name) VALUES ($1,$2) RETURNING id::text`, c.InstituteID, n)
	}
}
func (s *Server) listSessions(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT id::text AS id, name FROM academic_sessions WHERE institute_id=$1 ORDER BY name DESC`, c.InstituteID)
}
func (s *Server) createSession(w http.ResponseWriter, r *http.Request, c Claims) {
	if n, ok := nameOnly(w, r); ok {
		s.create(w, r, `INSERT INTO academic_sessions (institute_id, name) VALUES ($1,$2) RETURNING id::text`, c.InstituteID, n)
	}
}
func (s *Server) listTerms(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT t.id::text AS id, t.name, t.starts_on, t.ends_on, a.name AS session
		FROM terms t JOIN academic_sessions a ON a.id = t.session_id
		WHERE t.institute_id=$1 ORDER BY t.starts_on DESC NULLS LAST, t.name`, c.InstituteID)
}
func (s *Server) createTerm(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		SessionID string `json:"session_id"`
		Name      string `json:"name"`
		StartsOn  string `json:"starts_on"`
		EndsOn    string `json:"ends_on"`
	}
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Name) == "" || in.SessionID == "" {
		fail(w, 400, "session and term name are required")
		return
	}
	s.create(w, r, `INSERT INTO terms (institute_id, session_id, name, starts_on, ends_on)
		VALUES ($1,$2::uuid,$3,$4::date,$5::date) RETURNING id::text`,
		c.InstituteID, in.SessionID, strings.TrimSpace(in.Name), nilIfEmpty(in.StartsOn), nilIfEmpty(in.EndsOn))
}

func (s *Server) createEnrollment(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		StudentID string `json:"student_id"`
		ClassID   string `json:"class_id"`
		TermID    string `json:"term_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	// Ownership is checked first, so another school's ids look exactly like missing ones.
	s.create(w, r, `INSERT INTO enrollments (institute_id, student_id, class_id, term_id)
		SELECT $1, st.id, cl.id, t.id FROM students st, classes cl, terms t
		WHERE st.institute_id=$1 AND st.id::text=$2 AND cl.institute_id=$1 AND cl.id::text=$3
		AND t.institute_id=$1 AND t.id::text=$4 RETURNING id::text`, c.InstituteID, in.StudentID, in.ClassID, in.TermID)
}

func (s *Server) createAssignment(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		UserID    string `json:"user_id"`
		ClassID   string `json:"class_id"`
		SubjectID string `json:"subject_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	s.create(w, r, `INSERT INTO teacher_assignments (institute_id, user_id, class_id, subject_id)
		SELECT $1, m.user_id, $3::uuid, $4::uuid FROM institute_members m
		WHERE m.institute_id=$1 AND m.user_id=$2::uuid AND m.role='teacher' RETURNING id::text`,
		c.InstituteID, in.UserID, in.ClassID, nilIfEmpty(in.SubjectID))
}

func (s *Server) listAssignments(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT ta.id::text AS id, u.full_name AS teacher, cl.name AS class, COALESCE(sb.name,'All subjects') AS subject,
		cl.id::text AS class_id, sb.id::text AS subject_id
		FROM teacher_assignments ta JOIN users u ON u.id = ta.user_id
		JOIN classes cl ON cl.id = ta.class_id LEFT JOIN subjects sb ON sb.id = ta.subject_id
		WHERE ta.institute_id=$1 AND ($2 IN ('owner','admin') OR ta.user_id::text=$3)
		ORDER BY cl.name`, c.InstituteID, c.Role, c.UserID)
}

// teacherCan: owners and admins always; teachers only for classes (and subjects) assigned to them.
func (s *Server) teacherCan(r *http.Request, c Claims, classID, subjectID string) bool {
	if c.Role == "owner" || c.Role == "admin" {
		return true
	}
	var ok bool
	err := s.db.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM teacher_assignments
		WHERE institute_id=$1 AND user_id=$2::uuid AND class_id::text=$3
		AND ($4 = '' OR subject_id IS NULL OR subject_id::text=$4))`, c.InstituteID, c.UserID, classID, subjectID).Scan(&ok)
	return err == nil && ok
}

func (s *Server) classRoster(w http.ResponseWriter, r *http.Request, c Claims) {
	if !s.teacherCan(r, c, r.PathValue("id"), "") {
		fail(w, 403, "this class is not assigned to you")
		return
	}
	s.list(w, r, `SELECT st.id::text AS id, st.first_name, st.last_name, st.admission_number
		FROM enrollments e JOIN students st ON st.id = e.student_id
		WHERE e.institute_id=$1 AND e.class_id::text=$2 AND e.term_id::text=$3
		ORDER BY st.last_name, st.first_name`, c.InstituteID, r.PathValue("id"), r.URL.Query().Get("term_id"))
}

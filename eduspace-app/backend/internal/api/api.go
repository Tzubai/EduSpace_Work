package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	db     *pgxpool.Pool
	secret []byte
	mu     sync.Mutex
	fails  map[string][]time.Time // failed logins per IP
}

func New(db *pgxpool.Pool, secret string) *Server {
	return &Server{db: db, secret: []byte(secret), fails: map[string][]time.Time{}}
}

// Claims is what a login token carries: who, which school, which role.
type Claims struct {
	UserID       string `json:"u"`
	InstituteID  string `json:"i"`
	Role         string `json:"r"`
	Exp          int64  `json:"e"`
	TokenVersion int64  `json:"v"`
}

func (s *Server) sign(c Claims) string {
	c.Exp = time.Now().Add(12 * time.Hour).Unix()
	b, _ := json.Marshal(c)
	p := base64.RawURLEncoding.EncodeToString(b)
	m := hmac.New(sha256.New, s.secret)
	m.Write([]byte(p))
	return p + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (s *Server) verify(tok string) (Claims, bool) {
	var c Claims
	parts := strings.Split(tok, ".")
	if len(parts) != 2 {
		return c, false
	}
	m := hmac.New(sha256.New, s.secret)
	m.Write([]byte(parts[0]))
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(sig, m.Sum(nil)) {
		return c, false
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(b, &c) != nil || c.Exp <= time.Now().Unix() ||
		c.UserID == "" || c.InstituteID == "" || c.TokenVersion < 1 || !has([]string{"owner", "admin", "bursar", "teacher", "parent", "student"}, c.Role) {
		return c, false
	}
	return c, true
}

type handler func(w http.ResponseWriter, r *http.Request, c Claims)

// guard runs a handler only for a logged-in member of a school with an allowed role.
// The school always comes from the signed token, never from the request.
func (s *Server) guard(h handler, roles ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(auth, "Bearer ") {
			fail(w, 401, "please log in again")
			return
		}
		c, ok := s.verify(strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")))
		if !ok {
			fail(w, 401, "please log in again")
			return
		}
		if len(roles) > 0 && !has(roles, c.Role) {
			fail(w, 403, "your role is not allowed to do this")
			return
		}
		var still bool
		err := s.db.QueryRow(r.Context(), `SELECT EXISTS (
			SELECT 1 FROM institute_members m
			JOIN users u ON u.id=m.user_id
			JOIN institutes i ON i.id=m.institute_id
			WHERE m.institute_id=$1 AND m.user_id=$2 AND m.role=$3
			  AND u.is_active AND i.status='active'
			  AND u.token_version=$4)`,
			c.InstituteID, c.UserID, c.Role, c.TokenVersion).Scan(&still)
		if err != nil {
			fail(w, 500, "could not verify your account; please try again")
			return
		}
		if !still {
			fail(w, 401, "please log in again")
			return
		}
		h(w, r, c)
	}
}

func has(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func send(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, msg string) {
	send(w, status, map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		fail(w, 400, "invalid request body")
		return false
	}
	// Reject a second JSON value or trailing garbage instead of silently ignoring it.
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		fail(w, 400, "request body must contain exactly one JSON value")
		return false
	}
	return true
}

// dbFail turns database errors into plain messages without leaking details.
func dbFail(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 400, "a linked record was not found in your school")
		return
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		switch pe.Code {
		case "23505":
			fail(w, 409, "that record already exists")
			return
		case "23503":
			fail(w, 400, "a linked record was not found in your school, or this record is still in use")
			return
		case "23514", "22P02", "22007", "22008":
			fail(w, 400, "one of the values is not valid")
			return
		}
	}
	fail(w, 500, "something went wrong on our side")
}

// list runs a SELECT and returns every row as a JSON object.
func (s *Server) list(w http.ResponseWriter, r *http.Request, sql string, args ...any) {
	rs, err := s.db.Query(r.Context(), sql, args...)
	if err != nil {
		dbFail(w, err)
		return
	}
	defer rs.Close()
	out := []map[string]any{}
	for rs.Next() {
		vals, err := rs.Values()
		if err != nil {
			dbFail(w, err)
			return
		}
		row := map[string]any{}
		for i, f := range rs.FieldDescriptions() {
			row[f.Name] = vals[i]
		}
		out = append(out, row)
	}
	if rs.Err() != nil {
		dbFail(w, rs.Err())
		return
	}
	send(w, 200, out)
}

// create runs an INSERT ... RETURNING id::text and replies 201.
func (s *Server) create(w http.ResponseWriter, r *http.Request, sql string, args ...any) {
	var id string
	if err := s.db.QueryRow(r.Context(), sql, args...).Scan(&id); err != nil {
		dbFail(w, err)
		return
	}
	send(w, 201, map[string]string{"id": id})
}

func (s *Server) sendAudit(r *http.Request, c Claims, action, entityID string, metadata map[string]any) {
	_, _ = s.db.Exec(r.Context(), `INSERT INTO audit_events (institute_id,actor_user_id,action,entity_id,metadata)
		VALUES ($1::uuid,$2::uuid,$3,$4,$5::jsonb)`, c.InstituteID, c.UserID, action, entityID, mustJSON(metadata))
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	staff := []string{"owner", "admin"}
	money := []string{"owner", "admin", "bursar"}
	teach := []string{"owner", "admin", "teacher"}
	all := []string{"owner", "admin", "bursar", "teacher", "parent", "student"}
	v := "/api/v1"

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := s.db.Ping(r.Context()); err != nil {
			fail(w, 503, "database unavailable")
			return
		}
		send(w, 200, map[string]string{"status": "ok", "database": "ready"})
	})

	mux.HandleFunc("POST "+v+"/auth/register-institute", s.registerInstitute)
	mux.HandleFunc("POST "+v+"/auth/login", s.login)
	mux.HandleFunc("GET "+v+"/auth/me", s.guard(s.me))
	mux.HandleFunc("POST "+v+"/users", s.guard(s.createUser, staff...))
	mux.HandleFunc("GET "+v+"/users", s.guard(s.listUsers, staff...))
	mux.HandleFunc("PUT "+v+"/users/{id}", s.guard(s.updateUser, staff...))

	mux.HandleFunc("GET "+v+"/students", s.guard(s.listStudents, money...))
	mux.HandleFunc("POST "+v+"/students", s.guard(s.createStudent, staff...))
	mux.HandleFunc("GET "+v+"/students/{id}", s.guard(s.getStudent, staff...))
	mux.HandleFunc("PUT "+v+"/students/{id}", s.guard(s.updateStudent, staff...))
	mux.HandleFunc("DELETE "+v+"/students/{id}", s.guard(s.deleteStudent, staff...))
	mux.HandleFunc("POST "+v+"/guardians", s.guard(s.createGuardian, staff...))
	mux.HandleFunc("GET "+v+"/guardians", s.guard(s.listGuardians, staff...))

	mux.HandleFunc("GET "+v+"/classes", s.guard(s.listClasses, teach...))
	mux.HandleFunc("POST "+v+"/classes", s.guard(s.createClass, staff...))
	mux.HandleFunc("GET "+v+"/subjects", s.guard(s.listSubjects, teach...))
	mux.HandleFunc("POST "+v+"/subjects", s.guard(s.createSubject, staff...))
	mux.HandleFunc("GET "+v+"/sessions", s.guard(s.listSessions, append(teach, "bursar")...))
	mux.HandleFunc("POST "+v+"/sessions", s.guard(s.createSession, staff...))
	mux.HandleFunc("GET "+v+"/terms", s.guard(s.listTerms, append(teach, "bursar", "parent")...))
	mux.HandleFunc("POST "+v+"/terms", s.guard(s.createTerm, staff...))
	mux.HandleFunc("POST "+v+"/enrollments", s.guard(s.createEnrollment, staff...))
	mux.HandleFunc("GET "+v+"/classes/{id}/students", s.guard(s.classRoster, teach...))
	mux.HandleFunc("POST "+v+"/teacher-assignments", s.guard(s.createAssignment, staff...))
	mux.HandleFunc("GET "+v+"/teacher-assignments", s.guard(s.listAssignments, teach...))

	mux.HandleFunc("GET "+v+"/attendance", s.guard(s.getAttendance, teach...))
	mux.HandleFunc("POST "+v+"/attendance", s.guard(s.saveAttendance, teach...))

	mux.HandleFunc("GET "+v+"/fees/invoices", s.guard(s.listInvoices, money...))
	mux.HandleFunc("POST "+v+"/fees/invoices", s.guard(s.createInvoice, money...))
	mux.HandleFunc("POST "+v+"/fees/invoices/bulk", s.guard(s.createInvoicesForClass, money...))
	mux.HandleFunc("GET "+v+"/fees/payments", s.guard(s.listPayments, money...))
	mux.HandleFunc("POST "+v+"/fees/payments", s.guard(s.recordPayment, "owner", "bursar"))

	mux.HandleFunc("GET "+v+"/results/assessments", s.guard(s.listAssessments, teach...))
	mux.HandleFunc("POST "+v+"/results/assessments", s.guard(s.createAssessment, teach...))
	mux.HandleFunc("GET "+v+"/results/grades", s.guard(s.getGrades, teach...))
	mux.HandleFunc("POST "+v+"/results/grades", s.guard(s.saveGrades, teach...))
	mux.HandleFunc("GET "+v+"/results/report-card/{studentId}", s.guard(s.reportCard, "owner", "admin", "parent"))
	mux.HandleFunc("GET "+v+"/grading-scale", s.guard(s.getScale, all...))
	mux.HandleFunc("PUT "+v+"/grading-scale", s.guard(s.putScale, staff...))

	mux.HandleFunc("GET "+v+"/announcements", s.guard(s.listAnnouncements, all...))
	mux.HandleFunc("POST "+v+"/announcements", s.guard(s.createAnnouncement, staff...))
	mux.HandleFunc("GET "+v+"/dashboard", s.guard(s.dashboard, money...))
	mux.HandleFunc("GET "+v+"/parent/children", s.guard(s.parentChildren, "parent"))
	mux.HandleFunc("GET "+v+"/parent/children/{studentId}/summary", s.guard(s.parentStudentSummary, "parent"))
	mux.HandleFunc("GET "+v+"/parent/children/{studentId}/invoices", s.guard(s.parentInvoices, "parent"))
	mux.HandleFunc("GET "+v+"/parent/children/{studentId}/payments", s.guard(s.parentPayments, "parent"))
	mux.HandleFunc("GET "+v+"/parent/children/{studentId}/attendance", s.guard(s.parentAttendance, "parent"))
	mux.HandleFunc("GET "+v+"/parent/children/{studentId}/timetable", s.guard(s.parentTimetable, "parent"))
	mux.HandleFunc("GET "+v+"/parent/children/{studentId}/results", s.guard(s.parentResults, "parent"))
	mux.HandleFunc("GET "+v+"/subscriptions", s.guard(s.subscription, "owner"))
	mux.HandleFunc("GET "+v+"/audit-events", s.guard(s.listAuditEvents, "owner", "admin"))

	mux.HandleFunc("POST "+v+"/auth/forgot-password", s.forgotPassword)
	mux.HandleFunc("POST "+v+"/auth/reset-password", s.resetPassword)

	mux.HandleFunc("GET "+v+"/admissions", s.guard(s.listAdmissions, staff...))
	mux.HandleFunc("POST "+v+"/admissions", s.guard(s.createAdmission, staff...))
	mux.HandleFunc("PUT "+v+"/admissions/{id}", s.guard(s.updateAdmission, staff...))
	mux.HandleFunc("GET "+v+"/timetable", s.guard(s.listTimetable, teach...))
	mux.HandleFunc("POST "+v+"/timetable", s.guard(s.createTimetable, staff...))
	mux.HandleFunc("GET "+v+"/staff-attendance", s.guard(s.listStaffAttendance, staff...))
	mux.HandleFunc("POST "+v+"/staff-attendance", s.guard(s.saveStaffAttendance, staff...))
	mux.HandleFunc("GET "+v+"/events", s.guard(s.listEvents, all...))
	mux.HandleFunc("POST "+v+"/events", s.guard(s.createEvent, staff...))
	mux.HandleFunc("GET "+v+"/parent-messages", s.guard(s.listParentMessages, staff...))
	mux.HandleFunc("POST "+v+"/parent-messages", s.guard(s.createParentMessage, staff...))
	mux.HandleFunc("POST "+v+"/parent-messages/request", s.guard(s.createParentRequest, "parent"))
	mux.HandleFunc("GET "+v+"/notifications", s.guard(s.listNotifications, all...))

	mux.HandleFunc("GET "+v+"/schemes", s.guard(s.listSchemes, teach...))
	mux.HandleFunc("POST "+v+"/schemes", s.guard(s.saveScheme, teach...))
	mux.HandleFunc("GET "+v+"/exams", s.guard(s.listExams, teach...))
	mux.HandleFunc("POST "+v+"/exams", s.guard(s.createExam, staff...))
	mux.HandleFunc("PUT "+v+"/results/remarks", s.guard(s.saveRemarks, teach...))
	mux.HandleFunc("POST "+v+"/students/promote", s.guard(s.promoteStudent, staff...))
	mux.HandleFunc("POST "+v+"/attendance/correct", s.guard(s.correctAttendance, staff...))
	mux.HandleFunc("GET "+v+"/attendance/monthly", s.guard(s.monthlyAttendance, teach...))

	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		files := http.FileServer(http.Dir(dir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat(filepath.Join(dir, filepath.Clean("/"+r.URL.Path))); err != nil {
				http.ServeFile(w, r, filepath.Join(dir, "index.html"))
				return
			}
			files.ServeHTTP(w, r)
		})
	}
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: https:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; font-src 'self' data:")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

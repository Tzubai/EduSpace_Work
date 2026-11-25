package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ---------- attendance ----------

func (s *Server) saveAttendance(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		ClassID string `json:"class_id"`
		TermID  string `json:"term_id"`
		Day     string `json:"day"`
		Records []struct {
			StudentID   string `json:"student_id"`
			Status      string `json:"status"`
			ArrivalTime string `json:"arrival_time"`
			Reason      string `json:"reason"`
		} `json:"records"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !s.teacherCan(r, c, in.ClassID, "") {
		fail(w, 403, "this class is not assigned to you")
		return
	}
	if in.Day == "" || len(in.Records) == 0 {
		fail(w, 400, "day and at least one record are required")
		return
	}
	for _, rec := range in.Records {
		if !has([]string{"present", "absent", "late", "excused"}, rec.Status) {
			fail(w, 400, "attendance status must be present, absent, late or excused")
			return
		}
	}
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		for _, rec := range in.Records {
			tag, err := tx.Exec(r.Context(), `INSERT INTO attendance (institute_id,enrollment_id,day,status,arrival_time,reason)
				SELECT institute_id,id,$5::date,$6,NULLIF($7,'')::time,NULLIF($8,'') FROM enrollments
				WHERE institute_id=$1 AND student_id::text=$2 AND class_id::text=$3 AND term_id::text=$4
				ON CONFLICT (enrollment_id,day) DO UPDATE SET status=EXCLUDED.status,
				 arrival_time=EXCLUDED.arrival_time,reason=EXCLUDED.reason`,
				c.InstituteID, rec.StudentID, in.ClassID, in.TermID, in.Day, rec.Status, rec.ArrivalTime, rec.Reason)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return fmt.Errorf("student %s is not enrolled in this class for this term", rec.StudentID)
			}
		}
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "not enrolled") {
			fail(w, 400, err.Error())
			return
		}
		dbFail(w, err)
		return
	}
	send(w, 200, map[string]string{"status": "saved"})
}

func (s *Server) getAttendance(w http.ResponseWriter, r *http.Request, c Claims) {
	q := r.URL.Query()
	if !s.teacherCan(r, c, q.Get("class_id"), "") {
		fail(w, 403, "this class is not assigned to you")
		return
	}
	s.list(w, r, `SELECT st.id::text AS student_id, st.first_name, st.last_name, COALESCE(a.status,'') AS status
		FROM enrollments e JOIN students st ON st.id = e.student_id
		LEFT JOIN attendance a ON a.enrollment_id = e.id AND a.day = $4::date
		WHERE e.institute_id=$1 AND e.class_id::text=$2 AND e.term_id::text=$3
		ORDER BY st.last_name, st.first_name`, c.InstituteID, q.Get("class_id"), q.Get("term_id"), q.Get("day"))
}

// ---------- fees ----------

func (s *Server) createInvoice(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		StudentID   string `json:"student_id"`
		TermID      string `json:"term_id"`
		Description string `json:"description"`
		AmountKobo  int64  `json:"amount_kobo"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Description == "" {
		in.Description = "School fees"
	}
	if in.AmountKobo <= 0 || strings.TrimSpace(in.StudentID) == "" || strings.TrimSpace(in.TermID) == "" {
		fail(w, 400, "student, term and an amount greater than zero are required")
		return
	}
	s.create(w, r, `INSERT INTO fee_invoices (institute_id, student_id, term_id, description, amount_kobo)
		VALUES ($1,$2::uuid,$3::uuid,$4,$5) RETURNING id::text`, c.InstituteID, in.StudentID, in.TermID, in.Description, in.AmountKobo)
}

// createInvoicesForClass bills every student enrolled in a class for a term.
func (s *Server) createInvoicesForClass(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		ClassID     string `json:"class_id"`
		TermID      string `json:"term_id"`
		Description string `json:"description"`
		AmountKobo  int64  `json:"amount_kobo"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Description == "" {
		in.Description = "School fees"
	}
	if in.AmountKobo <= 0 || strings.TrimSpace(in.ClassID) == "" || strings.TrimSpace(in.TermID) == "" {
		fail(w, 400, "class, term and an amount greater than zero are required")
		return
	}
	tag, err := s.db.Exec(r.Context(), `INSERT INTO fee_invoices (institute_id, student_id, term_id, description, amount_kobo)
		SELECT institute_id, student_id, term_id, $4, $5 FROM enrollments
		WHERE institute_id=$1 AND class_id::text=$2 AND term_id::text=$3
		ON CONFLICT (institute_id, student_id, term_id, description) DO NOTHING`,
		c.InstituteID, in.ClassID, in.TermID, in.Description, in.AmountKobo)
	if err != nil {
		dbFail(w, err)
		return
	}
	send(w, 201, map[string]int64{"invoices_created": tag.RowsAffected()})
}

func (s *Server) listInvoices(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT i.id::text AS id, st.first_name, st.last_name, st.admission_number, t.name AS term,
		i.description, i.amount_kobo, COALESCE(SUM(p.amount_kobo),0)::bigint AS paid_kobo,
		(i.amount_kobo - COALESCE(SUM(p.amount_kobo),0))::bigint AS balance_kobo, i.status
		FROM fee_invoices i JOIN students st ON st.id = i.student_id JOIN terms t ON t.id = i.term_id
		LEFT JOIN payments p ON p.invoice_id = i.id
		WHERE i.institute_id=$1 AND ($2 = '' OR i.term_id::text=$2)
		GROUP BY i.id, st.id, t.id ORDER BY st.last_name, st.first_name`, c.InstituteID, r.URL.Query().Get("term_id"))
}

func (s *Server) recordPayment(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		InvoiceID  string `json:"invoice_id"`
		AmountKobo int64  `json:"amount_kobo"`
		Method     string `json:"method"`
		Reference  string `json:"reference"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.AmountKobo <= 0 {
		fail(w, 400, "amount must be greater than zero")
		return
	}
	var payID string
	var balance int64
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var total, paid int64
		if err := tx.QueryRow(r.Context(), `SELECT amount_kobo FROM fee_invoices
			WHERE institute_id=$1 AND id::text=$2 FOR UPDATE`, c.InstituteID, in.InvoiceID).Scan(&total); err != nil {
			return err
		}
		if err := tx.QueryRow(r.Context(), `SELECT COALESCE(SUM(amount_kobo),0)::bigint FROM payments
			WHERE institute_id=$1 AND invoice_id::text=$2`, c.InstituteID, in.InvoiceID).Scan(&paid); err != nil {
			return err
		}
		if paid+in.AmountKobo > total {
			return fmt.Errorf("overpay: balance is only %d kobo", total-paid)
		}
		if err := tx.QueryRow(r.Context(), `INSERT INTO payments (institute_id, invoice_id, amount_kobo, method, reference, recorded_by)
			VALUES ($1,$2::uuid,$3,$4,$5,$6::uuid) RETURNING id::text`,
			c.InstituteID, in.InvoiceID, in.AmountKobo, in.Method, nilIfEmpty(in.Reference), c.UserID).Scan(&payID); err != nil {
			return err
		}
		balance = total - paid - in.AmountKobo
		status := "part-paid"
		if balance == 0 {
			status = "paid"
		}
		_, err := tx.Exec(r.Context(), `UPDATE fee_invoices SET status=$3 WHERE institute_id=$1 AND id::text=$2`,
			c.InstituteID, in.InvoiceID, status)
		return err
	})
	if err != nil {
		if strings.HasPrefix(err.Error(), "overpay") {
			fail(w, 400, "payment is more than the outstanding balance ("+err.Error()[9:]+")")
			return
		}
		if err == pgx.ErrNoRows {
			fail(w, 404, "invoice not found")
			return
		}
		dbFail(w, err)
		return
	}
	send(w, 201, map[string]any{"id": payID, "balance_kobo": balance})
}

func (s *Server) listPayments(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT p.id::text AS id, st.first_name, st.last_name, i.description, p.amount_kobo, p.method,
		p.reference, p.paid_at FROM payments p JOIN fee_invoices i ON i.id = p.invoice_id
		JOIN students st ON st.id = i.student_id
		WHERE p.institute_id=$1 AND ($2 = '' OR p.invoice_id::text=$2) ORDER BY p.paid_at DESC LIMIT 200`,
		c.InstituteID, r.URL.Query().Get("invoice_id"))
}

// ---------- assessments, grades, report card ----------

func (s *Server) createAssessment(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		ClassID        string  `json:"class_id"`
		SubjectID      string  `json:"subject_id"`
		TermID         string  `json:"term_id"`
		Title          string  `json:"title"`
		MaxScore       float64 `json:"max_score"`
		AssessmentType string  `json:"assessment_type"`
		Weight         float64 `json:"weight"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !s.teacherCan(r, c, in.ClassID, in.SubjectID) {
		fail(w, 403, "this class and subject are not assigned to you")
		return
	}
	if strings.TrimSpace(in.Title) == "" || in.MaxScore <= 0 {
		fail(w, 400, "title and a maximum score greater than zero are required")
		return
	}
	if in.AssessmentType == "" {
		in.AssessmentType = "ca1"
	}
	if !has([]string{"ca1", "ca2", "exam", "assignment", "project", "practical"}, in.AssessmentType) {
		fail(w, 400, "invalid assessment type")
		return
	}
	if in.Weight <= 0 {
		in.Weight = 100
	}
	s.create(w, r, `INSERT INTO assessments (institute_id, class_id, subject_id, term_id, title, max_score, assessment_type, weight)
		VALUES ($1,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8) RETURNING id::text`,
		c.InstituteID, in.ClassID, in.SubjectID, in.TermID, strings.TrimSpace(in.Title), in.MaxScore, in.AssessmentType, in.Weight)
}

func (s *Server) listAssessments(w http.ResponseWriter, r *http.Request, c Claims) {
	q := r.URL.Query()
	if !s.teacherCan(r, c, q.Get("class_id"), "") {
		fail(w, 403, "this class is not assigned to you")
		return
	}
	s.list(w, r, `SELECT a.id::text AS id, a.title, a.max_score::float8 AS max_score, a.assessment_type, a.weight::float8 AS weight, sb.name AS subject, sb.id::text AS subject_id
		FROM assessments a JOIN subjects sb ON sb.id = a.subject_id
		WHERE a.institute_id=$1 AND a.class_id::text=$2 AND a.term_id::text=$3 ORDER BY sb.name, a.title`,
		c.InstituteID, q.Get("class_id"), q.Get("term_id"))
}

func (s *Server) assessmentInfo(r *http.Request, c Claims, id string) (class, subject, term string, max float64, err error) {
	err = s.db.QueryRow(r.Context(), `SELECT class_id::text, subject_id::text, term_id::text, max_score::float8
		FROM assessments WHERE institute_id=$1 AND id::text=$2`, c.InstituteID, id).Scan(&class, &subject, &term, &max)
	return
}

func (s *Server) getGrades(w http.ResponseWriter, r *http.Request, c Claims) {
	id := r.URL.Query().Get("assessment_id")
	class, subject, term, _, err := s.assessmentInfo(r, c, id)
	if err != nil {
		fail(w, 404, "assessment not found")
		return
	}
	if !s.teacherCan(r, c, class, subject) {
		fail(w, 403, "this class and subject are not assigned to you")
		return
	}
	s.list(w, r, `SELECT st.id::text AS student_id, st.first_name, st.last_name, g.score::float8 AS score
		FROM enrollments e JOIN students st ON st.id = e.student_id
		LEFT JOIN grades g ON g.student_id = st.id AND g.assessment_id::text = $4
		WHERE e.institute_id=$1 AND e.class_id::text=$2 AND e.term_id::text=$3
		ORDER BY st.last_name, st.first_name`, c.InstituteID, class, term, id)
}

func (s *Server) saveGrades(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		AssessmentID string `json:"assessment_id"`
		Scores       []struct {
			StudentID string  `json:"student_id"`
			Score     float64 `json:"score"`
		} `json:"scores"`
	}
	if !decode(w, r, &in) {
		return
	}
	class, subject, term, max, err := s.assessmentInfo(r, c, in.AssessmentID)
	if err != nil {
		fail(w, 404, "assessment not found")
		return
	}
	if !s.teacherCan(r, c, class, subject) {
		fail(w, 403, "this class and subject are not assigned to you")
		return
	}
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		for _, sc := range in.Scores {
			if sc.Score < 0 || sc.Score > max {
				return fmt.Errorf("invalid: scores must be between 0 and %v", max)
			}
			tag, err := tx.Exec(r.Context(), `INSERT INTO grades (institute_id, assessment_id, student_id, score)
				SELECT $1, $2::uuid, student_id, $4 FROM enrollments
				WHERE institute_id=$1 AND student_id::text=$3 AND class_id::text=$5 AND term_id::text=$6
				ON CONFLICT (assessment_id, student_id) DO UPDATE SET score = EXCLUDED.score`,
				c.InstituteID, in.AssessmentID, sc.StudentID, sc.Score, class, term)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return fmt.Errorf("invalid: student %s is not in this class for this term", sc.StudentID)
			}
		}
		return nil
	})
	if err != nil {
		if strings.HasPrefix(err.Error(), "invalid: ") {
			fail(w, 400, err.Error()[9:])
			return
		}
		dbFail(w, err)
		return
	}
	send(w, 200, map[string]string{"status": "saved"})
}

func (s *Server) getScale(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT min_score::float8 AS min_score, grade, remark FROM grade_bands
		WHERE institute_id=$1 ORDER BY min_score DESC`, c.InstituteID)
}

func (s *Server) putScale(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		Bands []struct {
			MinScore float64 `json:"min_score"`
			Grade    string  `json:"grade"`
			Remark   string  `json:"remark"`
		} `json:"bands"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Bands) == 0 {
		fail(w, 400, "at least one grade band is required")
		return
	}
	seen := make(map[float64]bool, len(in.Bands))
	for i := range in.Bands {
		in.Bands[i].Grade = strings.TrimSpace(in.Bands[i].Grade)
		in.Bands[i].Remark = strings.TrimSpace(in.Bands[i].Remark)
		if in.Bands[i].MinScore < 0 || in.Bands[i].MinScore > 100 || in.Bands[i].Grade == "" || seen[in.Bands[i].MinScore] {
			fail(w, 400, "grade bands need unique minimum scores from 0 to 100 and a grade label")
			return
		}
		seen[in.Bands[i].MinScore] = true
	}
	if !seen[0] {
		fail(w, 400, "the grading scale must include a 0 minimum score")
		return
	}
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), `DELETE FROM grade_bands WHERE institute_id=$1`, c.InstituteID); err != nil {
			return err
		}
		for _, b := range in.Bands {
			if _, err := tx.Exec(r.Context(), `INSERT INTO grade_bands (institute_id, min_score, grade, remark) VALUES ($1,$2,$3,$4)`,
				c.InstituteID, b.MinScore, b.Grade, b.Remark); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		dbFail(w, err)
		return
	}
	send(w, 200, map[string]string{"status": "saved"})
}

func (s *Server) reportCard(w http.ResponseWriter, r *http.Request, c Claims) {
	sid, term := r.PathValue("studentId"), r.URL.Query().Get("term_id")
	if c.Role == "parent" {
		var linked bool
		if err := s.db.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM student_guardians sg
			JOIN guardians g ON g.id = sg.guardian_id AND g.institute_id = sg.institute_id
			WHERE sg.institute_id=$1 AND sg.student_id::text=$2 AND g.user_id::text=$3)`,
			c.InstituteID, sid, c.UserID).Scan(&linked); err != nil {
			dbFail(w, err)
			return
		}
		if !linked {
			fail(w, 403, "this child is not linked to your account")
			return
		}
	}
	ctx := r.Context()
	var school, first, last, adm, termName, className string
	err := s.db.QueryRow(ctx, `SELECT i.name, st.first_name, st.last_name, st.admission_number, t.name, COALESCE(cl.name,'')
		FROM students st JOIN institutes i ON i.id = st.institute_id
		JOIN terms t ON t.institute_id = st.institute_id AND t.id::text = $3
		LEFT JOIN enrollments e ON e.student_id = st.id AND e.term_id = t.id
		LEFT JOIN classes cl ON cl.id = e.class_id
		WHERE st.institute_id=$1 AND st.id::text=$2`, c.InstituteID, sid, term).Scan(&school, &first, &last, &adm, &termName, &className)
	if err != nil {
		fail(w, 404, "student or term not found")
		return
	}
	bands := []struct {
		Min           float64
		Grade, Remark string
	}{}
	br, err := s.db.Query(ctx, `SELECT min_score::float8, grade, remark FROM grade_bands WHERE institute_id=$1 ORDER BY min_score DESC`, c.InstituteID)
	if err != nil {
		dbFail(w, err)
		return
	}
	for br.Next() {
		var b struct {
			Min           float64
			Grade, Remark string
		}
		if err := br.Scan(&b.Min, &b.Grade, &b.Remark); err != nil {
			br.Close()
			dbFail(w, err)
			return
		}
		bands = append(bands, b)
	}
	if err := br.Err(); err != nil {
		br.Close()
		dbFail(w, err)
		return
	}
	br.Close()

	subjects := []map[string]any{}
	rs, err := s.db.Query(ctx, `SELECT sb.name, SUM(g.score)::float8, SUM(a.max_score)::float8
		FROM grades g JOIN assessments a ON a.id = g.assessment_id JOIN subjects sb ON sb.id = a.subject_id
		WHERE g.institute_id=$1 AND g.student_id::text=$2 AND a.term_id::text=$3 GROUP BY sb.name ORDER BY sb.name`,
		c.InstituteID, sid, term)
	if err != nil {
		dbFail(w, err)
		return
	}
	var sumPct float64
	for rs.Next() {
		var name string
		var score, max float64
		if err := rs.Scan(&name, &score, &max); err != nil {
			rs.Close()
			dbFail(w, err)
			return
		}
		if max <= 0 {
			rs.Close()
			fail(w, 500, "an assessment has an invalid maximum score")
			return
		}
		pct := score / max * 100
		grade, remark := "", ""
		for _, b := range bands {
			if pct >= b.Min {
				grade, remark = b.Grade, b.Remark
				break
			}
		}
		sumPct += pct
		subjects = append(subjects, map[string]any{"subject": name, "score": score, "max_score": max,
			"percent": float64(int(pct*10+0.5)) / 10, "grade": grade, "remark": remark})
	}
	if err := rs.Err(); err != nil {
		rs.Close()
		dbFail(w, err)
		return
	}
	rs.Close()
	average := 0.0
	if len(subjects) > 0 {
		average = float64(int(sumPct/float64(len(subjects))*10+0.5)) / 10
	}

	var present, absent, late int
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FILTER (WHERE a.status='present'), COUNT(*) FILTER (WHERE a.status='absent'),
		COUNT(*) FILTER (WHERE a.status='late') FROM attendance a JOIN enrollments e ON e.id = a.enrollment_id
		WHERE e.institute_id=$1 AND e.student_id::text=$2 AND e.term_id::text=$3`, c.InstituteID, sid, term).Scan(&present, &absent, &late); err != nil {
		dbFail(w, err)
		return
	}
	var billed, paid int64
	if err := s.db.QueryRow(ctx, `SELECT COALESCE(SUM(i.amount_kobo),0)::bigint,
		COALESCE((SELECT SUM(p.amount_kobo) FROM payments p JOIN fee_invoices i2 ON i2.id = p.invoice_id
		WHERE i2.institute_id=$1 AND i2.student_id::text=$2 AND i2.term_id::text=$3),0)::bigint
		FROM fee_invoices i WHERE i.institute_id=$1 AND i.student_id::text=$2 AND i.term_id::text=$3`,
		c.InstituteID, sid, term).Scan(&billed, &paid); err != nil {
		dbFail(w, err)
		return
	}

	var teacherRemark, principalRemark, conduct, punctuality string
	_ = s.db.QueryRow(ctx, `SELECT COALESCE(teacher_remark,''),COALESCE(principal_remark,''),COALESCE(conduct,''),COALESCE(punctuality,'')
		FROM student_remarks WHERE institute_id=$1 AND student_id=$2::uuid AND term_id=$3::uuid`, c.InstituteID, sid, term).
		Scan(&teacherRemark, &principalRemark, &conduct, &punctuality)

	send(w, 200, map[string]any{"school": school, "student": first + " " + last, "admission_number": adm,
		"class": className, "term": termName, "subjects": subjects, "average_percent": average,
		"attendance": map[string]int{"present": present, "absent": absent, "late": late},
		"fees":       map[string]int64{"billed_kobo": billed, "paid_kobo": paid, "balance_kobo": billed - paid},
		"remarks":    map[string]string{"teacher": teacherRemark, "principal": principalRemark, "conduct": conduct, "punctuality": punctuality}})
}

// ---------- communication and summaries ----------

func (s *Server) listAnnouncements(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT id::text AS id, title, body, created_at FROM announcements
		WHERE institute_id=$1 ORDER BY created_at DESC LIMIT 50`, c.InstituteID)
}

func (s *Server) createAnnouncement(w http.ResponseWriter, r *http.Request, c Claims) {
	var in struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.Body) == "" {
		fail(w, 400, "title and message are required")
		return
	}
	s.create(w, r, `INSERT INTO announcements (institute_id, title, body, created_by) VALUES ($1,$2,$3,$4::uuid) RETURNING id::text`,
		c.InstituteID, strings.TrimSpace(in.Title), strings.TrimSpace(in.Body), c.UserID)
}

func (s *Server) parentChildren(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT st.id::text AS student_id,st.first_name,st.last_name,st.admission_number,
		(SELECT COUNT(*) FROM attendance a JOIN enrollments e ON e.id=a.enrollment_id WHERE e.student_id=st.id AND a.status IN ('present','late'))::int AS days_present,
		(SELECT COUNT(*) FROM attendance a JOIN enrollments e ON e.id=a.enrollment_id WHERE e.student_id=st.id AND a.status='absent')::int AS days_absent,
		COALESCE((SELECT SUM(amount_kobo) FROM fee_invoices WHERE student_id=st.id),0)::bigint AS billed_kobo,
		COALESCE((SELECT SUM(p.amount_kobo) FROM payments p JOIN fee_invoices i ON i.id=p.invoice_id WHERE i.student_id=st.id),0)::bigint AS paid_kobo
		FROM students st JOIN student_guardians sg ON sg.student_id=st.id
		JOIN guardians g ON g.id=sg.guardian_id AND g.institute_id=st.institute_id
		WHERE st.institute_id=$1 AND g.user_id::text=$2 ORDER BY st.first_name`, c.InstituteID, c.UserID)
}

func (s *Server) subscription(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT plan,status,term_ends_on FROM subscriptions WHERE institute_id=$1`, c.InstituteID)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request, c Claims) {
	var students, inactiveStudents, teachers, staff, classes, enrollments, present, marked int
	var billed, paid int64
	var academicAverage float64
	term := r.URL.Query().Get("term_id")
	ctx := r.Context()

	queries := []struct {
		sql  string
		args []any
		dest []any
	}{
		{`SELECT COUNT(*) FILTER (WHERE status='active'), COUNT(*) FILTER (WHERE status='inactive') FROM students WHERE institute_id=$1`, []any{c.InstituteID}, []any{&students, &inactiveStudents}},
		{`SELECT COUNT(*) FROM institute_members WHERE institute_id=$1 AND role='teacher'`, []any{c.InstituteID}, []any{&teachers}},
		{`SELECT COUNT(*) FROM institute_members WHERE institute_id=$1 AND role IN ('owner','admin','bursar','teacher')`, []any{c.InstituteID}, []any{&staff}},
		{`SELECT COUNT(*) FROM classes WHERE institute_id=$1`, []any{c.InstituteID}, []any{&classes}},
		{`SELECT COUNT(*) FROM enrollments WHERE institute_id=$1 AND ($2='' OR term_id::text=$2)`, []any{c.InstituteID, term}, []any{&enrollments}},
		{`SELECT COUNT(*) FILTER (WHERE status IN ('present','late')), COUNT(*) FROM attendance WHERE institute_id=$1 AND day=CURRENT_DATE`, []any{c.InstituteID}, []any{&present, &marked}},
		{`SELECT COALESCE(SUM(amount_kobo),0)::bigint FROM fee_invoices WHERE institute_id=$1 AND ($2='' OR term_id::text=$2)`, []any{c.InstituteID, term}, []any{&billed}},
		{`SELECT COALESCE(SUM(p.amount_kobo),0)::bigint FROM payments p JOIN fee_invoices i ON i.id=p.invoice_id WHERE p.institute_id=$1 AND ($2='' OR i.term_id::text=$2)`, []any{c.InstituteID, term}, []any{&paid}},
		{`SELECT COALESCE(AVG(g.score/a.max_score*100),0)::float8 FROM grades g JOIN assessments a ON a.id=g.assessment_id WHERE g.institute_id=$1 AND ($2='' OR a.term_id::text=$2)`, []any{c.InstituteID, term}, []any{&academicAverage}},
	}
	for _, q := range queries {
		if err := s.db.QueryRow(ctx, q.sql, q.args...).Scan(q.dest...); err != nil {
			dbFail(w, err)
			return
		}
	}

	recent := []map[string]any{}
	recentRows, err := s.db.Query(ctx, `SELECT action,'activity',created_at FROM audit_events WHERE institute_id=$1
		UNION ALL SELECT title,'announcement',created_at FROM announcements WHERE institute_id=$1
		ORDER BY created_at DESC LIMIT 8`, c.InstituteID)
	if err != nil {
		dbFail(w, err)
		return
	}
	defer recentRows.Close()
	for recentRows.Next() {
		var title, kind string
		var createdAt any
		if err := recentRows.Scan(&title, &kind, &createdAt); err != nil {
			dbFail(w, err)
			return
		}
		recent = append(recent, map[string]any{"title": title, "kind": kind, "created_at": createdAt})
	}

	events := []map[string]any{}
	eventRows, err := s.db.Query(ctx, `SELECT id::text,title,event_type,starts_at::text FROM school_events
		WHERE institute_id=$1 AND starts_at>=NOW() ORDER BY starts_at LIMIT 6`, c.InstituteID)
	if err != nil {
		dbFail(w, err)
		return
	}
	defer eventRows.Close()
	for eventRows.Next() {
		var id, title, eventType, starts string
		if err := eventRows.Scan(&id, &title, &eventType, &starts); err != nil {
			dbFail(w, err)
			return
		}
		events = append(events, map[string]any{"id": id, "title": title, "type": eventType, "starts_at": starts})
	}

	alerts := []map[string]any{}
	attendanceRate := 0
	if marked > 0 {
		attendanceRate = present * 100 / marked
		if attendanceRate < 85 {
			alerts = append(alerts, map[string]any{"level": "warning", "title": "Low attendance", "detail": "Attendance is below 85% today.", "action": "Attendance"})
		}
	}
	if billed-paid > 0 {
		alerts = append(alerts, map[string]any{"level": "attention", "title": "Outstanding fees", "detail": "There are uncollected fees for the selected term.", "action": "Fees"})
	}
	if inactiveStudents > 0 {
		alerts = append(alerts, map[string]any{"level": "info", "title": "Inactive student records", "detail": fmt.Sprintf("%d student record(s) need review.", inactiveStudents), "action": "Students"})
	}

	send(w, 200, map[string]any{
		"students": students, "inactive_students": inactiveStudents, "teachers": teachers, "staff": staff, "classes": classes, "enrollments": enrollments,
		"attendance_today": map[string]int{"present": present, "marked": marked, "rate": attendanceRate},
		"fees":             map[string]int64{"billed_kobo": billed, "collected_kobo": paid, "outstanding_kobo": billed - paid},
		"academic":         map[string]float64{"average_percent": academicAverage}, "alerts": alerts, "recent_activity": recent, "upcoming_events": events,
	})
}

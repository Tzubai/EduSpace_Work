package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) listAdmissions(w http.ResponseWriter, r *http.Request, c Claims) {
	status := r.URL.Query().Get("status")
	s.list(w, r, `SELECT a.id::text,a.application_number,a.applicant_name,a.guardian_name,a.guardian_phone,
		a.email,a.status,a.submitted_at,c.name AS applying_for_class,a.notes
		FROM admissions a JOIN classes c ON c.id=a.applying_for_class
		WHERE a.institute_id=$1 AND ($2='' OR a.status=$2)
		ORDER BY a.submitted_at DESC`, c.InstituteID, status)
}

func (s *Server) createAdmission(w http.ResponseWriter, r *http.Request, c Claims) {
	var body struct {
		ApplicationNumber string `json:"application_number"`
		ApplicantName     string `json:"applicant_name"`
		GuardianName      string `json:"guardian_name"`
		GuardianPhone     string `json:"guardian_phone"`
		Email             string `json:"email"`
		ClassID           string `json:"class_id"`
		PreviousSchool    string `json:"previous_school"`
		Notes             string `json:"notes"`
	}
	if !decode(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.ApplicationNumber) == "" || strings.TrimSpace(body.ApplicantName) == "" || strings.TrimSpace(body.GuardianName) == "" || strings.TrimSpace(body.ClassID) == "" {
		fail(w, 400, "application number, applicant, guardian and class are required")
		return
	}
	s.create(w, r, `INSERT INTO admissions (institute_id,application_number,applicant_name,guardian_name,guardian_phone,email,applying_for_class,previous_school,notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7::uuid,$8,$9) RETURNING id::text`, c.InstituteID, body.ApplicationNumber, body.ApplicantName, body.GuardianName, body.GuardianPhone, nilIfEmpty(body.Email), body.ClassID, nilIfEmpty(body.PreviousSchool), nilIfEmpty(body.Notes))
}

func (s *Server) updateAdmission(w http.ResponseWriter, r *http.Request, c Claims) {
	var body struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if !decode(w, r, &body) {
		return
	}
	if !has([]string{"submitted", "under_review", "accepted", "waitlisted", "rejected"}, body.Status) {
		fail(w, 400, "invalid admission status")
		return
	}
	tag, err := s.db.Exec(r.Context(), `UPDATE admissions SET status=$3,notes=COALESCE(NULLIF($4,''),notes)
		WHERE institute_id=$1 AND id::text=$2`, c.InstituteID, r.PathValue("id"), body.Status, body.Notes)
	if err != nil {
		dbFail(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "application not found")
		return
	}
	s.sendAudit(r, c, "admission.status_changed", r.PathValue("id"), map[string]any{"status": body.Status})
	send(w, 200, map[string]bool{"ok": true})
}

func (s *Server) listTimetable(w http.ResponseWriter, r *http.Request, c Claims) {
	term := r.URL.Query().Get("term_id")
	s.list(w, r, `SELECT t.id::text,t.day_of_week,t.starts_at,t.ends_at,t.room,
		c.name AS class_name,s.name AS subject_name,u.full_name AS teacher_name
		FROM timetable_slots t JOIN classes c ON c.id=t.class_id JOIN subjects s ON s.id=t.subject_id
		JOIN users u ON u.id=t.teacher_id
		WHERE t.institute_id=$1 AND ($2='' OR t.term_id::text=$2)
		ORDER BY t.day_of_week,t.starts_at,c.name`, c.InstituteID, term)
}

func (s *Server) createTimetable(w http.ResponseWriter, r *http.Request, c Claims) {
	var body struct {
		TermID    string `json:"term_id"`
		ClassID   string `json:"class_id"`
		SubjectID string `json:"subject_id"`
		TeacherID string `json:"teacher_id"`
		DayOfWeek int    `json:"day_of_week"`
		StartsAt  string `json:"starts_at"`
		EndsAt    string `json:"ends_at"`
		Room      string `json:"room"`
	}
	if !decode(w, r, &body) {
		return
	}
	if body.TermID == "" || body.ClassID == "" || body.SubjectID == "" || body.TeacherID == "" || body.StartsAt == "" || body.EndsAt == "" || body.DayOfWeek < 1 || body.DayOfWeek > 7 {
		fail(w, 400, "complete timetable details are required")
		return
	}
	s.create(w, r, `INSERT INTO timetable_slots (institute_id,term_id,class_id,subject_id,teacher_id,day_of_week,starts_at,ends_at,room)
		VALUES ($1,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7::time,$8::time,$9) RETURNING id::text`, c.InstituteID, body.TermID, body.ClassID, body.SubjectID, body.TeacherID, body.DayOfWeek, body.StartsAt, body.EndsAt, nilIfEmpty(body.Room))
}

func (s *Server) listStaffAttendance(w http.ResponseWriter, r *http.Request, c Claims) {
	day := r.URL.Query().Get("day")
	if day == "" {
		day = todayDate()
	}
	s.list(w, r, `SELECT sa.id::text,u.full_name,u.email,sa.status,sa.check_in_at,sa.check_out_at,sa.note
		FROM staff_attendance sa JOIN users u ON u.id=sa.user_id
		WHERE sa.institute_id=$1 AND sa.day=$2::date ORDER BY u.full_name`, c.InstituteID, day)
}

func (s *Server) saveStaffAttendance(w http.ResponseWriter, r *http.Request, c Claims) {
	var body struct {
		Day     string `json:"day"`
		Records []struct {
			UserID   string `json:"user_id"`
			Status   string `json:"status"`
			CheckIn  string `json:"check_in"`
			CheckOut string `json:"check_out"`
			Note     string `json:"note"`
		} `json:"records"`
	}
	if !decode(w, r, &body) {
		return
	}
	if body.Day == "" || len(body.Records) == 0 {
		fail(w, 400, "day and attendance records are required")
		return
	}
	for _, rec := range body.Records {
		if !has([]string{"present", "absent", "late", "leave"}, rec.Status) {
			fail(w, 400, "invalid staff attendance status")
			return
		}
		_, err := s.db.Exec(r.Context(), `INSERT INTO staff_attendance (institute_id,user_id,day,status,check_in_at,check_out_at,note,recorded_by)
			VALUES ($1,$2::uuid,$3::date,$4,NULLIF($5,'')::time,NULLIF($6,'')::time,NULLIF($7,''),$8::uuid)
			ON CONFLICT (institute_id,user_id,day) DO UPDATE SET status=EXCLUDED.status,
			check_in_at=EXCLUDED.check_in_at,check_out_at=EXCLUDED.check_out_at,note=EXCLUDED.note,recorded_by=EXCLUDED.recorded_by`,
			c.InstituteID, rec.UserID, body.Day, rec.Status, rec.CheckIn, rec.CheckOut, rec.Note, c.UserID)
		if err != nil {
			dbFail(w, err)
			return
		}
	}
	s.sendAudit(r, c, "staff_attendance.saved", body.Day, map[string]any{"records": len(body.Records)})
	send(w, 200, map[string]bool{"ok": true})
}

func todayDate() string { return time.Now().Format("2006-01-02") }

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT id::text,title,body,audience,channel,status,sent_at,created_at
		FROM notification_messages WHERE institute_id=$1 ORDER BY created_at DESC LIMIT 100`, c.InstituteID)
}

func (s *Server) createNotification(w http.ResponseWriter, r *http.Request, c Claims) {
	var body struct {
		Title    string `json:"title"`
		Body     string `json:"body"`
		Audience string `json:"audience"`
		Channel  string `json:"channel"`
		SendNow  bool   `json:"send_now"`
	}
	if !decode(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Title) == "" || strings.TrimSpace(body.Body) == "" {
		fail(w, 400, "title and message are required")
		return
	}
	if body.Audience == "" {
		body.Audience = "all"
	}
	if body.Channel == "" {
		body.Channel = "in_app"
	}
	if body.SendNow {
		s.create(w, r, `INSERT INTO notification_messages (institute_id,title,body,audience,channel,status,created_by,sent_at)
		VALUES ($1,$2,$3,$4,$5,'sent',$6::uuid,NOW()) RETURNING id::text`, c.InstituteID, body.Title, body.Body, body.Audience, body.Channel, c.UserID)
	} else {
		s.create(w, r, `INSERT INTO notification_messages (institute_id,title,body,audience,channel,status,created_by)
		VALUES ($1,$2,$3,$4,$5,'draft',$6::uuid) RETURNING id::text`, c.InstituteID, body.Title, body.Body, body.Audience, body.Channel, c.UserID)
	}
}

func (s *Server) listSchemes(w http.ResponseWriter, r *http.Request, c Claims) {
	term := r.URL.Query().Get("term_id")
	s.list(w, r, `SELECT sw.id::text,sw.title,sw.topics,sw.week_number,sw.status,c.name AS class_name,sb.name AS subject_name FROM schemes_of_work sw JOIN classes c ON c.id=sw.class_id JOIN subjects sb ON sb.id=sw.subject_id WHERE sw.institute_id=$1 AND ($2='' OR sw.term_id::text=$2) ORDER BY sw.week_number NULLS LAST,sw.title`, c.InstituteID, term)
}

func (s *Server) saveScheme(w http.ResponseWriter, r *http.Request, c Claims) {
	var b struct {
		TermID     string `json:"term_id"`
		ClassID    string `json:"class_id"`
		SubjectID  string `json:"subject_id"`
		Title      string `json:"title"`
		Topics     string `json:"topics"`
		WeekNumber int    `json:"week_number"`
		Status     string `json:"status"`
	}
	if !decode(w, r, &b) {
		return
	}
	if b.TermID == "" || b.ClassID == "" || b.SubjectID == "" || strings.TrimSpace(b.Title) == "" {
		fail(w, 400, "term, class, subject and title are required")
		return
	}
	if b.Status == "" {
		b.Status = "planned"
	}
	if !has([]string{"planned", "in_progress", "completed"}, b.Status) {
		fail(w, 400, "invalid scheme status")
		return
	}
	s.create(w, r, `INSERT INTO schemes_of_work(institute_id,class_id,subject_id,term_id,title,topics,week_number,status) VALUES($1,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8) RETURNING id::text`, c.InstituteID, b.ClassID, b.SubjectID, b.TermID, strings.TrimSpace(b.Title), nilIfEmpty(strings.TrimSpace(b.Topics)), b.WeekNumber, b.Status)
}

func (s *Server) listAuditEvents(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT ae.id::text,ae.action,ae.entity_id,ae.metadata,ae.created_at,u.full_name AS actor
		FROM audit_events ae LEFT JOIN users u ON u.id=ae.actor_user_id
		WHERE ae.institute_id=$1 ORDER BY ae.created_at DESC LIMIT 250`, c.InstituteID)
}

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT id::text,title,description,event_type,starts_at::text,ends_at::text,audience,created_at
		FROM school_events WHERE institute_id=$1 AND starts_at>=NOW()-INTERVAL '1 day'
		ORDER BY starts_at LIMIT 100`, c.InstituteID)
}

func (s *Server) createEvent(w http.ResponseWriter, r *http.Request, c Claims) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		EventType   string `json:"event_type"`
		StartsAt    string `json:"starts_at"`
		EndsAt      string `json:"ends_at"`
		Audience    string `json:"audience"`
	}
	if !decode(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Title) == "" || body.StartsAt == "" {
		fail(w, 400, "event title and start date are required")
		return
	}
	if body.EventType == "" {
		body.EventType = "school"
	}
	if body.Audience == "" {
		body.Audience = "all"
	}
	s.create(w, r, `INSERT INTO school_events(institute_id,title,description,event_type,starts_at,ends_at,audience,created_by)
		VALUES($1,$2,$3,$4,$5::timestamptz,NULLIF($6,'')::timestamptz,$7,$8::uuid) RETURNING id::text`, c.InstituteID, strings.TrimSpace(body.Title), nilIfEmpty(strings.TrimSpace(body.Description)), body.EventType, body.StartsAt, body.EndsAt, body.Audience, c.UserID)
}

func (s *Server) listParentMessages(w http.ResponseWriter, r *http.Request, c Claims) {
	s.list(w, r, `SELECT pm.id::text,pm.subject,pm.body,pm.channel,pm.sent_at,pm.read_at,g.full_name AS guardian_name
		FROM parent_messages pm JOIN guardians g ON g.id=pm.recipient_guardian_id
		WHERE pm.institute_id=$1 ORDER BY pm.sent_at DESC LIMIT 100`, c.InstituteID)
}

func (s *Server) createParentMessage(w http.ResponseWriter, r *http.Request, c Claims) {
	var body struct {
		GuardianID string `json:"guardian_id"`
		Subject    string `json:"subject"`
		Body       string `json:"body"`
		Channel    string `json:"channel"`
	}
	if !decode(w, r, &body) {
		return
	}
	if body.GuardianID == "" || strings.TrimSpace(body.Subject) == "" || strings.TrimSpace(body.Body) == "" {
		fail(w, 400, "guardian, subject and message are required")
		return
	}
	if body.Channel == "" {
		body.Channel = "in_app"
	}
	s.create(w, r, `INSERT INTO parent_messages(institute_id,sender_user_id,recipient_guardian_id,subject,body,channel)
		VALUES($1,$2::uuid,$3::uuid,$4,$5,$6) RETURNING id::text`, c.InstituteID, c.UserID, body.GuardianID, strings.TrimSpace(body.Subject), strings.TrimSpace(body.Body), body.Channel)
}

func (s *Server) createParentRequest(w http.ResponseWriter, r *http.Request, c Claims) {
	var b struct {
		StudentID string `json:"student_id"`
		Message   string `json:"message"`
	}
	if !decode(w, r, &b) {
		return
	}
	if b.StudentID == "" || strings.TrimSpace(b.Message) == "" {
		fail(w, 400, "child and message are required")
		return
	}
	if !s.parentLinked(r, c, b.StudentID) {
		fail(w, 403, "this child is not linked to your account")
		return
	}
	var guardianID string
	if err := s.db.QueryRow(r.Context(), `SELECT g.id::text FROM guardians g JOIN student_guardians sg ON sg.guardian_id=g.id WHERE g.institute_id=$1 AND g.user_id::text=$2 AND sg.student_id::text=$3 LIMIT 1`, c.InstituteID, c.UserID, b.StudentID).Scan(&guardianID); err != nil {
		dbFail(w, err)
		return
	}
	s.create(w, r, `INSERT INTO parent_messages(institute_id,sender_user_id,recipient_guardian_id,subject,body,channel) VALUES($1,$2::uuid,$3::uuid,'School message',$4,'in_app') RETURNING id::text`, c.InstituteID, c.UserID, guardianID, strings.TrimSpace(b.Message))
}

func (s *Server) listExams(w http.ResponseWriter, r *http.Request, c Claims) {
	term := r.URL.Query().Get("term_id")
	s.list(w, r, `SELECT es.id::text,es.title,es.starts_at::text,es.duration_minutes,es.venue,c.name AS class_name,sb.name AS subject_name FROM examination_schedules es JOIN classes c ON c.id=es.class_id JOIN subjects sb ON sb.id=es.subject_id WHERE es.institute_id=$1 AND ($2='' OR es.term_id::text=$2) ORDER BY es.starts_at`, c.InstituteID, term)
}

func (s *Server) createExam(w http.ResponseWriter, r *http.Request, c Claims) {
	var b struct {
		TermID    string `json:"term_id"`
		ClassID   string `json:"class_id"`
		SubjectID string `json:"subject_id"`
		Title     string `json:"title"`
		StartsAt  string `json:"starts_at"`
		Venue     string `json:"venue"`
		Duration  int    `json:"duration_minutes"`
	}
	if !decode(w, r, &b) {
		return
	}
	if b.TermID == "" || b.ClassID == "" || b.SubjectID == "" || strings.TrimSpace(b.Title) == "" || b.StartsAt == "" || b.Duration <= 0 {
		fail(w, 400, "term, class, subject, title, date and duration are required")
		return
	}
	s.create(w, r, `INSERT INTO examination_schedules(institute_id,term_id,class_id,subject_id,title,starts_at,duration_minutes,venue) VALUES($1,$2::uuid,$3::uuid,$4::uuid,$5,$6::timestamptz,$7,$8) RETURNING id::text`, c.InstituteID, b.TermID, b.ClassID, b.SubjectID, strings.TrimSpace(b.Title), b.StartsAt, b.Duration, nilIfEmpty(strings.TrimSpace(b.Venue)))
}

func (s *Server) saveRemarks(w http.ResponseWriter, r *http.Request, c Claims) {
	var b struct {
		StudentID       string `json:"student_id"`
		TermID          string `json:"term_id"`
		TeacherRemark   string `json:"teacher_remark"`
		PrincipalRemark string `json:"principal_remark"`
		Conduct         string `json:"conduct"`
		Punctuality     string `json:"punctuality"`
	}
	if !decode(w, r, &b) {
		return
	}
	if b.StudentID == "" || b.TermID == "" {
		fail(w, 400, "student and term are required")
		return
	}
	_, err := s.db.Exec(r.Context(), `INSERT INTO student_remarks(institute_id,student_id,term_id,teacher_remark,principal_remark,conduct,punctuality) VALUES($1,$2::uuid,$3::uuid,$4,$5,$6,$7) ON CONFLICT(student_id,term_id) DO UPDATE SET teacher_remark=EXCLUDED.teacher_remark,principal_remark=EXCLUDED.principal_remark,conduct=EXCLUDED.conduct,punctuality=EXCLUDED.punctuality,updated_at=NOW()`, c.InstituteID, b.StudentID, b.TermID, nilIfEmpty(b.TeacherRemark), nilIfEmpty(b.PrincipalRemark), nilIfEmpty(b.Conduct), nilIfEmpty(b.Punctuality))
	if err != nil {
		dbFail(w, err)
		return
	}
	s.sendAudit(r, c, "report.remarks.saved", b.StudentID, map[string]any{"term_id": b.TermID})
	send(w, 200, map[string]bool{"ok": true})
}

func (s *Server) promoteStudent(w http.ResponseWriter, r *http.Request, c Claims) {
	var b struct {
		StudentID   string `json:"student_id"`
		FromClassID string `json:"from_class_id"`
		ToClassID   string `json:"to_class_id"`
		TermID      string `json:"term_id"`
		Decision    string `json:"decision"`
		Reason      string `json:"reason"`
	}
	if !decode(w, r, &b) {
		return
	}
	if b.StudentID == "" || b.TermID == "" || !has([]string{"promoted", "retained", "graduated", "transferred"}, b.Decision) {
		fail(w, 400, "student, term and valid decision are required")
		return
	}
	var id string
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(r.Context(), `INSERT INTO student_promotions(institute_id,student_id,from_class_id,to_class_id,term_id,decision,reason,decided_by) VALUES($1,$2::uuid,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,$5::uuid,$6,$7,$8::uuid) RETURNING id::text`, c.InstituteID, b.StudentID, b.FromClassID, b.ToClassID, b.TermID, b.Decision, nilIfEmpty(strings.TrimSpace(b.Reason)), c.UserID).Scan(&id); err != nil {
			return err
		}
		if b.Decision == "graduated" || b.Decision == "transferred" {
			_, err := tx.Exec(r.Context(), `UPDATE students SET status='inactive' WHERE institute_id=$1 AND id=$2::uuid`, c.InstituteID, b.StudentID)
			return err
		}
		return nil
	})
	if err != nil {
		dbFail(w, err)
		return
	}
	s.sendAudit(r, c, "student.promotion_decided", b.StudentID, map[string]any{"decision": b.Decision})
	send(w, 201, map[string]string{"id": id})
}

func (s *Server) correctAttendance(w http.ResponseWriter, r *http.Request, c Claims) {
	var b struct {
		AttendanceID string `json:"attendance_id"`
		NewStatus    string `json:"new_status"`
		Reason       string `json:"reason"`
	}
	if !decode(w, r, &b) {
		return
	}
	if b.AttendanceID == "" || !has([]string{"present", "absent", "late", "excused"}, b.NewStatus) || strings.TrimSpace(b.Reason) == "" {
		fail(w, 400, "attendance record, new status and correction reason are required")
		return
	}
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var previous string
		if err := tx.QueryRow(r.Context(), `SELECT status FROM attendance WHERE institute_id=$1 AND id=$2::uuid FOR UPDATE`, c.InstituteID, b.AttendanceID).Scan(&previous); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE attendance SET status=$3,corrected_by=$4::uuid,corrected_at=NOW(),correction_reason=$5 WHERE institute_id=$1 AND id=$2::uuid`, c.InstituteID, b.AttendanceID, b.NewStatus, c.UserID, strings.TrimSpace(b.Reason)); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `INSERT INTO attendance_corrections(institute_id,attendance_id,previous_status,new_status,correction_reason,corrected_by) VALUES($1,$2::uuid,$3,$4,$5,$6::uuid)`, c.InstituteID, b.AttendanceID, previous, b.NewStatus, strings.TrimSpace(b.Reason), c.UserID)
		return err
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			fail(w, 404, "attendance record not found")
			return
		}
		dbFail(w, err)
		return
	}
	s.sendAudit(r, c, "attendance.corrected", b.AttendanceID, map[string]any{"status": b.NewStatus, "reason": b.Reason})
	send(w, 200, map[string]bool{"ok": true})
}

func (s *Server) monthlyAttendance(w http.ResponseWriter, r *http.Request, c Claims) {
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	s.list(w, r, `SELECT st.id::text AS student_id,st.first_name,st.last_name,COUNT(a.id) FILTER (WHERE a.status='present')::int AS present,COUNT(a.id) FILTER (WHERE a.status='absent')::int AS absent,COUNT(a.id) FILTER (WHERE a.status='late')::int AS late,COUNT(a.id) FILTER (WHERE a.status='excused')::int AS excused,COUNT(a.id)::int AS marked,CASE WHEN COUNT(a.id)=0 THEN 0 ELSE ROUND(COUNT(a.id) FILTER (WHERE a.status IN ('present','late'))::numeric*100/COUNT(a.id),1)::float8 END AS percentage FROM students st JOIN enrollments e ON e.student_id=st.id LEFT JOIN attendance a ON a.enrollment_id=e.id AND a.day>=($2||'-01')::date AND a.day<(($2||'-01')::date+INTERVAL '1 month') WHERE st.institute_id=$1 GROUP BY st.id ORDER BY st.last_name,st.first_name`, c.InstituteID, month)
}

func (s *Server) parentLinked(r *http.Request, c Claims, studentID string) bool {
	var linked bool
	if err := s.db.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM student_guardians sg JOIN guardians g ON g.id=sg.guardian_id WHERE sg.institute_id=$1 AND sg.student_id::text=$2 AND g.user_id::text=$3)`, c.InstituteID, studentID, c.UserID).Scan(&linked); err != nil {
		return false
	}
	return linked
}

func (s *Server) parentStudentSummary(w http.ResponseWriter, r *http.Request, c Claims) {
	studentID := r.PathValue("studentId")
	if !s.parentLinked(r, c, studentID) {
		fail(w, 403, "this child is not linked to your account")
		return
	}
	term := r.URL.Query().Get("term_id")
	var present, absent, late, excused int
	var billed, paid int64
	var average float64
	ctx := r.Context()
	queries := []struct {
		sql  string
		args []any
		dest []any
	}{
		{`SELECT COUNT(*) FILTER (WHERE a.status='present'),COUNT(*) FILTER (WHERE a.status='absent'),COUNT(*) FILTER (WHERE a.status='late'),COUNT(*) FILTER (WHERE a.status='excused') FROM attendance a JOIN enrollments e ON e.id=a.enrollment_id WHERE e.institute_id=$1 AND e.student_id::text=$2 AND ($3='' OR e.term_id::text=$3)`, []any{c.InstituteID, studentID, term}, []any{&present, &absent, &late, &excused}},
		{`SELECT COALESCE(SUM(i.amount_kobo),0)::bigint,COALESCE((SELECT SUM(p.amount_kobo) FROM payments p JOIN fee_invoices i2 ON i2.id=p.invoice_id WHERE i2.institute_id=$1 AND i2.student_id::text=$2 AND ($3='' OR i2.term_id::text=$3)),0)::bigint FROM fee_invoices i WHERE i.institute_id=$1 AND i.student_id::text=$2 AND ($3='' OR i.term_id::text=$3)`, []any{c.InstituteID, studentID, term}, []any{&billed, &paid}},
		{`SELECT COALESCE(AVG(g.score/a.max_score*100),0)::float8 FROM grades g JOIN assessments a ON a.id=g.assessment_id WHERE g.institute_id=$1 AND g.student_id::text=$2 AND ($3='' OR a.term_id::text=$3)`, []any{c.InstituteID, studentID, term}, []any{&average}},
	}
	for _, q := range queries {
		if err := s.db.QueryRow(ctx, q.sql, q.args...).Scan(q.dest...); err != nil {
			dbFail(w, err)
			return
		}
	}
	send(w, 200, map[string]any{
		"attendance": map[string]int{
			"present": present,
			"absent":  absent,
			"late":    late,
			"excused": excused,
		},
		"fees": map[string]int64{
			"billed_kobo":  billed,
			"paid_kobo":    paid,
			"balance_kobo": billed - paid,
		},
		"academic": map[string]float64{
			"average_percent": average,
		},
	})
}

func (s *Server) parentInvoices(w http.ResponseWriter, r *http.Request, c Claims) {
	studentID := r.PathValue("studentId")
	if !s.parentLinked(r, c, studentID) {
		fail(w, 403, "this child is not linked to your account")
		return
	}
	term := r.URL.Query().Get("term_id")
	s.list(w, r, `SELECT i.id::text,i.description,i.amount_kobo,COALESCE(SUM(p.amount_kobo),0)::bigint AS paid_kobo,(i.amount_kobo-COALESCE(SUM(p.amount_kobo),0))::bigint AS balance_kobo,i.status,t.name AS term FROM fee_invoices i JOIN terms t ON t.id=i.term_id LEFT JOIN payments p ON p.invoice_id=i.id WHERE i.institute_id=$1 AND i.student_id::text=$2 AND ($3='' OR i.term_id::text=$3) GROUP BY i.id,t.name ORDER BY i.created_at DESC`, c.InstituteID, studentID, term)
}

func (s *Server) parentPayments(w http.ResponseWriter, r *http.Request, c Claims) {
	studentID := r.PathValue("studentId")
	if !s.parentLinked(r, c, studentID) {
		fail(w, 403, "this child is not linked to your account")
		return
	}
	term := r.URL.Query().Get("term_id")
	s.list(w, r, `SELECT p.id::text,p.amount_kobo,p.method,p.reference,p.paid_at,i.description,t.name AS term FROM payments p JOIN fee_invoices i ON i.id=p.invoice_id JOIN terms t ON t.id=i.term_id WHERE p.institute_id=$1 AND i.student_id::text=$2 AND ($3='' OR i.term_id::text=$3) ORDER BY p.paid_at DESC`, c.InstituteID, studentID, term)
}

func (s *Server) parentAttendance(w http.ResponseWriter, r *http.Request, c Claims) {
	studentID := r.PathValue("studentId")
	if !s.parentLinked(r, c, studentID) {
		fail(w, 403, "this child is not linked to your account")
		return
	}
	term := r.URL.Query().Get("term_id")
	month := r.URL.Query().Get("month")
	s.list(w, r, `SELECT a.id::text,a.day,a.status,a.arrival_time,a.reason FROM attendance a JOIN enrollments e ON e.id=a.enrollment_id WHERE e.institute_id=$1 AND e.student_id::text=$2 AND ($3='' OR e.term_id::text=$3) AND ($4='' OR (a.day>=($4||'-01')::date AND a.day<(($4||'-01')::date+INTERVAL '1 month'))) ORDER BY a.day DESC LIMIT 366`, c.InstituteID, studentID, term, month)
}

func (s *Server) parentTimetable(w http.ResponseWriter, r *http.Request, c Claims) {
	studentID := r.PathValue("studentId")
	if !s.parentLinked(r, c, studentID) {
		fail(w, 403, "this child is not linked to your account")
		return
	}
	term := r.URL.Query().Get("term_id")
	s.list(w, r, `SELECT ts.day_of_week,ts.starts_at,ts.ends_at,ts.room,c.name AS class_name,sb.name AS subject_name,u.full_name AS teacher_name FROM timetable_slots ts JOIN enrollments e ON e.class_id=ts.class_id AND e.term_id=ts.term_id JOIN classes c ON c.id=ts.class_id JOIN subjects sb ON sb.id=ts.subject_id JOIN users u ON u.id=ts.teacher_id WHERE ts.institute_id=$1 AND e.student_id::text=$2 AND ($3='' OR ts.term_id::text=$3) ORDER BY ts.day_of_week,ts.starts_at`, c.InstituteID, studentID, term)
}

func (s *Server) parentResults(w http.ResponseWriter, r *http.Request, c Claims) {
	studentID := r.PathValue("studentId")
	if !s.parentLinked(r, c, studentID) {
		fail(w, 403, "this child is not linked to your account")
		return
	}
	term := r.URL.Query().Get("term_id")
	s.list(w, r, `SELECT sb.name AS subject,a.title,a.assessment_type,a.weight,g.score,a.max_score,ROUND(g.score/a.max_score*100,1)::float8 AS percent FROM grades g JOIN assessments a ON a.id=g.assessment_id JOIN subjects sb ON sb.id=a.subject_id WHERE g.institute_id=$1 AND g.student_id::text=$2 AND ($3='' OR a.term_id::text=$3) ORDER BY sb.name,a.assessment_type,a.title`, c.InstituteID, studentID, term)
}

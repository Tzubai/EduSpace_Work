-- Guard role values at the database boundary as well as in the API.
ALTER TABLE institute_members
    ADD CONSTRAINT institute_members_role_check
    CHECK (role IN ('owner', 'admin', 'bursar', 'teacher', 'parent', 'student'));

-- Common school-scoped lookups and joins. These indexes keep larger schools responsive.
CREATE INDEX IF NOT EXISTS institute_members_user_school_idx
    ON institute_members (user_id, institute_id);
CREATE INDEX IF NOT EXISTS students_school_name_idx
    ON students (institute_id, last_name, first_name);
CREATE INDEX IF NOT EXISTS enrollments_school_class_term_idx
    ON enrollments (institute_id, class_id, term_id);
CREATE INDEX IF NOT EXISTS teacher_assignments_school_user_idx
    ON teacher_assignments (institute_id, user_id, class_id);
CREATE INDEX IF NOT EXISTS attendance_school_day_idx
    ON attendance (institute_id, day);
CREATE INDEX IF NOT EXISTS assessments_school_class_term_idx
    ON assessments (institute_id, class_id, term_id);
CREATE INDEX IF NOT EXISTS grades_school_student_idx
    ON grades (institute_id, student_id, assessment_id);
CREATE INDEX IF NOT EXISTS invoices_school_term_student_idx
    ON fee_invoices (institute_id, term_id, student_id);
CREATE INDEX IF NOT EXISTS payments_school_invoice_paid_idx
    ON payments (institute_id, invoice_id, paid_at DESC);
CREATE INDEX IF NOT EXISTS student_guardians_school_guardian_idx
    ON student_guardians (institute_id, guardian_id);
CREATE INDEX IF NOT EXISTS announcements_school_created_idx
    ON announcements (institute_id, created_at DESC);

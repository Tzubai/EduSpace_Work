CREATE TABLE IF NOT EXISTS admissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    application_number VARCHAR(50) NOT NULL,
    applicant_name VARCHAR(150) NOT NULL,
    guardian_name VARCHAR(150) NOT NULL,
    guardian_phone VARCHAR(30),
    email VARCHAR(255),
    applying_for_class UUID NOT NULL,
    FOREIGN KEY (institute_id, applying_for_class) REFERENCES classes(institute_id, id),
    previous_school TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'submitted' CHECK (status IN ('submitted','under_review','accepted','waitlisted','rejected')),
    notes TEXT,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institute_id, application_number),
    UNIQUE (institute_id, id)
);

CREATE TABLE IF NOT EXISTS timetable_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    term_id UUID NOT NULL,
    class_id UUID NOT NULL,
    subject_id UUID NOT NULL,
    teacher_id UUID NOT NULL,
    day_of_week SMALLINT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    starts_at TIME NOT NULL,
    ends_at TIME NOT NULL,
    room VARCHAR(50),
    FOREIGN KEY (institute_id, term_id) REFERENCES terms(institute_id,id),
    FOREIGN KEY (institute_id, class_id) REFERENCES classes(institute_id,id),
    FOREIGN KEY (institute_id, subject_id) REFERENCES subjects(institute_id,id),
    FOREIGN KEY (teacher_id) REFERENCES users(id),
    UNIQUE (institute_id, class_id, term_id, day_of_week, starts_at)
);

CREATE TABLE IF NOT EXISTS staff_attendance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    user_id UUID NOT NULL,
    day DATE NOT NULL,
    status VARCHAR(12) NOT NULL CHECK (status IN ('present','absent','late','leave')),
    check_in_at TIME,
    check_out_at TIME,
    note TEXT,
    recorded_by UUID REFERENCES users(id),
    UNIQUE (institute_id, user_id, day)
);

CREATE TABLE IF NOT EXISTS notification_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    title VARCHAR(150) NOT NULL,
    body TEXT NOT NULL,
    audience VARCHAR(30) NOT NULL,
    channel VARCHAR(20) NOT NULL DEFAULT 'in_app',
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_by UUID REFERENCES users(id),
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS report_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    report_type VARCHAR(40) NOT NULL,
    filters JSONB NOT NULL DEFAULT '{}'::jsonb,
    generated_by UUID REFERENCES users(id),
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS admissions_school_status_idx ON admissions(institute_id,status,submitted_at DESC);
CREATE INDEX IF NOT EXISTS timetable_school_term_idx ON timetable_slots(institute_id,term_id,class_id,day_of_week);
CREATE INDEX IF NOT EXISTS staff_attendance_school_day_idx ON staff_attendance(institute_id,day,status);
CREATE INDEX IF NOT EXISTS notifications_school_created_idx ON notification_messages(institute_id,created_at DESC);

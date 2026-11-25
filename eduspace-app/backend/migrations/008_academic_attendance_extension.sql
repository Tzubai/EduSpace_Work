-- Academic structure and result extensions.
ALTER TABLE classes
    ADD COLUMN IF NOT EXISTS section VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS arm VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS class_teacher_id UUID;

ALTER TABLE assessments
    ADD COLUMN IF NOT EXISTS assessment_type VARCHAR(20) NOT NULL DEFAULT 'ca1',
    ADD COLUMN IF NOT EXISTS weight NUMERIC(5,2) NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS duration_minutes INTEGER;

ALTER TABLE assessments
    DROP CONSTRAINT IF EXISTS assessments_assessment_type_check;
ALTER TABLE assessments
    ADD CONSTRAINT assessments_assessment_type_check
    CHECK (assessment_type IN ('ca1','ca2','exam','assignment','project','practical'));

ALTER TABLE attendance
    DROP CONSTRAINT IF EXISTS attendance_status_check;
ALTER TABLE attendance
    ADD CONSTRAINT attendance_status_check
    CHECK (status IN ('present','absent','late','excused'));

ALTER TABLE attendance
    ADD COLUMN IF NOT EXISTS arrival_time TIME,
    ADD COLUMN IF NOT EXISTS reason TEXT,
    ADD COLUMN IF NOT EXISTS corrected_by UUID REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS corrected_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS correction_reason TEXT;

CREATE TABLE IF NOT EXISTS schemes_of_work (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    class_id UUID NOT NULL,
    subject_id UUID NOT NULL,
    term_id UUID NOT NULL,
    title VARCHAR(150) NOT NULL,
    topics TEXT,
    week_number INTEGER,
    status VARCHAR(20) NOT NULL DEFAULT 'planned' CHECK (status IN ('planned','in_progress','completed')),
    FOREIGN KEY (institute_id,class_id) REFERENCES classes(institute_id,id),
    FOREIGN KEY (institute_id,subject_id) REFERENCES subjects(institute_id,id),
    FOREIGN KEY (institute_id,term_id) REFERENCES terms(institute_id,id)
);

CREATE TABLE IF NOT EXISTS examination_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    term_id UUID NOT NULL,
    class_id UUID NOT NULL,
    subject_id UUID NOT NULL,
    title VARCHAR(150) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    duration_minutes INTEGER NOT NULL CHECK (duration_minutes > 0),
    venue VARCHAR(100),
    FOREIGN KEY (institute_id,term_id) REFERENCES terms(institute_id,id),
    FOREIGN KEY (institute_id,class_id) REFERENCES classes(institute_id,id),
    FOREIGN KEY (institute_id,subject_id) REFERENCES subjects(institute_id,id),
    UNIQUE (institute_id,class_id,subject_id,starts_at)
);

CREATE TABLE IF NOT EXISTS student_remarks (
    institute_id UUID NOT NULL REFERENCES institutes(id),
    student_id UUID NOT NULL,
    term_id UUID NOT NULL,
    teacher_remark TEXT,
    principal_remark TEXT,
    conduct VARCHAR(100),
    punctuality VARCHAR(100),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (student_id,term_id),
    FOREIGN KEY (institute_id,student_id) REFERENCES students(institute_id,id),
    FOREIGN KEY (institute_id,term_id) REFERENCES terms(institute_id,id)
);

CREATE TABLE IF NOT EXISTS student_promotions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    student_id UUID NOT NULL,
    from_class_id UUID,
    to_class_id UUID,
    term_id UUID NOT NULL,
    decision VARCHAR(20) NOT NULL CHECK (decision IN ('promoted','retained','graduated','transferred')),
    reason TEXT,
    decided_by UUID REFERENCES users(id),
    decided_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (institute_id,student_id) REFERENCES students(institute_id,id),
    FOREIGN KEY (institute_id,from_class_id) REFERENCES classes(institute_id,id),
    FOREIGN KEY (institute_id,to_class_id) REFERENCES classes(institute_id,id),
    FOREIGN KEY (institute_id,term_id) REFERENCES terms(institute_id,id)
);

CREATE TABLE IF NOT EXISTS attendance_corrections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    attendance_id UUID NOT NULL,
    previous_status VARCHAR(10) NOT NULL,
    new_status VARCHAR(10) NOT NULL,
    correction_reason TEXT NOT NULL,
    corrected_by UUID NOT NULL REFERENCES users(id),
    corrected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (attendance_id) REFERENCES attendance(id)
);

CREATE INDEX IF NOT EXISTS schemes_school_term_idx ON schemes_of_work(institute_id,term_id,class_id);
CREATE INDEX IF NOT EXISTS exams_school_term_idx ON examination_schedules(institute_id,term_id,starts_at);
CREATE INDEX IF NOT EXISTS attendance_correction_idx ON attendance_corrections(institute_id,attendance_id,corrected_at DESC);
CREATE INDEX IF NOT EXISTS promotions_school_term_idx ON student_promotions(institute_id,term_id,decision);

-- Every school-owned table carries institute_id. Composite foreign keys
-- (institute_id, id) make the database itself refuse links between schools.

CREATE TABLE academic_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    name VARCHAR(50) NOT NULL,
    UNIQUE (institute_id, name),
    UNIQUE (institute_id, id)
);

CREATE TABLE terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    session_id UUID NOT NULL,
    name VARCHAR(50) NOT NULL,
    starts_on DATE,
    ends_on DATE,
    FOREIGN KEY (institute_id, session_id) REFERENCES academic_sessions (institute_id, id),
    UNIQUE (session_id, name),
    UNIQUE (institute_id, id)
);

CREATE TABLE classes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    name VARCHAR(50) NOT NULL,
    UNIQUE (institute_id, name),
    UNIQUE (institute_id, id)
);

CREATE TABLE subjects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    name VARCHAR(100) NOT NULL,
    UNIQUE (institute_id, name),
    UNIQUE (institute_id, id)
);

CREATE TABLE students (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    admission_number VARCHAR(50) NOT NULL,
    date_of_birth DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institute_id, admission_number),
    UNIQUE (institute_id, id)
);

CREATE TABLE guardians (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    user_id UUID REFERENCES users(id),
    full_name VARCHAR(150) NOT NULL,
    phone VARCHAR(30),
    UNIQUE (institute_id, id)
);

CREATE TABLE student_guardians (
    institute_id UUID NOT NULL REFERENCES institutes(id),
    student_id UUID NOT NULL,
    guardian_id UUID NOT NULL,
    PRIMARY KEY (student_id, guardian_id),
    FOREIGN KEY (institute_id, student_id) REFERENCES students (institute_id, id),
    FOREIGN KEY (institute_id, guardian_id) REFERENCES guardians (institute_id, id)
);

CREATE TABLE teacher_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    user_id UUID NOT NULL REFERENCES users(id),
    class_id UUID NOT NULL,
    subject_id UUID,
    FOREIGN KEY (institute_id, class_id) REFERENCES classes (institute_id, id),
    FOREIGN KEY (institute_id, subject_id) REFERENCES subjects (institute_id, id)
);

CREATE TABLE enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    student_id UUID NOT NULL,
    class_id UUID NOT NULL,
    term_id UUID NOT NULL,
    FOREIGN KEY (institute_id, student_id) REFERENCES students (institute_id, id),
    FOREIGN KEY (institute_id, class_id) REFERENCES classes (institute_id, id),
    FOREIGN KEY (institute_id, term_id) REFERENCES terms (institute_id, id),
    UNIQUE (student_id, term_id),
    UNIQUE (institute_id, id)
);

CREATE TABLE attendance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    enrollment_id UUID NOT NULL,
    day DATE NOT NULL,
    status VARCHAR(10) NOT NULL CHECK (status IN ('present','absent','late')),
    FOREIGN KEY (institute_id, enrollment_id) REFERENCES enrollments (institute_id, id),
    UNIQUE (enrollment_id, day)
);

CREATE TABLE fee_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    student_id UUID NOT NULL,
    term_id UUID NOT NULL,
    description VARCHAR(150) NOT NULL DEFAULT 'School fees',
    amount_kobo BIGINT NOT NULL CHECK (amount_kobo > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'unpaid',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (institute_id, student_id) REFERENCES students (institute_id, id),
    FOREIGN KEY (institute_id, term_id) REFERENCES terms (institute_id, id),
    UNIQUE (institute_id, id)
);

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    invoice_id UUID NOT NULL,
    amount_kobo BIGINT NOT NULL CHECK (amount_kobo > 0),
    method VARCHAR(20) NOT NULL CHECK (method IN ('cash','transfer','pos')),
    reference VARCHAR(100),
    paid_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recorded_by UUID REFERENCES users(id),
    FOREIGN KEY (institute_id, invoice_id) REFERENCES fee_invoices (institute_id, id)
);

CREATE TABLE assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    class_id UUID NOT NULL,
    subject_id UUID NOT NULL,
    term_id UUID NOT NULL,
    title VARCHAR(100) NOT NULL,
    max_score NUMERIC(6,2) NOT NULL CHECK (max_score > 0),
    FOREIGN KEY (institute_id, class_id) REFERENCES classes (institute_id, id),
    FOREIGN KEY (institute_id, subject_id) REFERENCES subjects (institute_id, id),
    FOREIGN KEY (institute_id, term_id) REFERENCES terms (institute_id, id),
    UNIQUE (institute_id, id)
);

CREATE TABLE grades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    assessment_id UUID NOT NULL,
    student_id UUID NOT NULL,
    score NUMERIC(6,2) NOT NULL CHECK (score >= 0),
    FOREIGN KEY (institute_id, assessment_id) REFERENCES assessments (institute_id, id),
    FOREIGN KEY (institute_id, student_id) REFERENCES students (institute_id, id),
    UNIQUE (assessment_id, student_id)
);

-- Each school sets its own grading scale.
CREATE TABLE grade_bands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    min_score NUMERIC(5,2) NOT NULL,
    grade VARCHAR(5) NOT NULL,
    remark VARCHAR(50) NOT NULL DEFAULT '',
    UNIQUE (institute_id, min_score)
);

CREATE TABLE announcements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    title VARCHAR(150) NOT NULL,
    body TEXT NOT NULL,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL UNIQUE REFERENCES institutes(id),
    plan VARCHAR(20) NOT NULL DEFAULT 'starter',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    term_ends_on DATE
);

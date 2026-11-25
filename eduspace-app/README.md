# EduSpace — School Management Platform

EduSpace is being developed as a multi-school management platform for private schools. Each school is intended to have its own branded workspace, with role-based access for school administrators, principals, bursars, teachers and parents. The web experience is designed to be responsive across desktop, tablet and mobile screens.

## Current project contents

### Role-specific workspace update (2026-11-23)
- Teacher and Parent accounts now have a dedicated Dashboard entry in their navigation.
- Teacher dashboard shows assignment cards and direct links to attendance and scores.
- Parent dashboard shows linked-child summary metrics, fee balances, and a route to detailed child records.
- Dashboard widgets use existing school-scoped API data rather than fabricated metrics.
- Additional focus-visible keyboard indicators and responsive role-dashboard layouts.
### Public website
- EduSpace marketing homepage with the supplied logo treatment and navy/teal visual identity.
- Hero section, school-workspace preview, feature carousel, school-focused benefits, onboarding steps and registration calls to action.
- Six carousel topics: student administration, academic management, school finance, teachers and staff, parent engagement, and reports/oversight.
- Carousel supports previous/next controls, slide indicators, touch-friendly layout and timed rotation that pauses on interaction.

### School workspace foundation
- React + TypeScript frontend.
- Go HTTP API.
- PostgreSQL migrations.
- Existing modules for student records, people, setup, attendance, scores, fees, announcements and parent views.
- School-scoped queries and composite foreign keys are present in the current database design.

## Run it (Ubuntu)

```bash
# 1. Database (once)
sudo apt install -y postgresql
sudo -u postgres psql -c "CREATE USER eduspace WITH PASSWORD 'choose_a_password';"
sudo -u postgres psql -c "CREATE DATABASE eduspace OWNER eduspace;"

# 2. Backend
cd backend
go mod tidy
export DATABASE_URL='postgres://eduspace:choose_a_password@localhost:5432/eduspace'
export JWT_SECRET="$(openssl rand -hex 32)"
go run ./cmd/api

# 3. Frontend (second terminal)
cd frontend
npm ci
npm run dev
```

Open http://localhost:5173. The public homepage is shown first. The Sign In and Get Started buttons open the current login/registration interface.

For a single-server deployment, build the frontend with `npm run build` and configure the backend `STATIC_DIR` to the generated `frontend/dist` directory. Put production traffic behind HTTPS.

## Implementation status

EduSpace currently includes a compiling React/TypeScript frontend and Go/PostgreSQL backend with:

- School registration, login, password reset infrastructure, role-based guards and active-account revocation.
- Tenant-isolated students, guardians, classes, subjects, sessions, terms, enrollments and teacher assignments.
- Attendance, assessments, grades, report cards, invoices, payments and announcements.
- Admissions pipeline, timetable builder, staff attendance and notification records.
- Migration advisory locking, checksum validation, duplicate invoice protection and audit events.

The following areas still require provider or production configuration:

- EduSpace Super Admin console and platform-wide approval/suspension controls.
- Subscription trials, plans, recurring billing and capacity enforcement.
- Verified payment-gateway webhooks and reconciliation/refund workflows.
- Email/SMS delivery, backup/restore automation and production monitoring.
- Full PostgreSQL runtime and tenant-isolation verification.

Do not enter real student information until database backups, privacy controls and deployment hardening are complete. The smoke-test script creates fake tenants and records; run it only against a disposable database.

### Database smoke-test setup

An administrator can create a disposable database with:
The admin should have a complete overview of The admin should have a complete overview of what is happening in the school.
- Total students, teachers, and staff.
-
- Student enrollment statistics.
- Attendance overview.
- Fee collection and outstanding fees.
- Academic performance summaries.
- Recent activities.
- Upcoming events.
- Notifications and alerts.
- Quick actions.
- Reports and analytics.
- School-wide announcements.what is happening in the school.
- Total students, teachers, and staff.
- Student enrollment statistics.
- Attendance overview.
- Fee collection and outstanding fees.
- Academic performance summaries.
- Recent activities.
- Upcoming events.
- Notifications and alerts.
- Quick actions.
- Reports and analytics.
- School-wide announcements.
```bash
sudo -u postgres psql -c "CREATE USER eduspace WITH PASSWORD 'choose_a_password';"
sudo -u postgres psql -c "CREATE DATABASE eduspace OWNER eduspace;"
```

Then run:

```bash
cd backend
export DATABASE_URL='postgres://eduspace:choose_a_password@localhost:5432/eduspace'
export JWT_SECRET="$(openssl rand -hex 32)"
go run ./cmd/api

# In another terminal
cd backend
python3 scripts/smoke_test.py http://localhost:8080
```

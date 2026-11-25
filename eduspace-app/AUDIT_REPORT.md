# EduCare source-code review and correction report

Review basis: the supplied `educare-app.zip` archive (source files only). No production database, environment secrets, or live deployment were supplied.

## Corrections made

### Backend and API

- JWT verification now rejects expired tokens at the exact expiry boundary, empty identity claims, and unknown roles.
- Protected routes now require a correctly formatted Bearer authorization header.
- A database failure while checking a member's school membership is reported as a server error rather than incorrectly treating the user as logged out.
- JSON request decoding now rejects trailing JSON values and trailing garbage, in addition to unknown fields and oversized bodies.
- Registration and account creation validate bcrypt's 72-byte password limit. Previously, some code paths ignored bcrypt hashing errors; these are now handled.
- Login now distinguishes an unknown email from a database failure. Database outages are no longer disguised as invalid credentials.
- Login IP parsing now supports IPv4 and IPv6 host/port formats.
- Report-card data access now checks database errors when validating parent-child links and when reading grade bands, scores, attendance, and fees.
- Dashboard queries now stop and return an error if a statistic cannot be read, rather than silently showing zero values.
- Invoice creation validates required IDs and positive amounts before querying the database.
- Assessment creation validates that the maximum score is positive.
- Grading-scale updates validate score boundaries, unique minimum scores, non-empty grade labels, and require a zero-score band.
- The API server now has an idle timeout and graceful shutdown handling.

### Database

- Added migration `003_integrity_and_indexes.sql` to constrain membership roles and add indexes for frequently used school-scoped queries.
- The migration system already applies numbered migrations in order and records successful migrations; the new migration follows that mechanism.

### Frontend

- Date defaults now use the user's local calendar date rather than UTC, avoiding off-by-one dates near midnight.
- Currency conversion now safely rejects empty, invalid, non-finite, zero, and negative amounts instead of producing `NaN` or invalid payment values.
- Data-loading hooks clear stale errors when disabled and clear old data while a new resource is loading.

### Project hygiene

- Updated ignore rules so local environment files remain excluded while `.env.example` stays trackable.
- Expanded the README with dependency, password, migration, backup, and deployment notes.

## Verification performed

- Go source formatting: passed (`gofmt`).
- Python smoke-test script syntax: passed (`py_compile`).
- Frontend JSON manifests and lockfile: valid JSON.
- Go tests: could not run because the supplied project has no `go.sum`, and required Go modules were not available in the execution environment.
- Frontend production build: could not complete because `npm ci` could not finish fetching/installing dependencies in the execution environment; the partial local `node_modules` directory was not included in the corrected archive.
- PostgreSQL migrations and the 39-step smoke test were not run because no PostgreSQL server or configured test database was supplied.

## Required first run on the development machine

From `educare-app/`:

```bash
cd backend
go mod tidy
go test ./...
cd ../frontend
npm ci
npm run build
```

Then start PostgreSQL and the backend using the README instructions. Apply migrations against a backed-up development database and run:

```bash
python3 backend/scripts/smoke_test.py http://localhost:8080
```

Run that smoke test only against a disposable development database; it creates test schools and records.

## Important limitations / follow-up review

This is a source-level correction pass, not a certification for production use. The application still needs runtime integration testing with PostgreSQL, a review of the complete role/permission matrix with the school owner's requirements, backup-and-restore testing, HTTPS deployment configuration, and privacy/data-protection review before real student information is entered.

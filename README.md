# CampusDesk

Modular school management API built in Go. Phase 1 covers schools, students, users, academic structure, enrollments, guardians, and attendance.

## Architecture

```
cmd/server/          # application entrypoint
internal/
  app/               # wires HTTP router and modules
  config/            # environment configuration
  domain/            # shared domain types
  modules/           # feature modules (each self-contained)
    school/
    student/
    user/
    academic/
    enrollment/
    guardian/
    health/
  platform/          # database, HTTP helpers, errors
migrations/          # SQL schema migrations
```

Each module follows **repository → service → handler** and implements `modules.Module` to register routes.

## Prerequisites

- Go 1.17+
- PostgreSQL 16 (or Docker)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI (optional, for migrations)

## Quick start

```bash
# Start database
make docker-up

# Copy env and run migrations
cp .env.example .env
make migrate-up

# Run API
make run
```

API base: `http://localhost:8080/api/v1`

Health: `GET /health`, `GET /ready`

## API overview

| Resource | Endpoints |
|----------|-----------|
| Schools | `GET/POST /schools`, `GET/PUT/DELETE /schools/{id}`, `GET /schools/public` (unauthenticated directory: id/name/code, for the registration form) |
| Students | `GET/POST /students?school_id=`, `GET/PUT/DELETE /students/{id}` |
| Users | `GET/POST /users`, `GET /users?status=pending`, `POST /users/{id}/approve`, `POST /users/{id}/reject` (approve/reject require `super_admin` or `school_admin`), `POST /auth/login`, `POST /auth/register` |
| Academic years | `GET/POST /academic-years?school_id=` |
| Grade levels | `GET/POST /grade-levels?school_id=` |
| Class sections | `GET/POST /class-sections?school_id=&academic_year_id=` |
| Enrollments | `GET/POST /enrollments`, `PUT /enrollments/{id}` |
| Attendance | `GET/POST /attendance` |
| Guardians | `GET/POST /guardians`, `POST /guardians/link`, `GET /guardians/{id}` |

List endpoints support `limit` and `offset` query params (default limit 20).

## Roles

| Role | Scope |
|------|-------|
| `super_admin` | Sees and manages all schools |
| `school_admin` | Scoped to their own school |
| `teacher` | Scoped to their own school |
| `registrar` | Scoped to their own school |
| `parent` | Scoped to their own school |

## Registration & approval

Accounts can be provisioned two ways:

- **Admin-created** — `POST /users` (authenticated) creates a user that is immediately `approved` and active.
- **Self-registration** — `POST /auth/register` (public, no auth) lets someone request an account for
  `school_admin`, `teacher`, `registrar`, or `parent` (not `super_admin`) against an existing school. The
  account is created with `status: pending` and `is_active: false` — it cannot log in yet.

A `super_admin` or `school_admin` reviews pending requests via `GET /users?status=pending` and calls
`POST /users/{id}/approve` or `POST /users/{id}/reject`. Login checks the account's status first and
returns a specific message for pending (`your registration is pending admin approval`) or rejected
(`your registration was rejected`) accounts.

In the frontend, this is exposed as a "Create an account" link on the login page (`/register`) and a
"User Approvals" tab in Settings, visible only to admin roles.

## Example flow

```bash
# Create school
curl -s -X POST localhost:8080/api/v1/schools \
  -H 'Content-Type: application/json' \
  -d '{"name":"Springfield Elementary","code":"SPR-001"}'

# Create admin user directly (already approved)
curl -s -X POST localhost:8080/api/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@school.edu","password":"secret123","first_name":"Admin","last_name":"User","role":"school_admin"}'

# Self-register instead (created as pending — see "Registration & approval" above)
curl -s -X POST localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"teacher@school.edu","password":"secret123","first_name":"Jane","last_name":"Doe","role":"teacher","school_id":"<school-id>"}'

# An admin approves it (requires a super_admin/school_admin bearer token)
curl -s -X POST localhost:8080/api/v1/users/<user-id>/approve \
  -H "Authorization: Bearer <admin-token>"
```

## Development

```bash
make build    # compile binary to bin/campusdesk
make test     # run tests
```

## License

MIT

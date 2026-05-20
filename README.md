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
| Schools | `GET/POST /schools`, `GET/PUT/DELETE /schools/{id}` |
| Students | `GET/POST /students?school_id=`, `GET/PUT/DELETE /students/{id}` |
| Users | `GET/POST /users`, `POST /auth/login` |
| Academic years | `GET/POST /academic-years?school_id=` |
| Grade levels | `GET/POST /grade-levels?school_id=` |
| Class sections | `GET/POST /class-sections?school_id=&academic_year_id=` |
| Enrollments | `GET/POST /enrollments`, `PUT /enrollments/{id}` |
| Attendance | `GET/POST /attendance` |
| Guardians | `GET/POST /guardians`, `POST /guardians/link`, `GET /guardians/{id}` |

List endpoints support `limit` and `offset` query params (default limit 20).

## Example flow

```bash
# Create school
curl -s -X POST localhost:8080/api/v1/schools \
  -H 'Content-Type: application/json' \
  -d '{"name":"Springfield Elementary","code":"SPR-001"}'

# Create admin user
curl -s -X POST localhost:8080/api/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@school.edu","password":"secret123","first_name":"Admin","last_name":"User","role":"school_admin"}'
```

## Development

```bash
make build    # compile binary to bin/campusdesk
make test     # run tests
```

## License

MIT

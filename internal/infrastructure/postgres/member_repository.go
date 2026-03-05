package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/healthcare/member-service/internal/domain/member"
)

// memberRow is the flat database row representation of a Member.
type memberRow struct {
	MemberID        string         `db:"member_id"`
	FirstName       string         `db:"first_name"`
	LastName        string         `db:"last_name"`
	DateOfBirth     time.Time      `db:"date_of_birth"`
	Gender          string         `db:"gender"`
	Email           string         `db:"email"`
	Phone           string         `db:"phone"`
	Street          string         `db:"street"`
	City            string         `db:"city"`
	State           string         `db:"state"`
	ZipCode         string         `db:"zip_code"`
	Country         string         `db:"country"`
	PlanID          string         `db:"plan_id"`
	EnrollmentDate  time.Time      `db:"enrollment_date"`
	TerminationDate sql.NullTime   `db:"termination_date"`
	Status          string         `db:"status"`
	RiskScore       float64        `db:"risk_score"`
	CareLevel       string         `db:"care_level"`
	RiskUpdatedAt   time.Time      `db:"risk_updated_at"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
}

// Repository implements member.Repository using PostgreSQL via sqlx.
type Repository struct {
	db *sqlx.DB
}

// NewRepository constructs a PostgreSQL member repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Open creates and validates a new PostgreSQL connection pool.
func Open(dsn string, maxOpen, maxIdle int, lifetime time.Duration) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening postgres connection: %w", err)
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(lifetime)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}
	return db, nil
}

// Create inserts a new member record.
func (r *Repository) Create(ctx context.Context, m *member.Member) error {
	const q = `
		INSERT INTO members (
			member_id, first_name, last_name, date_of_birth, gender,
			email, phone, street, city, state, zip_code, country,
			plan_id, enrollment_date, termination_date,
			status, risk_score, care_level, risk_updated_at,
			created_at, updated_at
		) VALUES (
			:member_id, :first_name, :last_name, :date_of_birth, :gender,
			:email, :phone, :street, :city, :state, :zip_code, :country,
			:plan_id, :enrollment_date, :termination_date,
			:status, :risk_score, :care_level, :risk_updated_at,
			:created_at, :updated_at
		)`

	row := toRow(m)
	if _, err := r.db.NamedExecContext(ctx, q, row); err != nil {
		if isUniqueViolation(err) {
			return member.ErrMemberAlreadyExists
		}
		return fmt.Errorf("insert member: %w", err)
	}
	return nil
}

// GetByID retrieves a member by UUID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*member.Member, error) {
	const q = `SELECT * FROM members WHERE member_id = $1`
	var row memberRow
	if err := r.db.GetContext(ctx, &row, q, id.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, member.ErrMemberNotFound
		}
		return nil, fmt.Errorf("get member by id: %w", err)
	}
	return fromRow(row)
}

// GetByEmail retrieves a member by email address.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*member.Member, error) {
	const q = `SELECT * FROM members WHERE email = $1`
	var row memberRow
	if err := r.db.GetContext(ctx, &row, q, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, member.ErrMemberNotFound
		}
		return nil, fmt.Errorf("get member by email: %w", err)
	}
	return fromRow(row)
}

// Update persists changes to an existing member.
func (r *Repository) Update(ctx context.Context, m *member.Member) error {
	const q = `
		UPDATE members SET
			first_name = :first_name,
			last_name = :last_name,
			phone = :phone,
			street = :street,
			city = :city,
			state = :state,
			zip_code = :zip_code,
			country = :country,
			plan_id = :plan_id,
			enrollment_date = :enrollment_date,
			termination_date = :termination_date,
			status = :status,
			risk_score = :risk_score,
			care_level = :care_level,
			risk_updated_at = :risk_updated_at,
			updated_at = :updated_at
		WHERE member_id = :member_id`

	row := toRow(m)
	result, err := r.db.NamedExecContext(ctx, q, row)
	if err != nil {
		return fmt.Errorf("update member: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return member.ErrMemberNotFound
	}
	return nil
}

// Delete removes a member record.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM members WHERE member_id = $1`
	result, err := r.db.ExecContext(ctx, q, id.String())
	if err != nil {
		return fmt.Errorf("delete member: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return member.ErrMemberNotFound
	}
	return nil
}

// List returns a paginated, filtered list of members.
func (r *Repository) List(ctx context.Context, filter member.Filter, pagination member.Pagination) ([]*member.Member, int64, error) {
	if pagination.Page < 1 {
		pagination.Page = 1
	}
	if pagination.PageSize < 1 || pagination.PageSize > 100 {
		pagination.PageSize = 20
	}
	offset := (pagination.Page - 1) * pagination.PageSize

	where, args := buildListWhere(filter)

	countQ := "SELECT COUNT(*) FROM members" + where
	var total int64
	if err := r.db.GetContext(ctx, &total, countQ, args...); err != nil {
		return nil, 0, fmt.Errorf("count members: %w", err)
	}

	listQ := "SELECT * FROM members" + where +
		fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d OFFSET %d", pagination.PageSize, offset)

	var rows []memberRow
	if err := r.db.SelectContext(ctx, &rows, listQ, args...); err != nil {
		return nil, 0, fmt.Errorf("list members: %w", err)
	}

	members := make([]*member.Member, 0, len(rows))
	for _, row := range rows {
		m, err := fromRow(row)
		if err != nil {
			return nil, 0, err
		}
		members = append(members, m)
	}
	return members, total, nil
}

// buildListWhere constructs a WHERE clause from a Filter.
func buildListWhere(f member.Filter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	idx := 1

	if f.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, string(*f.Status))
		idx++
	}
	if f.CareLevel != nil {
		conditions = append(conditions, fmt.Sprintf("care_level = $%d", idx))
		args = append(args, string(*f.CareLevel))
		idx++
	}
	if f.PlanID != "" {
		conditions = append(conditions, fmt.Sprintf("plan_id = $%d", idx))
		args = append(args, f.PlanID)
		idx++
	}
	if f.SearchTerm != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d)",
			idx, idx+1, idx+2,
		))
		term := "%" + f.SearchTerm + "%"
		args = append(args, term, term, term)
	}

	if len(conditions) == 0 {
		return "", args
	}
	where := " WHERE "
	for i, c := range conditions {
		if i > 0 {
			where += " AND "
		}
		where += c
	}
	return where, args
}

// toRow converts a domain Member to a flat database row.
func toRow(m *member.Member) memberRow {
	row := memberRow{
		MemberID:       m.MemberID.String(),
		FirstName:      m.FirstName,
		LastName:       m.LastName,
		DateOfBirth:    m.DateOfBirth,
		Gender:         string(m.Gender),
		Email:          m.Contact.Email,
		Phone:          m.Contact.Phone,
		Street:         m.Contact.Address.Street,
		City:           m.Contact.Address.City,
		State:          m.Contact.Address.State,
		ZipCode:        m.Contact.Address.ZipCode,
		Country:        m.Contact.Address.Country,
		PlanID:         m.Enrollment.PlanID,
		EnrollmentDate: m.Enrollment.EnrollmentDate,
		Status:         string(m.Status),
		RiskScore:      m.Risk.RiskScore,
		CareLevel:      string(m.Risk.CareLevel),
		RiskUpdatedAt:  m.Risk.UpdatedAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
	if m.Enrollment.TerminationDate != nil {
		row.TerminationDate = sql.NullTime{Time: *m.Enrollment.TerminationDate, Valid: true}
	}
	return row
}

// fromRow converts a flat database row to a domain Member.
func fromRow(row memberRow) (*member.Member, error) {
	id, err := uuid.Parse(row.MemberID)
	if err != nil {
		return nil, fmt.Errorf("parsing member_id: %w", err)
	}

	m := &member.Member{
		MemberID:    id,
		FirstName:   row.FirstName,
		LastName:    row.LastName,
		DateOfBirth: row.DateOfBirth,
		Gender:      member.Gender(row.Gender),
		Contact: member.ContactInfo{
			Email: row.Email,
			Phone: row.Phone,
			Address: member.Address{
				Street:  row.Street,
				City:    row.City,
				State:   row.State,
				ZipCode: row.ZipCode,
				Country: row.Country,
			},
		},
		Enrollment: member.EnrollmentInfo{
			PlanID:         row.PlanID,
			EnrollmentDate: row.EnrollmentDate,
		},
		Risk: member.RiskProfile{
			RiskScore: row.RiskScore,
			CareLevel: member.CareLevel(row.CareLevel),
			UpdatedAt: row.RiskUpdatedAt,
		},
		Status:    member.MemberStatus(row.Status),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.TerminationDate.Valid {
		t := row.TerminationDate.Time
		m.Enrollment.TerminationDate = &t
	}
	return m, nil
}

// isUniqueViolation detects PostgreSQL unique constraint violations (code 23505).
func isUniqueViolation(err error) bool {
	return err != nil && fmt.Sprintf("%s", err) != "" &&
		containsStr(err.Error(), "23505")
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

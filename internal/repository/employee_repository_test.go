package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/employee-api-db-gin/internal/employee"
	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestRepositoryErrorMapping(t *testing.T) {
	unexpected := errors.New("connection unavailable")
	cases := []struct {
		name string
		err  error
		want error
	}{
		{"not found", gorm.ErrRecordNotFound, employee.ErrNotFound},
		{"duplicate key", gorm.ErrDuplicatedKey, employee.ErrConflict},
		{"unexpected error preserves cause", unexpected, unexpected},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := mapRepositoryError(tc.err); !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
}

func newTestRepository(t *testing.T) (*EmployeeRepository, sqlmock.Sqlmock) {
	t.Helper()
	pool, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{
		TranslateError:       true,
		DisableAutomaticPing: true,
		Logger:               logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		pool.Close()
	})
	return NewEmployeeRepository(db, time.Second), mock
}

func TestUpdateMissingEmployeeDoesNotInsert(t *testing.T) {
	repo, mock := newTestRepository(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE "employees" SET .* WHERE "id" = \$[0-9]+ RETURNING \*`).
		WithArgs("Anand", "a@example.com", "IT", int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "department"}))
	mock.ExpectCommit()
	_, err := repo.Update(context.Background(), 99, employee.EmployeeRequest{Name: "Anand", Email: "a@example.com", Department: "IT"})
	if !errors.Is(err, employee.ErrNotFound) {
		t.Fatalf("got %v want not found", err)
	}
}

func TestCreateReturnsGeneratedEmployee(t *testing.T) {
	repo, mock := newTestRepository(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "employees" .*RETURNING "id"`).
		WithArgs("Anand", "a@example.com", "IT").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
	mock.ExpectCommit()
	item, err := repo.Create(context.Background(), employee.EmployeeRequest{Name: "Anand", Email: "a@example.com", Department: "IT"})
	if err != nil || item.ID != 42 || item.Name != "Anand" {
		t.Fatalf("item=%+v err=%v", item, err)
	}
}

func TestEmptyListIsNonNil(t *testing.T) {
	repo, mock := newTestRepository(t)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "employees"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT \* FROM "employees" ORDER BY "id" LIMIT \$1`).WithArgs(20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "department"}))
	page, err := repo.List(context.Background(), employee.DefaultEmployeeListRequest())
	if err != nil || page.Items == nil || len(page.Items) != 0 || page.TotalItems != 0 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
}

func TestListAppliesSortLimitAndOffset(t *testing.T) {
	repo, mock := newTestRepository(t)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "employees"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery(`SELECT \* FROM "employees" ORDER BY "name" DESC,"id" LIMIT \$1 OFFSET \$2`).WithArgs(2, 2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "department"}).
			AddRow(5, "Bob", "bob@example.com", "IT").AddRow(2, "Alice", "alice@example.com", "HR"))
	page, err := repo.List(context.Background(), employee.EmployeeListRequest{Page: 2, PageSize: 2, SortBy: "name", SortOrder: "desc"})
	if err != nil || page.TotalItems != 5 || len(page.Items) != 2 || page.Items[0].ID != 5 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
}

func TestListRejectsUnsafeSortWithoutQueryingDatabase(t *testing.T) {
	repo, _ := newTestRepository(t)
	request := employee.DefaultEmployeeListRequest()
	request.SortBy = "name; DROP TABLE employees"
	if _, err := repo.List(context.Background(), request); !errors.Is(err, employee.ErrInvalid) {
		t.Fatalf("got %v want invalid input", err)
	}
}
func TestDeleteMissingEmployee(t *testing.T) {
	repo, mock := newTestRepository(t)
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "employees" WHERE id = \$1`).WithArgs(int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	if err := repo.Delete(context.Background(), 99); !errors.Is(err, employee.ErrNotFound) {
		t.Fatalf("got %v want not found", err)
	}
}

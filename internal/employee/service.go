package employee

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, request EmployeeListRequest) (EmployeePage, error) {
	if err := request.Validate(); err != nil {
		return EmployeePage{}, err
	}
	return s.repo.List(ctx, request)
}

func (s *Service) Get(ctx context.Context, id int64) (Employee, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) Create(ctx context.Context, in EmployeeRequest) (Employee, error) {
	in, err := validate(in)
	if err != nil {
		return Employee{}, err
	}
	return s.repo.Create(ctx, in)
}

func (s *Service) Update(ctx context.Context, id int64, in EmployeeRequest) (Employee, error) {
	in, err := validate(in)
	if err != nil {
		return Employee{}, err
	}
	return s.repo.Update(ctx, id, in)
}

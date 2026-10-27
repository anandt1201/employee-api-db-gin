package employee

import (
	"fmt"
	"net/mail"
	"strings"
)

func validate(in EmployeeRequest) (EmployeeRequest, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Department = strings.TrimSpace(in.Department)
	if len(in.Name) == 0 || len(in.Name) > 100 {
		return in, fmt.Errorf("%w: name must contain 1–100 bytes", ErrInvalid)
	}
	addr, err := mail.ParseAddress(in.Email)
	if err != nil || addr.Address != in.Email || len(in.Email) > 254 {
		return in, fmt.Errorf("%w: email must be a valid address", ErrInvalid)
	}
	if len(in.Department) == 0 || len(in.Department) > 100 {
		return in, fmt.Errorf("%w: department must contain 1–100 bytes", ErrInvalid)
	}
	return in, nil
}

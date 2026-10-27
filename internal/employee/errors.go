package employee

import "errors"

var ErrNotFound = errors.New("employee not found")
var ErrConflict = errors.New("email already exists")
var ErrInvalid = errors.New("invalid employee")

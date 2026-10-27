CREATE TABLE employees (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL CHECK (octet_length(name) BETWEEN 1 AND 100 AND btrim(name) <> ''),
    email TEXT NOT NULL CHECK (octet_length(email) BETWEEN 1 AND 254),
    department TEXT NOT NULL CHECK (octet_length(department) BETWEEN 1 AND 100 AND btrim(department) <> '')
);

-- The database enforces uniqueness atomically, including concurrent requests.
CREATE UNIQUE INDEX employees_email_lower_key ON employees (lower(email));

package student

import "github.com/uptrace/bun"

type Student struct {
	bun.BaseModel `bun:"table:students,alias:s"`

	ID        int    `bun:"id,pk,autoincrement" json:"id"`
	FirstName string `bun:"first_name,notnull" json:"first_name"`
	LastName  string `bun:"last_name,notnull" json:"last_name"`
	Email     string `bun:"email,unique,notnull" json:"email"`
	Major     string `bun:"major" json:"major"`
	Year      int    `bun:"year" json:"year"`
}

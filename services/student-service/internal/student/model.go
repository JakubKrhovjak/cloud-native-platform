package student

import "github.com/uptrace/bun"

type Student struct {
	bun.BaseModel `bun:"table:students,alias:s"`

	ID        int    `bun:"id,pk,autoincrement" json:"id"`
	FirstName string `bun:"first_name,notnull" json:"firstName" validate:"required"`
	LastName  string `bun:"last_name,notnull" json:"lastName" validate:"required"`
	Email     string `bun:"email,unique,notnull" json:"email" validate:"required,email"`
	Password  string `bun:"password,notnull" json:"-"` // Never expose password in JSON
	Major     string `bun:"major" json:"major"`
	Year      int    `bun:"year" json:"year" validate:"min=0,max=10"`
}

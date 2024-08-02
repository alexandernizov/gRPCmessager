package domain

type Chat struct {
	Uuid     string
	Owner    User
	Readonly bool
	Deadline int64
}

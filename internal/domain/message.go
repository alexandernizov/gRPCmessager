package domain

type Message struct {
	Uuid       string
	AuthorUuid string
	Body       string
	Published  int64
}

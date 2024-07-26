package domain

type Message struct {
	Uuid       string `redis:"uuid"`
	AuthorUuid string `redis:"authorUuid"`
	Body       string `redis:"body"`
	Published  int64  `redis:"published"`
}

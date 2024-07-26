package domain

type Chat struct {
	Uuid      string `redis:"uuid"`
	OwnerUuid string `redis:"ownerUuid"`
	Readonly  bool   `redis:"readOnly"`
	Deadline  int64  `redis:"ttl"`
}

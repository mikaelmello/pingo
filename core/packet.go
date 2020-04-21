package core

type rawPacket struct {
	content []byte
	length  int
	ttl     int
}

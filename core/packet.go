package core

// Raw packet read from the connection and used to pass information to the session
type rawPacket struct {
	content []byte
	length  int
	ttl     int
}

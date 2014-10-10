package link

import (
    "fmt"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/gob"
	"hash/adler32"
)

const (
	CONNECT = iota
	ACK
	PING
	DATA
)

type linkMessage struct {
	Kind uint8
	Data []byte
}

func Init() {
    gob.Register(linkMessage{})
}

func decodeMessage(b64data []byte) (linkMessage, error) {
	if b64data[len(b64data) - 1] == '+' {
	    b64data = b64data[0:len(b64data) - 1]
	}
	data := make([]byte,base64.StdEncoding.DecodedLen(len(b64data)))
	_,err := base64.StdEncoding.Decode(data,b64data)
	if err != nil {
		return linkMessage{}, err
	}
	if len(data) < 5 {
	    return linkMessage{},fmt.Errorf("message must be at least 5 bytes")
	}
    checksum :=  binary.BigEndian.Uint32(data)
    gobbuff := bytes.NewBuffer(data[4:])
    
    if checksum != adler32.Checksum(gobbuff.Bytes()) {
        return linkMessage{}, err
    }
    
    dec := gob.NewDecoder(gobbuff)
    
    var ret linkMessage
    err = dec.Decode(ret)
    if err != nil {
        return linkMessage{}, err
    }
    
	return ret,nil
}

func encodeMessage(m *linkMessage) ([]byte, error) {
	var rawencoded bytes.Buffer
	enc := gob.NewEncoder(&rawencoded)
	//First encode the value
	err := enc.Encode(m)
	if err != nil {
		return []byte{}, err
	}
    gobBytes := rawencoded.Bytes()
	cksumWithGobBytes := make([]byte,len(gobBytes) + 4)
	binary.BigEndian.PutUint32(cksumWithGobBytes,adler32.Checksum(gobBytes))
	for idx := range gobBytes {
	    cksumWithGobBytes[4 + idx] = gobBytes[idx]
	}
	var b64encoded []byte = make([]byte,base64.StdEncoding.EncodedLen(len(cksumWithGobBytes)) + 1)
	base64.StdEncoding.Encode(b64encoded,cksumWithGobBytes)
	// Add delimiter
	b64encoded[len(b64encoded) - 1] = '+'
	return b64encoded, nil
}

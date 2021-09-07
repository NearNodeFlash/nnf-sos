package common

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	ec "stash.us.cray.com/rabsw/nnf-ec/pkg/ec"
)

// Params -
func Params(r *http.Request) map[string]string {
	return mux.Vars(r)
}

// UnmarshalRequest -
func UnmarshalRequest(r *http.Request, model interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, &model)
}

// EncodeResponse -
func EncodeResponse(s interface{}, err error, w http.ResponseWriter) {
	if err != nil {
		log.WithError(err).Warn("Element Controller Error")
	}

	ec.EncodeResponse(s, err, w)
}

const (
	NamespaceMetadataSignature = 0x54424252 // "RBBT"
	NamespaceMetadataRevision  = 1
)

type NamespaceMetadata struct {
	Signature uint32
	Revision  uint16
	Rsvd      uint16
	Index     uint16
	Count     uint16
	Id        uuid.UUID
}

type NamespaceMetadataError struct {
	data *NamespaceMetadata
}

func NewNamespaceMetadataError(data *NamespaceMetadata) *NamespaceMetadataError {
	return &NamespaceMetadataError{data: data}
}

func (e *NamespaceMetadataError) Error() string {
	if NamespaceMetadataSignature != e.data.Signature {
		return fmt.Sprintf("Namespace Metadata Signature Invalid: Expected: %#08x Actual: %#08x", NamespaceMetadataSignature, e.data.Signature)
	}

	if NamespaceMetadataRevision != e.data.Revision {
		return fmt.Sprintf("Namespace Metadata Revision Invalid: Expected: %d Actual: %d", NamespaceMetadataRevision, e.data.Revision)
	}

	return "Unknown"
}

func (e *NamespaceMetadataError) Is(err error) bool {
	_, ok := err.(*NamespaceMetadataError)
	return ok
}

var (
	ErrNamespaceMetadata = NewNamespaceMetadataError(nil)
)

func EncodeNamespaceMetadata(pid uuid.UUID, index uint16, count uint16) ([]byte, error) {
	buf := new(bytes.Buffer)

	md := NamespaceMetadata{
		Signature: NamespaceMetadataSignature,
		Revision:  NamespaceMetadataRevision,
		Index:     index,
		Count:     count,
		Id:        pid,
	}

	err := binary.Write(buf, binary.LittleEndian, md)

	return buf.Bytes(), err
}

func DecodeNamespaceMetadata(buf []byte) (*NamespaceMetadata, error) {

	data := new(NamespaceMetadata)

	if err := binary.Read(bytes.NewReader(buf), binary.LittleEndian, data); err != nil {
		return nil, err
	}

	if (data.Signature != NamespaceMetadataSignature) || (data.Revision != NamespaceMetadataRevision) {
		return data, NewNamespaceMetadataError(data)
	}

	return data, nil
}

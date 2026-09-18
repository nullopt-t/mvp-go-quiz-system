package presenter

import (
	"encoding/base64"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Cursor represents an opaque keyset pagination token
type Cursor struct {
	Val string
	ID  primitive.ObjectID
}

// EncodeCursor serializes value and objectID to base64 url-safe token
func EncodeCursor(val string, id primitive.ObjectID) string {
	if id.IsZero() {
		return ""
	}
	raw := fmt.Sprintf("%s|%s", val, id.Hex())
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses an opaque token back to val and objectID
func DecodeCursor(token string) (Cursor, error) {
	if token == "" {
		return Cursor{}, nil
	}
	bytes, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return Cursor{}, err
	}
	parts := strings.Split(string(bytes), "|")
	if len(parts) != 2 {
		return Cursor{}, fmt.Errorf("malformed cursor token")
	}
	objID, err := primitive.ObjectIDFromHex(parts[1])
	if err != nil {
		return Cursor{}, err
	}
	return Cursor{
		Val: parts[0],
		ID:  objID,
	}, nil
}

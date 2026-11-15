package dto

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"job-management-service/internal/httpx"
)

// hexIDs transforma o lista de ObjectID in text. O lista goala devine [] in JSON, nu null.
func hexIDs(ids []bson.ObjectID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.Hex())
	}
	return out
}

// Ref este forma scurta a unei resurse dintr-o sub-colectie: doar id-ul si link-urile.
// La /computers/{id}/scripts e forma implicita; ?verbose cere reprezentarea completa (L3 2.1).
type Ref struct {
	ID    string      `json:"id"`
	Links httpx.Links `json:"_links"`
}

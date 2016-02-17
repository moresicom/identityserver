package models

import (
	"gopkg.in/mgo.v2"

	"github.com/itsyouonline/identityserver/db"
)

// Initialize models in DB, if required.
func InitModels() {
	// TODO: Use model tags to ensure indices/constraints.
	index := mgo.Index{
		Key:      []string{"username"},
		Unique:   true,
		DropDups: true,
	}

	db.EnsureIndex(COLLECTION_USERS, index)
}
package tgbot

import (
	"context"
	"strconv"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/db"
	"google.golang.org/api/iterator"
)

type Firebase[BOTDATA any, USERDATA any] struct {
	Firestore *firestore.Client
	Database  *db.Client
	Context   context.Context
}

func (fb *Firebase[BOTDATA, USERDATA]) GetUsers() ([]*User[USERDATA], error) {
	users := make([]*User[USERDATA], 0)

	iter := fb.Firestore.Collection("users").Documents(fb.Context)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var user User[USERDATA]
		doc.DataTo(&user)
		users = append(users, &user)
	}

	return users, nil
}

func (fb *Firebase[BOTDATA, USERDATA]) GetUser(id int64) (*User[USERDATA], error) {
	iter := fb.Firestore.Collection("users").Where("id", "==", id).Documents(fb.Context)

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var user *User[USERDATA]
	err = doc.DataTo(&user)
	return user, err
}

func (fb *Firebase[BOTDATA, USERDATA]) UpdateUser(user *User[USERDATA]) error {
	id := strconv.FormatInt(user.ID, 10)

	_, err := fb.Firestore.Collection("users").Doc(id).Set(fb.Context, user)

	return err
}

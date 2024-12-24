package tgbot

import (
	"context"
	"strconv"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/db"
	"google.golang.org/api/iterator"
)

type Firebase struct {
	Firestore *firestore.Client
	Database  *db.Client
	Context   context.Context
}

func (fb *Firebase) getUsers() ([]*User, error) {
	users := make([]*User, 0)

	iter := fb.Firestore.Collection("users").Documents(fb.Context)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var user User
		doc.DataTo(&user)
		users = append(users, &user)
	}

	return users, nil
}

func (fb *Firebase) getUser(id int64) (*User, error) {
	iter := fb.Firestore.Collection("users").Where("id", "==", id).Documents(fb.Context)

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var user *User
	err = doc.DataTo(&user)
	return user, err
}

func (fb *Firebase) updateUser(user *User) error {
	id := strconv.FormatInt(user.ID, 10)

	_, err := fb.Firestore.Collection("users").Doc(id).Set(fb.Context, user)

	return err
}

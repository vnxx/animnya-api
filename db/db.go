package db

import (
	"fmt"
	"os"
)

type DBInterface interface {
	Get(path string, id *string) (*[]byte, error)
	Save(path string, id *string, content *[]byte) error
}

func New() *DB {
	return &DB{}
}

type DB struct{}

func (db *DB) Get(path string, id *string) (*[]byte, error) {
	if id == nil {
		return nil, fmt.Errorf("ID_NOT_FOUND")
	}

	db.checkFolder(path)

	body, err := os.ReadFile("./.db/" + path + *id + ".animenya")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("NOT_FOUND")
		}
		return nil, err
	}

	return &body, nil
}

func (db *DB) Save(path string, id *string, content *[]byte) error {
	if id == nil {
		return fmt.Errorf("ID_NOT_FOUND")
	}

	if content == nil {
		return fmt.Errorf("CONTENT_NOT_FOUND")
	}

	db.checkFolder(path)

	var file *os.File
	var err error
	file, err = os.Create("./.db/" + path + *id + ".animenya")
	if err != nil {
		if !os.IsExist(err) {
			return err
		}

		file, err = os.OpenFile("./.db/"+path+*id+".animenya", os.O_RDWR, 0644)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	_, err = file.Write(*content)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) checkFolder(path string) error {
	if _, err := os.Stat("./.db"); os.IsNotExist(err) {
		err = os.Mkdir("./.db", os.ModePerm)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat("./.db/" + path); os.IsNotExist(err) {
		err = os.Mkdir("./.db/"+path, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

package db

import (
	"fmt"
	"os"
)

type DBInterface interface {
	Get(animeID *string) (*[]byte, error)
	Save(animeID *string, content *[]byte) error
}

func New() *DB {
	return &DB{}
}

type DB struct{}

func (db *DB) Get(animeID *string) (*[]byte, error) {
	if animeID == nil {
		return nil, fmt.Errorf("ID_NOT_FOUND")
	}

	body, err := os.ReadFile("./.db/" + *animeID + ".animenya")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("NOT_FOUND")
		}
		return nil, err
	}

	return &body, nil

}

func (db *DB) Save(animeID *string, content *[]byte) error {
	if animeID == nil {
		return fmt.Errorf("ID_NOT_FOUND")
	}

	if content == nil {
		return fmt.Errorf("CONTENT_NOT_FOUND")
	}

	if _, err := os.Stat("./.db"); os.IsNotExist(err) {
		err = os.Mkdir("./.db", os.ModePerm)
		if err != nil {
			return err
		}
	}

	var file *os.File
	var err error
	file, err = os.Create("./.db/" + *animeID + ".animenya")
	if err != nil {
		if !os.IsExist(err) {
			return err
		}

		file, err = os.OpenFile("./.db/"+*animeID+".animenya", os.O_RDWR, 0644)
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

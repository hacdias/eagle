package atproto

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/xrpc"
)

type bookHiveStatus string

const (
	bookHiveStatusFinished   bookHiveStatus = "buzz.bookhive.defs#finished"
	bookHiveStatusReading    bookHiveStatus = "buzz.bookhive.defs#reading"
	bookHiveStatusWantToRead bookHiveStatus = "buzz.bookhive.defs#wantToRead"
	bookHiveStatusAbandoned  bookHiveStatus = "buzz.bookhive.defs#abandoned"
)

type bookHiveBook struct {
	Stars       int            `json:"stars"`
	Title       string         `json:"title"`
	HiveID      string         `json:"hiveId"`
	Status      bookHiveStatus `json:"status"`
	Authors     string         `json:"authors"`
	CreatedAt   time.Time      `json:"createdAt"`
	StartedAt   time.Time      `json:"startedAt,omitempty"`
	FinishedAt  time.Time      `json:"finishedAt,omitempty"`
	Identifiers struct {
		HiveID      string `json:"hiveId"`
		ISBN10      string `json:"isbn10,omitempty"`
		ISBN13      string `json:"isbn13,omitempty"`
		GoodreadsID string `json:"goodreadsId,omitempty"`
	} `json:"identifiers"`
}

type reading struct {
	Name      string    `json:"name"`
	Author    string    `json:"author"`
	Date      time.Time `json:"date,omitempty"`
	Rating    int       `json:"rating,omitempty"`
	Publisher string    `json:"publisher,omitempty"`
	Pages     int       `json:"pages,omitempty"`
	UID       string    `json:"uid,omitempty"`
}

func getBookHiveBooks(ctx context.Context, client *xrpc.Client) ([]*bookHiveBook, error) {
	records, err := listRecords(ctx, client, "buzz.bookhive.book")
	if err != nil {
		return nil, err
	}

	books := []*bookHiveBook{}
	for _, record := range records {
		bean := &bookHiveBook{}
		err = json.Unmarshal(*record.Value, bean)
		if err != nil {
			return nil, err
		}
		books = append(books, bean)
	}

	return books, nil
}

func fetchAndProcessBookHiveBooks(ctx context.Context, client *xrpc.Client) ([]reading, error) {
	books, err := getBookHiveBooks(ctx, client)
	if err != nil {
		return nil, err
	}

	readings := []reading{}
	for _, book := range books {
		// For now, ignore books that ain't finished.
		if book.Status != bookHiveStatusFinished {
			continue
		}

		readings = append(readings, reading{
			Name:   book.Title,
			Author: book.Authors,
			Rating: book.Stars,
			Date:   book.FinishedAt,
			UID:    fmt.Sprintf("isbn:%s", book.Identifiers.ISBN13),
		})
	}

	return readings, nil
}

func (at *ATProto) UpdateReadings(ctx context.Context) error {
	// if at.arabicaFilename == "" {
	// 	return nil
	// }

	// client, err := at.getClient(ctx)
	// if err != nil {
	// 	return err
	// }

	// coffeeBags, err := fetchAndProcessArabicaData(ctx, client)
	// if err != nil {
	// 	return err
	// }

	// var oldCoffeeBags []coffeeBag
	// err = at.core.ReadJSON(at.arabicaFilename, &oldCoffeeBags)
	// if err != nil && !os.IsNotExist(err) {
	// 	return err
	// }

	// if reflect.DeepEqual(oldCoffeeBags, coffeeBags) {
	// 	return nil
	// }

	// coffeeBagsBytes, err := json.MarshalIndent(coffeeBags, "", "  ")
	// if err != nil {
	// 	return err
	// }

	// return at.core.WriteFile(at.arabicaFilename, coffeeBagsBytes, "atproto: synchronize readings data")
	return nil
}

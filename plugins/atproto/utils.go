package atproto

import (
	"bytes"
	"context"
	"encoding/json"
	"math/rand/v2"
	"reflect"

	"github.com/bluesky-social/indigo/api/agnostic"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"go.hacdias.com/eagle/server"
)

var clockId = uint(rand.Uint64())

type photoBlob struct {
	blob   *lexutil.LexBlob
	alt    string
	width  int
	height int
}

func uploadPhoto(ctx context.Context, client *xrpc.Client, photo *server.Photo) (*photoBlob, error) {
	resp, err := atproto.RepoUploadBlob(ctx, client, bytes.NewReader(photo.Data))
	if err != nil {
		return nil, err
	}

	return &photoBlob{
		blob: &lexutil.LexBlob{
			Ref:      resp.Blob.Ref,
			MimeType: photo.MimeType,
			Size:     resp.Blob.Size,
		},
		alt:    photo.Title,
		width:  photo.Width,
		height: photo.Height,
	}, nil
}

func uploadPhotos(ctx context.Context, client *xrpc.Client, photos []*server.Photo) ([]*photoBlob, error) {
	uploaded := make([]*photoBlob, 0, len(photos))

	for _, photo := range photos {
		up, err := uploadPhoto(ctx, client, photo)
		if err != nil {
			return nil, err
		}

		uploaded = append(uploaded, up)
	}

	return uploaded, nil
}

func deleteRecord(ctx context.Context, client *xrpc.Client, collection, recordKey string) error {
	_, err := atproto.RepoDeleteRecord(ctx, client, &atproto.RepoDeleteRecord_Input{
		Collection: collection,
		Repo:       client.Auth.Did,
		Rkey:       recordKey,
	})

	return err
}

func createRecord(ctx context.Context, client *xrpc.Client, collection string, recordKey *string, record map[string]any) (string, error) {
	result, err := agnostic.RepoCreateRecord(ctx, client, &agnostic.RepoCreateRecord_Input{
		Collection: collection,
		Repo:       client.Auth.Did,
		Record:     record,
		Rkey:       recordKey,
	})
	if err != nil {
		return "", err
	}

	return result.Uri, nil
}

func listRecords(ctx context.Context, client *xrpc.Client, collection string) ([]*agnostic.RepoListRecords_Record, error) {
	records := []*agnostic.RepoListRecords_Record{}
	cursor := ""

	for {
		resp, err := agnostic.RepoListRecords(ctx, client, collection, cursor, 100, client.Auth.Did, false)
		if err != nil {
			return nil, err
		}
		records = append(records, resp.Records...)
		if resp.Cursor != nil && *resp.Cursor != "" {
			cursor = *resp.Cursor
		} else {
			break
		}
	}

	return records, nil
}

func putRecord(ctx context.Context, client *xrpc.Client, collection, recordKey string, record map[string]any) (string, error) {
	// Check if the record exists and is the same, if so, return the existing URI
	if result, err := agnostic.RepoGetRecord(ctx, client, "", collection, client.Auth.Did, recordKey); err == nil {
		var currentRecord map[string]any
		err = json.Unmarshal(*result.Value, &currentRecord)
		if err != nil {
			return "", err
		}

		// Normalize new record by marshalling and unmarshalling it, ensuring
		// that the value types are the same.
		recordData, err := json.Marshal(record)
		if err != nil {
			return "", err
		}

		var normalizedRecord map[string]any
		err = json.Unmarshal(recordData, &normalizedRecord)
		if err != nil {
			return "", err
		}

		// Compare
		if reflect.DeepEqual(normalizedRecord, currentRecord) {
			return result.Uri, nil
		}
	}

	// Otherwise, update the record
	result, err := agnostic.RepoPutRecord(ctx, client, &agnostic.RepoPutRecord_Input{
		Collection: collection,
		Repo:       client.Auth.Did,
		Rkey:       recordKey,
		Record:     record,
	})
	if err != nil {
		return "", err
	}

	return result.Uri, nil
}

func blueskyPostToPhotoBlobs(posts []*blueskyPost) []*photoBlob {
	var photos []*photoBlob
	for _, post := range posts {
		if post.Embed == nil || post.Embed.EmbedImages == nil {
			continue
		}
		for _, img := range post.Embed.EmbedImages.Images {
			photo := &photoBlob{
				blob: img.Image,
				alt:  img.Alt,
			}
			if img.AspectRatio != nil {
				photo.width = int(img.AspectRatio.Width)
				photo.height = int(img.AspectRatio.Height)
			}
			photos = append(photos, photo)
		}
	}
	return photos
}

func uploadedPhotoBlobsToEmbeddings(photos []*photoBlob) []*bsky.EmbedImages_Image {
	embeddings := make([]*bsky.EmbedImages_Image, 0, len(photos))
	for _, photo := range photos {
		embedding := &bsky.EmbedImages_Image{
			Image: photo.blob,
			Alt:   photo.alt,
		}
		if photo.width > 0 && photo.height > 0 {
			embedding.AspectRatio = &bsky.EmbedDefs_AspectRatio{
				Width:  int64(photo.width),
				Height: int64(photo.height),
			}
		}
		embeddings = append(embeddings, embedding)
	}
	return embeddings
}

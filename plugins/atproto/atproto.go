package atproto

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"

	"github.com/bluesky-social/indigo/api/agnostic"
	"github.com/bluesky-social/indigo/api/atproto"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"go.hacdias.com/eagle/server"
)

func uploadPhoto(ctx context.Context, client *xrpc.Client, photo *server.Photo) (*lexutil.LexBlob, error) {
	resp, err := atproto.RepoUploadBlob(ctx, client, bytes.NewReader(photo.Data))
	if err != nil {
		return nil, err
	}

	return &lexutil.LexBlob{
		Ref:      resp.Blob.Ref,
		MimeType: photo.MimeType,
		Size:     resp.Blob.Size,
	}, nil
}

func deleteRecord(ctx context.Context, client *xrpc.Client, collection, recordKey string) error {
	_, err := atproto.RepoDeleteRecord(ctx, client, &atproto.RepoDeleteRecord_Input{
		Collection: collection,
		Repo:       client.Auth.Did,
		Rkey:       recordKey,
	})

	return err
}

func upsertRecord(ctx context.Context, client *xrpc.Client, collection, recordKey string, record map[string]any) (string, error) {
	// Create if there's no recordKey known
	if recordKey == "" {
		result, err := agnostic.RepoCreateRecord(ctx, client, &agnostic.RepoCreateRecord_Input{
			Collection: collection,
			Repo:       client.Auth.Did,
			Record:     record,
		})
		if err != nil {
			return "", err
		}

		return result.Uri, nil
	}

	// Check if the record exists and is the same, if so, return the existing URI
	if result, err := agnostic.RepoGetRecord(ctx, client, "", collection, client.Auth.Did, recordKey); err == nil {
		var currentRecord map[string]any
		err = json.Unmarshal(*result.Value, &currentRecord)
		if err != nil {
			return "", err
		}

		if reflect.DeepEqual(record, currentRecord) {
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

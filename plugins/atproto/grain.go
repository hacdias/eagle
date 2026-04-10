package atproto

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/uber/h3-go/v4"
	"go.hacdias.com/eagle/core"
)

func (at *ATProto) createGrainGallery(ctx context.Context, client *xrpc.Client, e *core.Entry, photos []*photoBlob) (string, error) {
	for i, photo := range photos {
		if photo.width <= 0 || photo.height <= 0 {
			return "", fmt.Errorf("photo %d has invalid dimensions (%dx%d): grain requires aspectRatio with minimum 1", i, photo.width, photo.height)
		}
	}

	// 1. Create the gallery record.
	title := e.Title
	if len(title) > 100 {
		title = title[:100]
	}

	galleryRecord := map[string]any{
		"$type":     "social.grain.gallery",
		"title":     title,
		"createdAt": e.Date.Format(syntax.AtprotoDatetimeLayout),
	}

	if e.Description != "" {
		galleryRecord["description"] = e.Description
	} else if summary := e.Summary(); summary != "" {
		galleryRecord["description"] = summary
	}

	if e.Location != nil {
		if e.Location.Latitude != 0 && e.Location.Longitude != 0 {
			cell, err := h3.LatLngToCell(h3.NewLatLng(e.Location.Latitude, e.Location.Longitude), 7)
			if err == nil {
				location := map[string]any{
					"$type": "community.lexicon.location.hthree",
					"value": cell.String(),
				}
				if e.Location.Name != "" {
					location["name"] = e.Location.Name
				} else {
					nameParts := []string{}
					if e.Location.Locality != "" {
						nameParts = append(nameParts, e.Location.Locality)
					}
					if e.Location.Region != "" {
						nameParts = append(nameParts, e.Location.Region)
					}
					if e.Location.Country != "" {
						nameParts = append(nameParts, e.Location.Country)
					}
					if len(nameParts) > 0 {
						location["name"] = strings.Join(nameParts, ", ")
					}

				}
				galleryRecord["location"] = location
			}
		}

		if e.Location.CountryCode != "" {
			address := map[string]any{
				"$type":   "community.lexicon.location.address",
				"country": e.Location.CountryCode,
			}
			if e.Location.PostalCode != "" {
				address["postalCode"] = e.Location.PostalCode
			}
			if e.Location.Region != "" {
				address["region"] = e.Location.Region
			}
			if e.Location.Locality != "" {
				address["locality"] = e.Location.Locality
			}
			if e.Location.Name != "" {
				address["name"] = e.Location.Name
			}
			galleryRecord["address"] = address
		}
	}

	galleryRecordKey := syntax.NewTID(e.Date.UnixMicro(), clockId).String()
	galleryURI, err := createRecord(ctx, client, "social.grain.gallery", &galleryRecordKey, galleryRecord)
	if err != nil {
		return "", fmt.Errorf("failed to create social.grain.gallery: %w", err)
	}
	at.log.Infow("created social.grain.gallery", "uri", galleryURI)

	// 2. Create photo records.
	photoURIs := make([]string, 0, len(photos))
	for i, photo := range photos {
		createdAt := e.Date.Add(time.Duration(i) * time.Second)
		recordKey := syntax.NewTID(createdAt.UnixMicro(), clockId).String()

		photoRecord := map[string]any{
			"$type": "social.grain.photo",
			"photo": photo.blob,
			"aspectRatio": map[string]any{
				"width":  photo.width,
				"height": photo.height,
			},
			"createdAt": createdAt.Format(syntax.AtprotoDatetimeLayout),
		}
		if photo.alt != "" {
			photoRecord["alt"] = photo.alt
		}

		photoURI, err := createRecord(ctx, client, "social.grain.photo", &recordKey, photoRecord)
		if err != nil {
			return "", fmt.Errorf("failed to create social.grain.photo: %w", err)
		}
		at.log.Infow("created social.grain.photo", "uri", photoURI)
		photoURIs = append(photoURIs, photoURI)
	}

	// 3. Create gallery item records linking photos to the gallery.
	for i, photoURI := range photoURIs {
		createdAt := e.Date.Add(time.Duration(i) * time.Second)
		recordKey := syntax.NewTID(createdAt.UnixMicro(), clockId).String()

		itemRecord := map[string]any{
			"$type":     "social.grain.gallery.item",
			"gallery":   galleryURI,
			"item":      photoURI,
			"position":  i,
			"createdAt": createdAt.Format(syntax.AtprotoDatetimeLayout),
		}

		itemURI, err := createRecord(ctx, client, "social.grain.gallery.item", &recordKey, itemRecord)
		if err != nil {
			return "", fmt.Errorf("failed to create social.grain.gallery.item: %w", err)
		}
		at.log.Infow("created social.grain.gallery.item", "uri", itemURI)
	}

	return galleryURI, nil
}

func (at *ATProto) deleteGrainGallery(ctx context.Context, client *xrpc.Client, uri syntax.ATURI) error {
	// TODO: Implement deletion of the gallery record, photo records, and gallery item records.
	// Maybe see if delete gallery xrpc method on grain.social becomes specified?
	return errors.New("not implemented")
}

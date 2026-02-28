package atproto

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/xrpc"
	"github.com/samber/lo"
)

type coffeeRoaster struct {
	Name     string `json:"name"`
	Location string `json:"location"`
	Website  string `json:"website"`
}

type coffeeBag struct {
	Name      string        `json:"name"`
	Origin    string        `json:"origin"`
	Roast     string        `json:"roast,omitempty"`
	Process   string        `json:"process,omitempty"`
	Variety   string        `json:"variety,omitempty"`
	Elevation string        `json:"elevation,omitempty"`
	Roaster   coffeeRoaster `json:"roaster"`
	Rating    int           `json:"rating,omitempty"`
	Date      string        `json:"date,omitempty"`
}

type arabicaBean struct {
	Name        string `json:"name"`
	Type        string `json:"$type"`
	Closed      bool   `json:"closed"`
	Origin      string `json:"origin"`
	Process     string `json:"process"`
	Variety     string `json:"variety"`
	CreatedAt   string `json:"createdAt"`
	RoastLevel  string `json:"roastLevel"`
	RoasterRef  string `json:"roasterRef"`
	Description string `json:"description"`
}

type arabicaRoaster struct {
	Name      string `json:"name"`
	Type      string `json:"$type"`
	Website   string `json:"website"`
	Location  string `json:"location"`
	CreatedAt string `json:"createdAt"`
}

func getArabicaBeans(ctx context.Context, client *xrpc.Client) ([]*arabicaBean, error) {
	beanRecords, err := listRecords(ctx, client, "social.arabica.alpha.bean")
	if err != nil {
		return nil, err
	}

	beans := []*arabicaBean{}
	for _, record := range beanRecords {
		bean := &arabicaBean{}
		err = json.Unmarshal(*record.Value, bean)
		if err != nil {
			return nil, err
		}
		beans = append(beans, bean)
	}

	return beans, nil
}

func getArabicaRoasters(ctx context.Context, client *xrpc.Client) (map[string]*arabicaRoaster, error) {
	roasterRecords, err := listRecords(ctx, client, "social.arabica.alpha.roaster")
	if err != nil {
		return nil, err
	}

	roasters := map[string]*arabicaRoaster{}
	for _, record := range roasterRecords {
		roaster := &arabicaRoaster{}
		err = json.Unmarshal(*record.Value, roaster)
		if err != nil {
			return nil, err
		}

		roasters[record.Uri] = roaster
	}

	return roasters, nil
}

func fetchAndProcessArabicaData(ctx context.Context, client *xrpc.Client) ([]coffeeBag, error) {
	beans, err := getArabicaBeans(ctx, client)
	if err != nil {
		return nil, err
	}

	roasters, err := getArabicaRoasters(ctx, client)
	if err != nil {
		return nil, err
	}

	coffeeBags := []coffeeBag{}
	for _, bean := range beans {
		roaster, ok := roasters[bean.RoasterRef]
		if !ok {
			return nil, fmt.Errorf("roaster %s not found for bean %s", bean.RoasterRef, bean.Name)
		}

		description := strings.Split(bean.Description, "\n")

		elevation, _ := lo.Find(description, func(v string) bool {
			return strings.HasPrefix(v, "Elevation:")
		})
		elevation = strings.TrimSpace(strings.TrimPrefix(elevation, "Elevation:"))

		ratingStr, _ := lo.Find(description, func(v string) bool {
			return strings.HasPrefix(v, "Rating:")
		})
		ratingStr = strings.TrimSpace(strings.TrimPrefix(ratingStr, "Rating:"))
		var rating int
		if ratingStr != "" {
			rating, err = strconv.Atoi(ratingStr)
			if err != nil {
				return nil, fmt.Errorf("invalid rating format for bean %s: %w", bean.Name, err)
			}
		}

		date, err := time.Parse(time.RFC3339, bean.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid date format for bean %s: %w", bean.Name, err)
		}

		coffeeBags = append(coffeeBags, coffeeBag{
			Name:      bean.Name,
			Origin:    bean.Origin,
			Process:   bean.Process,
			Roast:     bean.RoastLevel,
			Variety:   bean.Variety,
			Elevation: elevation,
			Date:      date.Format(time.DateOnly),
			Rating:    rating,
			Roaster: coffeeRoaster{
				Name:     roaster.Name,
				Location: roaster.Location,
				Website:  roaster.Website,
			},
		})
	}

	slices.SortStableFunc(coffeeBags, func(a, b coffeeBag) int {
		return strings.Compare(b.Date, a.Date)
	})

	return coffeeBags, nil
}

func (at *ATProto) UpdateCoffee(ctx context.Context) error {
	if at.arabicaFilename == "" {
		return nil
	}

	client, err := at.getClient(ctx)
	if err != nil {
		return err
	}

	coffeeBags, err := fetchAndProcessArabicaData(ctx, client)
	if err != nil {
		return err
	}

	var oldCoffeeBags []coffeeBag
	err = at.core.ReadJSON(at.arabicaFilename, &oldCoffeeBags)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if reflect.DeepEqual(oldCoffeeBags, coffeeBags) {
		return nil
	}

	coffeeBagsBytes, err := json.MarshalIndent(coffeeBags, "", "  ")
	if err != nil {
		return err
	}

	return at.core.WriteFile(at.arabicaFilename, coffeeBagsBytes, "atproto: synchronize arabica data")
}

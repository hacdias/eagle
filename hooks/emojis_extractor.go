package hooks

import (
	"github.com/forPelevin/gomoji"
	"github.com/hacdias/eagle/eagle"
	"github.com/thoas/go-funk"
)

type EmojisExtractor struct{}

func (t EmojisExtractor) FindEmojis(e *eagle.Entry) []string {
	results := gomoji.FindAll(e.Content)
	emojis := []string{}
	for _, emoji := range results {
		emojis = append(emojis, emoji.Character)
	}

	emojis = funk.UniqString(emojis)
	return emojis
}

func (t EmojisExtractor) EntryHook(e *eagle.Entry, isNew bool) error {
	if e.Listing == nil {
		return nil
	}

	e.Taxonomies["emojis"] = t.FindEmojis(e)
	return nil
}

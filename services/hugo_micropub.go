package services

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hacdias/eagle/middleware/micropub"
)

var typesWithLinks = map[micropub.Type]string{
	micropub.TypeRepost:   "repost-of",
	micropub.TypeLike:     "like-of",
	micropub.TypeReply:    "in-reply-of",
	micropub.TypeBookmark: "bookmark-of",
}

type Syndication struct {
	Related []string
	Targets []string
}

func (h *Hugo) FromMicropub(post *micropub.Request) (*HugoEntry, *Syndication, error) {
	entry := &HugoEntry{
		Content:  "",
		Metadata: map[string]interface{}{},
	}

	if published, ok := post.Properties.StringIf("published"); ok {
		entry.Metadata["date"] = published
	} else {
		entry.Metadata["date"] = time.Now().String()
	}

	postType := micropub.DiscoverType(post.Properties)

	switch postType {
	case micropub.TypeReply, micropub.TypeNote, micropub.TypeArticle:
		// It's fine.
	default:
		return nil, nil, errors.New("type not supported " + string(postType))
	}

	if content, ok := post.Properties.StringsIf("content"); ok {
		entry.Content = strings.TrimSpace(strings.Join(content, "\n"))
	}

	if name, ok := post.Properties.StringsIf("name"); ok {
		entry.Metadata["title"] = strings.TrimSpace(strings.Join(name, " "))
	}

	delete(post.Properties, "published")
	delete(post.Properties, "content")
	delete(post.Properties, "name")

	var synd *Syndication

	switch postType {
	case micropub.TypeRepost, micropub.TypeLike, micropub.TypeReply, micropub.TypeBookmark:
		links, ok := post.Properties.StringsIf(typesWithLinks[postType])
		if !ok {
			return nil, nil, errors.New("type " + string(postType) + " must refer to some link")
		}
		related, err := cleanRelated(links)
		if err != nil {
			return nil, nil, err
		}

		if len(related) > 0 {
			post.Properties[typesWithLinks[postType]] = related
		}

		if targets, ok := post.Commands.StringsIf("mp-syndicate-to"); ok {
			synd = &Syndication{
				Related: related,
				Targets: targets,
			}
		}
	}

	if categories, ok := post.Properties.StringsIf("category"); ok {
		entry.Metadata["tags"] = categories
		delete(post.Properties, "category")
	}

	entry.Metadata["properties"] = post.Properties

	if slugSlice, ok := post.Commands.StringsIf("mp-slug"); ok && len(slugSlice) == 1 {
		slug := strings.TrimSpace(strings.Join(slugSlice, "\n"))

		section := "micro"
		switch postType {
		case micropub.TypeArticle:
			section = "article"
		}

		year := time.Now().Year()
		month := time.Now().Month()
		entry.ID = fmt.Sprintf("/%s/%04d/%02ds/%s/", section, year, month, slug)
	} else {
		return nil, nil, errors.New("post must have a slug")
	}

	return entry, synd, nil
}

func cleanRelated(urls []string) ([]string, error) {
	clean := make([]string, len(urls))

	for i, u := range urls {
		if strings.HasPrefix(u, "https://twitter.com") && strings.Contains(u, "/status/") {
			u, err := url.Parse(u)
			if err != nil {
				return nil, err
			}

			for k := range u.Query() {
				u.Query().Del(k)
			}

			clean[i] = u.String()
		} else {
			clean[i] = u
		}
	}

	return clean, nil
}

func (h *Hugo) UpdateMicropub(entry *HugoEntry, mr *micropub.Request) (*HugoEntry, error) {
	//	tags := []string{}
	prop := map[string]interface{}{}

	/* if t, ok := entry.Metadata.StringsIf("tags"); ok {
		tags = t
	}

	if p, ok := entry.Metadata.MapIf("properties"); ok {

	} */

	// TODO

	for key, value := range mr.Updates.Replace {
		switch key {
		case "name":
			// meta.title = update.replace.name.join(' ').trim()
		case "category":
			// meta.tags = update.replace.category
		case "content":
			// content = update.replace.content.join(' ').trim()
		case "published":
			/*
					 if (!meta.publishDate && meta.date) {
				        meta.publishDate = meta.date
				      }

							meta.date = new Date(update.replace.published.join(' ').trim())
			*/
		default:
			prop[key] = value
		}
	}

	/*

		  }

		  for (const key in update.add) {
		    if (key === 'name') {
		      throw new Error('cannot add a new name')
		    } else if (key === 'category') {
		      meta.tags.push(...update.add.category)
		    } else if (key === 'content') {
		      content += update.add.join(' ').trim()
		    } else if (key === 'published') {
		      if (!meta.date) {
		        meta.date = new Date(update.add.published.join(' ').trim())
		      } else {
		        throw new Error('cannot replace published through add method')
		      }
		    } else {
		      meta.properties[key] = meta.properties[key] || []
		      meta.properties[key].push(...update.add[key])
		    }
		  }

		  if (Array.isArray(update.delete)) {
		    for (const key of update.delete) {
		      if (key === 'category') {
		        meta.tags = []
		      } else if (key === 'content') {
		        content = ''
		      } else {
		        delete meta.properties[key]
		      }
		    }
		  } else {
		    for (const [key, value] of Object.entries(update.delete)) {
		      if (key === 'content') {
		        content = ''
		      } if (key === 'category') {
		        meta.tags = meta.tags.filter(tag => !value.includes(tag))
		      } else {
		        meta.properties[key] = meta.properties[key]
		          .filter(tag => !value.includes(tag))
		      }
		    }
		  }

		  const res = getModifiers(content.trim())
		  content = res.content

		  if (res.modifiers.includes('+HOME')) {
		    meta.home = true
		  }

		  return { meta, content, modifiers: res.modifiers }
		}

	*/

	return nil, nil
}

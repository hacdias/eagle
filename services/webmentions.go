package services

import "github.com/hacdias/eagle/config"

type Webmentions struct {
	Domain    string
	Telegraph config.Telegraph
	Git       Git
	Media     Media
	Hugo      Hugo
}

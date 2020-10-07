package services

import (
	"sync"

	"github.com/hacdias/eagle/config"
)

type XRay struct {
	*sync.Mutex
	config.XRay
	StoragePath string
	Twitter     config.Twitter
}

func (x *XRay) Request() {

}
func (x *XRay) RequestAndSave(url string) {

}

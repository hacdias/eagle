package services

import "github.com/hacdias/eagle/config"

type XRay struct {
	config.XRay
	StoragePath string
	Twitter     config.Twitter
}

func (x *XRay) Request() {

}
func (x *XRay) RequestAndSave(url string) {

}

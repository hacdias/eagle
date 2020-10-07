package services

import "github.com/hacdias/eagle/config"

type Media config.BunnyCDN

func (m *Media) Upload(filename string, data []byte) (string, error) {
	return "", nil
}

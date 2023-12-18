package privatedns

import (
	"net"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const (
	initialInterval     = 500 * time.Millisecond
	randomizationFactor = 0.5
	multiplier          = 1.5
	maxInterval         = 1 * time.Minute
	maxElapsedTime      = 5 * time.Minute
)

func getIPAddress() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func getDnsBackoff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()

	b.InitialInterval = initialInterval
	b.RandomizationFactor = randomizationFactor
	b.Multiplier = multiplier
	b.MaxInterval = maxInterval
	b.MaxElapsedTime = maxElapsedTime

	return b
}

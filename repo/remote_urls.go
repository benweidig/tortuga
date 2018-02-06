package repo

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
)

type RemoteUrls struct {
	Push  string
	Fetch string
}

func NewRemoteUrls(stdOut bytes.Buffer) (RemoteUrls, error) {
	r := RemoteUrls{}

	scanner := bufio.NewScanner(&stdOut)

	for scanner.Scan() {
		row := scanner.Text()

		cols := strings.Fields(row)
		if len(cols) != 3 {
			return r, errors.New("Couldn't determinate remote urls from " + row)
		}

		remoteType := cols[2]
		url := cols[1]

		switch remoteType {
		case "(fetch)":
			r.Fetch = url
		case "(push)":
			r.Push = url
		}
	}

	return r, nil
}

package main

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"golang.org/x/term"
)

const RTSP_URL = "rtsp://localhost:8554/mypath"

func credentials() (string, string, error) {
	var username, password string

	fmt.Print("Username: ")
	fmt.Scan(&username)
	fmt.Println("Hello,", username)

	fmt.Print("Password: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return username, password, err
	}
	password = string(bytePassword)

	return strings.TrimSpace(username), strings.TrimSpace(password), nil
}

func main() {
	username, password, err := credentials()
	if err != nil {
		panic(err)
	}

	host := "192.168.0.1" // set to IP on local LAN
	port := 554           // default port Reolink uses

	// See: https://support.reolink.com/hc/en-us/articles/900000630706-Introduction-to-RTSP/
	// rtsp://<username>:<password>@<IP address>/Preview_<channel number>_<stream type>
	url := fmt.Sprintf("rtsp://%s:%s@%s:%d/Preview_01_main", username, password, host, port)

	c := gortsplib.Client{}

	u, err := base.ParseURL(url)
	if err != nil {
		panic(err)
	}

	err = c.Start(u.Scheme, u.Host)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	desc, _, err := c.Describe(u)
	if err != nil {
		panic(err)
	}

	fmt.Printf("available medias: %v\n", desc.Medias)
}

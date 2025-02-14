package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtph264"
	"github.com/cenkalti/backoff/v4"
	"github.com/pion/rtp"
)

type RTSPClient interface {
	Stream(url string) error
}

type GoRTSPLibClient struct {
	Client gortsplib.Client
}

func (c *GoRTSPLibClient) Stream(url string) error {
	fmt.Println(url)
	startTime := time.Now()

	u, err := base.ParseURL(url)
	if err != nil {
		return backoff.Permanent(fmt.Errorf("error parsing URL: %s", err))
	}

	err = c.Client.Start(u.Scheme, u.Host)
	if err != nil {
		return fmt.Errorf("error connecting to client: %s", err)
	}
	defer c.Client.Close()

	fmt.Println(startTime)
	fmt.Println("connecting to device...")

	desc, _, err := c.Client.Describe(u)
	if err != nil {
		return fmt.Errorf("error querying device for media: %s", err)
	}

	// find the H264 media and format
	var h264Format *format.H264
	h264Media := desc.FindFormat(&h264Format)
	if h264Media == nil {
		return fmt.Errorf("error attempting to find format: %s", "h264 media not found")
	}

	// find the mpeg-4 media and format
	var mpeg4Format *format.MPEG4Audio
	mpeg4Media := desc.FindFormat(&mpeg4Format)
	if mpeg4Media == nil {
		return fmt.Errorf("error attempting to find format: %s", "mpeg-4 media not found")
	}

	// setup RTP -> H264 decoder
	h264RTPDec, err := h264Format.CreateDecoder()
	if err != nil {
		return fmt.Errorf("error setting up H264 decoder: %s", err)
	}

	// setup RTP -> MPEG-4 audio decoder
	mpeg4AudioRTPDec, err := mpeg4Format.CreateDecoder()
	if err != nil {
		return fmt.Errorf("error setting up mpeg-4 decoder: %s", err)
	}

	// setup MPEG-TS muxer
	mpegtsMuxer := &mpegtsMuxer{
		fileName:         fmt.Sprintf("%s_living_room.ts", startTime.Format(time.RFC3339)),
		h264Format:       h264Format,
		mpeg4AudioFormat: mpeg4Format,
	}

	err = mpegtsMuxer.initialize()
	if err != nil {
		return fmt.Errorf("error initializing muxer: %s", err)
	}
	defer mpegtsMuxer.close()

	// setup all medias
	err = c.Client.SetupAll(desc.BaseURL, desc.Medias)
	if err != nil {
		return fmt.Errorf("setting up media: %s", err)
	}

	// called when a H264/RTP packet arrives
	c.Client.OnPacketRTP(h264Media, h264Format, func(pkt *rtp.Packet) {
		// decode timestamp
		pts, ok := c.Client.PacketPTS2(h264Media, pkt)
		if !ok {
			log.Printf("waiting for timestamp")
			return
		}

		// extract access unit from RTP packets
		au, err := h264RTPDec.Decode(pkt)
		if err != nil {
			if err != rtph264.ErrNonStartingPacketAndNoPrevious && err != rtph264.ErrMorePacketsNeeded {
				log.Printf("ERR: %v", err)
			}
			return
		}

		// encode the access unit into MPEG-TS
		err = mpegtsMuxer.writeH264(au, pts)
		if err != nil {
			log.Printf("ERR: %v", err)
			return
		}

		fmt.Printf("\rDuration: %v", time.Since(startTime))
	})

	// called when a MPEG-4 audio / RTP packet arrives
	c.Client.OnPacketRTP(mpeg4Media, mpeg4Format, func(pkt *rtp.Packet) {
		// decode timestamp
		pts, ok := c.Client.PacketPTS2(mpeg4Media, pkt)
		if !ok {
			log.Printf("waiting for timestamp")
			return
		}

		// extract access units from RTP packets
		aus, err := mpeg4AudioRTPDec.Decode(pkt)
		if err != nil {
			log.Printf("ERR: %v", err)
			return
		}

		// encode access units into MPEG-TS
		err = mpegtsMuxer.writeMPEG4Audio(aus, pts)
		if err != nil {
			log.Printf("ERR: %v", err)
			return
		}
	})

	fmt.Println("starting stream")

	// start playing
	_, err = c.Client.Play(nil)
	if err != nil {
		return fmt.Errorf("error sending PLAY query: %s", err)
	}

	// wait until a fatal error
	err = c.Client.Wait()

	fmt.Println("")
	if err != nil {
		return fmt.Errorf("connection closed with error: %s", err)
	}

	return nil
}

type ReolinkCameraClient struct {
	Username   string
	Password   string
	Host       string
	Port       uint16
	RTSPClient RTSPClient
}

func NewReolinkCameraClient(user, password, host string, port uint16, rtspClient RTSPClient) *ReolinkCameraClient {
	return &ReolinkCameraClient{
		Username:   user,
		Password:   password,
		Host:       host,
		Port:       port,
		RTSPClient: rtspClient,
	}
}

func (rcc *ReolinkCameraClient) GetRTSPURL() string {
	// See: https://support.reolink.com/hc/en-us/articles/900000630706-Introduction-to-RTSP/
	// rtsp://<username>:<password>@<IP address>/Preview_<channel number>_<stream type>
	url_template := "rtsp://%s:%s@%s:%d/Preview_01_main"
	return fmt.Sprintf(url_template, rcc.Username, rcc.Password, rcc.Host, rcc.Port)
}

func (rcc *ReolinkCameraClient) StreamRTSP() error {
	return rcc.RTSPClient.Stream(rcc.GetRTSPURL())
}

func main() {
	var username, password, host, port string
	var ok bool

	if username, ok = os.LookupEnv("CAM_USER"); !ok {
		panic("$CAM_USER not found")
	}

	if password, ok = os.LookupEnv("CAM_PASS"); !ok {
		panic("$CAM_PASS not found")
	}

	if host, ok = os.LookupEnv("CAM_HOST"); !ok {
		panic("$CAM_HOST not found")
	}

	if port, ok = os.LookupEnv("CAM_PORT"); !ok {
		panic("$CAM_PORT not found")
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		panic(err)
	}

	camera := NewReolinkCameraClient(
		username,
		password,
		host,
		uint16(portNum),
		&GoRTSPLibClient{
			Client: gortsplib.Client{
				OnPacketLost: func(err error) {
					fmt.Println("PACKET LOSS:", err)
				},
				OnServerRequest: func(r *base.Request) {
					fmt.Printf(`
Request
    Method: %s
	URL: %s
	Body: %s

					`, r.Method, r.URL, r.Body)
				},
				OnServerResponse: func(r *base.Response) {
					fmt.Printf(`
Response
	StatusCode: %d
	StatusMessage: %s
	String: %s

					`, r.StatusCode, r.StatusMessage, r.String())
				},
			},
		},
	)

	b := backoff.StopBackOff{}
	// b := backoff.NewExponentialBackOff(
	// 	backoff.WithMaxElapsedTime(30 * time.Second),
	// )

	err = backoff.Retry(camera.StreamRTSP, &b)
	if err != nil {
		panic(fmt.Errorf("could not maintain RTSP connection: %s", err))
	}
}

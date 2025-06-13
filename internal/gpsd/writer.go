package gpsd

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aokhrimenko/gpsd-simulator/internal/route"
)

const (
	DefaultVersionRelease    = "3.25"
	DefaultVersionRev        = "3.25"
	DefaultVersionProtoMajor = 3
	DefaultVersionProtoMinor = 25
	DefaultVersionDevicePath = "/dev/ttyUSB1"
	DefaultDeviceDriver      = "NMEA0183"
	DefaultDeviceActivated   = "2025-03-21T12:20:29.002Z"
	DefaultDeviceBps         = 9600
	DefaultDeviceParity      = "N"
	DefaultDeviceStopBits    = 1
	DefaultTpvMode           = 3

	WatchCommand  = `?WATCH=`
	CommandSuffix = ';'
)

type WriterConfig struct {
	VersionRelease    string
	VersionRev        string
	VersionProtoMajor uint
	VersionProtoMinor uint
	DevicePath        string
	DeviceDriver      string
	DeviceActivated   string
	DeviceBps         uint
	DeviceParity      string
	DeviceStopBits    uint
	TpvMode           uint
}

// {"class":"VERSION","release":"3.25","rev":"3.25","proto_major":3,"proto_minor":25}
type version struct {
	Class      string `json:"class"`
	Release    string `json:"release"`
	Rev        string `json:"rev"`
	ProtoMajor uint   `json:"proto_major"`
	ProtoMinor uint   `json:"proto_minor"`
}

// {"class":"TPV","device":"/dev/ttyUSB1","mode":3,"time":"2025-06-13T17:29:00.337902Z","lat":47.38184271474015,"lon":8.44824654879321,"alt":575,"altHAE":575,"track":91.13973909509252,"speed":15.277777777813657}
type tpv struct {
	Class  string        `json:"class"`
	Device string        `json:"device"`
	Mode   uint          `json:"mode"`
	Time   time.Time     `json:"time"`
	Lat    float64       `json:"lat"`
	Lon    float64       `json:"lon"`
	Alt    float64Fixed3 `json:"alt"`
	AltHAE float64Fixed3 `json:"altHAE"`
	Track  float64Fixed3 `json:"track"`
	Speed  float64Fixed3 `json:"speed"`
}

type float64Fixed2 float64

func (f float64Fixed2) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%.2f", f)), nil
}

type float64Fixed3 float64

func (f float64Fixed3) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%.2f", f)), nil
}

// {"class":"DEVICES","devices":[{"class":"DEVICE","path":"/dev/ttyUSB1","driver":"NMEA0183","activated":"2025-03-21T12:20:29.002Z","flags":1,"native":0,"bps":9600,"parity":"N","stopbits":1,"cycle":1.00}]}
type device struct {
	Class     string        `json:"class"`
	Path      string        `json:"path"`
	Driver    string        `json:"driver"`
	Activated string        `json:"activated"`
	Flags     uint          `json:"flags"`
	Native    uint          `json:"native"`
	Bps       uint          `json:"bps"`
	Parity    string        `json:"parity"`
	Stopbits  uint          `json:"stopbits"`
	Cycle     float64Fixed2 `json:"cycle"`
}
type devices struct {
	Class   string   `json:"class"`
	Devices []device `json:"devices"`
}

// {"class":"WATCH","enable":true,"json":true,"nmea":false,"raw":0,"scaled":false,"timing":false,"split24":false,"pps":false}
type watch struct {
	Class   string `json:"class"`
	Enable  bool   `json:"enable"`
	Json    bool   `json:"json"`
	Nmea    bool   `json:"nmea"`
	Raw     int    `json:"raw"`
	Scaled  bool   `json:"scaled"`
	Timing  bool   `json:"timing"`
	Split24 bool   `json:"split24"`
	Pps     bool   `json:"pps"`
}

func NewWriter(upstream io.Writer, config WriterConfig) *Writer {
	encoder := json.NewEncoder(upstream)
	encoder.SetEscapeHTML(false)

	return &Writer{
		encoder: encoder,
		config:  config,
		tpv: tpv{
			Class:  "TPV",
			Device: config.DevicePath,
			Mode:   config.TpvMode,
		},
	}
}

type Writer struct {
	encoder *json.Encoder
	config  WriterConfig
	tpv     tpv
}

func (w *Writer) WriteDevices() error {
	// {"class":"DEVICES","devices":[{"class":"DEVICE","path":"/dev/ttyUSB1","driver":"NMEA0183","activated":"2025-03-21T12:20:29.002Z","flags":1,"native":0,"bps":9600,"parity":"N","stopbits":1,"cycle":1.00}]}
	devicesData := devices{
		Class: "DEVICES",
		Devices: []device{
			{
				Class:     "DEVICE",
				Path:      w.config.DevicePath,
				Driver:    w.config.DeviceDriver,
				Activated: w.config.DeviceActivated,
				Flags:     1,
				Native:    0,
				Bps:       w.config.DeviceBps,
				Parity:    w.config.DeviceParity,
				Stopbits:  w.config.DeviceStopBits,
				Cycle:     float64Fixed2(1.0),
			},
		},
	}
	return w.encoder.Encode(devicesData)
}

func (w *Writer) WriteWatch() error {
	// {"class":"WATCH","enable":true,"json":true,"nmea":false,"raw":0,"scaled":false,"timing":false,"split24":false,"pps":false}
	watchData := watch{
		Class:   "WATCH",
		Enable:  true,
		Json:    true,
		Nmea:    false,
		Raw:     0,
		Scaled:  false,
		Timing:  false,
		Split24: false,
		Pps:     false,
	}

	return w.encoder.Encode(watchData)
}

func (w *Writer) WriteVersion() error {
	versionData := version{
		Class:      "VERSION",
		Release:    w.config.VersionRelease,
		Rev:        w.config.VersionRev,
		ProtoMajor: w.config.VersionProtoMajor,
		ProtoMinor: w.config.VersionProtoMinor,
	}

	return w.encoder.Encode(versionData)
}

func (w *Writer) WriteTPVReport(point route.Point) error {
	w.tpv.Time = time.Now().UTC()
	w.tpv.Lat = point.Lat
	w.tpv.Lon = point.Lon
	w.tpv.Alt = float64Fixed3(point.Elevation)
	w.tpv.AltHAE = float64Fixed3(point.Elevation)
	w.tpv.Track = float64Fixed3(point.Track)
	w.tpv.Speed = float64Fixed3(point.Speed)

	return w.encoder.Encode(w.tpv)
}

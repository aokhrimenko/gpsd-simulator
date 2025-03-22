package gpsd

import (
	"fmt"
	"io"
	"time"

	"github.com/aokhrimenko/gpsd-simulator/internal/route"
)

const (
	VersionLine = "{\"class\":\"VERSION\",\"release\":\"3.17\",\"rev\":\"3.17\",\"proto_major\":3,\"proto_minor\":12}\n"
	DevicesLine = "{\"class\":\"DEVICES\",\"devices\":[{\"class\":\"DEVICE\",\"path\":\"/dev/ttyUSB1\",\"driver\":\"NMEA0183\",\"activated\":\"2025-03-21T12:20:29.002Z\",\"flags\":1,\"native\":0,\"bps\":9600,\"parity\":\"N\",\"stopbits\":1,\"cycle\":1.00}]}\n"
	WatchLine   = "{\"class\":\"WATCH\",\"enable\":true,\"json\":true,\"nmea\":false,\"raw\":0,\"scaled\":false,\"timing\":false,\"split24\":false,\"pps\":false}\n"

	WatchCommand = `?WATCH=`

	TpvTemplate   = "{\"class\": \"TPV\", \"device\": \"/dev/ttyUSB1\", \"mode\": 3, \"time\": \"%s\", \"ept\": 0.005, \"lat\": %f, \"lon\": %f, \"alt\": %0.f, \"epv\": 18.400, \"track\": 186.1000, \"speed\": %.3f, \"climb\": 0.000, \"epc\": 36.80}\n"
	CommandSuffix = ';'
)

func WriteTPVReport(w io.Writer, point route.Point) error {
	now := time.Now().UTC()
	formattedTime := now.Format("2006-01-02T15:04:05.000Z")

	tpvString := fmt.Sprintf(TpvTemplate, formattedTime, point.Lat, point.Lon, point.Elevation, point.Speed)
	_, err := w.Write([]byte(tpvString))
	return err
}

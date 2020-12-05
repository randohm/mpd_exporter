package main

import (
    "flag"
    "log"
    "net/http"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/prometheus/client_golang/prometheus"
    //"os/exec"
    //"encoding/json"
    "strconv"
    //"strings"
    "fmt"
    "os"
    "github.com/fhs/gompd/mpd"
    "time"
)



const (
    swVersion = "0.0.1"
    swName = "mpd_exporter"
    defaultListenAddress = "0.0.0.0:9778"
    defaultMetricsPath = "/metrics"
    metricNamePrefix = "mpd_"

    defaultWatcherAddr = ":6600"
)



var (
    // Stats
    numSongsDesc = prometheus.NewDesc("mpd_stats_songs", "The number of songs in the collection", []string{"mpd_host"}, nil)
    numAlbumsDesc = prometheus.NewDesc("mpd_stats_albums", "The number of albums in the collection", []string{"mpd_host"}, nil)
    numArtistsDesc = prometheus.NewDesc("mpd_stats_artists", "The number of artists in the collection", []string{"mpd_host"}, nil)
    playtimeDesc = prometheus.NewDesc("mpd_stats_playtime", "Playtime of the collection", []string{"mpd_host"}, nil)
    outputDesc = prometheus.NewDesc("mpd_stats_output", "Output enabled", []string{"mpd_host", "output_id", "output_name", "plugin", "attribute"}, nil)

    // Songs
    songLenDesc = prometheus.NewDesc("mpd_song_length_seconds", "Song length in seconds.", []string{"mpd_host", "title", "album", "artist", "albumartist", "track", "format", "file"}, nil)
    songModDesc = prometheus.NewDesc("mpd_song_lastmodified_epoch", "Song last modified in epoch", []string{"mpd_host", "title", "album", "artist", "albumartist", "track", "format", "file"}, nil)
)



type MpdExporter struct {
    Addr    string
    Client  *mpd.Client
    Watcher *mpd.Watcher
    Stats   mpd.Attrs
    Status  mpd.Attrs
    Outputs []mpd.Attrs

    pass    string
}



func NewMpdExporter(mpdAddr, mpdPass string) (*MpdExporter, error) {
    e := MpdExporter{ Addr: mpdAddr, pass: mpdPass }
    return &e, nil
}



func (e *MpdExporter) Connect() error {
    var err error
    if e.pass == "" {
        e.Client, err = mpd.Dial("tcp", e.Addr)
    } else {
        e.Client, err = mpd.DialAuthenticated("tcp", e.Addr, e.pass)
    }
    if err != nil {
	    return err
    }
    return nil
}



func (e *MpdExporter) Describe(ch chan<- *prometheus.Desc) {
    ch <- numSongsDesc
    ch <- numAlbumsDesc
    ch <- numArtistsDesc
    ch <- playtimeDesc
    ch <- outputDesc
    ch <- songLenDesc
    ch <- songModDesc
}



func (e *MpdExporter) Collect(ch chan<- prometheus.Metric) {
    err := e.Connect()
    if err != nil {
        log.Print(err)
        return
    }
    defer e.Client.Close()

    e.collectStats(ch)
    e.collectSongStats(ch)
}



func (e *MpdExporter) collectSongStats(ch chan<- prometheus.Metric) {
    var err error

    list, err := e.Client.ListAllInfo("/")
    if err != nil {
        log.Print(err)
        return
    }

    for _, song := range list {
        //if i > 2 { break }
        var durationF float64
        if song["duration"] == "" {
            durationF = 0.0
        } else {
            durationF, err = strconv.ParseFloat(song["duration"], 64)
            if err != nil {
                log.Printf("parsing float '%s': %s", song["duration"], err)
                return
            }
        }
        ch <- prometheus.MustNewConstMetric(songLenDesc, prometheus.GaugeValue, durationF, e.Addr, song["Title"], song["Album"], song["Artist"], song["AlbumArtist"], song["Track"], song["Format"], song["file"])

        // Get epoch time from ISO
        //*
        var rYear, rMonth, rDay, rHour, rMinute, rSecond int
        _, err = fmt.Sscanf(song["Last-Modified"], "%4d-%2d-%2dT%2d:%2d:%2dZ", &rYear, &rMonth, &rDay, &rHour, &rMinute, &rSecond)
        epochTime := time.Date(rYear, time.Month(rMonth), rDay, rHour, rMinute, rSecond, 0, time.Local).Unix()
        ch <- prometheus.MustNewConstMetric(songModDesc, prometheus.GaugeValue, float64(epochTime), e.Addr, song["Title"], song["Album"], song["Artist"], song["AlbumArtist"], song["Track"], song["Format"], song["file"])
        //*/
    }
}
func (e *MpdExporter) collectStats(ch chan<- prometheus.Metric) {
    var err error

    // Collect data
    e.Stats, err = e.Client.Stats()
    if err != nil {
        log.Print(err)
        return
    }
    e.Status, err = e.Client.Status()
    if err != nil {
        log.Print(err)
        return
    }
    e.Outputs, err = e.Client.ListOutputs()
    if err != nil {
        log.Print(err)
        return
    }

    // Setup the metrics
    numSongsF, err := strconv.ParseFloat(e.Stats["songs"], 64)
    if err != nil {
        log.Print(err)
        return
    }
    ch <- prometheus.MustNewConstMetric(numSongsDesc, prometheus.GaugeValue, numSongsF, e.Addr)

    numAlbumsF, err := strconv.ParseFloat(e.Stats["albums"], 64)
    if err != nil {
        log.Print(err)
        return
    }
    ch <- prometheus.MustNewConstMetric(numAlbumsDesc, prometheus.GaugeValue, numAlbumsF, e.Addr)

    numArtistsF, err := strconv.ParseFloat(e.Stats["artists"], 64)
    if err != nil {
        log.Print(err)
        return
    }
    ch <- prometheus.MustNewConstMetric(numArtistsDesc, prometheus.GaugeValue, numArtistsF, e.Addr)

    playtimeF, err := strconv.ParseFloat(e.Stats["playtime"], 64)
    if err != nil {
        log.Print(err)
        return
    }
    ch <- prometheus.MustNewConstMetric(playtimeDesc, prometheus.GaugeValue, playtimeF, e.Addr)

    for _, o := range e.Outputs {
        enabledF, err := strconv.ParseFloat(o["outputenabled"], 64)
        if err != nil {
            log.Print(err)
            return
        }
        ch <- prometheus.MustNewConstMetric(outputDesc, prometheus.GaugeValue, enabledF, e.Addr, o["outputid"], o["outputname"], o["plugin"], o["attribute"])
    }
}



func (e *MpdExporter) test() {
    err := e.Connect()
    if err != nil {
        log.Print(err)
        return
    }
    defer e.Client.Close()

    e.Stats, err = e.Client.Stats()
    if err != nil {
        log.Print(err)
        return
    }
    log.Printf("%s", e.Stats)

    e.Status, err = e.Client.Status()
    if err != nil {
        log.Print(err)
        return
    }
    e.Outputs, err = e.Client.ListOutputs()
    if err != nil {
        log.Print(err)
        return
    }

    list, err := e.Client.ListAllInfo("/")
    if err != nil {
        log.Print(err)
        return
    }
    return

    for _, song := range list {
        //if i > 2 { break }
        log.Printf("%s", song)
        /*
        var durationF float64
        if song["duration"] == "" {
            durationF = 0.0
        } else {
            durationF, err = strconv.ParseFloat(song["duration"], 64)
            if err != nil {
                log.Printf("parsing float '%s': %s", song["duration"], err)
                return
            }
        }
        //*/

        // Get epoch time from ISO
        /*
        var rYear, rMonth, rDay, rHour, rMinute, rSecond int
        _, err = fmt.Sscanf(song["Last-Modified"], "%4d-%2d-%2dT%2d:%2d:%2dZ", &rYear, &rMonth, &rDay, &rHour, &rMinute, &rSecond)
        epochTime := time.Date(rYear, time.Month(rMonth), rDay, rHour, rMinute, rSecond, 0, time.Local).Unix()
        //*/
    }
}






func init() {
    log.SetFlags(log.Ldate|log.Ltime|log.Lshortfile)
}



func main() {
    listenAddress := flag.String("web.listen-address", defaultListenAddress, "Listen address for HTTP requests")
    metricsPath := flag.String("web.telemetry-path", defaultMetricsPath, "Path under which to expose metrics")
    showVersion := flag.Bool("version", false, "Show version and exit")
    mpdAddr := flag.String("mpd.addr", defaultWatcherAddr, "Address of mpd")
    mpdPass := flag.String("mpd.pass", "", "MPD password (optional)")
    flag.Parse()

    if *showVersion {
        fmt.Printf("%s v%s\n", swName, swVersion)
        os.Exit(0)
    }


    exporter, err := NewMpdExporter(*mpdAddr, *mpdPass)
    if err != nil {
        log.Fatal(err)
    }
    prometheus.MustRegister(exporter)
    /*
    exporter.test()
    return
    //*/

    //*
    http.Handle(*metricsPath, promhttp.Handler())
    log.Printf("Listening on %s", *listenAddress)
    http.ListenAndServe(*listenAddress, nil)
    //*/
}

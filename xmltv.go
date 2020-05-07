package main

import (
  "bytes"
  "encoding/xml"
  "fmt"
  "io/ioutil"
  "path/filepath"
  "runtime"
  "strings"
  "time"
)

func CreateXMLTV2(filename string) (err error) {

  defer func() {
    runtime.GC()
  }()

  Config.File = strings.TrimSuffix(filename, filepath.Ext(filename))

  var generator xml.Attr
  generator.Name = xml.Name{Local: AppName}
  generator.Value = AppName

  var source xml.Attr
  source.Name = xml.Name{Local: "source-info-name"}
  source.Value = "Schedules Direct"

  var info xml.Attr
  info.Name = xml.Name{Local: "source-info-url"}
  info.Value = "http://schedulesdirect.org"

  buf := &bytes.Buffer{}
  buf.WriteString(xml.Header)

  enc := xml.NewEncoder(buf)
  enc.Indent("", "  ")

  var he = func(err error) {
    if err != nil {
      ShowErr(err)
      return
    }
  }

  err = Config.Open()
  if err != nil {
    return
  }

  err = Cache.Open()
  if err != nil {
    return
  }

  Cache.Init()
  err = Cache.Open()
  if err != nil {
    ShowErr(err)
    return
  }

  showInfo("G2G", fmt.Sprintf("Create XMLTV File [%s]", Config.Files.XMLTV))

  he(enc.EncodeToken(xml.StartElement{Name: xml.Name{Local: "tv"}, Attr: []xml.Attr{generator, source, info}}))

  // XMLTV Channels
  for _, cache := range Cache.Channel {

    var xmlCha channel // struct_config.go

    xmlCha.ID = fmt.Sprintf("%s.%s.schedulesdirect.org", AppName, cache.StationID)
    xmlCha.Icon = cache.getLogo()
    xmlCha.DisplayName = append(xmlCha.DisplayName, DisplayName{Value: cache.Callsign})
    xmlCha.DisplayName = append(xmlCha.DisplayName, DisplayName{Value: cache.Name})

    he(enc.Encode(xmlCha))

  }

  // XMLTV Programs
  for _, cache := range Cache.Channel {

    var program = getProgram(cache)
    he(enc.Encode(program))

  }

  he(enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "tv"}}))
  he(enc.Flush())

  // write the whole body at once
  err = ioutil.WriteFile(Config.Files.XMLTV, buf.Bytes(), 0644)
  if err != nil {
    panic(err)
  }

  return
}

// CreateXMLTV : Create XMLTV file from cache file
/*
func CreateXMLTV(filename string) (err error) {

  defer runtime.GC()

  Config.File = strings.TrimSuffix(filename, filepath.Ext(filename))

  err = Config.Open()
  if err != nil {
    return
  }

  err = Cache.Open()
  if err != nil {
    return
  }

  Cache.Init()
  err = Cache.Open()
  if err != nil {
    ShowErr(err)
    return
  }

  showInfo("G2G", fmt.Sprintf("Create XMLTV File [%s]", Config.Files.XMLTV))

  var xmltv = &XMLTV{Generator: AppName, Source: "Schedules Direct", Info: "http://schedulesdirect.org"}

  for _, ch := range Cache.Channel {

    var channel ChannelXML

    channel.ID = fmt.Sprintf("%s.%s.schedulesdirect.org", AppName, ch.StationID)
    channel.Icon = ch.getLogo()
    channel.DisplayName = append(channel.DisplayName, DisplayName{Value: ch.Callsign})
    channel.DisplayName = append(channel.DisplayName, DisplayName{Value: ch.Name})

    xmltv.Channel = append(xmltv.Channel, channel)

    xmltv.Program = append(xmltv.Program, getProgram(ch)...)

  }

  var content, _ = xml.MarshalIndent(xmltv, "  ", "    ")
  var xmlOutput = []byte(xml.Header + string(content))
  _ = xmlOutput
  //fmt.Println(string(xmlOutput))
  writeByteToFile(Config.Files.XMLTV, xmlOutput)
  //time.Sleep(20 * time.Second)

  xmltv = &XMLTV{}

  return
}
*/

// Channel infos
func (channel *G2GCache) getLogo() (icon Icon) {

  icon.Src = channel.Logo.URL
  icon.Height = channel.Logo.Height
  icon.Width = channel.Logo.Width

  return
}

func getProgram(channel G2GCache) (p []programme) {

  if schedule, ok := Cache.Schedule[channel.StationID]; ok {

    for _, s := range schedule {

      var pro programme

      // Channel ID
      pro.Channel = fmt.Sprintf("%s.%s.schedulesdirect.org", AppName, channel.StationID)

      // Start and Stop time
      timeLayout := "2006-01-02 15:04:05 +0000 UTC"
      t, err := time.Parse(timeLayout, s.AirDateTime.Format(timeLayout))
      if err != nil {
        ShowErr(err)
        return
      }

      var dateArray = strings.Fields(t.String())
      var offset = " " + dateArray[2]
      var startTime = t.Format("20060102150405") + offset
      var stopTime = t.Add(time.Second*time.Duration(s.Duration)).Format("20060102150405") + offset
      pro.Start = startTime
      pro.Stop = stopTime

      // Title
      var lang = "en"
      if len(channel.BroadcastLanguage) != 0 {
        lang = channel.BroadcastLanguage[0]
      }

      pro.Title = Cache.GetTitle(s.ProgramID, lang)

      // Sub Title

      pro.SubTitle = Cache.GetSubTitle(s.ProgramID, lang)

      // Description
      pro.Desc = Cache.GetDescs(s.ProgramID, pro.SubTitle.Value)

      // Category
      pro.Categorys = Cache.GetCategory(s.ProgramID)

      // EpisodeNum
      pro.EpisodeNums = Cache.GetEpisodeNum(s.ProgramID)

      // Icon
      pro.Icon = Cache.GetIcon(s.ProgramID[0:10])

      // Video
      for _, v := range s.VideoProperties {

        switch strings.ToLower(v) {

        case "hdtv", "sdtv", "uhdtv", "3d":
          pro.Video.Quality = strings.ToLower(v)

        }

      }

      // Audio
      for _, a := range s.AudioProperties {

        switch a {

        case "stereo", "dvs":
          pro.Audio.Stereo = "dolby digital"
        case "DD 5.1", "Atmos":
          pro.Audio.Surround = a
        case "dubbed", "mono":
          pro.Audio.Mono = "mono"
        default:
          pro.Audio.Mono = "mono"

        }

      }

      // New / PreviouslyShown
      if s.New {
        pro.New = &New{Value: ""}
      } else {
        var prev = Cache.GetPreviouslyShown(s.ProgramID)
        pro.PreviouslyShown = &prev
      }

      // Live
      if s.LiveTapeDelay == "Live" {
        pro.Live = &Live{Value: ""}
      }

      p = append(p, pro)

    }

  }

  return
}

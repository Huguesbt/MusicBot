package main

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	kkdaiYoutube "github.com/kkdai/youtube/v2"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	Client kkdaiYoutube.Client
)

func getDuration(stringRawFull, stringRawOffset string) (stringRemain string) {
	var stringFull string
	var duration TimeDuration
	var partial time.Duration

	stringFull = strings.Replace(stringRawFull, "P", "", 1)
	stringFull = strings.Replace(stringFull, "T", "", 1)
	stringFull = strings.ToLower(stringFull)

	var secondsFull, secondsOffset int
	value := strings.Split(stringFull, "d")
	if len(value) == 2 {
		secondsFull, _ = strconv.Atoi(value[0])
		// get the days in seconds
		secondsFull = secondsFull * 86400
		// get the format 1h1m1s in seconds
		partial, _ = time.ParseDuration(value[1])
		secondsFull = secondsFull + int(partial.Seconds())
	} else {
		partial, _ = time.ParseDuration(stringFull)
		secondsFull = int(partial.Seconds())
	}

	if stringRawOffset != "" {
		value = strings.Split(stringRawOffset, "s")
		if len(value) == 2 {
			secondsOffset, _ = strconv.Atoi(value[0])
		}
	}
	// substact the time offset
	duration.Second = secondsFull - secondsOffset

	if duration.Second <= 0 {
		return "0:00"
	}

	// print the time
	t := AddTimeDuration(duration)
	if t.Day == 0 && t.Hour == 0 {
		return fmt.Sprintf("%02d:%02d", t.Minute, t.Second)
	}
	if t.Day == 0 {
		return fmt.Sprintf("%02d:%02d:%02d", t.Hour, t.Minute, t.Second)
	}
	return fmt.Sprintf("%d:%02d:%02d:%02d", t.Day, t.Hour, t.Minute, t.Second)
}

func YoutubeFind(searchString string, v *VoiceInstance, m *discordgo.MessageCreate) (song_struct PkgSong, err error) {
	service, err := youtube.NewService(context.Background(), option.WithAPIKey(o.YoutubeToken))
	if err != nil {
		log.Printf("Error creating new YouTube client: %v", err)
		return
	}

	var timeOffset string
	if strings.Contains(searchString, "?t=") || strings.Contains(searchString, "&feature=youtu.be&t=") {
		var split []string
		switch {
		case strings.Contains(searchString, "?t="):
			split = strings.Split(searchString, "?t=")
			break

		case strings.Contains(searchString, "&feature=youtu.be&t="):
			split = strings.Split(searchString, "&feature=youtu.be&t=")
			break
		}
		searchString = split[0]
		timeOffset = split[1]

		if !strings.ContainsAny(timeOffset, "h | m | s") {
			timeOffset = timeOffset + "s" // secons
		}
	}

	call := service.Search.List([]string{"id", "snippet"}).Q(searchString).MaxResults(1)
	response, err := call.Do()
	if err != nil {
		log.Printf("error making search API call: %v", err)
		return
	}

	var (
		audioId, audioTitle string
	)

	for _, item := range response.Items {
		audioId = item.Id.VideoId
		audioTitle = item.Snippet.Title
	}
	if audioId == "" {
		ChMessageSend(m.ChannelID, "Sorry, I can't found this song.")
		return
	}

	vid := getVideoInfo(audioId)
	format := vid.Formats.FindByQuality("360p")
	urlStr, err := Client.GetStreamURL(vid, format)
	if err != nil {
		log.Println(err)
		ChMessageSend(m.ChannelID, "Sorry, nothing found for query: "+strings.Trim(searchString, " "))
		return
	}
	videoURL, _ := url.Parse(urlStr)

	videos := service.Videos.List([]string{"contentDetails"}).Id(vid.ID)
	resp, err := videos.Do()

	duration := resp.Items[0].ContentDetails.Duration
	durationString := getDuration(duration, timeOffset)

	var videoURLString string
	if videoURL != nil {
		if timeOffset != "" {
			offset, _ := time.ParseDuration(timeOffset)
			query := videoURL.Query()
			query.Set("begin", fmt.Sprint(int64(offset/time.Millisecond)))
			videoURL.RawQuery = query.Encode()
		}
		videoURLString = videoURL.String()
	} else {
		log.Println("Video URL not found")
	}

	guildID := SearchGuild(m.ChannelID)
	member, _ := v.session.GuildMember(guildID, m.Author.ID)
	name := ""
	if member.Nick == "" {
		name = m.Author.Username
	} else {
		name = member.Nick
	}

	song := Song{
		m.ChannelID,
		name,
		m.Author.ID,
		vid.ID,
		audioTitle,
		durationString,
		videoURLString,
	}

	song_struct.data = song
	song_struct.v = v

	return
}

func getVideoInfo(videoID string) (video *kkdaiYoutube.Video) {
	var (
		err error
	)

	video, err = Client.GetVideo(videoID)
	if err != nil {
		panic(err)
	}

	return
}

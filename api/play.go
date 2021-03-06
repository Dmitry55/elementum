package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/elgatito/elementum/bittorrent"
	"github.com/elgatito/elementum/database"
	"github.com/elgatito/elementum/util"
	"github.com/elgatito/elementum/xbmc"

	"github.com/gin-gonic/gin"
	"github.com/sanity-io/litter"
)

// Download ...
func Download(s *bittorrent.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		rURL, _ := url.Parse(fmt.Sprintf("%s%s", util.GetContextHTTPHost(ctx), strings.Replace(ctx.Request.RequestURI, "/download", "/play", 1)+"&background=true"))
		ctx.Redirect(302, rURL.String())
		return
	}
}

// Play ...
func Play(s *bittorrent.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Index is the file index to automatically select.
		// Starts with 0, -1 or empty means - not defined.
		// OIndex is the original file index to automatically select.
		// 		Order is just like in the torrent file, without changes.
		index := ctx.Query("index")
		oindex := ctx.Query("oindex")

		uri := ctx.Query("uri")
		resume := ctx.Query("resume")
		doresume := ctx.DefaultQuery("doresume", "true")
		query := ctx.Query("query")
		contentType := ctx.Query("type")
		tmdb := ctx.Query("tmdb")
		show := ctx.Query("show")
		season := ctx.Query("season")
		episode := ctx.Query("episode")
		background := ctx.DefaultQuery("background", "false")

		if uri == "" && resume == "" {
			return
		}

		fileIndex := -1
		if index != "" {
			if position, err := strconv.Atoi(index); err == nil && position >= 0 {
				fileIndex = position
			}
		}

		originalIndex := -1
		if oindex != "" {
			if position, err := strconv.Atoi(oindex); err == nil && position >= 0 {
				originalIndex = position
			}
		}

		tmdbID := 0
		if tmdb != "" {
			if id, err := strconv.Atoi(tmdb); err == nil && id > 0 {
				tmdbID = id
			}
		}

		showID := 0
		if show != "" {
			if id, err := strconv.Atoi(show); err == nil && id > 0 {
				showID = id
			}
		}

		seasonNumber := 0
		if season != "" {
			if number, err := strconv.Atoi(season); err == nil && number > 0 {
				seasonNumber = number
			}
		}

		episodeNumber := 0
		if episode != "" {
			if number, err := strconv.Atoi(episode); err == nil && number > 0 {
				episodeNumber = number
			}
		}

		params := bittorrent.PlayerParams{
			URI:            uri,
			OriginalIndex:  originalIndex,
			FileIndex:      fileIndex,
			ResumeHash:     resume,
			ResumePlayback: doresume != "false",
			KodiPosition:   -1,
			ContentType:    contentType,
			TMDBId:         tmdbID,
			ShowID:         showID,
			Season:         seasonNumber,
			Episode:        episodeNumber,
			Query:          query,
			Background:     background == "true",
		}

		player := bittorrent.NewPlayer(s, params)
		log.Infof("Playing item: %s", litter.Sdump(params))
		if t := s.GetTorrentByHash(resume); resume != "" && t != nil {
			player.SetTorrent(t)
		}
		if player.Buffer() != nil || !player.HasChosenFile() {
			player.Close()
			return
		}

		rURL, _ := url.Parse(fmt.Sprintf("%s/files/%s", util.GetContextHTTPHost(ctx), player.PlayURL()))
		ctx.Redirect(302, rURL.String())
	}
}

// PlayTorrent ...
func PlayTorrent(ctx *gin.Context) {
	retval := xbmc.DialogInsert()
	if retval["path"] == "" {
		log.Errorf("No path from insert dialog: %#v", retval)
		return
	}
	xbmc.PlayURLWithTimeout(URLQuery(URLForXBMC("/play"), "uri", retval["path"]))

	ctx.String(200, "")
	return
}

// PlayURI ...
func PlayURI(s *bittorrent.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		uri := ctx.Request.FormValue("uri")
		file, header, fileError := ctx.Request.FormFile("file")

		if file != nil && header != nil && fileError == nil {
			t, err := saveTorrentFile(file, header)
			if err == nil && t != "" {
				uri = t
			}
		}

		index := ctx.Query("index")
		resume := ctx.Query("resume")

		if uri == "" && resume == "" {
			return
		}

		if uri != "" {
			xbmc.PlayURL(URLQuery(URLForXBMC("/play"), "uri", uri, "index", index))
		} else {
			var (
				tmdb        string
				show        string
				season      string
				episode     string
				query       string
				contentType string
			)
			t := s.GetTorrentByHash(resume)

			if t != nil {
				infoHash := t.InfoHash()
				dbItem := database.GetStorm().GetBTItem(infoHash)
				if dbItem != nil && dbItem.Type != "" {
					contentType = dbItem.Type
					if contentType == movieType {
						tmdb = strconv.Itoa(dbItem.ID)
					} else {
						show = strconv.Itoa(dbItem.ShowID)
						season = strconv.Itoa(dbItem.Season)
						episode = strconv.Itoa(dbItem.Episode)
					}
					query = dbItem.Query
				}
			}
			xbmc.PlayURL(URLQuery(URLForXBMC("/play"),
				"resume", resume,
				"index", index,
				"tmdb", tmdb,
				"show", show,
				"season", season,
				"episode", episode,
				"query", query,
				"type", contentType))
		}
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.String(200, "")
	}
}

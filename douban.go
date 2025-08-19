package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"resty.dev/v3"
)

var (
	SubjectCategoryUnknow    SubjectCategory = "unknow"      // 未知
	SubjectCategoryMovieOrTv SubjectCategory = "movie_or_tv" // 电影/剧集
	SubjectCategoryBook      SubjectCategory = "book"        // 书籍
	SubjectCategoryMusic     SubjectCategory = "music"       // 音乐
	SubjectCategoryGame      SubjectCategory = "game"        // 游戏
	UserStatusUnknow         UserStatus      = "unknow"      // 未知
	UserStatusFuture         UserStatus      = "future"      // 想看/想听/想玩
	UserStatusNow            UserStatus      = "now"         // 在看/在听/在玩
	UserStatusPast           UserStatus      = "past"        // 看过/听过/玩过
)

type (
	SubjectCategory string
	UserStatus      string
)

type douban struct {
	SubjectID       int             `json:"subject_id"`
	SubjectTitle    string          `json:"subject_title"`
	SubjectCover    string          `json:"subject_cover"`
	SubjectLink     string          `json:"subject_link"`
	SubjectCategory SubjectCategory `json:"subject_category"`

	UserStatus  UserStatus `json:"user_status"`
	UserRating  int8       `json:"user_rating"`
	UserPubDate int64      `json:"user_pub_date"`

	ExtCoverURL  string `json:"ext_cover_url"`
	ExtCoverHash string `json:"ext_cover_hash"`
}

type doubanXML struct {
	Channel struct {
		Title       string `xml:"title"`       // hyakkiyakou 的收藏
		Link        string `xml:"link"`        // https://www.douban.com/people/157489011/
		Description string `xml:"description"` // hyakkiyakou 的收藏：想看、在看和看过的书和电影，想听、在听和听过的音乐
		PubDate     string `xml:"pubDate"`     // Thu, 03 Apr 2025 19:19:34 GMT
		Items       []*struct {
			Title       string `xml:"title"`       // 在看三国演义
			Link        string `xml:"link"`        // https://movie.douban.com/subject/1830528/
			Description string `xml:"description"` //<table><tr> <td width="80px"><a href="https://movie.douban.com/subject/1830528/" title="三国演义"> <img src="https://img9.doubanio.com/view/photo/s_ratio_poster/public/p2558480666.webp" alt="三国演义"></a></td> <td> </td></tr></table>
			PubDate     string `xml:"pubDate"`     // Thu, 27 Mar 2025 05:23:24 GMT
		} `xml:"item"`
	} `xml:"channel"`
}

func doubanFetch() ([]*douban, error) {
	var result []*douban

	rssURL := fmt.Sprintf("https://www.douban.com/feed/people/%s/interests", C.DoubanID)
	client := resty.New()
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Error closing resty client: %v", err)
		}
	}()

	raw := &doubanXML{}
	if _, err := client.R().
		SetTimeout(C.Timeout).
		SetRetryCount(3).
		SetHeader("Accept", "application/xml").
		SetHeader("User-Agent", C.UserAgent).
		SetResult(raw).
		Get(rssURL); err != nil {
		return nil, err
	}
	if raw.Channel.Items == nil {
		return nil, fmt.Errorf("RSS channel items are missing, URL: %s", rssURL)
	}

	for _, v := range raw.Channel.Items {
		var subjectID int
		var subjectTitle string
		var subjectCover string
		var subjectLink string
		var subjectCategory SubjectCategory
		var userStatus UserStatus
		var userRating int8
		var userPubDate int64

		titleRunes := []rune(v.Title)
		action := string(titleRunes[0:2])

		// ID
		if match := regexp.MustCompile(`subject/(\d+)`).FindStringSubmatch(v.Description); len(match) > 1 {
			sid, err := strconv.Atoi(match[1])
			if err != nil {
				return nil, err
			}
			subjectID = sid
		}

		// Title
		subjectTitle = string(titleRunes[2:])

		// Cover
		if match := regexp.MustCompile(`src="(https://[^"]+\.(jpg|png|webp))"`).FindStringSubmatch(v.Description); len(match) > 1 {
			subjectCover = match[1]
		}

		// Link
		subjectLink = v.Link

		// Category
		subjectCategory = SubjectCategoryUnknow
		{
			switch {
			case strings.Contains(action, "看"):
				subjectCategory = SubjectCategoryMovieOrTv
			case strings.Contains(action, "读"):
				subjectCategory = SubjectCategoryBook
			case strings.Contains(action, "听"):
				subjectCategory = SubjectCategoryMusic
			case strings.Contains(action, "玩"):
				subjectCategory = SubjectCategoryGame
			}
		}

		// Status
		userStatus = UserStatusUnknow
		{
			switch {
			case strings.Contains(action, "想"):
				userStatus = UserStatusFuture
			case strings.Contains(action, "在"):
				userStatus = UserStatusNow
			case strings.Contains(action, "过"):
				userStatus = UserStatusPast
			}
		}

		// Rating
		userRating = int8(0)
		{
			switch {
			case strings.Contains(v.Description, "推荐: 很差"):
				userRating = 1
			case strings.Contains(v.Description, "推荐: 较差"):
				userRating = 2
			case strings.Contains(v.Description, "推荐: 还行"):
				userRating = 3
			case strings.Contains(v.Description, "推荐: 推荐"):
				userRating = 4
			case strings.Contains(v.Description, "推荐: 力荐"):
				userRating = 5
			}
		}

		// PubDate
		userPubDateTime, err := time.Parse(time.RFC1123, v.PubDate)
		if err != nil {
			return nil, err
		}
		userPubDate = userPubDateTime.Unix()

		result = append(result, &douban{
			SubjectID:       subjectID,
			SubjectTitle:    subjectTitle,
			SubjectCover:    subjectCover,
			SubjectLink:     subjectLink,
			SubjectCategory: subjectCategory,
			UserStatus:      userStatus,
			UserRating:      userRating,
			UserPubDate:     userPubDate,
		})
	}

	return result, nil
}

func doubanSave(items []*douban) error {
	same, err := doubanCompare(items)
	if err != nil {
		return err
	}
	if same {
		log.Println("File is up to date; skipping.")
		return nil
	}

	if err = fileSave(items, filepath.Join(C.SaveDir, C.IndexFileName)); err != nil {
		return err
	}
	if err = doubanSaveCover(items); err != nil {
		return err
	}

	return doubanUpdateExt(items)
}

func doubanSaveCover(items []*douban) error {
	for _, i := range items {
		if err := fileDownload(C.SaveDir, i.SubjectCover); err != nil {
			return err
		}
	}
	return nil
}

func doubanCleanup() error {
	items, err := doubanLoad()
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	fs, err := os.ReadDir(C.SaveDir)
	if err != nil {
		return err
	}
	for _, f := range fs {
		if f.Name() == C.IndexFileName {
			continue
		}
		ok := false
		for _, i := range items {
			if f.Name() == path.Base(i.SubjectCover) {
				ok = true
				break
			}
		}
		if !ok {
			log.Println("Removing file ", f.Name())
			if err = os.Remove(filepath.Join(C.SaveDir, f.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

func doubanUpdateExt(items []*douban) error {
	fs, err := os.ReadDir(C.SaveDir)
	if err != nil {
		return err
	}
	for _, f := range fs {
		if f.Name() == C.IndexFileName {
			continue
		}
		for k, v := range items {
			if f.Name() == path.Base(v.SubjectCover) {
				link, err := url.JoinPath(C.PublicURL, f.Name())
				if err != nil {
					return err
				}
				items[k].ExtCoverURL = link
				hash, err := fileMD5(filepath.Join(C.SaveDir, f.Name()))
				if err != nil {
					return err
				}
				items[k].ExtCoverHash = hash
			}
		}
	}

	return fileSave(items, filepath.Join(C.SaveDir, C.IndexFileName))
}

func doubanCompare(new []*douban) (bool, error) {
	old, err := doubanLoad()
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Local index file %s not found", filepath.Join(C.SaveDir, C.IndexFileName))
			return false, nil
		}
		return false, err
	}

	if len(new) != len(old) {
		return false, nil
	}

	for i := range old {
		old[i].ExtCoverURL = ""
		old[i].ExtCoverHash = ""
		if !reflect.DeepEqual(*old[i], *new[i]) {
			return false, nil
		}
	}

	return true, nil
}

func doubanLoad() ([]*douban, error) {
	var result []*douban

	f, err := os.Open(filepath.Join(C.SaveDir, C.IndexFileName))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Println("Error closing file:", err)
		}
	}()

	d := json.NewDecoder(f)
	err = d.Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

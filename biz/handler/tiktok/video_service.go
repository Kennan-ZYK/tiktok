package tiktok

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"simple-tiktok/biz/dal/db"
	tiktok "simple-tiktok/biz/model/tiktok"
	"simple-tiktok/pkg/consts"
	"simple-tiktok/pkg/errno"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/disintegration/imaging"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

// Feed .
// @router /douyin/feed/ [GET]
func Feed(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tiktok.FeedRequest
	err = c.BindAndValidate(&req)
	if err != nil {
		log.Printf("参数BindAndValidate失败: %v\n", err.Error())
		c.JSON(http.StatusBadRequest, tiktok.FeedResponse{
			StatusCode: errno.ParamErr.ErrCode,
			StatusMsg:  &errno.ParamErr.ErrMsg,
		})
		return
	}
	user := c.Value(consts.IdentityKeyID).(*tiktok.User)
	resp := new(tiktok.FeedResponse)

	var t int64

	resp.VideoList, t, err = db.GetFeedVideo(ctx, *req.LatestTime, user.ID)
	if err != nil {
		log.Printf("获取最新视频失败: %v\n", err.Error())
		c.JSON(http.StatusBadRequest, tiktok.FeedResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return
	}
	log.Println(*resp.VideoList[0])
	resp.StatusCode = errno.Success.ErrCode
	resp.StatusMsg = &errno.Success.ErrMsg
	tmp := t
	resp.NextTime = &tmp

	c.JSON(http.StatusOK, resp)
}

// UploadVideo .
// @router /douyin/publish/action/ [POST]
func UploadVideo(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tiktok.UploadVideoRequest
	err = c.BindAndValidate(&req)
	if err != nil {
		log.Printf("参数BindAndValidate失败: %v\n", err.Error())
		c.JSON(http.StatusBadRequest, tiktok.UploadVideoResponse{
			StatusCode: errno.ParamErr.ErrCode,
			StatusMsg:  &errno.ParamErr.ErrMsg,
		})
		return
	}

	fileHeader, err := c.FormFile("data")
	if err != nil {
		log.Printf("参数FormFile失败: %v\n", err.Error())
		c.JSON(http.StatusBadRequest, tiktok.UploadVideoResponse{
			StatusCode: errno.ParamErr.ErrCode,
			StatusMsg:  &errno.ParamErr.ErrMsg,
		})
		return
	}

	tmp := strings.Split(fileHeader.Filename, ".")
	user := c.Value(consts.IdentityKeyID).(*tiktok.User)
	fileName, err := db.CreateVideoAndGetId(ctx, req.Title, tmp[len(tmp)-1], user.ID)
	if err != nil {
		log.Printf("获取视频id失败: %v\n", err.Error())
		c.JSON(http.StatusBadRequest, tiktok.UploadVideoResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return
	}
	SaveVideo(c, fileHeader, fileName, tmp[len(tmp)-1], true)

	c.JSON(http.StatusOK, tiktok.UploadVideoResponse{
		StatusCode: errno.Success.ErrCode,
		StatusMsg:  &errno.Success.ErrMsg,
	})
}

func SaveVideo(c *app.RequestContext, fileHeader *multipart.FileHeader, fileName string, tp string, SaveCover bool) {
	Name := fileName + "." + tp
	videoPath := fmt.Sprintf("./file/upload/video/%s", Name)
	c.SaveUploadedFile(fileHeader, videoPath)
	if SaveCover {
		time.Sleep(1 * time.Second)
		err := SaveSnapshot(c, videoPath)
		if err != nil {
			log.Printf("保存封面失败: %v\n", err.Error())
			c.JSON(http.StatusBadRequest, tiktok.UploadVideoResponse{
				StatusCode: errno.ServiceErr.ErrCode,
				StatusMsg:  &errno.ServiceErr.ErrMsg,
			})
			return
		}
	}
}

func SaveSnapshot(c *app.RequestContext, videoPath string) (err error) {
	snapshotPath := "." + strings.Split(videoPath, ".")[1]
	snapshotPath = strings.Replace(snapshotPath, "video", "img", 1)
	buf := bytes.NewBuffer(nil)
	err = ffmpeg.Input(videoPath).Filter("select", ffmpeg.Args{fmt.Sprintf("gte(n,%d)", 1)}).
		Output("pipe:", ffmpeg.KwArgs{"vframes": 1, "format": "image2", "vcodec": "mjpeg"}).
		WithOutput(buf, os.Stdout).
		Run()

	if err != nil {
		log.Printf("生成缩略图失败: %v\n", err.Error())
		c.JSON(http.StatusBadRequest, tiktok.UploadVideoResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return err
	}

	img, err := imaging.Decode(buf)
	if err != nil {
		log.Printf("生成缩略图失败: %v\n", err.Error())
		c.JSON(http.StatusBadRequest, tiktok.UploadVideoResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return err
	}

	err = imaging.Save(img, snapshotPath+".jpg")
	if err != nil {
		log.Printf("保存缩略图失败: %v\n", err.Error())
		c.JSON(http.StatusBadRequest, tiktok.UploadVideoResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return err
	}

	return nil
}

// GetPublishList .
// @router /douyin/publish/list/ [GET]
func GetPublishList(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tiktok.GetPublishRequest
	err = c.BindAndValidate(&req)
	if err != nil {
		log.Printf("参数BindAndValidate失败: %v\n", err.Error())
		c.JSON(http.StatusBadRequest, tiktok.GetPublishResponse{
			StatusCode: errno.ParamErr.ErrCode,
			StatusMsg:  &errno.ParamErr.ErrMsg,
		})
		return
	}

	resp := new(tiktok.GetPublishResponse)

	user_id, err := strconv.ParseUint(req.UserID, 10, 32)
	if err != nil {
		log.Printf("user_id转化失败: %v\n", err.Error())
		c.JSON(http.StatusBadRequest, tiktok.GetPublishResponse{
			StatusCode: errno.ParamErr.ErrCode,
			StatusMsg:  &errno.ParamErr.ErrMsg,
		})
		return
	}
	resp.VideoList, _ = db.GetPublishList(ctx, uint(user_id))
	resp.StatusCode = errno.Success.ErrCode
	resp.StatusMsg = &errno.Success.ErrMsg

	c.JSON(http.StatusOK, resp)
}

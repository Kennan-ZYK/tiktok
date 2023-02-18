// Code generated by hertz generator.

package tiktok

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"

	"simple-tiktok/biz/dal/db"
	tiktok "simple-tiktok/biz/model/tiktok"
	"simple-tiktok/biz/mw"
	"simple-tiktok/pkg/consts"
	"simple-tiktok/pkg/errno"

	"github.com/cloudwego/hertz/pkg/app"
)

// CreateUser .
// @router /douyin/user/register/ [POST]
func CreateUser(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tiktok.CreateUserRequest
	err = c.BindAndValidate(&req)
	if err != nil {
		log.Printf("参数BindAndValidate失败: %v\n", err.Error())
		c.JSON(http.StatusBadRequest, tiktok.CreateUserResponse{
			StatusCode: errno.ParamErr.ErrCode,
			StatusMsg:  &errno.ParamErr.ErrMsg,
		})
		return
	}

	users, err := db.QueryUser(ctx, req.Username)
	if err != nil {
		log.Printf("查询用户失败: %v\n", err.Error())
		c.JSON(http.StatusInternalServerError, tiktok.CreateUserResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return
	}
	if len(users) != 0 {
		c.JSON(http.StatusBadRequest, tiktok.CreateUserResponse{
			StatusCode: errno.UserAlreadyExistErr.ErrCode,
			StatusMsg:  &errno.UserAlreadyExistErr.ErrMsg,
		})
		return
	}

	h := md5.New()
	if _, err = io.WriteString(h, req.Password); err != nil {
		log.Printf("md5加密错误: %v\n", err.Error())
		c.JSON(http.StatusInternalServerError, tiktok.CreateUserResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return
	}

	password := fmt.Sprintf("%x", h.Sum(nil))
	_, err = db.CreateUser(ctx, req.Username, password)
	if err != nil {
		log.Printf("创建用户失败: %v\n", err.Error())
		c.JSON(http.StatusInternalServerError, tiktok.CreateUserResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return
	}

	mw.JwtMiddleware.LoginHandler(ctx, c)
}

// CheckUser .
// @router /douyin/user/login/ [POST]
func CheckUser(ctx context.Context, c *app.RequestContext) {
	mw.JwtMiddleware.LoginHandler(ctx, c)
}

// GetUser .
// @router /douyin/user/ [GET]
func GetUser(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tiktok.GetUserRequest
	err = c.BindAndValidate(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, tiktok.GetUserResponse{
			StatusCode: errno.ParamErr.ErrCode,
			StatusMsg:  &errno.ParamErr.ErrMsg,
		})
		return
	}

	followers, err := db.QueryFollower(ctx, uint(req.UserID))
	if err != nil {
		log.Printf("查询用户: %d粉丝失败: %v\n", req.UserID, err.Error())
		c.JSON(http.StatusInternalServerError, tiktok.GetUserResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return
	}

	follows, err := db.QueryFollow(ctx, uint(req.UserID))
	if err != nil {
		log.Printf("查询用户: %d关注失败: %v\n", req.UserID, err.Error())
		c.JSON(http.StatusInternalServerError, tiktok.GetUserResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return
	}

	user := c.Value(consts.IdentityKeyID).(*tiktok.User)
	followersCount := int64(len(followers))
	followsCount := int64(len(follows))
	c.JSON(http.StatusOK, tiktok.GetUserResponse{
		StatusCode: errno.Success.ErrCode,
		StatusMsg:  &errno.Success.ErrMsg,
		User: &tiktok.User{
			ID:            req.UserID,
			Name:          user.Name,
			FollowCount:   &followsCount,
			FollowerCount: &followersCount,
			IsFollow:      true,
		},
	})
}

// FollowUser .
// @router /douyin/relation/action/ [POST]
func FollowUser(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tiktok.FollowUserRequest
	err = c.BindAndValidate(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, tiktok.FollowUserResponse{
			StatusCode: errno.ParamErr.ErrCode,
			StatusMsg:  &errno.ParamErr.ErrMsg,
		})
		return
	}

	users, err := db.MGetUsers(ctx, []int64{
		req.ToUserID,
	})
	if err != nil {
		log.Printf("查询用户: %v失败: %v", req.ToUserID, err.Error())
		c.JSON(http.StatusInternalServerError, tiktok.FollowUserResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return
	}
	if len(users) != 1 {
		log.Printf("关注/取关用户: %v不存在", req.ToUserID)
		c.JSON(http.StatusOK, tiktok.FollowUserResponse{
			StatusCode: errno.UserNotExistErr.ErrCode,
			StatusMsg:  &errno.UserNotExistErr.ErrMsg,
		})
		return
	}

	user := c.Value(consts.IdentityKeyID).(*tiktok.User)
	var u1ID, u1Name = uint(user.ID), user.Name
	var u2ID, u2Name = users[0].ID, users[0].Username
	switch req.ActionType {
	case consts.FollowUser:
		err := db.FollowUser(ctx, u1ID, u1Name, u2ID, u2Name)
		if err != nil {
			log.Printf("用户: %d 关注用户: %d失败: %v\n", user.ID, req.ToUserID, err.Error())
			c.JSON(http.StatusInternalServerError, tiktok.FollowUserResponse{
				StatusCode: errno.ServiceErr.ErrCode,
				StatusMsg:  &errno.ServiceErr.ErrMsg,
			})
			return
		}
	case consts.UnFollowUser:
		err := db.UnFollowUser(ctx, u1ID, u2ID)
		if err != nil {
			log.Printf("用户: %d 取消关注用户: %d失败: %v\n", user.ID, req.ToUserID, err.Error())
			c.JSON(http.StatusInternalServerError, tiktok.FollowUserResponse{
				StatusCode: errno.ServiceErr.ErrCode,
				StatusMsg:  &errno.ServiceErr.ErrMsg,
			})
			return
		}
	}

	c.JSON(http.StatusOK, tiktok.FollowUserResponse{
		StatusCode: errno.Success.ErrCode,
		StatusMsg:  &errno.Success.ErrMsg,
	})
}

// GetFollow .
// @router /douyin/relation/follow/list/ [GET]
func GetFollow(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tiktok.GetFollowRequest
	err = c.BindAndValidate(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, tiktok.FollowUserResponse{
			StatusCode: errno.ParamErr.ErrCode,
			StatusMsg:  &errno.ParamErr.ErrMsg,
		})
		return
	}

	users, err := db.QueryFollow(ctx, uint(req.UserID))
	if err != nil {
		log.Printf("查询用户: %v的关注失败: %v\n", req.UserID, err.Error())
		c.JSON(http.StatusInternalServerError, tiktok.FollowUserResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return
	}

	n := len(users)
	follows := make([]*tiktok.User, n)
	for i := 0; i < n; i++ {
		follows[i] = &tiktok.User{
			ID:       int64(users[i].UserID),
			Name:     users[i].Username,
			IsFollow: true,
		}
	}
	// TODO: 缓存到redis中，避免重复查询

	c.JSON(http.StatusOK, tiktok.GetFollowResponse{
		StatusCode: errno.Success.ErrCode,
		StatusMsg:  &errno.Success.ErrMsg,
		UserList:   follows,
	})
}

// GetFollower .
// @router /douyin/relation/follower/list/ [GET]
func GetFollower(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tiktok.GetFollowerRequest
	err = c.BindAndValidate(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, tiktok.FollowUserResponse{
			StatusCode: errno.ParamErr.ErrCode,
			StatusMsg:  &errno.ParamErr.ErrMsg,
		})
		return
	}

	users, err := db.QueryFollower(ctx, uint(req.UserID))
	if err != nil {
		log.Printf("查询用户: %v的粉丝失败: %v\n", req.UserID, err.Error())
		c.JSON(http.StatusInternalServerError, tiktok.FollowUserResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return
	}

	n := len(users)
	followers := make([]*tiktok.User, n)
	m := make(map[int64]*tiktok.User, n)
	// 所有粉丝的uid列表
	uids := make([]uint, n)
	for i := 0; i < n; i++ {
		followers[i] = &tiktok.User{
			ID:   int64(users[i].FollowerID),
			Name: users[i].FollowerName,
		}
		m[followers[i].ID] = followers[i]
		uids[i] = users[i].FollowerID
	}

	uid := c.Value(consts.IdentityKeyID).(*tiktok.User).ID
	// 查询粉丝列表中哪些用户被当前用户关注了
	uids, err = db.MGetFollow(ctx, uint(uid), uids)
	if err != nil {
		log.Printf("查询用户: %v的粉丝中已关注用户失败: %v\n", uid, err.Error())
		c.JSON(http.StatusInternalServerError, tiktok.FollowUserResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return
	}

	for _, uid := range uids {
		m[int64(uid)].IsFollow = true
	}

	// TODO: 缓存到redis中，避免重复查询

	c.JSON(http.StatusOK, tiktok.GetFollowerResponse{
		StatusCode: errno.Success.ErrCode,
		StatusMsg:  &errno.Success.ErrMsg,
		UserList:   followers,
	})
}

// GetFriend .
// @router /douyin/relation/friend/list/ [GET]
func GetFriend(ctx context.Context, c *app.RequestContext) {
	var err error
	var req tiktok.GetFriendRequest
	err = c.BindAndValidate(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, tiktok.GetFriendResponse{
			StatusCode: errno.ParamErr.ErrCode,
			StatusMsg:  &errno.ParamErr.ErrMsg,
		})
		return
	}

	users, err := db.QueryFollower(ctx, uint(req.UserID))
	if err != nil {
		log.Printf("查询用户: %v的粉丝失败: %v\n", req.UserID, err.Error())
		c.JSON(http.StatusInternalServerError, tiktok.GetFriendResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return
	}

	n := len(users)
	m := make(map[uint]*tiktok.FriendUser, n)
	// 所有粉丝的uid列表
	uids := make([]uint, n)
	for i := 0; i < n; i++ {
		user := users[i]
		m[user.FollowerID] = &tiktok.FriendUser{
			ID:     int64(user.FollowerID),
			Name:   user.FollowerName,
			Avatar: "https://simple-tiktok-1300912551.cos.ap-guangzhou.myqcloud.com/avatar.jpg",
		}
		uids[i] = user.FollowerID
	}

	uid := c.Value(consts.IdentityKeyID).(*tiktok.User).ID
	// 查询粉丝列表中哪些用户被当前用户关注了
	uids, err = db.MGetFollow(ctx, uint(uid), uids)
	if err != nil {
		log.Printf("查询用户: %v的粉丝中已关注用户失败: %v\n", uid, err.Error())
		c.JSON(http.StatusInternalServerError, tiktok.GetFriendResponse{
			StatusCode: errno.ServiceErr.ErrCode,
			StatusMsg:  &errno.ServiceErr.ErrMsg,
		})
		return
	}

	friends := make([]*tiktok.FriendUser, 0, len(uids))
	for _, uid := range uids {
		friend := m[uid]
		friend.IsFollow = true
		friends = append(friends, friend)
	}

	u1 := uint(uid)
	// 查询与各个好友之间的最新消息
	for _, u2 := range uids {
		msg, err := db.GetMessage(ctx, u1, u2)
		if err != nil {
			log.Printf("查询用户: %v的与用户: %v之间的最新消息失败: %v\n", u1, u2, err.Error())
			continue
		}
		friend := m[u2]
		friend.Message = &msg
	}

	c.JSON(http.StatusOK, tiktok.GetFriendResponse{
		StatusCode: errno.Success.ErrCode,
		StatusMsg:  &errno.Success.ErrMsg,
		UserList:   friends,
	})
}

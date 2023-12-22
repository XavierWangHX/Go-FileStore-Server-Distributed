package handler

import (
	"FileStore/db"
	"FileStore/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

const pwd_salt = "*#890"

func SigninGetHandler(c *gin.Context) {
	c.Redirect(http.StatusFound, "/static/view/signin.html")
}
func SigninPostHandler(c *gin.Context) {
	username := c.Request.FormValue("username")
	passwd := c.Request.FormValue("password")
	enc_passwd := util.Sha1([]byte(passwd + pwd_salt))
	passwd_check := db.UserSignin(username, enc_passwd)
	if !passwd_check {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Signin Failed!",
			"code": -1,
		})
		return
	}
	token := GenToken(username)
	res := db.UpdateToken(username, token)
	if !res {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Update Token Failed!",
			"code": -2,
		})
		return
	}
	resp := util.RespMsg{
		Code: 0,
		Msg:  "OK",
		Data: struct {
			Location string
			Username string
			Token    string
		}{
			Location: "/static/view/home.html",
			Username: username,
			Token:    token,
		},
	}
	c.Data(http.StatusOK, "application/json", resp.JSONBytes())
}

func UserInfoHandler(c *gin.Context) {

	username := c.Request.FormValue("username")
	user, err := db.GetUserInfo(username)
	if err != nil {
		fmt.Println("Failed to get user info")
		c.JSON(http.StatusForbidden, gin.H{
			"msg":  "Get User Info Error",
			"code": -1,
		})
		return
	}
	resp := util.RespMsg{
		Code: 0,
		Msg:  "OK",
		Data: user,
	}
	c.Data(http.StatusOK, "application/json", resp.JSONBytes())

}

func SignupGetHandler(c *gin.Context) {
	c.Redirect(http.StatusFound, "/static/view/signup.html")
}
func SignupPostHandler(c *gin.Context) {

	username := c.Request.FormValue("username")
	passwd := c.Request.FormValue("password")
	if len(username) < 3 || len(passwd) < 5 {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Username or Password is too short, please retype again!",
			"code": -1,
		})
		return
	}
	enc_pwd := util.Sha1([]byte(passwd + pwd_salt))
	suc := db.UserSignUp(username, enc_pwd)
	if suc {
		resp := util.RespMsg{
			Code: 0,
			Msg:  "OK",
			Data: "/user/signin",
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
	} else {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "Signup Failed!",
			"code": -2,
		})
	}

}
func GenToken(username string) string {
	// 40位字符:md5(username+timestamp+token_salt)+timestamp[:8]
	ts := fmt.Sprintf("%x", time.Now().Unix())
	tokenPrefix := util.MD5([]byte(username + ts + "_tokensalt"))
	return tokenPrefix + ts[:8]
}

func IsTokenValid(token string) bool {
	if len(token) != 40 {
		return false
	}
	return true
}
